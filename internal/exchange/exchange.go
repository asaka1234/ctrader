package exchange

import (
	"fmt"
	. "github.com/robaho/fixed"
	"logtech.com/exchange/ltrader/entity"
	"logtech.com/exchange/ltrader/pkg/constant"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	. "logtech.com/exchange/ltrader/pkg/common"
)

// ------------------------------------------------------
// 存的是最新的bid/ask的两个单子.(TODO 没看出作用来)
type quotePair struct {
	bid sessionOrder
	ask sessionOrder
}

// ------------------------------------------------------
//代表一个clint链接conn.(grpc或fix),主要用于对结果的返回(毕竟不同协议的conn的结构是不一样的)

type exchangeClient interface {
	SendOrderStatus(so sessionOrder)
	SendTrades(trades []trade)
	SessionID() string
}

// ------------------------------------------------------
//一个session代表一个symbol的处理机

type session struct {
	sync.Mutex
	id     string
	orders map[entity.OrderID]*entity.Order //id -> order
	quotes map[entity.Instrument]quotePair  //每个pair的所有原始orderBook  (因为symbol没有独立的group，所以只能通过map来区分)
	client exchangeClient                   //TODO 这个要搞清楚作用.
}

// ------------------------------------------------------
//代表当前symbol的session中的一个order

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

// 一个session代表一个conn长连接
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

	//之所以要session order目的是为了找到他的client长连接.
	so := sessionOrder{client, order, time.Now()}

	//开始真正的去撮合
	trades, err := ob.add(so)
	if err != nil {
		return -1, err
	}

	//TODO 全量的所有book,没有走增量??
	book := ob.buildBook()
	/*
		sendMarketData是对结果的广播
		sendTrades是针对请求者的定向返回
	*/
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

// rfq相当于是询价请求
// TODO 但是我不理解的是为什么:实际执行撮合了?并且存了一份最新的bid/ask的最新的order？保存起来的作用是什么呢？
func (e *exchange) Quote(client exchangeClient, instrument entity.Instrument, bidPrice Fixed, bidQuantity Fixed, askPrice Fixed, askQuantity Fixed) error {
	ob := e.lockOrderBook(instrument)
	defer ob.Unlock()

	s := e.lockSession(client)
	defer s.Unlock()

	//把quote清空
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
		//相当于下了一个order
		order := entity.LimitOrder(instrument, constant.Buy, bidPrice, bidQuantity)
		order.ExchangeId = "quote.bid." + strconv.FormatInt(instrument.ID(), 10)
		so := sessionOrder{client, order, time.Now()}
		//赋值到bid里, 所以这里的bid应该是最新的单子
		qp.bid = so
		//做撮合，产生成交
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

	//获取orderbook
	book := ob.buildBook()
	sendMarketData(MarketEvent{book, trades})

	//发送trade成交出去
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
