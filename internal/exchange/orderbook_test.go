package exchange

import (
	"fmt"
	"github.com/robaho/go-trader/entity"
	"github.com/robaho/go-trader/pkg/constant"
	"testing"
	"time"

	. "github.com/robaho/go-trader/pkg/common"
)

func TestWaitForKey(t *testing.T) {
	//time.Sleep(20*time.Second)
}

func TestOrderBook(t *testing.T) {
	// no need for locking here...
	var ob = orderBook{}

	var i = entity.NewInstrument(1, "abc")

	var o1 = entity.LimitOrder(i, constant.Buy, NewDecimal("100"), NewDecimal("10"))
	o1.ExchangeId = "1"
	var o2 = entity.LimitOrder(i, constant.Sell, NewDecimal("110"), NewDecimal("10"))
	o2.ExchangeId = "1"

	var s1 = sessionOrder{nil, o1, time.Now()}
	var s2 = sessionOrder{nil, o2, time.Now()}

	ob.add(s1)
	ob.add(s2)

	b := ob.buildBook()
	if len(b.Bids) != 1 {
		t.Error("incorrect bids", b.Bids, ob)
	}
	if len(b.Asks) != 1 {
		t.Error("incorrect asks", b.Asks, ob)
	}

	var o3 = entity.LimitOrder(i, constant.Buy, NewDecimal("100"), NewDecimal("10"))
	o3.ExchangeId = "3"
	var o4 = entity.LimitOrder(i, constant.Buy, NewDecimal("99"), NewDecimal("30"))
	o4.ExchangeId = "4"

	var s3 = sessionOrder{nil, o3, time.Now()}
	var s4 = sessionOrder{nil, o4, time.Now()}

	ob.add(s3)
	ob.add(s4)

	fmt.Println("the order book is ", &ob)
	b = ob.buildBook()
	fmt.Println("the book is ", b)

	if len(b.Bids) != 2 {
		t.Error("incorrect bids", b.Bids, ob)
	}
	if len(b.Asks) != 1 {
		t.Error("incorrect asks", b.Asks, ob)
	}
	if !b.Bids[0].Quantity.Equal(NewDecimal("20")) {
		t.Error("wrong quantity", b.Bids)
	}

	err := ob.remove(s4)
	if err != nil {
		t.Error("unexpected ", err)
	}

	fmt.Println("the order book is ", &ob)
	b = ob.buildBook()
	fmt.Println("the book is ", b)

	if len(b.Bids) != 1 {
		t.Error("incorrect bids", b.Bids, &ob)
	}
	if len(b.Asks) != 1 {
		t.Error("incorrect asks", b.Asks, &ob)
	}
	if !b.Bids[0].Quantity.Equal(NewDecimal("20")) {
		t.Error("wrong quantity", b.Bids)
	}

	err = ob.remove(s3)
	if err != nil {
		t.Error("unexpected ", err)
	}

	fmt.Println("the order book is ", &ob)
	b = ob.buildBook()
	fmt.Println("the book is ", b)

	if len(b.Bids) != 1 {
		t.Error("incorrect bids", b.Bids, &ob)
	}
	if len(b.Asks) != 1 {
		t.Error("incorrect asks", b.Asks, &ob)
	}
	if !b.Bids[0].Quantity.Equal(NewDecimal("10")) {
		t.Error("wrong quantity", b.Bids)
	}
}

func TestOrderMatch(t *testing.T) {
	// no need for locking here...
	var ob = orderBook{}

	var i = entity.NewInstrument(1, "abc")

	var o1 = entity.LimitOrder(i, constant.Buy, NewDecimal("110"), NewDecimal("20"))
	o1.ExchangeId = "1"
	var o2 = entity.LimitOrder(i, constant.Sell, NewDecimal("100"), NewDecimal("10"))
	o2.ExchangeId = "2"

	var s1 = sessionOrder{nil, o1, time.Now()}
	var s2 = sessionOrder{nil, o2, time.Now()}

	ob.add(s1)

	trades, _ := ob.add(s2)

	b := ob.buildBook()
	if len(b.Bids) != 1 {
		t.Error("incorrect bids", b.Bids, ob)
	}
	if len(b.Asks) != 0 {
		t.Error("incorrect asks", b.Asks, ob)
	}
	if len(trades) != 1 {
		t.Error("wrong trades", trades)
	}
	if !trades[0].quantity.Equal(NewDecimal("10")) {
		t.Error("wrong trade qty", trades)
	}

}
func TestOrderMatchSweep(t *testing.T) {
	// no need for locking here...
	var ob = orderBook{}

	var i = entity.NewInstrument(1, "abc")

	var o1 = entity.LimitOrder(i, constant.Buy, NewDecimal("100"), NewDecimal("20"))
	o1.ExchangeId = "1"

	var o2 = entity.LimitOrder(i, constant.Buy, NewDecimal("90"), NewDecimal("20"))
	o1.ExchangeId = "1"

	var o3 = entity.LimitOrder(i, constant.Sell, NewDecimal("80"), NewDecimal("30"))
	o2.ExchangeId = "2"

	var s1 = sessionOrder{nil, o1, time.Now()}
	var s2 = sessionOrder{nil, o2, time.Now()}
	var s3 = sessionOrder{nil, o3, time.Now()}

	ob.add(s1)
	ob.add(s2)

	trades, _ := ob.add(s3)

	b := ob.buildBook()
	if len(b.Bids) != 1 {
		t.Error("incorrect bids", b.Bids, ob)
	}
	if len(b.Asks) != 0 {
		t.Error("incorrect asks", b.Asks, ob)
	}
	if len(trades) != 2 {
		t.Error("wrong trades", trades)
	}
	if !trades[0].quantity.Equal(NewDecimal("20")) {
		t.Error("wrong trade qty", trades)
	}
	if !trades[1].quantity.Equal(NewDecimal("10")) {
		t.Error("wrong trade qty", trades)
	}

}
