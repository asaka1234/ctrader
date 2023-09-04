package exchange

import (
	"fmt"
	. "github.com/robaho/fixed"
	"logtech.com/exchange/ltrader/conf"
	"logtech.com/exchange/ltrader/entity"
	"logtech.com/exchange/ltrader/pkg/constant"
	"log"
	"strconv"

	"github.com/pkg/errors"
	. "logtech.com/exchange/ltrader/pkg/common"
	"logtech.com/exchange/ltrader/pkg/protocol"
)

//----------负责对外提供grpc接口----------------------------------

type grpcServer struct {
	e *exchange
}

type grpcClient struct {
	conn     protocol.Exchange_ConnectionServer
	loggedIn bool
	user     string
}

func (c *grpcClient) SendOrderStatus(so sessionOrder) {
	rpt := &protocol.ExecutionReport{}
	rpt.Symbol = so.order.Symbol()
	rpt.ExOrdId = so.order.ExchangeId
	rpt.ReportType = protocol.ExecutionReport_Status
	switch so.order.OrderState {
	case constant.New, constant.Booked:
		rpt.OrderState = protocol.ExecutionReport_Booked
	case constant.PartialFill:
		rpt.OrderState = protocol.ExecutionReport_Partial
	case constant.Filled:
		rpt.OrderState = protocol.ExecutionReport_Filled
	case constant.Cancelled:
		rpt.OrderState = protocol.ExecutionReport_Cancelled
	case constant.Rejected:
		rpt.OrderState = protocol.ExecutionReport_Rejected
	}
	rpt.RejectReason = so.order.RejectReason
	rpt.ClOrdId = int32(so.order.Id)
	rpt.Quantity = ToFloat(so.order.Quantity)
	rpt.Price = ToFloat(so.order.Price)
	rpt.Remaining = ToFloat(so.order.Remaining)
	if so.order.OrderSide == constant.Buy {
		rpt.Side = protocol.CreateOrderRequest_Buy
	} else {
		rpt.Side = protocol.CreateOrderRequest_Sell
	}
	reply := &protocol.OutMessage_Execrpt{Execrpt: rpt}
	so.client.(*grpcClient).conn.Send(&protocol.OutMessage{Reply: reply})
}

func (c *grpcClient) SendTrades(trades []trade) {
	for _, k := range trades {
		c.sendTradeExecutionReport(k.buyer, k.price, k.quantity, k.buyRemaining)
		c.sendTradeExecutionReport(k.seller, k.price, k.quantity, k.sellRemaining)
	}
}

// sessionId只要唯一就好了
func (c *grpcClient) SessionID() string {
	return fmt.Sprint(c.conn)
}

func (c *grpcClient) String() string {
	return c.SessionID()
}

func (c *grpcClient) sendTradeExecutionReport(so sessionOrder, price Fixed, quantity Fixed, remaining Fixed) {
	rpt := &protocol.ExecutionReport{}
	rpt.Symbol = so.order.Symbol()
	rpt.ExOrdId = so.order.ExchangeId
	rpt.ReportType = protocol.ExecutionReport_Fill
	rpt.ClOrdId = int32(so.order.Id)
	rpt.Quantity = ToFloat(so.order.Quantity)
	rpt.Price = ToFloat(so.order.Price)
	rpt.LastPrice = ToFloat(price)
	rpt.LastQuantity = ToFloat(quantity)
	if so.order.OrderSide == constant.Buy {
		rpt.Side = protocol.CreateOrderRequest_Buy
	} else {
		rpt.Side = protocol.CreateOrderRequest_Sell
	}
	switch so.order.OrderState {
	case constant.New, constant.Booked:
		rpt.OrderState = protocol.ExecutionReport_Booked
	case constant.PartialFill:
		rpt.OrderState = protocol.ExecutionReport_Partial
	case constant.Filled:
		rpt.OrderState = protocol.ExecutionReport_Filled
	case constant.Cancelled:
		rpt.OrderState = protocol.ExecutionReport_Cancelled
	case constant.Rejected:
		rpt.OrderState = protocol.ExecutionReport_Rejected
	}

	if !remaining.Equal(ZERO) {
		rpt.OrderState = protocol.ExecutionReport_Partial
	}

	rpt.Remaining = ToFloat(remaining)
	reply := &protocol.OutMessage_Execrpt{Execrpt: rpt}
	so.client.(*grpcClient).conn.Send(&protocol.OutMessage{Reply: reply})
}

