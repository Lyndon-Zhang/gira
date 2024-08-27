package syncs

import "sync"

func OnceFunc(f func()) func() {
	var once sync.Once
	return func() {
		once.Do(f)
	}
}
