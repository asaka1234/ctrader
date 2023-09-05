package logger

import (
	"fmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// error logger
var errorLogger *zap.Logger //SugaredLogger
var atomicLevel zap.AtomicLevel

var levelMap = map[string]zapcore.Level{
	"debug":  zapcore.DebugLevel,
	"info":   zapcore.InfoLevel,
	"warn":   zapcore.WarnLevel,
	"error":  zapcore.ErrorLevel,
	"dpanic": zapcore.DPanicLevel,
	"panic":  zapcore.PanicLevel,
	"fatal":  zapcore.FatalLevel,
}

func getLoggerLevel(lvl string) zapcore.Level {
	if level, ok := levelMap[lvl]; ok {
		return level
	}
	return zapcore.InfoLevel
}

// Setup initialize the log instance
func Setup() {

	//获取日志的存储路径
	//filePath := getLogFilePath()
	//fileName := getLogFileName()
	//fileFullName := filePath + fileName
	//获取配置的日志级别(TODO:这里是不是要先pull拉取到才可以)
	//给个默认值,反正后边还会修改 cy
	atomicLevel = zap.NewAtomicLevel()
	cfg := zap.Config{
		Encoding:         "console",
		Level:            atomicLevel,
		OutputPaths:      []string{"stderr"},
		ErrorOutputPaths: []string{"stderr"},
		//InitialFields:    map[string]interface{}{"foo": "bar"},
		EncoderConfig: zapcore.EncoderConfig{
			MessageKey: "message",

			LevelKey:    "level",
			EncodeLevel: zapcore.CapitalLevelEncoder,

			TimeKey:    "time",
			EncodeTime: zapcore.ISO8601TimeEncoder,

			CallerKey:    "caller",
			EncodeCaller: zapcore.ShortCallerEncoder,
		},
	}
	var err error
	errorLogger, err = cfg.Build()
	if err != nil {
		panic(err)
	}
	errorLogger = errorLogger.WithOptions(zap.AddCallerSkip(1))
}

// 增加公共字段
func AddHook(commonFields map[string]string) {
	fields := make([]zap.Field, 0)
	for key, val := range commonFields {
		fields = append(fields, zap.String(key, val))
	}
	errorLogger.WithOptions(zap.Fields(fields...))
}

func SetLevel(lvl string) {
	atomicLevel.SetLevel(getLoggerLevel(lvl))
}

func GetLevel() zapcore.Level {
	return atomicLevel.Level()
}

func Debug(msg string, args ...zap.Field) {
	errorLogger.Debug(msg, args...)
}

func Debugf(template string, args ...interface{}) {
	msg := fmt.Sprintf(template, args...)
	errorLogger.Debug(msg)
}

func Info(msg string, args ...zap.Field) {
	errorLogger.Info(msg, args...)
}

func Infof(template string, args ...interface{}) {
	msg := fmt.Sprintf(template, args...)
	errorLogger.Info(msg)
}

func Warn(msg string, args ...zap.Field) {
	errorLogger.Warn(msg, args...)
}

func Warnf(template string, args ...interface{}) {
	msg := fmt.Sprintf(template, args...)
	errorLogger.Warn(msg)
}

func Error(msg string, args ...zap.Field) {
	errorLogger.Error(msg, args...)
}

func Errorf(template string, args ...interface{}) {
	msg := fmt.Sprintf(template, args...)
	errorLogger.Error(msg)
}

func DPanic(msg string, args ...zap.Field) {
	errorLogger.DPanic(msg, args...)
}

func Panic(msg string, args ...zap.Field) {
	errorLogger.Panic(msg, args...)
}

func Fatal(msg string, args ...zap.Field) {
	errorLogger.Fatal(msg, args...)
}
