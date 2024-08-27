package logger

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/Lyndon-Zhang/gira"
	"github.com/Lyndon-Zhang/gira/errors"
	"github.com/Lyndon-Zhang/gira/facade"
	"github.com/natefinch/lumberjack"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Logger struct {
	logger *zap.Logger
	sugar  *zap.SugaredLogger
}

func (l *Logger) Infow(msg string, kvs ...interface{}) {
	l.sugar.Infow(msg, kvs...)
}

func (l *Logger) Errorw(msg string, kvs ...interface{}) {
	l.sugar.Errorw(msg, kvs...)
}

func (l *Logger) Debugw(msg string, kvs ...interface{}) {
	l.sugar.Debugw(msg, kvs...)
}

func (l *Logger) Fatalw(msg string, kvs ...interface{}) {
	l.sugar.Fatalw(msg, kvs...)
}

func (l *Logger) Warnw(msg string, kvs ...interface{}) {
	l.sugar.Warnw(msg, kvs...)
}

func (l *Logger) Info(args ...interface{}) {
	l.sugar.Info(args...)
}

func (l *Logger) Error(args ...interface{}) {
	l.sugar.Error(args...)
}

func (l *Logger) Debug(args ...interface{}) {
	l.sugar.Debug(args...)
}

func (l *Logger) Fatal(args ...interface{}) {
	l.sugar.Fatal(args...)
}

func (l *Logger) Warn(args ...interface{}) {
	l.sugar.Warn(args...)
}

func (l *Logger) Infof(format string, args ...interface{}) {
	l.sugar.Infof(format, args...)
}

func (l *Logger) Errorf(format string, args ...interface{}) {
	l.sugar.Errorf(format, args...)
}

func (l *Logger) Debugf(format string, args ...interface{}) {
	l.sugar.Debugf(format, args...)
}

func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.sugar.Fatalf(format, args...)
}

func (l *Logger) Warnf(format string, args ...interface{}) {
	l.sugar.Warnf(format, args...)
}

func (l *Logger) Named(s string) gira.Logger {
	logger := l.logger.Named(s)
	sugar := logger.Sugar()
	return &Logger{
		sugar:  sugar,
		logger: logger,
	}
}

const (
	EncoderKey_time       = "time"
	EncoderKey_level      = "level"
	EncoderKey_name       = "logger"
	EncoderKey_caller     = "caller"
	EncoderKey_message    = "msg"
	EncoderKey_stacktrace = "stacktrace"
)

type MongoSink struct {
	collection  string
	writeOption *options.InsertOneOptions
}

func (s *MongoSink) Write(data []byte) (n int, err error) {
	client := facade.GetLogDbClient()
	if client == nil {
		return
	}
	switch c := client.(type) {
	case gira.MongoClient:
		doc := bson.M{}
		if err = json.Unmarshal(data, &doc); err != nil {
			return
		}
		doc["app_id"] = facade.GetAppId()
		doc["app_full_name"] = facade.GetAppFullName()
		database := c.GetMongoDatabase()
		_, err = database.Collection(s.collection).InsertOne(context.Background(), doc, s.writeOption)
		return
	default:
		err = errors.ErrDbNotSupport
		return
	}
}

func NewDefaultCliLogger(logDir string) (*Logger, error) {
	// 配置日志输出
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        EncoderKey_time,
		LevelKey:       EncoderKey_level,
		NameKey:        EncoderKey_name,
		CallerKey:      EncoderKey_caller,
		MessageKey:     EncoderKey_message,
		StacktraceKey:  EncoderKey_stacktrace,
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
	consoleCore := zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoderConfig), // 控制台输出格式
		zapcore.AddSync(os.Stdout),               // 输出到控制台
		zap.NewAtomicLevelAt(zap.DebugLevel),
	)
	// 滚动日志配置
	rollingCfg := zapcore.EncoderConfig{
		TimeKey:        EncoderKey_time,
		LevelKey:       EncoderKey_level,
		NameKey:        EncoderKey_name,
		CallerKey:      EncoderKey_caller,
		MessageKey:     EncoderKey_message,
		StacktraceKey:  EncoderKey_stacktrace,
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
	rollingCore := zapcore.NewCore(
		zapcore.NewConsoleEncoder(rollingCfg), // 滚动日志输出格式
		// zapcore.NewJSONEncoder(rollingCfg), // 滚动日志输出格式
		zapcore.AddSync(&lumberjack.Logger{
			Filename:   filepath.Join(logDir, "cli.log"), // 日志文件路径
			MaxSize:    10 * 1024,                        // 每个日志文件的最大大小，单位为 MB
			MaxBackups: 10,                               // 保留的旧日志文件的最大个数
			MaxAge:     30,                               // 保留的旧日志文件的最大天数
			Compress:   true,                             // 是否压缩旧日志文件
		}),
		zap.NewAtomicLevelAt(zap.DebugLevel),
	)
	// 创建日志对象
	logger := zap.New(zapcore.NewTee(consoleCore, rollingCore))
	logger = logger.WithOptions(zap.WithCaller(true), zap.AddCallerSkip(2))
	sugar := logger.Sugar()
	l := &Logger{
		logger: logger,
		sugar:  sugar,
	}
	return l, nil
}

