package model

import (
	"logtech.com/exchange/ltrader/pkg/database"
	"time"
)

type User struct {
	ID                        int       `gorm:"column:id;primary_key"`              //
	CountryCode               string    `gorm:"column:country_code"`                //
	MobileNumber              string    `gorm:"column:mobile_number"`               //
	Email                     string    `gorm:"column:email"`                       //
	LoginPwd                  string    `gorm:"column:login_pword"`                 //
	CapitalPwd                string    `gorm:"column:capital_pword"`               //
	AuthType                  int       `gorm:"column:auth_type"`                   //1A2B3CC
	AuthLevel                 int       `gorm:"column:auth_level"`                  //0123C2
	Nickname                  string    `gorm:"column:nickname"`                    //+
	LoginStatus               int       `gorm:"column:login_status"`                //01
	LoginExpireTime           time.Time `gorm:"column:loginexpire_time"`            //
	ExcStatus                 int       `gorm:"column:exc_status"`                  //01
	ExcExpireTime             time.Time `gorm:"column:excexpire_time"`              //
	WithdrawStatus            int       `gorm:"column:withdraw_status"`             //01
	WithdrawExpireTime        time.Time `gorm:"column:withdrawexpire_time"`         //
	LockExpireTime            time.Time `gorm:"column:lockexpire_time"`             //+100
	Ctime                     time.Time `gorm:"column:ctime"`                       //
	Mtime                     time.Time `gorm:"column:mtime"`                       //
	RealNameTime              time.Time `gorm:"column:realname_time"`               //
	CertificateTime           time.Time `gorm:"column:certificate_time"`            //C2
	LastLoginTime             time.Time `gorm:"column:last_login_time"`             //
	GoogleAuthenticatorStatus int       `gorm:"column:google_authenticator_status"` //:0-,1-
	GoogleAuthenticatorKey    string    `gorm:"column:google_authenticator_key"`    //key
	MobileAuthenticatorStatus int       `gorm:"column:mobile_authenticator_status"` //:0-,1-
	UserType                  int       `gorm:"column:user_type"`                   //
	ChannelID                 string    `gorm:"column:channel_id"`                  //
}

// 保存最新成交
func SaveUser(trade User) error {
	klineDb, _ := database.GetCoreDb()
	return klineDb.Model(trade).Create(&trade).Error
}
