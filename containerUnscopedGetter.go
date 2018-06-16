package di

import (
	"errors"
	"fmt"
)

// containerUnscopedGetter contains all the functions that are useful
// to retrieve an object from a container when the object
// is defined in a narrower scope.
type containerUnscopedGetter struct{}

func (g *containerUnscopedGetter) UnscopedSafeGet(ctn *container, name string) (interface{}, error) {
	def, ok := ctn.definitions[name]
	if !ok {
		return nil, fmt.Errorf("could not find a Definition for `%s` in the Container", name)
	}

	if !stringSliceContains(ctn.SubScopes(), def.Scope) {
		return ctn.SafeGet(name)
	}

	ctn.m.Lock()
	unscopedChild := ctn.unscopedChild
	ctn.m.Unlock()

	child := &container{
		containerCore: unscopedChild,
		built:         ctn.built,
		logger:        ctn.logger,
	}

	if child.containerCore != nil {
		return child.UnscopedSafeGet(name)
	}

	child, err := g.addUnscopedChild(ctn)
	if err != nil {
		return nil, err
	}

	return child.UnscopedSafeGet(name)
}

func (g *containerUnscopedGetter) addUnscopedChild(ctn *container) (*container, error) {
	child, err := ctn.containerLineage.createChild(ctn)
	if err != nil {
		return nil, err
	}

	ctn.m.Lock()

	if ctn.closed {
		ctn.m.Unlock()
		return nil, errors.New("the Container is closed")
	}

	ctn.unscopedChild = child.containerCore

	ctn.m.Unlock()

	return child, nil
}

func (g *containerUnscopedGetter) UnscopedGet(ctn *container, name string) interface{} {
	obj, _ := ctn.UnscopedSafeGet(name)
	return obj
}

func (g *containerUnscopedGetter) UnscopedFill(ctn *container, name string, dst interface{}) error {
	obj, err := ctn.UnscopedSafeGet(name)
	if err != nil {
		return err
	}

	return fill(obj, dst)
}
