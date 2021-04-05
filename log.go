package clientcontroller

import (
	"os"

	kitlog "github.com/go-kit/kit/log"
	kitlevel "github.com/go-kit/kit/log/level"
)

var (
	msgKey = "_msg"
)

type Logger interface {
	Debug(msg string, keyvals ...interface{})
	Info(msg string, keyvals ...interface{})
	Error(msg string, keyvals ...interface{})

	With(keyvals ...interface{}) Logger
}

type defaultLogger struct {
	logger kitlog.Logger
}

func newDefaultLogger() Logger {
	return &defaultLogger{
		logger: kitlog.NewLogfmtLogger(kitlog.NewSyncWriter(os.Stdout)),
	}
}

func (d *defaultLogger) Info(msg string, keyvals ...interface{}) {
	lWithLevel := kitlevel.Info(d.logger)
	kitlog.With(lWithLevel, msgKey, msg).Log(keyvals...)
}
func (d *defaultLogger) Debug(msg string, keyvals ...interface{}) {
	lWithLevel := kitlevel.Debug(d.logger)
	kitlog.With(lWithLevel, msgKey, msg).Log(keyvals...)
}
func (d *defaultLogger) Error(msg string, keyvals ...interface{}) {
	lWithLevel := kitlevel.Error(d.logger)
	kitlog.With(lWithLevel, msgKey, msg).Log(keyvals...)
}
func (d *defaultLogger) With(keyvals ...interface{}) Logger {
	return &defaultLogger{
		logger: kitlog.With(d.logger, keyvals),
	}
}
