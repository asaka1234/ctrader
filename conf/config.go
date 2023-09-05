package conf

import (
	gookit_conf "github.com/gookit/config/v2"
	gookit_yml "github.com/gookit/config/v2/yaml"
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

func ParseConf(filename string, dst any, initial bool) error {
	//parse
	if initial {
		gookit_conf.AddDriver(gookit_yml.Driver)
		gookit_conf.WithOptions(gookit_conf.ParseDefault)
		gookit_conf.WithOptions(func(opt *gookit_conf.Options) { opt.DecoderConfig.TagName = "config" })
	}
	err := gookit_conf.LoadFiles(gookit_conf.Yaml, filename)
	if err != nil {
		//log.Infof("配置文件加载失败，请校验配置文件 Failed to load setting, Error in Unmarshal, err=%v", err.Error())
		if initial {
			os.Exit(0)
		}
		return err
	}
	err = gookit_conf.Decode(dst)
	if err != nil {
		//log.Infof("配置文件解析失败，请校验配置文件 Failed to load setting, Error in Unmarshal, err=%v", err.Error())
		if initial {
			os.Exit(0)
		}
		return err
	}
	return nil
}
