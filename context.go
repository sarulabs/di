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
	// The difference is that the object can still be retrieved
	// even if it belongs to a narrower scope.
	// Do do so NastySafeGet creates a subcontext.
	// When the created object is no longer needed,
	// it is important to use the Clean method to Delete these contexts.
	NastySafeGet(name string) (interface{}, error)

	// NastyGet is similar to NastySafeGet but it does not return the error.
	NastyGet(name string) interface{}

	// NastyFill is similar to NastySafeGet but copies the object in dst instead of returning it.
	NastyFill(name string, dst interface{}) error

	// Clean deletes the subcontext created by NastySafeGet, NastyGet or NastyFill.
	Clean()

	// Delete removes all the references to the objects that have been build in this context.
	// Before removing the references, it calls the Close method
	// from the object Definition on each object.
	// It will also call Delete on each child and remove its reference in the parent Context.
	Delete()

	// IsClosed retuns true if the Context has been deleted.
	IsClosed() bool
}

type contextCore struct {
	m           sync.Mutex
	closed      bool
	logger      Logger
	scope       string
	scopes      []string
	definitions map[string]Definition
	parent      *contextCore
	children    []*contextCore
	nastyChild  *contextCore
	objects     map[string]interface{}
}

func (ctx *contextCore) Definitions() map[string]Definition {
	defs := map[string]Definition{}

	for name, def := range ctx.definitions {
		defs[name] = def
	}

	return defs
}

func (ctx *contextCore) Scope() string {
	return ctx.scope
}

func (ctx *contextCore) Scopes() []string {
	scopes := make([]string, len(ctx.scopes))
	copy(scopes, ctx.scopes)
	return scopes
}

func (ctx *contextCore) ParentScopes() []string {
	scopes := ctx.Scopes()

	for i, s := range scopes {
		if s == ctx.scope {
			return scopes[:i]
		}
	}

	return []string{}
}

func (ctx *contextCore) SubScopes() []string {
	scopes := ctx.Scopes()

	for i, s := range scopes {
		if s == ctx.scope {
			return scopes[i+1:]
		}
	}

	return []string{}
}

func (ctx *contextCore) getParent() *contextCore {
	ctx.m.Lock()
	defer ctx.m.Unlock()
	return ctx.parent
}

func (ctx *contextCore) getParentWithScope(scope string) *contextCore {
	parent := ctx.getParent()

	for parent != nil {
		if parent.scope == scope {
			return parent
		}
		parent = parent.getParent()
	}

	return nil
}

func (ctx *contextCore) Delete() {
	ctx.m.Lock()

	// copy children, parent and objects so they can be closed outside of the locked area
	children := make([]*contextCore, len(ctx.children))
	copy(children, ctx.children)
	ctx.children = nil

	nastyChild := ctx.nastyChild
	ctx.nastyChild = nil

	parent := ctx.parent
	ctx.parent = nil

	objects := map[string]interface{}{}
	for name, obj := range ctx.objects {
		objects[name] = obj
	}
	ctx.objects = nil

	ctx.closed = true

	ctx.m.Unlock()

	// delete children
	for _, child := range children {
		child.Delete()
	}

	if nastyChild != nil {
		nastyChild.Delete()
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

	// close objects
	for name, obj := range objects {
		def := ctx.definitions[name]
		ctx.close(obj, def)
	}
}

func (ctx *contextCore) close(obj interface{}, def Definition) {
	defer func() {
		if r := recover(); r != nil {
			msg := fmt.Sprintf("could not close `%s` err=%s stack=%s", def.Name, r, debug.Stack())
			ctx.logger.Error(msg)
		}
	}()

	def.Close(obj)
	return
}

func (ctx *contextCore) IsClosed() bool {
	ctx.m.Lock()
	defer ctx.m.Unlock()
	return ctx.closed
}

// context is the implementation of the Context interface
type context struct {
	*contextCore
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
	parent := ctx.getParentWithScope(def.Scope)
	if parent == nil {
		return nil, fmt.Errorf(
			"Definition of `%s` requires `%s` scope which does not match this Context scope or any of its parents scope",
			def.Name,
			def.Scope,
		)
	}

	p := context{
		contextCore: parent,
		built:       ctx.built,
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

	ctx.m.Lock()
	nastyChild := ctx.nastyChild
	ctx.m.Unlock()

	child := &context{
		contextCore: nastyChild,
		built:       ctx.built,
	}

	if nastyChild == nil {
		child, err = ctx.addNastyChild()
		if err != nil {
			return nil, err
		}
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

func (ctx context) Clean() {
	ctx.m.Lock()
	child := ctx.nastyChild
	ctx.nastyChild = nil
	ctx.m.Unlock()

	if child != nil {
		child.Delete()
	}
}
