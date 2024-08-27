package gira

import (
	"reflect"

	"github.com/Lyndon-Zhang/gira/errors"
)

func TypeOf[T any]() reflect.Type {
	a := new(T)
	return reflect.TypeOf(a)
}

type Component interface {
	GetObject() *Object
	SetObject(object *Object) error
	Create() error
	Awake() error
	Start() error
	Destory() error
	GetComponents(t reflect.Type) []Component
	AddComponent(c Component) error
}

type BaseComponent struct {
	object *Object
}

func (c *BaseComponent) AddComponent(component Component) error {
	o := c.GetObject()
	if o == nil {
		return errors.ErrNullObject
	}
	return o.AddComponent(component)
}

func (c *BaseComponent) SetObject(object *Object) error {
	c.object = object
	return nil
}

func (c *BaseComponent) GetComponents(t reflect.Type) []Component {
	o := c.GetObject()
	if o == nil {
		return nil
	}
	return o.GetComponents(t)
}

func (c *BaseComponent) GetObject() *Object {
	return c.object
}

func (c *BaseComponent) Create() error {
	return nil
}

func (c *BaseComponent) Awake() error {
	return nil
}

func (c *BaseComponent) Start() error {
	return nil
}

func (c *BaseComponent) Destory() error {
	return nil
}
