package di

import (
	"errors"
	"fmt"
	"runtime/debug"
)

// context is the implementation of the Context interface
type context struct {
	// contextCore contains the context data.
	// Serveral contexts can share the same contextCore.
	// In this case these contexts represent the same entity,
	// but at a different stage in an object construction.
	*contextCore

	// built contains the name of the Definition beeing built by this context.
	// It is used to avoid cycles in object Definitions.
	// Each time a Context is passed in parameter of the Build function
	// of a definition, this is in fact a new context.
	// This context is created with a built attribute
	// updated with the name of the Definition.
	built []string
}

func (ctx context) Parent() Context {
	return context{
		contextCore: ctx.getParent(),
		built:       ctx.built,
	}
}

func (ctx context) SubContext() (Context, error) {
	child, err := ctx.createChild()
	if err != nil {
		return nil, err
	}

	ctx.m.Lock()

	if ctx.closed {
		return nil, errors.New("the Context is closed")
	}

	ctx.children = append(ctx.children, child.contextCore)

	ctx.m.Unlock()

	return child, nil
}

func (ctx context) createChild() (*context, error) {
	subscopes := ctx.SubScopes()

	if len(subscopes) == 0 {
		return nil, fmt.Errorf("there is no narrower scope than `%s`", ctx.scope)
	}

	return &context{
		contextCore: &contextCore{
			logger:      ctx.logger,
			scope:       subscopes[0],
			scopes:      ctx.scopes,
			definitions: ctx.definitions,
			parent:      ctx.contextCore,
			children:    []*contextCore{},
			nastyChild:  nil,
			objects:     map[string]interface{}{},
		},
		built: ctx.built,
	}, nil
}

func (ctx context) SafeGet(name string) (interface{}, error) {
	obj, err := ctx.get(name)
	if err != nil {
		ctx.logger.Error(fmt.Sprintf("could not build `%s` err=%s", name, err))
	}

	return obj, err
}

func (ctx context) Get(name string) interface{} {
	obj, _ := ctx.SafeGet(name)
	return obj
}

func (ctx context) Fill(name string, dst interface{}) error {
	obj, err := ctx.SafeGet(name)
	if err != nil {
		return err
	}

	return fill(obj, dst)
}

func (ctx context) get(name string) (interface{}, error) {
	def, ok := ctx.definitions[name]
	if !ok {
		return nil, fmt.Errorf("could not find a Definition for `%s` in the Context", name)
	}

	if stringSliceContains(ctx.built, name) {
		return nil, fmt.Errorf("there is a cycle in object definitions : %v", ctx.built)
	}

	if ctx.scope != def.Scope {
		return ctx.getInParent(def)
	}

	return ctx.getInThisContext(def)
}

func (ctx context) getInThisContext(def Definition) (interface{}, error) {
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

	obj, err := ctx.build(def)
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

func (ctx context) getInParent(def Definition) (interface{}, error) {
	p := context{
		contextCore: ctx.getParent(),
		built:       ctx.built,
	}

	if p.contextCore == nil {
		return nil, fmt.Errorf(
			"Definition of `%s` requires `%s` scope which does not match this Context scope or any of its parents scope",
			def.Name,
			def.Scope,
		)
	}

	if p.scope != def.Scope {
		return p.getInParent(def)
	}

	return p.getInThisContext(def)
}

func (ctx context) build(def Definition) (obj interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("could not build `%s` err=%s stack=%s", def.Name, r, debug.Stack())
		}
	}()

	obj, err = def.Build(&context{
		contextCore: ctx.contextCore,
		built:       append(ctx.built, def.Name),
	})
	return
}

func (ctx context) NastySafeGet(name string) (interface{}, error) {
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

	child, err := ctx.addNastyChild()
	if err != nil {
		return nil, err
	}

	return child.NastySafeGet(name)
}

func (ctx context) addNastyChild() (*context, error) {
	child, err := ctx.createChild()
	if err != nil {
		return nil, err
	}

	ctx.m.Lock()

	if ctx.closed {
		return nil, errors.New("the Context is closed")
	}

	ctx.nastyChild = child.contextCore

	ctx.m.Unlock()

	return child, nil
}

func (ctx context) NastyGet(name string) interface{} {
	obj, _ := ctx.NastySafeGet(name)
	return obj
}

func (ctx context) NastyFill(name string, dst interface{}) error {
	obj, err := ctx.NastySafeGet(name)
	if err != nil {
		return err
	}

	return fill(obj, dst)
}
