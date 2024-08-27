package log

import (
	"github.com/Lyndon-Zhang/gira"
	"github.com/Lyndon-Zhang/gira/logger"
)

var defaultLogger gira.Logger

func init() {
	defaultLogger = logger.NewDefaultLogger()
}

func GetDefaultLogger() gira.Logger {
	return defaultLogger
}

func Config(config gira.LogConfig) error {
	var err error
	if defaultLogger, err = logger.NewConfigLogger(config); err != nil {
		return err
	} else {
		return nil
	}
}

func ConfigAsCli(logDir string) error {
	var err error
	if defaultLogger, err = logger.NewDefaultCliLogger(logDir); err != nil {
		return err
	} else {
		return nil
	}
}

func Info(args ...interface{}) {
	defaultLogger.Info(args...)
}

func Debug(args ...interface{}) {
	defaultLogger.Debug(args...)
}

func Error(args ...interface{}) {
	defaultLogger.Error(args...)
}

func Println(args ...interface{}) {
	defaultLogger.Info(args...)
}

func Fatal(args ...interface{}) {
	defaultLogger.Fatal(args...)
}

func Warn(args ...interface{}) {
	defaultLogger.Warn(args...)
}

func Infow(msg string, kvs ...interface{}) {
	defaultLogger.Infow(msg, kvs...)
}

func Debugw(msg string, kvs ...interface{}) {
	defaultLogger.Debugw(msg, kvs...)
}

func Errorw(msg string, kvs ...interface{}) {
	defaultLogger.Errorw(msg, kvs...)
}
func Fatalw(msg string, kvs ...interface{}) {
	defaultLogger.Fatalw(msg, kvs...)
}
func Warnw(msg string, kvs ...interface{}) {
	defaultLogger.Warnw(msg, kvs...)
}

func Printf(format string, args ...interface{}) {
	defaultLogger.Infof(format, args...)
}
func Infof(format string, args ...interface{}) {
	defaultLogger.Infof(format, args...)
}

func Debugf(format string, args ...interface{}) {
	defaultLogger.Debugf(format, args...)
}

func Errorf(format string, args ...interface{}) {
	defaultLogger.Errorf(format, args...)
}

func Fatalf(format string, args ...interface{}) {
	defaultLogger.Fatalf(format, args...)
}

func Warnf(format string, args ...interface{}) {
	defaultLogger.Warnf(format, args...)
}
