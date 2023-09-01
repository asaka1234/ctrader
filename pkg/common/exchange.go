package common

import (
	"errors"
	. "github.com/robaho/fixed"
	"github.com/robaho/go-trader/entity"
	"github.com/robaho/go-trader/pkg/constant"
	"time"
)

type ExchangeConnector interface {
	IsConnected() bool
	Connect() error
	Disconnect() error

	CreateOrder(order *entity.Order) (entity.OrderID, error)
	ModifyOrder(order entity.OrderID, price Fixed, quantity Fixed) error
	CancelOrder(order entity.OrderID) error

	Quote(instrument entity.Instrument, bidPrice Fixed, bidQuantity Fixed, askPrice Fixed, askQuantity Fixed) error

	GetExchangeCode() string

	// ask exchange to create the instrument if it does not already exist, and assign a numeric instrument id
	// the instruments are not persisted across exchange restarts
	CreateInstrument(symbol string)
	// ask exchange for configured instruments, will be emitted via onInstrument() on the callback. this call
	// blocks until all instruments are received
	DownloadInstruments() error
}

// a fill on an order or quote
type Fill struct {
	Instrument entity.Instrument
	IsQuote    bool
	// Order will be nil on quote trade, the order is unlocked
	Order      *entity.Order
	ExchangeID string
	Quantity   Fixed
	Price      Fixed
	Side       constant.Side
	IsLegTrade bool
}

// an exchange trade, not necessarily initiated by the current client
type Trade struct {
	Instrument entity.Instrument
	Quantity   Fixed
	Price      Fixed
	ExchangeID string
	TradeTime  time.Time
}

type ConnectorCallback interface {
	OnBook(*Book)
	// the following is for intra-day instrument addition, or initial startup
	OnInstrument(entity.Instrument)
	// the callback will have the order locked, and will unlock when the callback returns
	OnOrderStatus(*entity.Order)
	OnFill(*Fill)
	OnTrade(*Trade)
}

var AlreadyConnected = errors.New("already connected")
var NotConnected = errors.New("not connected")
var ConnectionFailed = errors.New("connection failed")
var OrderNotFound = errors.New("order not found")
var InvalidConnector = errors.New("invalid connector")
var UnknownInstrument = errors.New("unknown instrument")
var UnsupportedOrderType = errors.New("unsupported order type")
var DownloadFailed = errors.New("download failed")
