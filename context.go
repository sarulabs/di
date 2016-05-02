package di

import (
	"errors"
	"fmt"
	"runtime/debug"
	"sync"
)

// Context represents a dependency injection container.
// A Context has a scope and may have a parent with a wider scope
// and children with a narrower scope.
// Objects can be retrieved from the Context.
// If the desired object does not already exist in the Context,
// it is built thanks to the object Definition.
// The following requests to this object will return the same object.
type Context interface {
	// Definition returns the map of the available Definitions
	// ordered by Definition name.
	Definitions() map[string]Definition

	// Scope returns the Context scope.
	Scope() string

	// Scopes returns the list of available scopes.
	Scopes() []string

	// ParentScopes returns the list of scopes  wider than the Context scope.
	ParentScopes() []string

	// SubContext creates a new Context in the next subscope
	// that will have this Container as parent.
	SubScopes() []string

	// Parent returns the parent Context.
	Parent() Context

	// SubContext creates a new Context in the next subscope
	// that will have this Container as parent.
	SubContext() (Context, error)

	// SafeGet retrieves an object from the Context.
	// If the object does not already exist, it is created and saved in the Context.
	// If the item can't be created, it returns an error.
	SafeGet(name string) (interface{}, error)

	// Get is similar to SafeGet but it does not return the error.
	Get(name string) interface{}

	// Fill is similar to SafeGet but it does not return the object.
	// Instead it fills the provided object with the value returned by SafeGet.
	// The provided object must be a pointer to the value returned by SafeGet.
	Fill(name string, dst interface{}) error

	// Delete removes all the references to the objects that have been build in this context.
	// Before removing the references, it calls the Close method
	// from the object Definition on each object.
	// It will also call Delete on each child and remove its reference in the parent Context.
	Delete()

	// IsClosed retuns true if the Context has been deleted.
	IsClosed() bool
}

type contextData struct {
	m           sync.Mutex
	closed      bool
	scope       string
	scopes      []string
	definitions map[string]Definition
	parent      *contextData
	children    []*contextData
	objects     map[string]interface{}
}

func (ctx *contextData) getParent() *contextData {
	ctx.m.Lock()
	defer ctx.m.Unlock()
	return ctx.parent
}

func (ctx *contextData) parentWithScope(scope string) *contextData {
	parent := ctx.getParent()

	for parent != nil {
		if parent.scope == scope {
			return parent
		}
		parent = parent.getParent()
	}

	return nil
}

func (ctx *contextData) Delete() {
	ctx.m.Lock()

	// copy children, parent and objects so they can be removed outside of the locked area
	children := make([]*contextData, len(ctx.children))
	copy(children, ctx.children)

	parent := ctx.parent

	objects := map[string]interface{}{}

	for name, obj := range ctx.objects {
		objects[name] = obj
	}

	ctx.closed = true

	ctx.m.Unlock()

	// delete children
	for _, child := range children {
		child.Delete()
	}

	// remove reference from parent
	if parent != nil {
		parent.m.Lock()

		for i, child := range parent.children {
			if ctx == child {
				parent.children = append(parent.children[:i], parent.children[i+1:]...)
				break
			}
		}

		parent.m.Unlock()
	}

	// close items
	for name, obj := range objects {
		def := ctx.definitions[name]
		ctx.close(obj, def)
	}

	// remove references
	ctx.m.Lock()
	ctx.parent = nil
	ctx.children = nil
	ctx.objects = nil
	ctx.m.Unlock()
}

func (ctx *contextData) close(obj interface{}, def Definition) {
	defer func() {
		recover()
	}()

	def.Close(obj)
	return
}

func (ctx *contextData) IsClosed() bool {
	ctx.m.Lock()
	defer ctx.m.Unlock()
	return ctx.closed
}

// context is the implementation of the Context interface
type context struct {
	*contextData
	building []string
}

func (ctx context) Definitions() map[string]Definition {
	defs := map[string]Definition{}

	for name, def := range ctx.definitions {
		defs[name] = def
	}

	return defs
}

func (ctx context) Scope() string {
	return ctx.scope
}

func (ctx context) Scopes() []string {
	scopes := make([]string, len(ctx.scopes))
	copy(scopes, ctx.scopes)
	return scopes
}

func (ctx context) ParentScopes() []string {
	scopes := ctx.Scopes()

	for i, s := range scopes {
		if s == ctx.scope {
			return scopes[:i]
		}
	}

	return []string{}
}

func (ctx context) SubScopes() []string {
	scopes := ctx.Scopes()

	for i, s := range scopes {
		if s == ctx.scope {
			return scopes[i+1:]
		}
	}

	return []string{}
}

func (ctx context) Parent() Context {
	return context{
		contextData: ctx.getParent(),
		building:    ctx.building,
	}
}

func (ctx context) SubContext() (Context, error) {
	subscopes := ctx.SubScopes()

	if len(subscopes) == 0 {
		return nil, fmt.Errorf("there is no narrower scope than `%s`", ctx.scope)
	}

	child := &context{
		contextData: &contextData{
			scope:       subscopes[0],
			scopes:      ctx.scopes,
			definitions: ctx.definitions,
			parent:      ctx.contextData,
			children:    []*contextData{},
			objects:     map[string]interface{}{},
		},
		building: ctx.building,
	}

	ctx.m.Lock()

	if ctx.closed {
		return nil, errors.New("the Context is closed")
	}

	ctx.children = append(ctx.children, child.contextData)

	ctx.m.Unlock()

	return child, nil
}

func (ctx context) SafeGet(name string) (interface{}, error) {
	def, ok := ctx.definitions[name]
	if !ok {
		return nil, fmt.Errorf("could not find a Definition for `%s` in the Context", name)
	}

	if stringSliceContains(ctx.building, name) {
		return nil, fmt.Errorf("there is a cycle in object definitions : %v", ctx.building)
	}

	if ctx.scope != def.Scope {
		return ctx.getInParent(def)
	}

	return ctx.getInThisContext(def)
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
	parent := ctx.parentWithScope(def.Scope)
	if parent == nil {
		return nil, fmt.Errorf(
			"Definition of `%s` requires `%s` scope which does not match this Context scope or any of its parents scope",
			def.Name,
			def.Scope,
		)
	}

	p := context{
		contextData: parent,
		building:    ctx.building,
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
			err = fmt.Errorf("panic : %s - stack : %s", r, debug.Stack())
		}
	}()

	obj, err = def.Build(&context{
		contextData: ctx.contextData,
		building:    append(ctx.building, def.Name),
	})
	return
}
