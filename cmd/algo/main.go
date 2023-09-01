package main

import (
	"fmt"
	"github.com/robaho/fixed"
	"github.com/robaho/go-trader/conf"
	"github.com/robaho/go-trader/entity"
	"github.com/robaho/go-trader/pkg/constant"
	"github.com/spf13/cobra"
	"log"
	"os"
	"time"

	. "github.com/robaho/go-trader/pkg/common"
	"github.com/robaho/go-trader/pkg/connector"
)

var (
	module = "algo" //回测
	fix    string
	config string
	offset float64
	symbol string
)

var playbackCmd = &cobra.Command{
	Use:   module,
	Short: "exchange algo",
	Run: func(cmd *cobra.Command, args []string) {

		start()
		println("algo service run...")
	},
}

func init() {
	playbackCmd.PersistentFlags().StringVarP(&symbol, "symbol", "s", "IBM", "set the symbol")
	playbackCmd.PersistentFlags().StringVarP(&fix, "fix", "f", "resources/qf_algo_settings.cfg", "set the fix session file")
	playbackCmd.PersistentFlags().StringVarP(&config, "config", "c", "resources/lt-trader.yml", "set the exchange properties file")
	playbackCmd.PersistentFlags().Float64VarP(&offset, "offset", "o", 1.0, "price offset for entry exit")
}

func main() {
	if err := playbackCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// simple "algo" that buys a instrument, with an exit price offset - double the offset for exiting a loser.
// It tracks and reports total profit every 10 seconds.
// Very simple since it only handles an initial buy with quantities of 1.
func start() {
	var callback = MyAlgo{state: preInstrument}
	callback.symbol = symbol
	callback.offset = fixed.NewF(offset)

	//1. 解析配置文件
	err := conf.ParseConf(config, conf.AppConfig, true)
	/*
		p, err := NewProperties(*props)
		if err != nil {
			panic(err)
		}
		p.SetString("fix", *fix)
	*/

	exchange = connector.NewConnector(&callback, fix, nil)

	exchange.Connect()
	if !exchange.IsConnected() {
		panic("exchange is not connected")
	}

	err = exchange.DownloadInstruments()
	if err != nil {
		panic(err)
	}

	instrument := conf.IMap.GetBySymbol(callback.symbol)
	if instrument == nil {
		log.Fatal("unable symbol", symbol)
	}

	fmt.Println("running algo on", instrument.Symbol(), "...")

	for {
		time.Sleep(time.Duration(10) * time.Second)
		tp := callback.totalProfit
		if tp.LessThan(fixed.ZERO) {
			fmt.Println("<<<<< total profit", tp)
		} else {
			fmt.Println(">>>>> total profit", tp)
		}
	}
}

//-------------------------

type algoState int

const (
	preInstrument algoState = iota
	preEntry
	waitBuy
	waitExit
	waitSell
	preExit
)

var exchange ExchangeConnector

type MyAlgo struct {
	symbol      string
	instrument  entity.Instrument
	entryPrice  fixed.Fixed
	offset      fixed.Fixed
	totalProfit fixed.Fixed
	state       algoState
	runs        int
	nextEntry   time.Time
}

func (a *MyAlgo) OnBook(book *Book) {
	if book.Instrument != a.instrument {
		return
	}

	fmt.Println(book)

	switch a.state {
	case preEntry:
		if time.Now().Before(a.nextEntry) {
			return
		}
		if book.HasAsks() {
			exchange.CreateOrder(entity.LimitOrder(a.instrument, constant.Buy, book.Asks[0].Price, NewDecimal("1")))
			a.state = waitBuy
			a.runs++
		}
	case waitExit:
		if book.HasBids() {
			price := book.Bids[0].Price
			if price.GreaterThanOrEqual(a.entryPrice.Add(a.offset)) { // exit winner
				exchange.CreateOrder(entity.MarketOrder(a.instrument, constant.Sell, NewDecimal("1")))
				a.state = waitSell
			} else if price.LessThanOrEqual(a.entryPrice.Sub(a.offset)) { // exit loser ( 2 x the offset )
				exchange.CreateOrder(entity.MarketOrder(a.instrument, constant.Sell, NewDecimal("1")))
				a.state = waitSell
			}
		}
	}
}

func (a *MyAlgo) OnInstrument(instrument entity.Instrument) {
	if a.state == preInstrument && instrument.Symbol() == a.symbol {
		a.instrument = instrument
		a.state = preEntry
		fmt.Println("assigned instrument")
	}
}

func (*MyAlgo) OnOrderStatus(order *entity.Order) {
}

func (a *MyAlgo) OnFill(fill *Fill) {
	if a.state == waitBuy {
		a.entryPrice = fill.Price
		fmt.Println("entered market at ", fill.Price)
		a.state = waitExit
	}
	if a.state == waitSell {
		profit := fill.Price.Sub(a.entryPrice)
		fmt.Println("exited market at ", fill.Price)
		if profit.GreaterThan(fixed.ZERO) {
			fmt.Println("!!!! winner ", profit)
		} else {
			fmt.Println("____ loser ", profit)
		}
		a.totalProfit = a.totalProfit.Add(profit)
		a.state = preEntry
		a.nextEntry = time.Now().Add(time.Second)
	}
	//fmt.Println("fill", fill, "total profit",a.totalProfit)
}

func (*MyAlgo) OnTrade(trade *Trade) {
	//fmt.Println("trade", trade)
}
