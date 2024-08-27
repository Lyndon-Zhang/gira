package gira

type Cron interface {
	AddFunc(spec string, cmd func()) error
}
