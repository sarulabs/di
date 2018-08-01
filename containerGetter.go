package di

import (
	"fmt"
)

// containerGetter contains all the functions that are useful
// to retrieve an object from a container.
type containerGetter struct{}

func (g *containerGetter) Get(ctn *container, name string) interface{} {
	obj, err := ctn.SafeGet(name)
	if err != nil {
		panic(err)
	}

	return obj
}

func (g *containerGetter) Fill(ctn *container, name string, dst interface{}) error {
	obj, err := ctn.SafeGet(name)
	if err != nil {
		return err
	}

	return fill(obj, dst)
}

func (g *containerGetter) SafeGet(ctn *container, name string) (interface{}, error) {
	def, ok := ctn.definitions[name]
	if !ok {
		return nil, fmt.Errorf("could get `%s` because the definition does not exist", name)
	}

	if ctn.scope != def.Scope {
		return g.getInParent(ctn, def)
	}

	return g.getInThisContainer(ctn, def)
}

func (g *containerGetter) getInThisContainer(ctn *container, def Def) (interface{}, error) {
	ctn.m.RLock()
	closed := ctn.closed
	obj, ok := ctn.objects[def.Name]
	ctn.m.RUnlock()

	if closed {
		return nil, fmt.Errorf("could not get `%s` because the container has been deleted", def.Name)
	}

	if ok {
		return obj, nil
	}

	return g.buildInThisContainer(ctn, def)
}

func (g *containerGetter) buildInThisContainer(ctn *container, def Def) (interface{}, error) {
	obj, err := g.build(ctn, def)
	if err != nil {
		return nil, err
	}

	ctn.m.Lock()

	if ctn.closed {
		ctn.m.Unlock()
		err = ctn.containerSlayer.closeObject(obj, def)
		return nil, fmt.Errorf(
			"could not get `%s` because the container has been deleted, the object has been created and closed%s",
			def.Name, g.formatCloseErr(err),
		)
	}

	ctn.objects[def.Name] = obj

	ctn.m.Unlock()

	return obj, nil
}

func (g *containerGetter) formatCloseErr(err error) string {
	if err == nil {
		return ""
	}
	return " (with an error: " + err.Error() + ")"
}

func (g *containerGetter) getInParent(ctn *container, def Def) (interface{}, error) {
	p := ctn.containerLineage.parent(ctn)

	if p.containerCore == nil {
		return nil, fmt.Errorf(
			"could not get `%s` because it requires `%s` scope which does not match this container scope or any of its parents scope",
			def.Name, def.Scope,
		)
	}

	if p.scope != def.Scope {
		return g.getInParent(p, def)
	}

	return g.getInThisContainer(p, def)
}

func (g *containerGetter) build(ctn *container, def Def) (obj interface{}, err error) {
	if ctn.builtList.Has(def.Name) {
		return nil, fmt.Errorf(
			"could not build `%s` because there is a cycle in the object definitions (%v)",
			def.Name, ctn.builtList.OrderedList(),
		)
	}

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("could not build `%s` because the build function panicked: %s", def.Name, r)
		}
	}()

	obj, err = def.Build(&container{
		containerCore: ctn.containerCore,
		builtList:     ctn.builtList.Add(def.Name),
	})

	if err != nil {
		return nil, fmt.Errorf("could not build `%s`: %s", def.Name, err.Error())
	}

	return obj, nil
}