func NewConfigLogger(config gira.LogConfig) (*Logger, error) {
	cores := make([]zapcore.Core, 0)
	// 1.控制台输出
	if config.Console {
		var level zap.AtomicLevel
		err := level.UnmarshalText([]byte(config.Level))
		if err != nil {
			return nil, err
		}
		enabler := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
			return lvl >= level.Level()
		})
		encoderConfig := zapcore.EncoderConfig{
			TimeKey:        EncoderKey_time,
			LevelKey:       EncoderKey_level,
			NameKey:        EncoderKey_name,
			CallerKey:      EncoderKey_caller,
			MessageKey:     EncoderKey_message,
			StacktraceKey:  EncoderKey_stacktrace,
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		}
		consoleCore := zapcore.NewCore(
			zapcore.NewConsoleEncoder(encoderConfig), // 控制台输出格式
			zapcore.AddSync(os.Stdout),               // 输出到控制台
			enabler,
		)
		cores = append(cores, consoleCore)
	}

	// 2.滚动文件输出
	for _, file := range config.Files {
		var enabler zap.LevelEnablerFunc
		if file.Filter != "" {
			var level zap.AtomicLevel
			err := level.UnmarshalText([]byte(file.Filter))
			if err != nil {
				return nil, err
			}
			enabler = zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
				return lvl == level.Level()
			})
		} else if file.Level != "" {
			var level zap.AtomicLevel
			err := level.UnmarshalText([]byte(file.Level))
			if err != nil {
				return nil, err
			}
			enabler = zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
				return lvl >= level.Level()
			})
		} else if config.Level != "" {
			var level zap.AtomicLevel
			err := level.UnmarshalText([]byte(config.Level))
			if err != nil {
				return nil, err
			}
			enabler = zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
				return lvl >= level.Level()
			})
		}
		encoderCfg := zapcore.EncoderConfig{
			TimeKey:        EncoderKey_time,
			LevelKey:       EncoderKey_level,
			NameKey:        EncoderKey_name,
			CallerKey:      EncoderKey_caller,
			MessageKey:     EncoderKey_message,
			StacktraceKey:  EncoderKey_stacktrace,
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		}
		var format zapcore.Encoder
		if file.Format == "console" {
			format = zapcore.NewConsoleEncoder(encoderCfg)
		} else {
			format = zapcore.NewJSONEncoder(encoderCfg)
		}
		rollingCore := zapcore.NewCore(
			format, // 滚动日志输出格式
			zapcore.AddSync(&lumberjack.Logger{
				Filename:   file.Path,       // 日志文件路径
				MaxSize:    file.MaxSize,    // 每个日志文件的最大大小，单位为 MB
				MaxBackups: file.MaxBackups, // 保留的旧日志文件的最大个数
				MaxAge:     file.MaxAge,     // 保留的旧日志文件的最大天数
				Compress:   file.Compress,   // 是否压缩旧日志文件
			}),
			enabler,
		)
		cores = append(cores, rollingCore)
	}
	// 3.数据库输出
	if config.Db {
		var level zap.AtomicLevel
		err := level.UnmarshalText([]byte(config.DbLevel))
		if err != nil {
			return nil, err
		}
		enabler := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
			return lvl >= level.Level()
		})
		encoderCfg := zapcore.EncoderConfig{
			TimeKey:        EncoderKey_time,
			LevelKey:       EncoderKey_level,
			NameKey:        EncoderKey_name,
			CallerKey:      EncoderKey_caller,
			MessageKey:     EncoderKey_message,
			StacktraceKey:  EncoderKey_stacktrace,
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		}
		mongoCore := zapcore.NewCore(
			zapcore.NewJSONEncoder(encoderCfg), // 滚动日志输出格式
			zapcore.AddSync(&MongoSink{
				collection: "log",
				writeOption: options.InsertOne().
					SetBypassDocumentValidation(true),
			}),
			enabler,
		)
		cores = append(cores, mongoCore)
	}

	// 创建日志对象
	logger := zap.New(zapcore.NewTee(cores...))
	logger = logger.WithOptions(zap.WithCaller(true), zap.AddCallerSkip(2))
	sugar := logger.Sugar()
	l := &Logger{
		logger: logger,
		sugar:  sugar,
	}
	return l, nil
}

func NewDefaultLogger() *Logger {
	logger, _ := zap.NewDevelopment()
	logger = logger.WithOptions(zap.AddCallerSkip(2))
	sugar := logger.Sugar()
	l := &Logger{
		logger: logger,
		sugar:  sugar,
	}
	return l
}
