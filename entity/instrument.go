package entity

// 一个代表一个币对可以认为（id + pair）
type Instrument interface {
	ID() int64
	Symbol() string
	Group() string
}

type base struct {
	id     int64
	symbol string
	group  string
}

type instrumentImpl struct {
	base
}

func (e *instrumentImpl) String() string {
	return e.symbol
}

func NewInstrument(id int64, symbol string) Instrument {
	e := instrumentImpl{base{id, symbol, symbol}}
	return e
}

func (b base) ID() int64 {
	return b.id
}
func (b base) Symbol() string {
	return b.symbol
}
func (b base) Group() string {
	return b.group
}
func (b base) String() string {
	return b.symbol
}
