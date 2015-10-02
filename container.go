// Package "di" contains a very simple dependency injection container.
// It allows you to store and retrieve global services or parameters.
package di

import (
	"errors"
	"fmt"
	"reflect"
)

const (
	// Types of entry in the container
	Factory = 1 + iota
	Singleton
	Instance
	Alias
)

// Container should be constructed using NewContainer in order to work
type Container struct {
	types     map[string]int
	factories map[string]func() interface{}
	instances map[string]interface{}
	aliases   map[string]string
}

// NewContainer builds a Container by initializing all its fields.
func NewContainer() *Container {
	return &Container{
		types:     map[string]int{},
		factories: map[string]func() interface{}{},
		instances: map[string]interface{}{},
		aliases:   map[string]string{},
	}
}

// AlreadySetError is returned when you try to set an already existing entry in the container.
type AlreadySetError struct {
	entry string
}

func (e *AlreadySetError) Error() string {
	return fmt.Sprintf("container already contains an entry for `%s`", e.entry)
}

// NotSetError is returned when you try to get a nonexistent entry in the container.
type NotSetError struct {
	entry string
}

func (e *NotSetError) Error() string {
	return fmt.Sprintf("container does not contain an entry for `%s`", e.entry)
}

// Make retrieves item from the container.
// The item is placed in the given destination.
// The destination should be a pointer to the entry type.
// If the item does not exist in the container or if it can not be
// placed in the destination, an error is return and the destination
// is not modified.
func (c *Container) Make(name string, dest interface{}) error {
	n := c.ResolveName(name)

	if n == "" {
		return &NotSetError{name}
	}

	t := c.types[n]
	i, built := c.instances[n]

	if !built || t == Factory {
		i = c.factories[n]()
	}
	if !built && t == Singleton {
		c.instances[n] = i
	}

	return c.fill(i, dest)
}

// fill copy an interface into another using reflection.
func (c *Container) fill(src, dest interface{}) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.New("destination should be a pointer to the entry type")
		} else {
			err = nil
		}
	}()

	reflect.ValueOf(dest).Elem().Set(reflect.ValueOf(src))
	return
}

// func (c *Container) fill(src, dest interface{}) error {
// 	destV := reflect.ValueOf(dest)

// 	if destV.Kind() != reflect.Ptr {
// 		return errors.New("destination should be a pointer")
// 	}

// 	if !destV.IsValid() {
// 		return errors.New("destination is not valid")
// 	}

// 	if destV == reflect.Zero(destV.Type()) {
// 		return errors.New("destination has zero value")
// 	}

// 	destEV := destV.Elem()
// 	srcV := reflect.ValueOf(src)

// 	if destEV.Type() != srcV.Type() {
// 		return fmt.Errorf("destination should be a pointer to type %s but has type %s", srcV.Type(), destV.Type())
// 	}

// 	if !destEV.CanSet() {
// 		return errors.New("destination can not be set")
// 	}

// 	destEV.Set(srcV)

// 	return nil
// }

// ResolveName returns the real name of an entry by fowling aliases.
func (c *Container) ResolveName(name string) string {
	switch t, _ := c.types[name]; t {
	case Alias:
		return c.ResolveName(c.aliases[name])
	case Factory, Singleton, Instance:
		return name
	}
	return ""
}

// Factory binds an item to the container. The item is defined by a factory function.
// Each call to Make will call the factory and return its result.
// If there is already an entry with this name, an error is returned and the container keeps the old entry.
func (c *Container) Factory(name string, factory func() interface{}) error {
	if _, ok := c.types[name]; ok {
		return &AlreadySetError{name}
	}
	c.types[name] = Factory
	c.factories[name] = factory
	return nil
}

// Singleton does the same as Factory except that the first call
// to Make will build the item and the next ones will return
// the same item without calling the factory again.
func (c *Container) Singleton(name string, factory func() interface{}) error {
	if _, ok := c.types[name]; ok {
		return &AlreadySetError{name}
	}
	c.types[name] = Singleton
	c.factories[name] = factory
	return nil
}

// Instance directly bind an item to the container.
// Each call to Make will return this item.
// If there is already an entry with this name, an error is returned and the container keeps the old entry.
func (c *Container) Instance(name string, instance interface{}) error {
	if _, ok := c.types[name]; ok {
		return &AlreadySetError{name}
	}
	c.types[name] = Instance
	c.instances[name] = instance
	return nil
}

// Alias define an alias for an already registered entry.
// If there is already an entry with the name of the alias,
// or if the destination is not an existing entry,
// an error is returned and the container is not modified.
func (c *Container) Alias(alias, entry string) error {
	if _, ok := c.types[alias]; ok {
		return &AlreadySetError{entry}
	}
	if _, ok := c.types[entry]; !ok {
		return &NotSetError{entry}
	}
	c.types[alias] = Alias
	c.aliases[alias] = entry
	return nil
}
