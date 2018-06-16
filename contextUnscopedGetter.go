package di

import (
	"errors"
	"fmt"
)

// contextUnscopedGetter contains all the functions that are useful
// to retrieve an object from a context when the object
// is defined in a narrower scope.
type contextUnscopedGetter struct{}

func (g *contextUnscopedGetter) UnscopedSafeGet(ctx *context, name string) (interface{}, error) {
	def, ok := ctx.definitions[name]
	if !ok {
		return nil, fmt.Errorf("could not find a Definition for `%s` in the Context", name)
	}

	if !stringSliceContains(ctx.SubScopes(), def.Scope) {
		return ctx.SafeGet(name)
	}

	ctx.m.Lock()
	unscopedChild := ctx.unscopedChild
	ctx.m.Unlock()

	child := &context{
		contextCore: unscopedChild,
		built:       ctx.built,
		logger:      ctx.logger,
	}

	if child.contextCore != nil {
		return child.UnscopedSafeGet(name)
	}

	child, err := g.addUnscopedChild(ctx)
	if err != nil {
		return nil, err
	}

	return child.UnscopedSafeGet(name)
}

func (g *contextUnscopedGetter) addUnscopedChild(ctx *context) (*context, error) {
	child, err := ctx.contextLineage.createChild(ctx)
	if err != nil {
		return nil, err
	}

	ctx.m.Lock()

	if ctx.closed {
		ctx.m.Unlock()
		return nil, errors.New("the Context is closed")
	}

	ctx.unscopedChild = child.contextCore

	ctx.m.Unlock()

	return child, nil
}

func (g *contextUnscopedGetter) UnscopedGet(ctx *context, name string) interface{} {
	obj, _ := ctx.UnscopedSafeGet(name)
	return obj
}

func (g *contextUnscopedGetter) UnscopedFill(ctx *context, name string, dst interface{}) error {
	obj, err := ctx.UnscopedSafeGet(name)
	if err != nil {
		return err
	}

	return fill(obj, dst)
}
