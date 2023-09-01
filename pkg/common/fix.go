package common

import (
	"github.com/quickfixgo/enum"
	. "github.com/robaho/fixed"
	"github.com/robaho/go-trader/pkg/constant"
	"github.com/shopspring/decimal"
)

func ToFixed(d decimal.Decimal) Fixed {
	f, _ := d.Float64()
	return NewDecimalF(f)
}
func ToDecimal(f Fixed) decimal.Decimal {
	return decimal.NewFromFloat(f.Float())
}

func MapToFixSide(side constant.Side) enum.Side {
	switch side {
	case constant.Buy:
		return enum.Side_BUY
	case constant.Sell:
		return enum.Side_SELL
	}
	panic("unsupported side " + side)
}

func MapFromFixSide(side enum.Side) constant.Side {
	switch side {
	case enum.Side_BUY:
		return constant.Buy
	case enum.Side_SELL:
		return constant.Sell
	}
	panic("unsupported side " + side)
}

func MapToFixOrdStatus(state constant.OrderState) enum.OrdStatus {
	switch state {
	case constant.Booked:
		return enum.OrdStatus_NEW
	case constant.PartialFill:
		return enum.OrdStatus_PARTIALLY_FILLED
	case constant.Filled:
		return enum.OrdStatus_FILLED
	case constant.Cancelled:
		return enum.OrdStatus_CANCELED
	case constant.Rejected:
		return enum.OrdStatus_REJECTED
	}
	panic("unknown OrderState " + state)
}

func MapFromFixOrdStatus(ordStatus enum.OrdStatus) constant.OrderState {
	switch ordStatus {
	case enum.OrdStatus_NEW:
		return constant.Booked
	case enum.OrdStatus_CANCELED:
		return constant.Cancelled
	case enum.OrdStatus_PARTIALLY_FILLED:
		return constant.PartialFill
	case enum.OrdStatus_FILLED:
		return constant.Filled
	case enum.OrdStatus_REJECTED:
		return constant.Rejected
	}
	panic("unsupported order status " + ordStatus)
}
