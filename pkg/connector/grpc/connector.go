package grpc

import (
	"context"
	. "github.com/robaho/fixed"
	"github.com/robaho/go-trader/conf"
	"github.com/robaho/go-trader/entity"
	. "github.com/robaho/go-trader/pkg/common"
	"github.com/robaho/go-trader/pkg/constant"
	"github.com/robaho/go-trader/pkg/protocol"
	"google.golang.org/grpc"
	"io"
	"log"
	"os"
	"strings"
	"sync"
)

type grpcConnector struct {
	connected bool
	callback  ConnectorCallback
	nextOrder int64
	nextQuote int64
	// holds OrderID->*Order, concurrent since notifications/updates may arrive while order is being processed
	orders   sync.Map
	conn     *grpc.ClientConn
	stream   protocol.Exchange_ConnectionClient
	loggedIn StatusBool
	// true after all instruments are downloaded from exchange
	downloaded StatusBool
	//props      Properties
	log io.Writer
}

func (c *grpcConnector) IsConnected() bool {
	return c.connected
}

func (c *grpcConnector) Connect() error {
	if c.connected {
		return AlreadyConnected
	}

	addr := conf.AppConfig.GrpcHost + ":" + conf.AppConfig.GrpcPort

	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		return err
	}

	c.conn = conn

	client := protocol.NewExchangeClient(conn)

	//timeoutSecs := time.Second * time.Duration(timeout)

	ctx := context.Background()
	stream, err := client.Connection(ctx)

	if err != nil {
		conn.Close()
		return err
	}

	c.stream = stream

	log.Println("connection to exchange OK, sending login")

	request := &protocol.InMessage_Login{Login: &protocol.LoginRequest{Username: "guest", Password: "guest"}}

	err = stream.Send(&protocol.InMessage{Request: request})
	if err != nil {
		return err
	}

	go func() {
		for {
			msg, err := stream.Recv()
			if err != nil {
				log.Println("unable to receive message", err)
				c.Disconnect()
				return
			}

			switch msg.GetReply().(type) {
			case *protocol.OutMessage_Login:
				response := msg.GetReply().(*protocol.OutMessage_Login).Login
				if response.Error != "" {
					log.Println("unable to login", response.Error)
				} else {
					c.loggedIn.SetTrue()
				}
			case *protocol.OutMessage_Reject:
				response := msg.GetReply().(*protocol.OutMessage_Reject).Reject
				if response.Error != "" {
					log.Println("request rejected", response.Error)
				}
			case *protocol.OutMessage_Secdef:
				sec := msg.GetReply().(*protocol.OutMessage_Secdef).Secdef
				if sec.InstrumentID == 0 { // end of instrument download
					c.downloaded.SetTrue()
					continue
				}

				instrument := entity.NewInstrument(int64(sec.InstrumentID), sec.Symbol)

				conf.IMap.Put(instrument)

				c.callback.OnInstrument(instrument)
			case *protocol.OutMessage_Execrpt:
				rpt := msg.GetReply().(*protocol.OutMessage_Execrpt).Execrpt
				c.handleExecutionReport(rpt)
			}
		}
	}()

	// wait for login up to 30 seconds
	if !c.loggedIn.WaitForTrue(30 * 1000) {
		return ConnectionFailed
	}

	log.Println("login OK")

	c.connected = true

	return nil
}

func (c *grpcConnector) Disconnect() error {
	if !c.connected {
		return NotConnected
	}
	c.conn.Close()
	c.connected = false
	c.loggedIn.SetFalse()
	return nil
}

func (c *grpcConnector) CreateInstrument(symbol string) {

	request := &protocol.InMessage_Secdefreq{Secdefreq: &protocol.SecurityDefinitionRequest{Symbol: symbol}}

	err := c.stream.Send(&protocol.InMessage{Request: request})
	if err != nil {
		log.Println("unable to send SecurityDefinitionRequest", err)
	}
}

func (c *grpcConnector) DownloadInstruments() error {
	if !c.loggedIn.IsTrue() {
		return NotConnected
	}

	c.downloaded.SetFalse()

	request := &protocol.InMessage_Download{Download: &protocol.DownloadRequest{}}

	err := c.stream.Send(&protocol.InMessage{Request: request})
	if err != nil {
		log.Println("unable to send DownloadRequest", err)
	}

	// wait for login up to 30 seconds
	if !c.downloaded.WaitForTrue(30 * 1000) {
		return DownloadFailed
	}
	return nil
}

func (c *grpcConnector) CreateOrder(order *entity.Order) (entity.OrderID, error) {
	if !c.loggedIn.IsTrue() {
		return -1, NotConnected
	}

	if order.OrderType != constant.Limit && order.OrderType != constant.Market {
		return -1, UnsupportedOrderType
	}

	c.nextOrder = c.nextOrder + 1

	var orderID = entity.OrderID(c.nextOrder)
	order.Id = orderID
	c.orders.Store(orderID, order)

	co := protocol.CreateOrderRequest{}
	co.ClOrdId = int32(orderID)
	co.Symbol = order.Symbol()
	co.Price = ToFloat(order.Price)
	co.Quantity = ToFloat(order.Quantity)
	switch order.OrderType {
	case constant.Market:
		co.OrderType = protocol.CreateOrderRequest_Market
	case constant.Limit:
		co.OrderType = protocol.CreateOrderRequest_Limit
	}
	switch order.Side {
	case constant.Buy:
		co.OrderSide = protocol.CreateOrderRequest_Buy
	case constant.Sell:
		co.OrderSide = protocol.CreateOrderRequest_Sell
	}

	request := &protocol.InMessage_Create{Create: &co}
	err := c.stream.Send(&protocol.InMessage{Request: request})
	return orderID, err
}

