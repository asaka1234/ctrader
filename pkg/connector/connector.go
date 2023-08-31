package connector

import (
	"github.com/robaho/go-trader/conf"
	"github.com/robaho/go-trader/pkg/common"
	"github.com/robaho/go-trader/pkg/connector/grpc"
	"github.com/robaho/go-trader/pkg/connector/marketdata"
	"github.com/robaho/go-trader/pkg/connector/qfix"
	"io"
)

func NewConnector(callback common.ConnectorCallback, fixSettingFilename string, logOutput io.Writer) common.ExchangeConnector {
	var c common.ExchangeConnector

	if "grpc" == conf.AppConfig.Protocol {
		c = grpc.NewConnector(callback, logOutput)
	} else {
		c = qfix.NewConnector(callback, fixSettingFilename, logOutput)
	}

	marketdata.StartMarketDataReceiver(c, callback, logOutput)
	return c
}
