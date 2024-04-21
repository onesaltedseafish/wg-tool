// Package main wg-tool server side
package main

import (
	"context"

	"github.com/onesaltedseafish/go-utils/log"
	gormLog "github.com/onesaltedseafish/go-utils/log/gorm"
	config "github.com/onesaltedseafish/wg-tool"
	"github.com/onesaltedseafish/wg-tool/models"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	logger     *log.Logger
	gormLogger *gormLog.Logger
	ctx        = context.Background()
)

func initLog() {
	logLevel, _ := zapcore.ParseLevel(config.Config.LogLevel)
	logOpt := log.CommonLogOpt.WithDirectory(config.Config.LogDirectory).WithLogLevel(logLevel)
	logger = log.GetLogger("wg-tool-server", &logOpt)
	gormLogger = gormLog.NewLogger("wg-tool-server-db", &logOpt)
}

// 初始化 DB
func initDb() {
	dialector := models.InitSqlite(config.Config.SqlitePath)
	if _, err := models.InitDb(dialector, true, gormLogger); err != nil {
		logger.Fatal(ctx, "init db failed", zap.Error(err))
	}
}

func main() {
	config.InitConfig()
	initLog()
	initDb()
	logger.Info(ctx, "start wg-tool server", zap.Any("config", config.Config))
}
