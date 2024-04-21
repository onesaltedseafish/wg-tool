// Package config 提供基本的配置能力
package config

import (
	"context"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	// Config 全局的配置文件
	Config   = newConfig()
	ctx      = context.Background()
	validate = validator.New(validator.WithRequiredStructEnabled())
)

const (
	configName = "wg-tool"
)

var (
	configPossiblePath = []string{".", "/etc/wg-tool"}
)

type config struct {
	Token        string `mapstructure:"token" validate:"required"`
	SqlitePath   string `mapstructure:"sqlite"`
	LogLevel     string `mapstructure:"log_level"`
	LogDirectory string `mapstructure:"log_dir"`
}

func newConfig() config {
	return config{
		SqlitePath:   "./wg-tool-default.db",
		LogLevel:     "info",
		LogDirectory: "./logs",
	}
}

// InitConfig 初始化配置
func InitConfig() {
	logger, _ := zap.NewDevelopment()
	viper.SetConfigName("wg-tool")
	viper.SetConfigType("yaml")
	for _, v := range configPossiblePath {
		viper.AddConfigPath(v)
	}
	// get config
	if err := viper.ReadInConfig(); err != nil {
		logger.Fatal("ReadInConfig failed", zap.Error(err))
	}
	// unmarshal
	if err := viper.Unmarshal(&Config); err != nil {
		logger.Fatal("unmarshal config failed", zap.Error(err))
	}
	// 校验 config
	if err := validate.Struct(Config); err != nil {
		logger.Fatal("validate config failed", zap.Error(err))
	}
	if _, err := zapcore.ParseLevel(Config.LogLevel); err != nil {
		logger.Fatal("config log_leval invalid", zap.String("ori", Config.LogLevel))
	}
}
