package model

import (
	"gorm.io/gorm"
	"logtech.com/exchange/ltrader/pkg/database"
	log "logtech.com/exchange/ltrader/pkg/logger"
	"time"
)

//-----i18n 所有文案等的多语言模板---------------------------

type ConfigLanguage struct {
	ID        int       `gorm:"column:id;primary_key"` //
	ConfigKey string    `gorm:"column:config_key"`     //配置key ->一个配置 对应多个语言版本 (deviceIpChangeSwitchEmailTip）
	LangKey   string    `gorm:"column:lang_key"`       //语言key -> en_US
	Content   string    `gorm:"column:content"`        //具体的模板内容
	Meta      string    `gorm:"column:meta"`           //描述
	Ctime     time.Time `gorm:"column:ctime"`          //
	Mtime     time.Time `gorm:"column:mtime"`          //
}

// TableName sets the insert table name for this struct type
func (c *ConfigLanguage) TableName() string {
	return "config_language"
}

// 依据key来获取配置信息
func GetConfigLanguage(configKey string, langKey string) *ConfigLanguage {
	db, _ := database.GetCoreDb()
	info := ConfigLanguage{}
	err := db.Model(ConfigLanguage{}).Where("config_key=?", configKey).Where("lang_key=?", langKey).First(&info).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			log.Warnf("ConfigLanguage:GetConfig, get conf fail, configKey=%+v, langKey=%+v,err=%+v", configKey, langKey, err)
		}
		return nil
	}
	return &info
}
