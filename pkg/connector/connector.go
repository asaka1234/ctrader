package connector

import (
	"logtech.com/exchange/ltrader/conf"
	"logtech.com/exchange/ltrader/pkg/common"
	"logtech.com/exchange/ltrader/pkg/connector/grpc"
	"logtech.com/exchange/ltrader/pkg/connector/marketdata"
	"logtech.com/exchange/ltrader/pkg/connector/qfix"
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
