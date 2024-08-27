package gira

import (
	"context"
	"reflect"

	"log"

	"golang.org/x/sync/errgroup"
)

type Scene struct {
	object     *Object
	cancelCtx  context.Context
	cancelFunc context.CancelFunc
	errCtx     context.Context
	errGroup   *errgroup.Group
}

func CreateScene() *Scene {
	//app := App()
	cancelCtx, cancel := context.WithCancel(context.TODO())
	errGroup, errCtx := errgroup.WithContext(cancelCtx)
	scene := &Scene{
		cancelCtx:  cancelCtx,
		cancelFunc: cancel,
		errCtx:     errCtx,
		errGroup:   errGroup,
	}
	scene.object = scene.CreateObject()
	return scene
}

func (s *Scene) CreateObject() *Object {
	object := &Object{
		componentDict: make(map[reflect.Type][]Component),
		scene:         s,
	}
	return object
}

func (s *Scene) forver() error {
	select {
	case <-s.cancelCtx.Done():
		{
			log.Println("scene done")
			break
		}
	}
	return nil
}

func (s *Scene) Cancel() {
	s.cancelFunc()
}

func (s *Scene) Go(f func() error) {
	s.errGroup.Go(f)
}
