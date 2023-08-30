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
	props       string
	instruments string
	port        string
	profile     bool
)

var exchangeCmd = &cobra.Command{
	Use:   module,
	Short: "exchange service",
	Run: func(cmd *cobra.Command, args []string) {

		println("service server run...")
	},
}

func init() {
	exchangeCmd.PersistentFlags().StringVarP(&fix, "fix", "f", "configs/qf_got_settings", "set the fix session file")
	exchangeCmd.PersistentFlags().StringVarP(&props, "props", "p", "resources/lt-trader.yml", "set the exchange properties file")
	exchangeCmd.PersistentFlags().StringVarP(&instruments, "instruments", "i", "configs/instruments.txt", "the instrument file")
	exchangeCmd.PersistentFlags().StringVarP(&port, "port", "P", "8080", "set the web server port")
	exchangeCmd.PersistentFlags().BoolVarP(&profile, "profile", "c", false, "create CPU profiling output")
}

func main() {

	//解析配置文件
	err := conf.ParseConf(props, conf.AppConfig, true)
	err = common.IMap.Load(instruments)
	if err != nil {
		fmt.Println("unable to load instruments", err)
	}

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

	var ex = &exchange.TheExchange
	//相当于启动engine
	ex.Start()

	_ = acceptor.Start()
	defer acceptor.Stop()

	// start grpc protocol
	grpc_port := conf.AppConfig.GrpcPort
	//grpc_port := p.GetString("grpc_port", "5000")

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

	//又启动一个web-server
	exchange.StartWebServer(":" + port)
	fmt.Println("web server access available at :" + port)

	if profile {
		runtime.SetBlockProfileRate(1)
	}

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
			for _, symbol := range common.IMap.AllSymbols() {
				instrument := common.IMap.GetBySymbol(symbol)
				fmt.Println(instrument)
			}
		} else {
			fmt.Println("Unknown command, '", s, "' use 'help'")
		}
	again:
		fmt.Print("Command?")
	}
}
