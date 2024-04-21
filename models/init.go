package models

import (
	gormlog "github.com/onesaltedseafish/go-utils/log/gorm"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var (
	universalDb *gorm.DB
)

// InitDb 初始化 DB
func InitDb(dialector gorm.Dialector, migrate bool, logger *gormlog.Logger) (*gorm.DB, error) {
	var config gorm.Config
	var db *gorm.DB
	var err error
	if logger != nil {
		config.Logger = logger
	}
	db, err = gorm.Open(dialector, &config)
	if err != nil {
		return nil, err
	}
	if migrate {
		err = db.AutoMigrate(Peer{})
		if err != nil {
			return nil, err
		}
	}
	universalDb = db
	return db, nil
}

// InitSqlite 初始化 sqlite
func InitSqlite(sqlitePath string) gorm.Dialector {
	return sqlite.Open(sqlitePath)
}

// GetDb 获取全局 DB
func GetDb() *gorm.DB {
	return universalDb
}
