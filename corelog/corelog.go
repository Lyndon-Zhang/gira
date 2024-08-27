package corelog

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

func Infow(msg string, keysAndValues ...interface{}) {
	defaultLogger.Infow(msg, keysAndValues...)
}

func Debugw(msg string, keysAndValues ...interface{}) {
	defaultLogger.Debugw(msg, keysAndValues...)
}

func Errorw(msg string, keysAndValues ...interface{}) {
	defaultLogger.Errorw(msg, keysAndValues...)
}
func Fatalw(msg string, keysAndValues ...interface{}) {
	defaultLogger.Fatalw(msg, keysAndValues...)
}
func Warnw(msg string, keysAndValues ...interface{}) {
	defaultLogger.Warnw(msg, keysAndValues...)
}

func Printf(template string, args ...interface{}) {
	defaultLogger.Infof(template, args...)
}
func Infof(template string, args ...interface{}) {
	defaultLogger.Infof(template, args...)
}

func Debugf(template string, args ...interface{}) {
	defaultLogger.Debugf(template, args...)
}

func Errorf(template string, args ...interface{}) {
	defaultLogger.Errorf(template, args...)
}

func Fatalf(template string, args ...interface{}) {
	defaultLogger.Fatalf(template, args...)
}

func Warnf(template string, args ...interface{}) {
	defaultLogger.Warnf(template, args...)
}
