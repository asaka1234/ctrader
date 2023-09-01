package entity

import (
	. "github.com/robaho/fixed"
	"github.com/robaho/go-trader/pkg/constant"
	"strconv"
	"sync"
)

type OrderID int32

//-------------------------------------------

type Order struct {
	sync.RWMutex
	Instrument
	Id         OrderID
	ExchangeId string
	Price      Fixed
	constant.OrderSide
	Quantity  Fixed
	Remaining Fixed
	constant.OrderType
	constant.OrderState
	RejectReason string
}

func (order *Order) String() string {
	return "oid " + order.Id.String() +
		" eoid " + order.ExchangeId +
		" " + order.Instrument.Symbol() +
		" " + order.Quantity.String() + "@" + order.Price.String() +
		" remaining " + order.Remaining.String() +
		" " + string(order.OrderState)
}
func (order *Order) IsActive() bool {
	return order.OrderState != constant.Filled && order.OrderState != constant.Cancelled && order.OrderState != constant.Rejected
}

func MarketOrder(instrument Instrument, side constant.OrderSide, quantity Fixed) *Order {
	order := newOrder(instrument, side, quantity)
	order.Price = ZERO
	order.OrderType = constant.Market
	return order
}

func LimitOrder(instrument Instrument, side constant.OrderSide, price Fixed, quantity Fixed) *Order {
	order := newOrder(instrument, side, quantity)
	order.Price = price
	order.OrderType = constant.Limit
	return order
}
func newOrder(instrument Instrument, side constant.OrderSide, qty Fixed) *Order {
	order := new(Order)
	order.Instrument = instrument
	order.OrderSide = side
	order.Quantity = qty
	order.Remaining = qty
	order.OrderState = constant.New
	return order
}

//---------工具------------------------------

func (id OrderID) String() string {
	return strconv.Itoa(int(id))
}
func NewOrderID(id string) OrderID {
	i, _ := strconv.Atoi(id)
	return OrderID(i)
}
