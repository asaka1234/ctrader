package logger

import (
	"testing"
)

func TestWrite(t *testing.T) {
	Setup()
	Infof("this is info")
	Debugf("this is debug")
	Error("this is error")
	Errorf("this is error")
}
