package exchange

import (
	"fmt"
	. "github.com/robaho/fixed"
	"github.com/robaho/go-trader/entity"
	"github.com/robaho/go-trader/pkg/constant"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	. "github.com/robaho/go-trader/pkg/common"
)

// ------------------------------------------------------
//TODO 作用是什么?

type quotePair struct {
	bid sessionOrder
	ask sessionOrder
}

// ------------------------------------------------------
//TODO 作用是什么?

type exchangeClient interface {
	SendOrderStatus(so sessionOrder)
	SendTrades(trades []trade)
	SessionID() string
}

// ------------------------------------------------------
//TODO 作用是什么?

type session struct {
	sync.Mutex
	id     string
	orders map[entity.OrderID]*entity.Order
	quotes map[entity.Instrument]quotePair
	client exchangeClient
}

// ------------------------------------------------------
//TODO 作用是什么?

type sessionOrder struct {
	client exchangeClient
	order  *entity.Order
	time   time.Time
}

func (so sessionOrder) String() string {
	return fmt.Sprint(so.client.SessionID(), so.order)
}

// return the "effective price" of an order - so market orders can always be at the top
func (so *sessionOrder) getPrice() Fixed {
	if so.order.OrderType == constant.Market {
		if so.order.OrderSide == constant.Buy {
			return NewDecimal("9999999999999")
		} else {
			return ZERO
		}
	}
	return so.order.Price
}

// -----------------------------------------------------------
// 交易引擎

var TheExchange exchange

type exchange struct {
	connected  bool
	callbacks  []ConnectorCallback
	orderBooks sync.Map // map of Instrument to *orderBook
	sessions   sync.Map // map of string to session
	nextOrder  int32
}

func (e *exchange) newSession(client exchangeClient) *session {
	s := session{}
	s.id = client.SessionID()
	s.orders = make(map[entity.OrderID]*entity.Order)
	s.quotes = make(map[entity.Instrument]quotePair)
	s.client = client

	e.sessions.Store(client, &s)

	return &s
}

// locking the session is probably not needed, as quickfix ensures a single thread
// processes all of the work for a "fix session"
// still, if it is uncontended it is very cheap
func (e *exchange) lockSession(client exchangeClient) *session {
	s, ok := e.sessions.Load(client)
	if !ok {
		fmt.Println("new session", client)
		s = e.newSession(client)
	}
	s.(*session).Lock()
	return s.(*session)
}

func (e *exchange) lockOrderBook(instrument entity.Instrument) *orderBook {
	ob, ok := e.orderBooks.Load(instrument)
	if !ok {
		ob = &orderBook{Instrument: instrument}
		ob, _ = e.orderBooks.LoadOrStore(instrument, ob)
	}
	_ob := ob.(*orderBook)
	_ob.Lock()
	return _ob
}

// 上游传过来一个order, 需要撮合和发送出去
func (e *exchange) CreateOrder(client exchangeClient, order *entity.Order) (entity.OrderID, error) {
	//拿到orderbook
	ob := e.lockOrderBook(order.Instrument)
	defer ob.Unlock()

	//类似seq_id,是自增的
	nextOrder := atomic.AddInt32(&e.nextOrder, 1)

	s := e.lockSession(client)
	defer s.Unlock()

	var orderID = order.Id

	order.ExchangeId = strconv.Itoa(int(nextOrder))

	s.orders[orderID] = order

	so := sessionOrder{client, order, time.Now()}

	//开始真正的去撮合
	trades, err := ob.add(so)
	if err != nil {
		return -1, err
	}

	book := ob.buildBook()
	//对撮合的结果做push推送
	sendMarketData(MarketEvent{book, trades})
	client.SendTrades(trades)
	if len(trades) == 0 || order.OrderState == constant.Cancelled {
		client.SendOrderStatus(so)
	}

	return orderID, nil
}