// 系统函数
// 收到client的connect请求后, 会进入这里. 相当于这里是read入口
func (s *grpcServer) Connection(conn protocol.Exchange_ConnectionServer) error {

	//client代表一个connect长连接
	client := &grpcClient{conn: conn}
	//一个client对应一个session
	s.e.newSession(client)

	log.Println("grpc session connect", client)
	defer func() {
		log.Println("grpc session disconnect", client)
		//TODO 要看一下作用
		s.e.SessionDisconnect(client)
	}()

	for {
		//开始不断从这个conn中接收数据
		msg, err := conn.Recv()

		if err != nil {
			log.Println("recv failed", err)
			return err
		}

		switch msg.Request.(type) {
		case *protocol.InMessage_Login:
			//login登录请求
			err = s.login(conn, client, msg.GetRequest().(*protocol.InMessage_Login).Login)
			if err != nil {
				return err
			}
			continue
		}
		if !client.loggedIn {
			reply := &protocol.OutMessage_Reject{Reject: &protocol.SessionReject{Error: "session not logged in"}}
			err = conn.Send(&protocol.OutMessage{Reply: reply})
			continue
		}
		//--------------------------------------------

		switch msg.Request.(type) {
		case *protocol.InMessage_Download:
			//todo 要明确下作用
			s.download(conn, client)
		case *protocol.InMessage_Massquote:
			//todo 要明确下作用
			err = s.massquote(conn, client, msg.GetRequest().(*protocol.InMessage_Massquote).Massquote)
		case *protocol.InMessage_Create:
			//进入一个新order,开始撮合和做行情推送
			err = s.create(conn, client, msg.GetRequest().(*protocol.InMessage_Create).Create)
		case *protocol.InMessage_Modify:
			//修改order -> TODO 为什么会有修改能力?
			err = s.modify(conn, client, msg.GetRequest().(*protocol.InMessage_Modify).Modify)
		case *protocol.InMessage_Cancel:
			//取消order
			err = s.cancel(conn, client, msg.GetRequest().(*protocol.InMessage_Cancel).Cancel)
		}

		if err != nil {
			log.Println("recv failed", err)
			return err
		}
	}
}
func (s *grpcServer) login(conn protocol.Exchange_ConnectionServer, client *grpcClient, request *protocol.LoginRequest) error {
	log.Println("login received", request)
	var err error = nil
	reply := &protocol.OutMessage_Login{Login: &protocol.LoginReply{Error: toErrS(err)}}
	err = conn.Send(&protocol.OutMessage{Reply: reply})
	client.loggedIn = true
	client.user = request.Username
	return err
}
func (s *grpcServer) download(conn protocol.Exchange_ConnectionServer, client *grpcClient) {
	log.Println("downloading...")
	for _, symbol := range conf.IMap.AllSymbols() {
		instrument := conf.IMap.GetBySymbol(symbol)
		sec := &protocol.OutMessage_Secdef{Secdef: &protocol.SecurityDefinition{Symbol: symbol, InstrumentID: instrument.ID()}}
		err := conn.Send(&protocol.OutMessage{Reply: sec})
		if err != nil {
			return
		}
	}
	sec := &protocol.OutMessage_Secdef{Secdef: &protocol.SecurityDefinition{Symbol: endOfDownload.Symbol(), InstrumentID: endOfDownload.ID()}}
	conn.Send(&protocol.OutMessage{Reply: sec})
	log.Println("downloading complete")
}

// mass quote message 批量报价
func (s *grpcServer) massquote(server protocol.Exchange_ConnectionServer, client *grpcClient, q *protocol.MassQuoteRequest) error {
	instrument := conf.IMap.GetBySymbol(q.Symbol)
	if instrument == nil {
		return errors.New("unknown symbol " + q.Symbol)
	}
	//TODO 这个价这个数量,你接不接?
	return s.e.Quote(client, instrument, NewDecimalF(q.BidPrice), NewDecimalF(q.BidQuantity), NewDecimalF(q.AskPrice), NewDecimalF(q.AskQuantity))
}
func (s *grpcServer) create(conn protocol.Exchange_ConnectionServer, client *grpcClient, request *protocol.CreateOrderRequest) error {

	instrument := conf.IMap.GetBySymbol(request.Symbol)
	if instrument == nil {
		reply := &protocol.OutMessage_Reject{Reject: &protocol.SessionReject{Error: "unknown symbol " + request.Symbol}}
		return conn.Send(&protocol.OutMessage{Reply: reply})
	}

	var order *entity.Order
	var side constant.OrderSide

	if request.OrderSide == protocol.CreateOrderRequest_Buy {
		side = constant.Buy
	} else {
		side = constant.Sell
	}

	if request.OrderType == protocol.CreateOrderRequest_Limit {
		order = entity.LimitOrder(instrument, side, NewDecimalF(request.Price), NewDecimalF(request.Quantity))
	} else {
		order = entity.MarketOrder(instrument, side, NewDecimalF(request.Quantity))
	}
	order.Id = entity.NewOrderID(strconv.Itoa(int(request.ClOrdId)))
	s.e.CreateOrder(client, order)
	return nil
}
func (s *grpcServer) modify(server protocol.Exchange_ConnectionServer, client *grpcClient, request *protocol.ModifyOrderRequest) error {
	price := NewDecimalF(request.Price)
	qty := NewDecimalF(request.Quantity)
	s.e.ModifyOrder(client, entity.NewOrderID(strconv.Itoa(int(request.ClOrdId))), price, qty)
	return nil
}
func (s *grpcServer) cancel(server protocol.Exchange_ConnectionServer, client *grpcClient, request *protocol.CancelOrderRequest) error {
	s.e.CancelOrder(client, entity.NewOrderID(strconv.Itoa(int(request.ClOrdId))))
	return nil
}

func toErrS(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func NewGrpcServer() protocol.ExchangeServer {
	s := grpcServer{e: &TheExchange}
	return &s
}
