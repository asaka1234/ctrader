package model

import (
	"gorm.io/gorm"
	"logtech.com/exchange/ltrader/pkg/database"
	log "logtech.com/exchange/ltrader/pkg/logger"
	"time"
)

// 语言配置
type ExLanguageConfig struct {
	ID               int       `gorm:"column:id;primary_key"`     //
	IsOpen           int       `gorm:"column:is_open"`            //是否启用 1启用 0关闭
	Type             int       `gorm:"column:type"`               //类型 1条件设置
	Key              string    `gorm:"column:key"`                //语言key
	Language         string    `gorm:"column:language"`           //
	Title            string    `gorm:"column:title"`              //语言名称
	Icon             string    `gorm:"column:icon"`               //语言图标
	SmsHeader        string    `gorm:"column:sms_header"`         //短信机构名（短信里的title）
	EmailHeader      string    `gorm:"column:email_header"`       //邮件机构名
	OtcPayMode       string    `gorm:"column:otc_pay_mode"`       //支付方式
	OtcCountry       string    `gorm:"column:otc_country"`        //国家
	OtcPaycoin       string    `gorm:"column:otc_paycoin"`        //支付币种
	PhoneCountryName string    `gorm:"column:phone_country_name"` //xml中电话号码国家编码名字
	OperateOpen      int       `gorm:"column:operate_open"`       //后台是否开启该语言；1开启，0关闭
	CmsTypeID        string    `gorm:"column:cms_type_id"`        //后台cms系统分类id;英文逗号分隔前者为帮助中心分类id，后者为footer分类id
	NameOrder        string    `gorm:"column:name_order"`         //不同国家姓名顺序，familyName姓，givenName名
	Sort             int       `gorm:"column:sort"`               //排序
	Lang             string    `gorm:"column:lang"`               //语言缩写
	MoneySymbol      string    `gorm:"column:money_symbol"`       //货币符号
	Ctime            time.Time `gorm:"column:ctime"`              //
	Mtime            time.Time `gorm:"column:mtime"`              //
	SeoKeywords      string    `gorm:"column:seo_keywords"`       //爬虫_keywords
	SeoDescription   string    `gorm:"column:seo_description"`    //爬虫_description
	SeoTitle         string    `gorm:"column:seo_title"`          //爬虫_title
	SeoPageContent   string    `gorm:"column:seo_page_content"`   //爬虫_page_content
}

// TableName sets the insert table name for this struct type
func (e *ExLanguageConfig) TableName() string {
	return "ex_language_config"
}

// key = zh_CN en_US ..
func GetExLanguageConfig(key string) *ExLanguageConfig {
	db, _ := database.GetCoreDb()
	info := ExLanguageConfig{}
	err := db.Model(ExLanguageConfig{}).Where("key=?", key).First(&info).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			log.Warnf("ConfigKvStore:GetConfig, get conf fail, key=%+v, err=%+v", key, err)
		}
		return nil
	}
	return &info
}
