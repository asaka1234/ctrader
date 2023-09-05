package redis

import (
	"context"
	jsoniter "github.com/json-iterator/go"
	"github.com/redis/go-redis/v9"
	log "logtech.com/exchange/ltrader/pkg/logger"
	"strings"
	"time"
)

var (
	client *redis.ClusterClient
)

func Init(address, auth string) {
	addr := strings.Split(address, ",")

	//----------------------------------
	var option redis.ClusterOptions
	option = redis.ClusterOptions{
		Addrs:           addr,
		Password:        auth,
		ConnMaxIdleTime: 120 * time.Second,
		PoolSize:        10000,
	}
	//----------------------------------
	client = redis.NewClusterClient(&option)
	err := client.Ping(context.Background()).Err()
	if err != nil {
		log.Errorf("Redis:Init error,err=%+v", err)
	}
	log.Info("redis init SUCC")
	go monitorRedisConnection(addr, auth)
}

func Set(k string, v interface{}) (bool, error) {
	//defer client.Close()
	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	value, _ := json.Marshal(v)
	if _, err := client.Set(context.Background(), k, value, 0).Result(); err != nil {
		log.Errorf("redis: set key=%+v fail, value=%+v, err=%+v", k, v, err)
		return false, err
	}
	return true, nil
}

func Setnx(k string, v interface{}) (bool, error) {
	//defer client.Close()
	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	value, _ := json.Marshal(v)
	if _, err := client.SetNX(context.Background(), k, value, 0).Result(); err != nil {
		log.Errorf("redis: setnx key=%+v fail, value=%+v, err=%+v", k, v, err)
		return false, err
	}
	return true, nil
}

func Get(k string, r interface{}) (err error) {
	//defer client.Close()

	data, err := client.Get(context.Background(), k).Result()
	if err != nil {
		log.Warnf("redis: get key=%+v fail, err=%+v", k, err)
		return err
	}
	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	err = json.Unmarshal([]byte(data), r)
	if err != nil {
		return err
	}
	return nil
}

func HSet(hashKey string, field string, v interface{}) (bool, error) {
	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	value, _ := json.Marshal(v)
	_, err := client.HSet(context.Background(), hashKey, field, value).Result()
	if err != nil {
		log.Errorf("Redis:HSet hasKey=%v, field=%+v fail, value=%+v, err=%+v", hashKey, field, v, err)
		return false, err
	}
	return true, nil
}

func HVals(hashKey string) ([]string, error) {
	v, err := client.HVals(context.Background(), hashKey).Result()
	if err != nil {
		log.Errorf("Redis:HVals hasKey=%v, err=%+v", hashKey, err)
		return nil, err
	}
	return v, nil
}

func HGet(hashKey string, field string) (string, error) {
	v, err := client.HGet(context.Background(), hashKey, field).Result()
	if err != nil {
		log.Errorf("Redis:HVals hasKey=%v, err=%+v", hashKey, err)
		return "", err
	}
	return v, nil
}

func IncrBy(hashKey string, val int64) (int64, error) {
	v, err := client.IncrBy(context.Background(), hashKey, val).Result()
	if err != nil {
		log.Errorf("Redis:HVals hasKey=%v, err=%+v", hashKey, err)
		return -1, err
	}
	return v, nil
}

func monitorRedisConnection(address []string, auth string) {
	var err error

	for {
		err = client.Ping(context.Background()).Err()
		if err != nil {
			log.Debugf("redis Connection err  %v", err)
			newClient, err := refresh(address, auth)
			// 重新赋值句柄
			if err == nil {
				client = newClient
			}
		}
		//log.Debugf("monitorConnection ----> kline db")
		time.Sleep(time.Second)
	}
}

func refresh(address []string, auth string) (*redis.ClusterClient, error) {
	var option redis.ClusterOptions
	option = redis.ClusterOptions{
		Addrs:           address,
		Password:        auth,
		ConnMaxIdleTime: 120 * time.Second,
		PoolSize:        10000,
	}
	//----------------------------------
	newClient := redis.NewClusterClient(&option)
	err := newClient.Ping(context.Background()).Err()
	if err != nil {
		log.Errorf("Redis:refresh Ping error,err=%+v", err)
		return nil, err
	}
	return newClient, nil
}
