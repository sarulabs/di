package di

import (
	"errors"
	"fmt"
)

// containerUnscopedGetter contains all the functions that are useful
// to retrieve an object from a container when the object
// is defined in a more specific scope.
type containerUnscopedGetter struct{}

func (g *containerUnscopedGetter) UnscopedGet(ctn *container, name string) interface{} {
	obj, err := ctn.UnscopedSafeGet(name)
	if err != nil {
		panic(err)
	}

	return obj
}

func (g *containerUnscopedGetter) UnscopedFill(ctn *container, name string, dst interface{}) error {
	obj, err := ctn.UnscopedSafeGet(name)
	if err != nil {
		return err
	}

	return fill(obj, dst)
}

func (g *containerUnscopedGetter) UnscopedSafeGet(ctn *container, name string) (interface{}, error) {
	def, ok := ctn.definitions[name]
	if !ok {
		return nil, fmt.Errorf("could not get `%s` because the definition does not exist", name)
	}

	if !ScopeList(ctn.SubScopes()).Contains(def.Scope) {
		return ctn.SafeGet(name)
	}

	return g.unscopedSafeGet(ctn, def)
}

func (g *containerUnscopedGetter) unscopedSafeGet(ctn *container, def Def) (interface{}, error) {
	if ctn.scope == def.Scope {
		return ctn.SafeGet(def.Name)
	}

	child, err := g.getUnscopedChild(ctn)
	if err != nil {
		return nil, fmt.Errorf("could not get `%s` because %+v", def.Name, err)
	}

	return g.unscopedSafeGet(child, def)
}

func (g *containerUnscopedGetter) getUnscopedChild(ctn *container) (*container, error) {
	ctn.m.Lock()
	unscopedChild := ctn.unscopedChild
	ctn.m.Unlock()

	if unscopedChild == nil {
		return g.addUnscopedChild(ctn)
	}

	return &container{
		containerCore: unscopedChild,
	}, nil
}

func (g *containerUnscopedGetter) addUnscopedChild(ctn *container) (*container, error) {
	child, err := ctn.containerLineage.createChild(ctn)
	if err != nil {
		return nil, err
	}

	ctn.m.Lock()

	if ctn.closed {
		ctn.m.Unlock()
		return nil, errors.New("the container is closed")
	}

	ctn.unscopedChild = child.containerCore

	ctn.m.Unlock()

	return child, nil
}