func (e *exchange) ModifyOrder(client exchangeClient, orderId entity.OrderID, price Fixed, quantity Fixed) error {
	s := e.lockSession(client)
	defer s.Unlock()

	order, ok := s.orders[orderId]
	if !ok {
		return OrderNotFound
	}

	ob := e.lockOrderBook(order.Instrument)
	defer ob.Unlock()

	so := sessionOrder{client, order, time.Now()}
	err := ob.remove(so)
	if err != nil {
		client.SendOrderStatus(so)
		return nil
	}

	order.Price = price
	order.Quantity = quantity
	order.Remaining = quantity

	trades, err := ob.add(so)
	if err != nil {
		return nil
	}
	book := ob.buildBook()
	sendMarketData(MarketEvent{book, trades})
	client.SendTrades(trades)
	if len(trades) == 0 {
		client.SendOrderStatus(so)
	}

	return nil
}

func (e *exchange) CancelOrder(client exchangeClient, orderId entity.OrderID) error {
	s := e.lockSession(client)
	defer s.Unlock()

	order, ok := s.orders[orderId]
	if !ok {
		return OrderNotFound
	}
	ob := e.lockOrderBook(order.Instrument)
	defer ob.Unlock()

	so := sessionOrder{client, order, time.Now()}
	err := ob.remove(so)
	if err != nil {
		return err
	}
	book := ob.buildBook()
	sendMarketData(MarketEvent{book: book})
	client.SendOrderStatus(so)

	return nil
}

func (e *exchange) Quote(client exchangeClient, instrument entity.Instrument, bidPrice Fixed, bidQuantity Fixed, askPrice Fixed, askQuantity Fixed) error {
	ob := e.lockOrderBook(instrument)
	defer ob.Unlock()

	s := e.lockSession(client)
	defer s.Unlock()

	qp, ok := s.quotes[instrument]
	if ok {
		if qp.bid.order != nil {
			ob.remove(qp.bid)
			qp.bid.order = nil
		}
		if qp.ask.order != nil {
			ob.remove(qp.ask)
			qp.ask.order = nil
		}
	} else {
		qp = quotePair{}
	}
	var trades []trade
	if !bidPrice.IsZero() {
		order := entity.LimitOrder(instrument, constant.Buy, bidPrice, bidQuantity)
		order.ExchangeId = "quote.bid." + strconv.FormatInt(instrument.ID(), 10)
		so := sessionOrder{client, order, time.Now()}
		qp.bid = so
		bidTrades, _ := ob.add(so)
		if bidTrades != nil {
			trades = append(trades, bidTrades...)
		}
	}
	if !askPrice.IsZero() {
		order := entity.LimitOrder(instrument, constant.Sell, askPrice, askQuantity)
		order.ExchangeId = "quote.ask." + strconv.FormatInt(instrument.ID(), 10)
		so := sessionOrder{client, order, time.Now()}
		qp.ask = so
		askTrades, _ := ob.add(so)
		if askTrades != nil {
			trades = append(trades, askTrades...)
		}
	}
	s.quotes[instrument] = qp

	book := ob.buildBook()
	sendMarketData(MarketEvent{book, trades})

	client.SendTrades(trades)

	return nil
}

func (e *exchange) ListSessions() string {
	var s []string

	e.sessions.Range(func(key, value interface{}) bool {
		s = append(s, key.(exchangeClient).SessionID())
		return true
	})
	return strings.Join(s, ",")
}

func (e *exchange) SessionDisconnect(client exchangeClient) {
	orderCount := 0
	quoteCount := 0

	s := e.lockSession(client)
	defer s.Unlock()

	for _, v := range s.orders {
		ob := e.lockOrderBook(v.Instrument)
		so := sessionOrder{client: client, order: v}
		ob.remove(so)
		client.SendOrderStatus(so)
		sendMarketData(MarketEvent{book: ob.buildBook()})
		ob.Unlock()
		orderCount++
	}
	for k, v := range s.quotes {
		ob := e.lockOrderBook(k)
		ob.remove(v.bid)
		ob.remove(v.ask)
		sendMarketData(MarketEvent{book: ob.buildBook()})
		ob.Unlock()
		quoteCount++
	}
	fmt.Println("session", client.SessionID(), "disconnected, cancelled", orderCount, "orders", quoteCount, "quotes")
}

func (e *exchange) Start() {
	startMarketData()
}
