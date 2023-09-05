package redis

import (
	"testing"
			"fmt"
	)
type Message struct {
	EventRep string `json:"event_rep"`
	Channel string `json:"channel"`
	Data interface{} `json:"data"`
	Tick interface{} `json:"tick"`
	Ts int64 `json:"ts"`
	Status string `json:"status"`
}
func TestGet(t *testing.T) {
	Init("39.106.139.147:7005","test")
	var data Message
	Get("546luobin", &data)
	fmt.Printf("get value: %+v", data)
}

func TestSet(t *testing.T) {
	v := Message{
		EventRep:"",
		Tick: []interface{}{1,"3",2},
	}

	Init("39.106.139.147:7001","test")
	Set("546luobin",v)
	fmt.Printf("set value: %+v", v)
}
