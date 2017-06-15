package di

import (
	"errors"
	"fmt"
	"runtime/debug"
)

// contextGetter contains all the functions that are useful
// to retrieve an object from a context.
type contextGetter struct{}

func (g *contextGetter) SafeGet(ctx *context, name string) (interface{}, error) {
	obj, err := g.get(ctx, name)
	if err != nil {
		ctx.logger.Error(fmt.Sprintf("could not build `%s` err=%s", name, err))
	}

	return obj, err
}

func (g *contextGetter) Get(ctx *context, name string) interface{} {
	obj, _ := ctx.SafeGet(name)
	return obj
}

func (g *contextGetter) Fill(ctx *context, name string, dst interface{}) error {
	obj, err := ctx.SafeGet(name)
	if err != nil {
		return err
	}

	return fill(obj, dst)
}

func (g *contextGetter) get(ctx *context, name string) (interface{}, error) {
	def, ok := ctx.definitions[name]
	if !ok {
		return nil, fmt.Errorf("could not find a Definition for `%s` in the Context", name)
	}

	if stringSliceContains(ctx.built, name) {
		return nil, fmt.Errorf("there is a cycle in object definitions : %v", ctx.built)
	}

	if ctx.scope != def.Scope {
		return g.getInParent(ctx, def)
	}

	return g.getInThisContext(ctx, def)
}

func (g *contextGetter) getInThisContext(ctx *context, def Definition) (interface{}, error) {
	ctx.m.Lock()
	closed := ctx.closed
	obj, ok := ctx.objects[def.Name]
	ctx.m.Unlock()

	if closed {
		return nil, errors.New("the Context has been deleted")
	}

	if ok {
		return obj, nil
	}

	return g.buildInThisContext(ctx, def)
}

func (g *contextGetter) buildInThisContext(ctx *context, def Definition) (interface{}, error) {
	obj, err := g.build(ctx, def)
	if err != nil {
		return nil, err
	}

	ctx.m.Lock()

	if ctx.closed {
		ctx.m.Unlock()
		ctx.Delete()
		return nil, errors.New("the Context has been deleted")
	}

	ctx.objects[def.Name] = obj

	ctx.m.Unlock()

	return obj, nil
}

func (g *contextGetter) getInParent(ctx *context, def Definition) (interface{}, error) {
	p, _ := ctx.contextLineage.Parent(ctx).(*context)

	if p.contextCore == nil {
		return nil, fmt.Errorf(
			"Definition of `%s` requires `%s` scope which does not match this Context scope or any of its parents scope",
			def.Name,
			def.Scope,
		)
	}

	if p.scope != def.Scope {
		return g.getInParent(p, def)
	}

	return g.getInThisContext(p, def)
}

func (g *contextGetter) build(ctx *context, def Definition) (obj interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("could not build `%s` err=%s stack=%s", def.Name, r, debug.Stack())
		}
	}()

	obj, err = def.Build(&context{
		contextCore: ctx.contextCore,
		built:       append(ctx.built, def.Name),
		logger:      ctx.logger,
	})
	return
}
