package exchange

import (
	. "github.com/robaho/fixed"
	"logtech.com/exchange/ltrader/entity"
	"logtech.com/exchange/ltrader/pkg/constant"
	"sync"
)
import (
	"fmt"
	"sort"
	"sync/atomic"
	"time"

	. "logtech.com/exchange/ltrader/pkg/common"
)

// orderbook和book区别
// orderbook： 是match里的,包含所有order列表
// book : 是market行情服务里的, 不包含所有order，只是每一档的price和Volume,
type orderBook struct {
	sync.Mutex
	entity.Instrument
	bids []sessionOrder
	asks []sessionOrder
}

type trade struct {
	buyer    sessionOrder
	seller   sessionOrder
	price    Fixed
	quantity Fixed
	tradeId  int64
	when     time.Time

	buyRemaining  Fixed
	sellRemaining Fixed
}

func (ob *orderBook) String() string {
	return fmt.Sprint("bids:", ob.bids, "asks:", ob.asks)
}

// 实际入口
func (ob *orderBook) add(so sessionOrder) ([]trade, error) {
	so.order.OrderState = constant.Booked

	//先把新的order加入到orderbook里
	if so.order.OrderSide == constant.Buy {
		ob.bids = insertSort(ob.bids, so, 1)
	} else {
		ob.asks = insertSort(ob.asks, so, -1)
	}

	//随后开始撮合
	// match and build trades
	var trades = matchTrades(ob)

	// cancel any remaining market order
	//市价单,剩余的部分就撤单！
	if so.order.OrderType == constant.Market && so.order.IsActive() {
		so.order.OrderState = constant.Cancelled
		ob.remove(so)
	}

	return trades, nil
}

func insertSort(orders []sessionOrder, so sessionOrder, direction int) []sessionOrder {
	index := sort.Search(len(orders), func(i int) bool {
		cmp := so.getPrice().Cmp(orders[i].getPrice()) * direction
		if cmp == 0 {
			cmp = CmpTime(so.time, orders[i].time)
		}
		return cmp >= 0
	})

	return append(orders[:index], append([]sessionOrder{so}, orders[index:]...)...)
}

var nextTradeID int64 = 0

// 撮合逻辑
// 感觉跟我们认知的match是不一样的
func matchTrades(book *orderBook) []trade {
	var trades []trade
	var tradeID int64 = 0
	var when = time.Now()

	for len(book.bids) > 0 && len(book.asks) > 0 {
		bid := book.bids[0]
		ask := book.asks[0]

		if !bid.getPrice().GreaterThanOrEqual(ask.getPrice()) {
			break
		}

		var price Fixed
		// need to use price of resting order
		if bid.time.Before(ask.time) {
			price = bid.order.Price
		} else {
			price = ask.order.Price
		}

		var qty = MinDecimal(bid.order.Remaining, ask.order.Remaining)

		var trade = trade{}

		if tradeID == 0 {
			// use same tradeID for all trades
			tradeID = atomic.AddInt64(&nextTradeID, 1)
		}

		trade.price = price
		trade.quantity = qty
		trade.buyer = bid
		trade.seller = ask
		trade.tradeId = tradeID
		trade.when = when

		fill(bid.order, qty, price)
		fill(ask.order, qty, price)

		trade.buyRemaining = bid.order.Remaining
		trade.sellRemaining = ask.order.Remaining

		trades = append(trades, trade)

		if bid.order.Remaining.Equal(ZERO) {
			book.remove(bid)
		}
		if ask.order.Remaining.Equal(ZERO) {
			book.remove(ask)
		}
	}
	return trades
}

func fill(order *entity.Order, qty Fixed, price Fixed) {
	order.Remaining = order.Remaining.Sub(qty)
	if order.Remaining.Equal(ZERO) {
		order.OrderState = constant.Filled
	} else {
		order.OrderState = constant.PartialFill
	}
}

func (ob *orderBook) remove(so sessionOrder) error {

	var removed bool

	removeFN := func(orders *[]sessionOrder, so sessionOrder) bool {
		for i, v := range *orders {
			if v.order == so.order {
				*orders = append((*orders)[:i], (*orders)[i+1:]...)
				return true
			}
		}
		return false
	}

	if so.order.OrderSide == constant.Buy {
		removed = removeFN(&ob.bids, so)
	} else {
		removed = removeFN(&ob.asks, so)
	}

	if !removed {
		return OrderNotFound
	}

	if so.order.IsActive() {
		so.order.OrderState = constant.Cancelled
	}

	return nil
}

// 构造book
func (ob *orderBook) buildBook() *Book {
	var book = new(Book)

	book.Instrument = ob.Instrument
	book.Bids = createBookLevels(ob.bids)
	book.Asks = createBookLevels(ob.asks)

	return book
}

// 把每一档的qty累加, 所以qty相当于是volume
func createBookLevels(orders []sessionOrder) []BookLevel {
	var levels []BookLevel

	if len(orders) == 0 {
		return levels
	}

	price := orders[0].order.Price
	quantity := ZERO

	for _, v := range orders {
		if v.order.Price.Equal(price) {
			quantity = quantity.Add(v.order.Remaining)
		} else {
			bl := BookLevel{Price: price, Quantity: quantity}
			levels = append(levels, bl)
			price = v.order.Price
			quantity = v.order.Remaining
		}
	}
	bl := BookLevel{Price: price, Quantity: quantity}
	levels = append(levels, bl)
	return levels
}
