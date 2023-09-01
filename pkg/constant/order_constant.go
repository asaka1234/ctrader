package constant

//-----------订单的状态--------------

type Side string
type OrderState string
type OrderType string

const (
	Buy  Side = "buy"
	Sell Side = "sell"
)

const (
	Market OrderType = "market"
	Limit  OrderType = "limit"
)

const (
	New         OrderState = "new"
	Booked      OrderState = "booked"
	PartialFill OrderState = "partial" //部分成交
	Filled      OrderState = "filled"  //全部成交
	Cancelled   OrderState = "cancelled"
	Rejected    OrderState = "rejected"
)
