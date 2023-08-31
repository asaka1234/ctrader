package main

import (
	"bufio"
	"fmt"
	"github.com/robaho/go-trader/conf"
	"github.com/robaho/go-trader/pkg/protocol"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"log"
	"net"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/quickfixgo/quickfix"
	"github.com/robaho/go-trader/internal/exchange"
	"github.com/robaho/go-trader/pkg/common"
)

import _ "net/http/pprof"

var (
	module      = "exchange"
	fix         string
	config      string
	instruments string
	port        string
	profile     bool
)

var exchangeCmd = &cobra.Command{
	Use:   module,
	Short: "exchange service",
	Run: func(cmd *cobra.Command, args []string) {
		start()
		println("service server run...")
	},
}

func init() {
	//qf -> quickfix
	exchangeCmd.PersistentFlags().StringVarP(&fix, "fix", "f", "resources/qf_acceptor_settings.cfg", "set the fix session file")
	exchangeCmd.PersistentFlags().StringVarP(&config, "config", "c", "resources/lt-trader.yml", "set the exchange properties file")
	exchangeCmd.PersistentFlags().StringVarP(&instruments, "instruments", "i", "resources/instruments.txt", "the instrument file")
	exchangeCmd.PersistentFlags().StringVarP(&port, "port", "P", "8080", "set the statics server port")
	exchangeCmd.PersistentFlags().BoolVarP(&profile, "profile", "p", false, "create CPU profiling output")
}

func main() {
	if err := exchangeCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func start() {

	//1. 解析配置文件
	err := conf.ParseConf(config, conf.AppConfig, true)
	//2. 解析支持的币对
	err = conf.IMap.Load(instruments)
	if err != nil {
		fmt.Println("unable to load instruments", err)
	}
	//3. 解析fixapi的配置
	cfg, err := os.Open(fix)
	if err != nil {
		panic(err)
	}
	appSettings, err := quickfix.ParseSettings(cfg)
	if err != nil {
		panic(err)
	}
	storeFactory := quickfix.NewMemoryStoreFactory()
	//logFactory, _ := quickfix.NewFileLogFactory(appSettings)
	useLogging, err := appSettings.GlobalSettings().BoolSetting("Logging")
	var logFactory quickfix.LogFactory
	if useLogging {
		logFactory = quickfix.NewScreenLogFactory()
	} else {
		logFactory = quickfix.NewNullLogFactory()
	}
	//服务端,接受client传过来的数据
	acceptor, err := quickfix.NewAcceptor(&exchange.App, storeFactory, appSettings, logFactory)
	if err != nil {
		panic(err)
	}

	//-------感觉这里是exchange的引擎--------------------------
	var ex = &exchange.TheExchange
	//相当于启动engine，本质是启动了行情+push服务
	ex.Start()

	//1. 启动fix acceptor (类似启动web-server,从而client可以通过fix protocol来交互)
	_ = acceptor.Start()
	defer acceptor.Stop()

	//2. 启动grpc服务, client可以通过grpc来通信交互
	grpc_port := conf.AppConfig.GrpcPort
	lis, err := net.Listen("tcp", ":"+grpc_port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	} else {
		log.Println("accepting grpc connections at ", lis.Addr())
	}
	s := grpc.NewServer()
	protocol.RegisterExchangeServer(s, exchange.NewGrpcServer())
	// Register reflection service on gRPC server.
	reflection.Register(s)

	go func() {
		if err := s.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	//3. 启动一个web-server
	exchange.StartWebServer(":" + port)
	fmt.Println("statics server access available at :" + port)

	if profile {
		runtime.SetBlockProfileRate(1)
	}

	//-----------------------启动之后,也可以继续通过console来输入一些参数,指导程序做一定操作和输出------------------------------------
	watching := sync.Map{}

	fmt.Println("use 'help' to get a list of commands")
	fmt.Print("Command?")

	//输入命令行参数的解析
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		s := scanner.Text()
		parts := strings.Fields(s)
		if len(parts) == 0 {
			goto again
		}
		if "help" == parts[0] {
			fmt.Println("The available commands are: quit, sessions, book SYMBOL, watch SYMBOL, unwatch SYMBOL, list")
		} else if "quit" == parts[0] {
			break
		} else if "sessions" == parts[0] {
			fmt.Println("Active sessions: ", ex.ListSessions())
		} else if "book" == parts[0] {
			//获取当前的orderBook
			book := exchange.GetBook(parts[1])
			if book != nil {
				fmt.Println(book)
			}
		} else if "watch" == parts[0] && len(parts) == 2 {
			fmt.Println("You are now watching ", parts[1], ", use 'unwatch ", parts[1], "' to stop.")
			watching.Store(parts[1], "watching")
			go func(symbol string) {
				var lastBook *common.Book = nil
				for {
					if _, ok := watching.Load(symbol); !ok {
						break
					}
					book := exchange.GetBook(symbol)
					if book != nil {
						if lastBook != book {
							fmt.Println(book)
							lastBook = book
						}
					}
					time.Sleep(1 * time.Second)
				}
			}(parts[1])
		} else if "unwatch" == parts[0] && len(parts) == 2 {
			watching.Delete(parts[1])
			fmt.Println("You are no longer watching ", parts[1])
		} else if "list" == parts[0] {
			for _, symbol := range conf.IMap.AllSymbols() {
				instrument := conf.IMap.GetBySymbol(symbol)
				fmt.Println(instrument)
			}
		} else {
			fmt.Println("Unknown command, '", s, "' use 'help'")
		}
	again:
		fmt.Print("Command?")
	}
}