func (c *grpcConnector) ModifyOrder(id entity.OrderID, price Fixed, quantity Fixed) error {
	if !c.loggedIn.IsTrue() {
		return NotConnected
	}
	order := c.GetOrder(id)
	if order == nil {
		return OrderNotFound
	}
	order.Lock()
	defer order.Unlock()

	order.Price = price
	order.Quantity = quantity

	co := protocol.ModifyOrderRequest{}
	co.ClOrdId = int32(order.Id)
	co.Price = ToFloat(order.Price)
	co.Quantity = ToFloat(order.Quantity)

	request := &protocol.InMessage_Modify{Modify: &co}
	err := c.stream.Send(&protocol.InMessage{Request: request})
	return err
}

func (c *grpcConnector) CancelOrder(id entity.OrderID) error {
	if !c.loggedIn.IsTrue() {
		return NotConnected
	}
	order := c.GetOrder(id)
	if order == nil {
		return OrderNotFound
	}
	order.Lock()
	defer order.Unlock()

	co := protocol.CancelOrderRequest{}
	co.ClOrdId = int32(order.Id)

	request := &protocol.InMessage_Cancel{Cancel: &co}
	err := c.stream.Send(&protocol.InMessage{Request: request})
	return err
}

func (c *grpcConnector) Quote(instrument entity.Instrument, bidPrice Fixed, bidQuantity Fixed, askPrice Fixed, askQuantity Fixed) error {

	if !c.loggedIn.IsTrue() {
		return NotConnected
	}

	c.nextQuote += 1

	request := &protocol.InMessage_Massquote{Massquote: &protocol.MassQuoteRequest{
		Symbol:   instrument.Symbol(),
		BidPrice: ToFloat(bidPrice), BidQuantity: ToFloat(bidQuantity),
		AskPrice: ToFloat(askPrice), AskQuantity: ToFloat(askQuantity)}}

	err := c.stream.Send(&protocol.InMessage{Request: request})
	if err != nil {
		log.Println("unable to send MassQuote", err)
	}

	return err
}

func (c *grpcConnector) GetExchangeCode() string {
	return "GOT"
}
func (c *grpcConnector) GetOrder(id entity.OrderID) *entity.Order {
	_order, ok := c.orders.Load(id)
	if !ok {
		return nil
	}
	return _order.(*entity.Order)
}

func (c *grpcConnector) handleExecutionReport(rpt *protocol.ExecutionReport) {
	exchangeId := rpt.ExOrdId
	var id entity.OrderID
	var order *entity.Order
	if strings.HasPrefix(exchangeId, "quote.") {
		// quote fill
		id = entity.OrderID(0)
	} else {
		id = entity.OrderID(int(rpt.ClOrdId))
		order = c.GetOrder(id)
		if order == nil {
			log.Println("unknown order ", id)
			return
		}
	}

	instrument := conf.IMap.GetBySymbol(rpt.Symbol)
	if instrument == nil {
		log.Println("unknown symbol in execution report ", rpt.Symbol)
	}

	var state constant.OrderState

	switch rpt.OrderState {
	case protocol.ExecutionReport_Booked:
		state = constant.Booked
	case protocol.ExecutionReport_Cancelled:
		state = constant.Cancelled
	case protocol.ExecutionReport_Partial:
		state = constant.PartialFill
	case protocol.ExecutionReport_Filled:
		state = constant.Filled
	case protocol.ExecutionReport_Rejected:
		state = constant.Rejected
	}

	if order != nil {
		order.Lock()
		defer order.Unlock()

		order.ExchangeId = exchangeId
		order.Remaining = NewDecimalF(rpt.Remaining)
		order.Price = NewDecimalF(rpt.Price)
		order.Quantity = NewDecimalF(rpt.Quantity)

		order.OrderState = state
	}

	if rpt.ReportType == protocol.ExecutionReport_Fill {
		lastPx := NewF(rpt.LastPrice)
		lastQty := NewF(rpt.LastQuantity)

		var side constant.Side
		if rpt.Side == protocol.CreateOrderRequest_Buy {
			side = constant.Buy
		} else {
			side = constant.Sell
		}

		fill := &Fill{instrument, id == 0, order, exchangeId, lastQty, lastPx, side, false}
		c.callback.OnFill(fill)
	}

	if order != nil {
		c.callback.OnOrderStatus(order)
	}

}

func NewConnector(callback ConnectorCallback, logOutput io.Writer) ExchangeConnector {
	if logOutput == nil {
		logOutput = os.Stdout
	}
	c := &grpcConnector{log: logOutput}
	c.callback = callback

	return c
}
