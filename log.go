package gira

type Logger interface {
	Infow(msg string, kvs ...interface{})
	Errorw(msg string, kvs ...interface{})
	Debugw(msg string, kvs ...interface{})
	Fatalw(msg string, kvs ...interface{})
	Warnw(msg string, kvs ...interface{})
	Info(args ...interface{})
	Error(args ...interface{})
	Debug(args ...interface{})
	Fatal(args ...interface{})
	Warn(args ...interface{})
	Infof(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Debugf(format string, args ...interface{})
	Fatalf(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Named(s string) Logger
}
