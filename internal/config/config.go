package config

import (
	"errors"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type Config struct {
	AppPort         string
	CacheSize       int
	CacheDir        string
	LogLevel        logrus.Level
	ShutdownTimeout time.Duration
	DisableLogging  bool
}

func Load(configPath string) (*Config, error) {
	v := viper.New()

	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(configPath)
	v.AutomaticEnv()

	v.SetDefault("app_port", "8080")
	v.SetDefault("cache_size", 100)
	v.SetDefault("cache_dir", "./cache")
	v.SetDefault("log_level", "info")
	v.SetDefault("shutdown_timeout", "5s")
	v.SetDefault("disable_logging", false)

	// Читаем файл конфигурации
	if err := v.ReadInConfig(); err != nil {
		// Если файл не найден, это не ошибка, используем значения по умолчанию и переменные окружения
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) {
			return nil, err
		}
	}

	cfg := &Config{}

	cfg.AppPort = v.GetString("app_port")
	cfg.CacheSize = v.GetInt("cache_size")
	cfg.CacheDir = v.GetString("cache_dir")

	logLevelStr := v.GetString("log_level")
	logLevel, err := logrus.ParseLevel(logLevelStr)
	if err != nil {
		logLevel = logrus.InfoLevel
	}
	cfg.LogLevel = logLevel

	shutdownTimeoutStr := v.GetString("shutdown_timeout")
	shutdownTimeout, err := time.ParseDuration(shutdownTimeoutStr)
	if err != nil {
		shutdownTimeout = 5 * time.Second
	}
	cfg.ShutdownTimeout = shutdownTimeout

	cfg.DisableLogging = v.GetBool("disable_logging")

	return cfg, nil
}
