package di

import (
	"errors"
	"fmt"
)

// contextNastyGetter contains all the functions that are useful
// to retrieve an object from a context when the object
// is defined in a narrower scope.
type contextNastyGetter struct{}

func (g *contextNastyGetter) NastySafeGet(ctx context, name string) (interface{}, error) {
	def, ok := ctx.definitions[name]
	if !ok {
		return nil, fmt.Errorf("could not find a Definition for `%s` in the Context", name)
	}

	if !stringSliceContains(ctx.SubScopes(), def.Scope) {
		return ctx.SafeGet(name)
	}

	ctx.m.Lock()
	nastyChild := ctx.nastyChild
	ctx.m.Unlock()

	child := &context{
		contextCore: nastyChild,
		built:       ctx.built,
	}

	if child.contextCore != nil {
		return child.NastySafeGet(name)
	}

	child, err := g.addNastyChild(ctx)
	if err != nil {
		return nil, err
	}

	return child.NastySafeGet(name)
}

func (g *contextNastyGetter) addNastyChild(ctx context) (*context, error) {
	child, err := ctx.contextLineage.createChild(ctx)
	if err != nil {
		return nil, err
	}

	ctx.m.Lock()

	if ctx.closed {
		ctx.m.Unlock()
		return nil, errors.New("the Context is closed")
	}

	ctx.nastyChild = child.contextCore

	ctx.m.Unlock()

	return child, nil
}

func (g *contextNastyGetter) NastyGet(ctx context, name string) interface{} {
	obj, _ := ctx.NastySafeGet(name)
	return obj
}

func (g *contextNastyGetter) NastyFill(ctx context, name string, dst interface{}) error {
	obj, err := ctx.NastySafeGet(name)
	if err != nil {
		return err
	}

	return fill(obj, dst)
}
