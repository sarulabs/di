package di

import (
	"errors"
	"fmt"
	"runtime/debug"
)

// Context represents a dependency injection container.
// A Context has a scope and may have a parent with a wider scope
// and children with a narrower scope.
// Objects can be retrieved from the Context.
// If the desired object does not already exist in the Context,
// it is built thanks to the object Definition.
// The following attempts to get this object will return the same object.
type Context interface {
	// Definition returns the map of the available Definitions ordered by Definition name.
	// These Definitions represent all the objects that this Context can build.
	Definitions() map[string]Definition

	// Scope returns the Context scope.
	Scope() string

	// Scopes returns the list of available scopes.
	Scopes() []string

	// ParentScopes returns the list of scopes  wider than the Context scope.
	ParentScopes() []string

	// SubScopes returns the list of scopes narrower than the Context scope.
	SubScopes() []string

	// Parent returns the parent Context.
	Parent() Context

	// SubContext creates a new Context in the next subscope
	// that will have this Container as parent.
	SubContext() (Context, error)

	// SafeGet retrieves an object from the Context.
	// The object has to belong to this scope or a wider one.
	// If the object does not already exist, it is created and saved in the Context.
	// If the object can't be created, it returns an error.
	SafeGet(name string) (interface{}, error)

	// Get is similar to SafeGet but it does not return the error.
	Get(name string) interface{}

	// Fill is similar to SafeGet but it does not return the object.
	// Instead it fills the provided object with the value returned by SafeGet.
	// The provided object must be a pointer to the value returned by SafeGet.
	Fill(name string, dst interface{}) error

	// NastySafeGet retrieves an object from the Context, like SafeGet.
	// The difference is that the object can be retrieved
	// even if it belongs to a narrower scope.
	// To do so NastySafeGet creates a subcontext.
	// When the created object is no longer needed,
	// it is important to use the Clean method to Delete this subcontext.
	NastySafeGet(name string) (interface{}, error)

	// NastyGet is similar to NastySafeGet but it does not return the error.
	NastyGet(name string) interface{}

	// NastyFill is similar to NastySafeGet but copies the object in dst instead of returning it.
	NastyFill(name string, dst interface{}) error

	// Clean deletes the subcontext created by NastySafeGet, NastyGet or NastyFill.
	Clean()

	// Delete takes all the objects saved in this Context
	// and calls the Close function of their Definition on them.
	// It will also call Delete on each child and remove its reference in the parent Context.
	// After deletion, the Context can no longer be used.
	Delete()

	// IsClosed retuns true if the Context has been deleted.
	IsClosed() bool
}

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
		msg := fmt.Sprintf("could not build `%s` err=%s", name, err)
		ctx.logger.Error(msg)
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
	obj, err := ctx.reuseObject(def.Name)
	if err == nil {
		return obj, err
	}

	obj, err = ctx.build(def)
	if err != nil {
		return nil, err
	}

	err = ctx.saveObject(def.Name, obj)
	if err != nil {
		return nil, err
	}

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

func (ctx context) reuseObject(name string) (interface{}, error) {
	ctx.m.Lock()
	obj, ok := ctx.objects[name]
	ctx.m.Unlock()

	if ok {
		return obj, nil
	}

	return nil, fmt.Errorf("could not find `%s`", name)
}

func (ctx context) saveObject(name string, obj interface{}) error {
	ctx.m.Lock()

	if ctx.closed {
		ctx.m.Unlock()
		ctx.Delete()
		return errors.New("the Context has been deleted")
	}

	ctx.objects[name] = obj

	ctx.m.Unlock()

	return nil
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

	var err error

	child := &context{
		contextCore: ctx.getNastyChild(),
		built:       ctx.built,
	}

	if child.contextCore != nil {
		return child.NastySafeGet(name)
	}

	child, err = ctx.addNastyChild()
	if err != nil {
		return nil, err
	}

	return child.NastySafeGet(name)
}

func (ctx context) getNastyChild() *contextCore {
	ctx.m.Lock()
	defer ctx.m.Unlock()
	return ctx.nastyChild
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

func (ctx context) Clean() {
	ctx.m.Lock()
	child := ctx.nastyChild
	ctx.nastyChild = nil
	ctx.m.Unlock()

	if child != nil {
		child.Delete()
	}
}
