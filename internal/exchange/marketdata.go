package exchange

import (
	"bytes"
	"container/list"
	"encoding/binary"
	"fmt"
	. "github.com/robaho/fixed"
	"logtech.com/exchange/ltrader/conf"
	"logtech.com/exchange/ltrader/entity"
	"golang.org/x/net/ipv4"
	"log"
	"net"
	"strconv"
	"sync"
	"sync/atomic"

	. "logtech.com/exchange/ltrader/pkg/common"
	"logtech.com/exchange/ltrader/pkg/protocol"
)

//用来做行情加工和推送. 相当于是market和push-center两个服务的合集

// market data caches the latest books, and publishes books and exchange trades via multicast

var bookCache sync.Map  //key->symbol value是book
var statsCache sync.Map //key -> symbol. value是ticker

var eventChannel chan MarketEvent
var lastSentBook map[entity.Instrument]uint64 // to avoid publishing exact same book multiple times due to coalescing
var sequence uint64                           //每个book都有一个自增的seq_id,避免顺序颠倒
var udpCon *net.UDPConn
var pUdpCon *ipv4.PacketConn
var subMutex sync.Mutex
var subscriptions []chan *Book //TODO  貌似没用到.应该是还没开发完毕
var buffers = &SPSC{}

type MarketEvent struct {
	book   *Book
	trades []trade
}

// 这个统计是ticker
// 累计一段时间的高开低首/涨跌幅等
type Statistics struct {
	Symbol     string
	BidQty     Fixed //BBO档一的qty
	BidPrice   Fixed //BBO档一的price
	AskQty     Fixed //BBO档一的qty
	AskPrice   Fixed //BBO档一的price
	Volume     Fixed
	High       Fixed
	Low        Fixed
	HasHighLow bool
}

func subscribe(sub chan *Book) {
	subMutex.Lock()
	defer subMutex.Unlock()

	subscriptions = append(subscriptions, sub)
}

func unsubscribe(sub chan *Book) {
	subMutex.Lock()
	defer subMutex.Unlock()

	copy := subscriptions[:0]
	for _, v := range subscriptions {
		if v != sub {
			copy = append(copy, v)
		}
	}
	subscriptions = copy
}

func sendMarketData(event MarketEvent) {
	cacheBook(event.book)
	eventChannel <- event
}

func cacheBook(book *Book) {
	book.Sequence = atomic.AddUint64(&sequence, 1)
	bookCache.Store(book.Instrument, book)
}

func GetLatestBook(instrument entity.Instrument) *Book {
	v, ok := bookCache.Load(instrument)
	if !ok {
		return nil
	}
	return v.(*Book)
}

func GetBook(symbol string) *Book {
	i := conf.IMap.GetBySymbol(symbol)
	if i == nil {
		return nil
	}
	return GetLatestBook(i)
}

func newBuffer() *bytes.Buffer {
	placeholder := make([]byte, 8)

	p := buffers.get()

	if p != nil {
		p.Write(placeholder) // leave room for packet number
		return p
	}

	buf := new(bytes.Buffer)
	buf.Grow(protocol.MaxMsgSize)
	buf.Write(placeholder) // leave room for packet number
	return buf
}

// 感觉就是发送行情(book + trades).同时更新了ticker, 但是也没见发送，只是更新了内存. 应该是还没实现完毕
func publish() {
	stats := make(map[entity.Instrument]*Statistics)

	buf := newBuffer()

	for {
		//收到撮合结果
		event := <-eventChannel

		//TODO 要看一下具体作用
		book := getLatestBook(event.book)
		trades := coalesceTrades(event.trades)

		s, ok := stats[book.Instrument]
		if !ok {
			s = &Statistics{}
			s.Symbol = book.Instrument.Symbol()
			stats[book.Instrument] = s
		}

		if book.HasBids() {
			//档一
			s.BidPrice = book.Bids[0].Price
			s.BidQty = book.Bids[0].Quantity
		}
		if book.HasAsks() {
			//档一
			s.AskPrice = book.Asks[0].Price
			s.AskQty = book.Asks[0].Quantity
		}

		for _, t := range trades {
			s.Volume = s.Volume.Add(t.Quantity)
			if !s.HasHighLow {
				s.High = t.Price
				s.Low = s.High
				s.HasHighLow = true
			} else {
				if t.Price.GreaterThan(s.High) {
					s.High = t.Price
				}
				if t.Price.LessThan(s.Low) {
					s.Low = t.Price
				}
			}
		}
		//更新Ticker行情
		statsCache.Store(book.Instrument, s)

		buf2 := newBuffer()
		protocol.EncodeMarketEvent(buf2, book, trades)

		//用udp发出去
		if len(eventChannel) == 0 || buf2.Len()+buf.Len() > protocol.MaxMsgSize {
			if buf.Len() == 8 { // the group packet is empty, so just use this one
				sendPacket(buf2.Bytes())
			} else {
				if buf2.Len()+buf.Len() > protocol.MaxMsgSize {
					sendPacket(buf.Bytes())
					sendPacket(buf2.Bytes())
				} else {
					// copy current into the group packet, skipping packet number
					buf.Write(buf2.Bytes()[8:])
					sendPacket(buf.Bytes())
				}
				buf = newBuffer()
			}
		} else {
			// there is another update so delay sending, and this fits, so copy to group packet
			buf.Write(buf2.Bytes()[8:])
		}

		// TODO 要看一下这一块发给内部作用是什么？？？
		// publish to internal subscribers
		for _, sub := range subscriptions {
			sub <- book
		}
	}
}

