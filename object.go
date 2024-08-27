package gira

import (
	"reflect"

	"github.com/Lyndon-Zhang/gira/errors"
)

type Object struct {
	bbb           int
	scene         *Scene
	componentDict map[reflect.Type][]Component
}

func AddComponent(self Component, c Component) error {
	o := self.GetObject()
	if o == nil {
		return errors.ErrNullObject
	}
	return o.AddComponent(c)
}

func GetComponent[T Component](self Component, t T) (c T) {
	typ := reflect.TypeOf(t)
	o := self.GetObject()
	if o == nil {
		return
	}
	ci := o.GetComponent(typ)
	var ok bool
	if c, ok = ci.(T); ok {
		return
	}
	return
}

func GetComponents[T Component](self Component, t T) []Component {
	typ := reflect.TypeOf(t)
	o := self.GetObject()
	if o == nil {
		return nil
	}
	return o.GetComponents(typ)
}

func (o *Object) GetComponents(t reflect.Type) []Component {
	if arr, ok := o.componentDict[t]; ok {
		return arr
	} else {
		return nil
	}
}

func (o *Object) GetComponent(t reflect.Type) Component {
	if arr, ok := o.componentDict[t]; ok {
		if len(arr) > 0 {
			return arr[0]
		} else {
			return nil
		}
	} else {
		return nil
	}
}

func (o *Object) AddComponent(component Component) error {
	if component == nil {
		return errors.ErrNullPointer
	}
	t := reflect.TypeOf(component)
	if arr, ok := o.componentDict[t]; ok {
		arr = append(arr, component)
	} else {
		arr = make([]Component, 0)
		arr = append(arr, component)
		o.componentDict[t] = arr
	}
	component.SetObject(o)
	if err := component.Awake(); err != nil {
		return err
	}
	return nil
}
