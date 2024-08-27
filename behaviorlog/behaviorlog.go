package behaviorlog

import (
	"github.com/Lyndon-Zhang/gira"
	"github.com/Lyndon-Zhang/gira/logger"
)

var defaultLogger gira.Logger

func init() {
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
	if defaultLogger == nil {
		return
	}
	defaultLogger.Info(args...)
}

func Debug(args ...interface{}) {
	if defaultLogger == nil {
		return
	}
	defaultLogger.Debug(args...)
}

func Error(args ...interface{}) {
	if defaultLogger == nil {
		return
	}
	defaultLogger.Error(args...)
}

func Println(args ...interface{}) {
	if defaultLogger == nil {
		return
	}
	defaultLogger.Info(args...)
}

func Fatal(args ...interface{}) {
	if defaultLogger == nil {
		return
	}
	defaultLogger.Fatal(args...)
}

func Warn(args ...interface{}) {
	if defaultLogger == nil {
		return
	}
	defaultLogger.Warn(args...)
}

func Infow(msg string, keysAndValues ...interface{}) {
	if defaultLogger == nil {
		return
	}
	defaultLogger.Infow(msg, keysAndValues...)
}

func Debugw(msg string, keysAndValues ...interface{}) {
	if defaultLogger == nil {
		return
	}
	defaultLogger.Debugw(msg, keysAndValues...)
}

func Errorw(msg string, keysAndValues ...interface{}) {
	if defaultLogger == nil {
		return
	}
	defaultLogger.Errorw(msg, keysAndValues...)
}
func Fatalw(msg string, keysAndValues ...interface{}) {
	if defaultLogger == nil {
		return
	}
	defaultLogger.Fatalw(msg, keysAndValues...)
}
func Warnw(msg string, keysAndValues ...interface{}) {
	if defaultLogger == nil {
		return
	}
	defaultLogger.Warnw(msg, keysAndValues...)
}

func Printf(template string, args ...interface{}) {
	if defaultLogger == nil {
		return
	}
	defaultLogger.Infof(template, args...)
}

func Infof(template string, args ...interface{}) {
	if defaultLogger == nil {
		return
	}
	defaultLogger.Infof(template, args...)
}

func Debugf(template string, args ...interface{}) {
	if defaultLogger == nil {
		return
	}
	defaultLogger.Debugf(template, args...)
}

func Errorf(template string, args ...interface{}) {
	if defaultLogger == nil {
		return
	}
	defaultLogger.Errorf(template, args...)
}

func Fatalf(template string, args ...interface{}) {
	if defaultLogger == nil {
		return
	}
	defaultLogger.Fatalf(template, args...)
}

func Warnf(template string, args ...interface{}) {
	if defaultLogger == nil {
		return
	}
	defaultLogger.Warnf(template, args...)
}
