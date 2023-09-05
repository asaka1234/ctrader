package model

import (
	"gorm.io/gorm"
	"logtech.com/exchange/ltrader/pkg/database"
	log "logtech.com/exchange/ltrader/pkg/logger"
	"time"
)

type ConfigKvStore struct {
	ID        int       `gorm:"column:id;primary_key"` //
	ConfigKey string    `gorm:"column:config_key"`     //唯一key
	Value     string    `gorm:"column:value"`          //配置值
	Title     string    `gorm:"column:title"`          //标题
	Meta      string    `gorm:"column:meta"`           //描述
	Ctime     time.Time `gorm:"column:ctime"`          //
	Mtime     time.Time `gorm:"column:mtime"`          //
	Status    int       `gorm:"column:status"`         //状态：1开启，0关闭
	IsOpen    int       `gorm:"column:is_open"`        //是否在后台配置管理显示：1显示，0隐藏
	IsRelease int       `gorm:"column:is_release"`     //是否需要重新发布 1是，0否
}

// TableName sets the insert table name for this struct type
func (c *ConfigKvStore) TableName() string {
	return "config_kv_store"
}

// 依据key来获取配置信息
func GetConfigKvStore(key string) *ConfigKvStore {
	db, _ := database.GetCoreDb()
	info := ConfigKvStore{}
	err := db.Model(ConfigKvStore{}).Where("config_key=?", key).First(&info).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			log.Warnf("ConfigKvStore:GetConfig, get conf fail, key=%+v, err=%+v", key, err)
		}
		return nil
	}
	return &info
}
