package di

import (
	"errors"
	"fmt"
	"runtime/debug"
)

// containerGetter contains all the functions that are useful
// to retrieve an object from a container.
type containerGetter struct{}

func (g *containerGetter) SafeGet(ctn *container, name string) (interface{}, error) {
	obj, err := g.get(ctn, name)
	if err != nil {
		ctn.logger.Error(fmt.Sprintf("could not build `%s` err=%s", name, err))
	}

	return obj, err
}

func (g *containerGetter) Get(ctn *container, name string) interface{} {
	obj, _ := ctn.SafeGet(name)
	return obj
}

func (g *containerGetter) Fill(ctn *container, name string, dst interface{}) error {
	obj, err := ctn.SafeGet(name)
	if err != nil {
		return err
	}

	return fill(obj, dst)
}

func (g *containerGetter) get(ctn *container, name string) (interface{}, error) {
	def, ok := ctn.definitions[name]
	if !ok {
		return nil, fmt.Errorf("could not find a Definition for `%s` in the Container", name)
	}

	if stringSliceContains(ctn.built, name) {
		return nil, fmt.Errorf("there is a cycle in object definitions : %v", ctn.built)
	}

	if ctn.scope != def.Scope {
		return g.getInParent(ctn, def)
	}

	return g.getInThisContainer(ctn, def)
}

func (g *containerGetter) getInThisContainer(ctn *container, def Definition) (interface{}, error) {
	ctn.m.Lock()
	closed := ctn.closed
	obj, ok := ctn.objects[def.Name]
	ctn.m.Unlock()

	if closed {
		return nil, errors.New("the Container has been deleted")
	}

	if ok {
		return obj, nil
	}

	return g.buildInThisContainer(ctn, def)
}

func (g *containerGetter) buildInThisContainer(ctn *container, def Definition) (interface{}, error) {
	obj, err := g.build(ctn, def)
	if err != nil {
		return nil, err
	}

	ctn.m.Lock()

	if ctn.closed {
		ctn.m.Unlock()
		ctn.Delete()
		return nil, errors.New("the Container has been deleted")
	}

	ctn.objects[def.Name] = obj

	ctn.m.Unlock()

	return obj, nil
}

func (g *containerGetter) getInParent(ctn *container, def Definition) (interface{}, error) {
	p, _ := ctn.containerLineage.Parent(ctn).(*container)

	if p.containerCore == nil {
		return nil, fmt.Errorf(
			"Definition of `%s` requires `%s` scope which does not match this Container scope or any of its parents scope",
			def.Name,
			def.Scope,
		)
	}

	if p.scope != def.Scope {
		return g.getInParent(p, def)
	}

	return g.getInThisContainer(p, def)
}

func (g *containerGetter) build(ctn *container, def Definition) (obj interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("could not build `%s` err=%s stack=%s", def.Name, r, debug.Stack())
		}
	}()

	obj, err = def.Build(&container{
		containerCore: ctn.containerCore,
		built:         append(ctn.built, def.Name),
		logger:        ctn.logger,
	})
	return
}