func getLatestBook(book *Book) *Book {
	lastSeq, ok := lastSentBook[book.Instrument]
	if ok {
		//必须是最新的才可以，book是通过seq_id来维护新旧的.
		//TODO 没找到在哪赋值的.
		if lastSeq >= book.Sequence {
			return nil
		}
	}
	return book
}

func getStatistics(instrument entity.Instrument) *Statistics {
	stats, ok := statsCache.Load(instrument)
	if ok {
		return stats.(*Statistics)
	}
	return nil
}

// 联系的trade-list, 类似滑块. 前后挨着price相同的做合并.
// trade是不关心trade_id的，更多知识price+qty
func coalesceTrades(trades []trade) []Trade {
	var Trades []Trade

	// coalesce all trades at same price
	last := 0
	for i, v := range trades {
		if i > 0 && v.price.Equal(Trades[last].Price) {
			Trades[last].Quantity = Trades[last].Quantity.Add(v.quantity)
			continue
		}
		exchangeID := strconv.FormatInt(v.tradeId, 10)
		t := Trade{Instrument: v.seller.order.Instrument, Price: v.price, Quantity: v.quantity, ExchangeID: exchangeID, TradeTime: v.when}
		Trades = append(Trades, t)
		last = len(Trades) - 1
	}
	return Trades
}

var packetNumber uint64

func sendPacket(data []byte) {

	packetNumber++

	binary.LittleEndian.PutUint64(data, packetNumber)

	//udp去发送
	_, err := udpCon.Write(data)
	if err != nil {
		fmt.Println("error sending packet", err)
	}

	rememberPacket(packetNumber, data)
}

// 感觉是推送行情数据
func startMarketData() {
	eventChannel = make(chan MarketEvent, 1000)
	lastSentBook = make(map[entity.Instrument]uint64)

	// read settings and create socket
	saddr := conf.AppConfig.MulticastAddr
	intf := conf.AppConfig.MulticastIntf
	_intf, err := net.InterfaceByName(intf)
	if err != nil {
		panic("unable to read multicast interface")
	}

	fmt.Println("publishing marketdata at", saddr)

	addr, err := net.ResolveUDPAddr("udp", saddr)
	if err != nil {
		panic(err)
	}

	rport := conf.AppConfig.ReplayPort
	c, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		panic(err)
	}
	c.SetWriteBuffer(1024 * 1024)

	udpCon = c
	pUdpCon = ipv4.NewPacketConn(udpCon)
	pUdpCon.SetMulticastInterface(_intf)

	go func() {
		publish()
	}()

	go func() {
		ln, err := net.Listen("tcp", "0.0.0.0:"+rport)
		if err != nil {
			log.Fatal("unable to listen on replay port", err)
		} else {
			log.Println("listening for replay requests on", ln.Addr())
		}
		for {
			conn, _ := ln.Accept()

			go func(conn net.Conn) {
				var request protocol.ReplayRequest
				for {
					err := binary.Read(conn, binary.LittleEndian, &request)
					if err != nil {
						log.Println("failure to read replay request", err)
						return
					}
					err = resendPackets(conn, request)
					if err != nil {
						log.Println("failure to resend replay packets", err)
						return
					}
				}
			}(conn)
		}
	}()
}

type Packet struct {
	number uint64
	data   []byte
}
type PacketHistory struct {
	sync.RWMutex
	packets list.List
}

// 积攒一个list
var history PacketHistory

func rememberPacket(packetNumber uint64, data []byte) {
	history.Lock()
	defer history.Unlock()

	if history.packets.Len() > 10000 {
		p := history.packets.Remove(history.packets.Front()).(*Packet)
		buffers.put(bytes.NewBuffer(p.data[:0]))
	}

	packet := Packet{packetNumber, data}

	history.packets.PushBack(&packet)
}

func resendPackets(conn net.Conn, request protocol.ReplayRequest) error {
	history.RLock()
	defer history.RUnlock()

	expected := int(request.End - request.Start)
	var count = 0

	for e := history.packets.Front(); e != nil; e = e.Next() {
		p := e.Value.(*Packet)
		if p.number < request.Start {
			continue
		}
		if p.number >= request.End {
			break
		}
		count++
		var len = uint16(len(p.data))
		err := binary.Write(conn, binary.LittleEndian, &len)
		if err != nil {
			fmt.Println("unable to write replay packet header", err)
			return err
		}
		_, err = conn.Write(p.data)
		if err != nil {
			fmt.Println("unable to write replay packets", err)
			return err
		}
	}
	if count != expected {
		fmt.Println("replay failed", request, "missing", expected-count)
	} else {
		fmt.Println("replay complete", request, "count", count)
	}
	return nil
}
