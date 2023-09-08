package conf

import (
	"github.com/jinzhu/configor"
	"logtech.com/exchange/ltrader/pkg/logger"
	"os"
)

//--------------------------------------------------

type Config struct {
	MulticastAddr string `config:"multicast_addr" default:"224.0.0.100:9999"`
	MulticastIntf string `config:"multicast_intf" default:"lo0"`
	ReplayHost    string `config:"replay_host" default:"localhost"`
	ReplayPort    string `config:"replay_port" default:"9999"`
	GrpcPort      string `config:"grpc_port" default:"5000"`
	GrpcHost      string `config:"grpc_host" default:"localhost"` //protocol sets the client connect protocol, the server always enables both -> grpc|fix
	Protocol      string `config:"protocol" default:"fix"`

	DB struct {
		ExchangeDsn string `config:"ExchangeDsn" default:""`
		MaxConn     int    `config:"MaxConn" default:"50"`
		MaxIdle     int    `config:"MaxIdle" default:"10"`
	} `config:"DB"`

	Redis struct {
		Url  string `config:"Url" default:""`
		Auth string `config:"Auth" default:""`
	} `config:"Redis"`
}

// 是全局的配置加载
var AppConfig = &Config{}

func ParseConf(filename string) error {

	err := configor.Load(AppConfig, filename)
	if err != nil {
		logger.Infof("配置文件解析失败，请校验配置文件 Failed to load setting, Error in configor.Load, err=%v", err.Error())
		os.Exit(-1)
	}
	return nil
}
