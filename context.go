package di

import (
	"errors"
	"fmt"
	"runtime/debug"
	"sync"
)

// Context is a dependency injection container.
// The Context has a scope and may have a parent with a wider scope
// and children with a narrower scope.
// Objects can be retrieved from the Context.
// If the desired object does not already exist in the Context,
// it is built thanks to the object Definition.
// The following requests to this object will return the same object.
type Context struct {
	m           sync.Mutex
	closed      bool
	scope       string
	scopes      []string
	definitions map[string]Definition
	parent      *Context
	children    []*Context
	objects     map[string]interface{}
}

// Scope returns the Context scope.
func (ctx *Context) Scope() string {
	return ctx.scope
}

// Scopes returns the list of available scopes.
func (ctx *Context) Scopes() []string {
	scopes := make([]string, len(ctx.scopes))
	copy(scopes, ctx.scopes)
	return scopes
}

// ParentScopes returns the list of scopes  wider than the Context scope.
func (ctx *Context) ParentScopes() []string {
	scopes := ctx.Scopes()

	for i, s := range scopes {
		if s == ctx.scope {
			return scopes[:i]
		}
	}

	return []string{}
}

// SubScopes returns the list of the scopes narrower than the Context scope.
func (ctx *Context) SubScopes() []string {
	scopes := ctx.Scopes()

	for i, s := range scopes {
		if s == ctx.scope {
			return scopes[i+1:]
		}
	}

	return []string{}
}

// HasSubScope returns true if scope is one of the Context subscopes.
func (ctx *Context) HasSubScope(scope string) bool {
	return stringSliceContains(ctx.SubScopes(), scope)
}

// Parent returns the parent Context.
func (ctx *Context) Parent() *Context {
	ctx.m.Lock()
	defer ctx.m.Unlock()
	return ctx.parent
}

// ParentWithScope looks over the parents to find one with the given scope.
func (ctx *Context) ParentWithScope(scope string) *Context {
	parent := ctx.Parent()

	for parent != nil {
		if parent.scope == scope {
			return parent
		}
		parent = parent.Parent()
	}

	return nil
}

// SubContext creates a new Context in the next subscope
// that will have this Container as parent.
func (ctx *Context) SubContext() (*Context, error) {
	subscopes := ctx.SubScopes()

	if len(subscopes) == 0 {
		return nil, fmt.Errorf("there is no narrower scope than `%s`", ctx.scope)
	}

	child := &Context{
		scope:       subscopes[0],
		scopes:      ctx.scopes,
		definitions: ctx.definitions,
		parent:      ctx,
		children:    []*Context{},
		objects:     map[string]interface{}{},
	}

	ctx.m.Lock()

	if ctx.closed {
		return nil, errors.New("the Context is closed")
	}

	ctx.children = append(ctx.children, child)

	ctx.m.Unlock()

	return child, nil
}

// SafeGet retrieves an object from the Context.
// If the object does not already exist, it is created and saved in the Context.
// If the item can't be created, it returns an error.
func (ctx *Context) SafeGet(name string) (interface{}, error) {
	def, ok := ctx.definitions[name]
	if !ok {
		return nil, fmt.Errorf("could not find a Definition for `%s` in the Context", name)
	}

	if ctx.scope != def.Scope {
		return ctx.buildInParent(def)
	}

	return ctx.buildInThisContext(def)
}

func (ctx *Context) buildInThisContext(def Definition) (interface{}, error) {
	// try to reuse an already made item
	ctx.m.Lock()
	obj, ok := ctx.objects[def.Name]
	ctx.m.Unlock()

	if ok {
		return obj, nil
	}

	// the object needs to be created
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

func (ctx *Context) buildInParent(def Definition) (interface{}, error) {
	parent := ctx.ParentWithScope(def.Scope)
	if parent == nil {
		return nil, fmt.Errorf(
			"Definition of `%s` requires `%s` scope which does not match this Context scope or any of its parents scope",
			def.Name,
			def.Scope,
		)
	}

	return parent.buildInThisContext(def)
}

func (ctx *Context) build(def Definition) (obj interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic : %s - stack : %s", r, debug.Stack())
		}
	}()

	obj, err = def.Build(ctx)
	return
}

// Get is similar to SafeGet but it does not return the error.
func (ctx *Context) Get(name string) interface{} {
	obj, _ := ctx.SafeGet(name)
	return obj
}

// Fill is similar to SafeGet but it does not return the object.
// Instead it fills the provided object with the value returned by SafeGet.
// The provided object must be a pointer to the value returned by SafeGet.
func (ctx *Context) Fill(name string, dst interface{}) error {
	obj, err := ctx.SafeGet(name)
	if err != nil {
		return err
	}

	return fill(obj, dst)
}

// Delete removes all the references to the objects that have been build in this context.
// Before removing the references, it calls the Close method
// from the object Definition on each object.
// It will also call Delete on each child and remove its reference in the parent Context.
func (ctx *Context) Delete() {
	ctx.m.Lock()

	// copy children, parent and objects so they can be removed outside of the locked area
	children := make([]*Context, len(ctx.children))
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

func (ctx *Context) close(obj interface{}, def Definition) {
	defer func() {
		recover()
	}()

	def.Close(obj)
	return
}
