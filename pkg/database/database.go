package database

import (
	"database/sql"
	"gorm.io/driver/mysql"
	_ "gorm.io/driver/mysql"
	"gorm.io/gorm"
	gorm_logger "gorm.io/gorm/logger"
	"logtech.com/exchange/ltrader/conf"
	log "logtech.com/exchange/ltrader/pkg/logger"
	"time"
)

var coreDb *gorm.DB
var klineMasterDb *gorm.DB
var klineSlaveDb *gorm.DB

func InitCore(dsn string, maxConn, maxIdle int) {

	var err error
	coreDb, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: gorm_logger.Default.LogMode(gorm_logger.Error),
	})
	if err != nil {
		panic(err)
	} else {
		sqlDB, err := coreDb.DB()
		if err != nil {
			panic(err)
		} else {
			sqlDB.SetMaxIdleConns(maxIdle)
			sqlDB.SetMaxOpenConns(maxConn)
			sqlDB.SetConnMaxLifetime(time.Hour * 1)
			sqlDB.Ping()
			go monitorConnection(sqlDB, dsn)
		}
		log.Info("core DB init SUCC")
	}

}

func InitKlineMasterDb(dsn string, maxConn, maxIdle int) {
	var err error
	klineMasterDb, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: gorm_logger.Default.LogMode(gorm_logger.Error),
	})
	if err != nil {
		panic(err)
	} else {
		sqlDB, err := klineMasterDb.DB()
		if err != nil {
			panic(err)
		} else {
			sqlDB.SetMaxIdleConns(maxIdle)
			sqlDB.SetMaxOpenConns(maxConn)
			sqlDB.SetConnMaxLifetime(time.Hour * 1)
			sqlDB.Ping()
			go monitorConnection(sqlDB, dsn)
		}
		log.Info("kline DB init SUCC")
	}
}

func InitKlineSlaveDb(dsn string, maxConn, maxIdle int) {
	var err error
	klineSlaveDb, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: gorm_logger.Default.LogMode(gorm_logger.Error),
	})
	if err != nil {
		panic(err)
	} else {
		sqlDB, err := klineSlaveDb.DB()
		if err != nil {
			panic(err)
		} else {
			sqlDB.SetMaxIdleConns(maxIdle)
			sqlDB.SetMaxOpenConns(maxConn)
			sqlDB.SetConnMaxLifetime(time.Hour * 1)
			sqlDB.Ping()
			go monitorConnection(sqlDB, dsn)
		}
		log.Info("kline DB init SUCC")
	}
}

func GetCoreDb() (*gorm.DB, error) {
	return coreDb, nil
}

func GetKlineDb() (*gorm.DB, error) {
	return klineSlaveDb, nil
}

func GetKlineSlaveDb() (*gorm.DB, error) {
	return klineSlaveDb, nil
}

func GetKlineMasterDb() (*gorm.DB, error) {
	return klineMasterDb, nil
}

func CloseAllDb() error {
	sqlDB, err := coreDb.DB()
	if err != nil {
		return nil
	}
	err = sqlDB.Close()
	if err != nil {
		return nil
	}

	//-----------

	sqlDB, err = klineMasterDb.DB()
	if err != nil {
		return nil
	}
	err = sqlDB.Close()
	if err != nil {
		return nil
	}

	//-----------
	sqlDB, err = klineSlaveDb.DB()
	if err != nil {
		return nil
	}
	return sqlDB.Close()
}

func refresh(url string) (newSql *sql.DB, err error) {
	klineSlaveDb, err = gorm.Open(mysql.Open(url), &gorm.Config{
		Logger: gorm_logger.Default.LogMode(gorm_logger.Error),
	})
	if err != nil {
		log.Errorf("kline db refresh error %v", err)
		return nil, err
	} else {
		sqlDB, err := klineSlaveDb.DB()
		if err != nil {
			panic(err)
		} else {
			sqlDB.SetMaxIdleConns(conf.AppConfig.DB.MaxConn)
			sqlDB.SetMaxOpenConns(conf.AppConfig.DB.MaxConn)
			sqlDB.SetConnMaxLifetime(time.Hour * 1)
		}
		return sqlDB, nil
	}
}

func monitorConnection(sql *sql.DB, url string) {
	var err error

	for {
		err = sql.Ping()
		if err != nil {
			newSql, err := refresh(url)
			// 重新赋值句柄
			if err == nil {
				sql = newSql
			}
		}
		log.Infof("monitorConnection ----> kline db")
		time.Sleep(5 * time.Second)
	}
}
