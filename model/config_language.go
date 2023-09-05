package model

import "time"

//-----所有文案等的多语言模板---------------------------

type ConfigLanguage struct {
	ID        int       `gorm:"column:id;primary_key"` //
	ConfigKey string    `gorm:"column:config_key"`     //配置key
	LangKey   string    `gorm:"column:lang_key"`       //语言key
	Content   string    `gorm:"column:content"`        //
	Meta      string    `gorm:"column:meta"`           //描述
	Ctime     time.Time `gorm:"column:ctime"`          //
	Mtime     time.Time `gorm:"column:mtime"`          //
}

// TableName sets the insert table name for this struct type
func (c *ConfigLanguage) TableName() string {
	return "config_language"
}
