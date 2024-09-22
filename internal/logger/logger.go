package logger

import (
	"io"

	"github.com/romangricuk/image-previewer/internal/config"
	"github.com/sirupsen/logrus"
)

// Интерфейс Logger с методами для различных уровней логирования.
type Logger interface {
	Info(args ...interface{})
	Infof(format string, args ...interface{})
	Debug(args ...interface{})
	Debugf(format string, args ...interface{})
	Warn(args ...interface{})
	Warnf(format string, args ...interface{})
	Error(args ...interface{})
	Errorf(format string, args ...interface{})
	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})
	SetOutput(output io.Writer)
}

// Структура AppLogger, реализующая интерфейс Logger.
type AppLogger struct {
	logger *logrus.Logger
}

// Фабричная функция для создания нового логгера.
func New(cfg *config.Config) Logger {
	log := logrus.New()
	log.SetLevel(cfg.LogLevel)
	log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
	if cfg.DisableLogging {
		log.SetOutput(io.Discard)
	}
	return &AppLogger{
		logger: log,
	}
}

// NewTestLogger создает логгер для использования в тестах.
func NewTestLogger() Logger {
	log := logrus.New()
	log.SetLevel(logrus.FatalLevel)
	log.SetOutput(io.Discard)

	return &AppLogger{
		logger: log,
	}
}

// Реализация методов интерфейса Logger

func (l *AppLogger) Info(args ...interface{}) {
	l.logger.Info(args...)
}

func (l *AppLogger) Infof(format string, args ...interface{}) {
	l.logger.Infof(format, args...)
}

func (l *AppLogger) Debug(args ...interface{}) {
	l.logger.Debug(args...)
}

func (l *AppLogger) Debugf(format string, args ...interface{}) {
	l.logger.Debugf(format, args...)
}

func (l *AppLogger) Warn(args ...interface{}) {
	l.logger.Warn(args...)
}

func (l *AppLogger) Warnf(format string, args ...interface{}) {
	l.logger.Warnf(format, args...)
}

func (l *AppLogger) Error(args ...interface{}) {
	l.logger.Error(args...)
}

func (l *AppLogger) Errorf(format string, args ...interface{}) {
	l.logger.Errorf(format, args...)
}

func (l *AppLogger) Fatal(args ...interface{}) {
	l.logger.Fatal(args...)
}

func (l *AppLogger) Fatalf(format string, args ...interface{}) {
	l.logger.Fatalf(format, args...)
}

func (l *AppLogger) SetOutput(output io.Writer) {
	l.logger.SetOutput(output)
}
