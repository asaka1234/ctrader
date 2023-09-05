package logger

import (
	"fmt"
	"os"
	"time"
)

// getLogFilePath get the log file save path
func getLogFilePath() string {
	strLogPath := "logs/"
	_, err := os.Stat(strLogPath)
	if err != nil {
		fmt.Println(err)
		err2 := os.Mkdir(strLogPath, os.ModePerm)
		if err2 != nil {
			fmt.Println("Mkdir logs err", err)
			os.Exit(-1)
		}
	}
	return fmt.Sprintf("%s", strLogPath) //setting.AppSetting.RuntimeRootPath, setting.AppSetting.LogSavePath)
}

// getLogFileName get the save name of the log file
func getLogFileName() string {
	return fmt.Sprintf("%s.%s.%s",
		"access", //conf.LocalConfig.Application.Name,
		"log",
		time.Now().Format("20060102"),
	)
}
