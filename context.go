package di

import (
	"errors"
	"fmt"
	"sync"
)

// Context can build items thanks to their definition contained in a ContextManager.
type Context struct {
	m              sync.Mutex
	scope          string
	contextManager *ContextManager
	parent         *Context
	children       []*Context
	items          map[string]interface{}
}

// Scope returns the name of the context scope.
func (ctx *Context) Scope() string {
	return ctx.scope
}

// ParentScopes returns the list of this context parent scopes.
func (ctx *Context) ParentScopes() []string {
	if manager := ctx.ContextManager(); manager != nil {
		for i, s := range manager.scopes {
			if s == ctx.scope {
				return manager.scopes[:i]
			}
		}
	}
	return []string{}
}

// SubScopes returns the list of this context subscopes.
func (ctx *Context) SubScopes() []string {
	if manager := ctx.ContextManager(); manager != nil {
		for i, s := range manager.scopes {
			if s == ctx.scope {
				return manager.scopes[i+1:]
			}
		}
	}
	return []string{}
}

// HasSubScope returns true if scope is one of this context subscopes.
func (ctx *Context) HasSubScope(scope string) bool {
	return stringSliceContains(ctx.SubScopes(), scope)
}

// ContextManager returns the ContextManager that has generated this Context.
func (ctx *Context) ContextManager() *ContextManager {
	ctx.m.Lock()
	defer ctx.m.Unlock()
	return ctx.contextManager
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

// SubContext creates a new Context in a subscope
// that will have this Container as parent.
func (ctx *Context) SubContext(scope string) (*Context, error) {
	if !ctx.HasSubScope(scope) {
		return nil, fmt.Errorf("you need to call SubContext with a subscope of `%s` and `%s` is not", ctx.scope, scope)
	}

	return ctx.subContext(scope, ctx.SubScopes())
}

func (ctx *Context) subContext(scope string, subscopes []string) (*Context, error) {
	ctx.m.Lock()

	if ctx.contextManager == nil {
		ctx.m.Unlock()
		return nil, fmt.Errorf("could not create sub-context of closed `%s` context", ctx.scope)
	}

	child := &Context{
		scope:          subscopes[0],
		contextManager: ctx.contextManager,
		parent:         ctx,
		children:       []*Context{},
		items:          map[string]interface{}{},
	}

	ctx.children = append(ctx.children, child)

	ctx.m.Unlock()

	if child.scope == scope {
		return child, nil
	}

	return child.subContext(scope, subscopes[1:])
}

// SafeMake creates a new item.
// If the item can't be created, it returns an error.
func (ctx *Context) SafeMake(name string) (interface{}, error) {
	manager := ctx.ContextManager()
	if manager == nil {
		return nil, errors.New("context has been deleted")
	}

	n, err := manager.ResolveName(name)
	if err != nil {
		return nil, err
	}

	// name is registered, check if it matches an Instance in the ContextManager
	if instance, ok := manager.instances[n]; ok {
		return instance, nil
	}

	// it's not an Instance, so it's a Maker
	// try to find the Maker in the ContextManager
	maker, ok := manager.makers[n]
	if !ok {
		return nil, fmt.Errorf("could not find Maker for `%s` in the ContextManager", name)
	}

	// if the Maker scope doesn't math this Context scope
	// try to make the item in a parent Context matching the Maker scope
	if ctx.scope != maker.Scope {
		return ctx.makeInParent(maker)
	}

	// it's the suitable Maker in the right scope, provide the item
	return ctx.makeInThisContext(maker)
}

func (ctx *Context) makeInThisContext(maker Maker) (interface{}, error) {
	// try to reuse an already made item
	ctx.m.Lock()
	item, ok := ctx.items[maker.Name]
	ctx.m.Unlock()

	if ok {
		return item, nil
	}

	// the item has not been made yet, so create it
	item, err := ctx.makeItem(maker)
	if err != nil {
		return nil, err
	}

	// ensure the Context is not closed before adding the item
	ctx.m.Lock()
	defer ctx.m.Unlock()

	if ctx.contextManager == nil {
		return nil, errors.New("context has been deleted")
	}

	ctx.items[maker.Name] = item

	return item, nil
}

func (ctx *Context) makeInParent(maker Maker) (interface{}, error) {
	parent := ctx.ParentWithScope(maker.Scope)
	if parent == nil {
		return nil, fmt.Errorf(
			"Maker for `%s` requires `%s` scope which does not match this Context scope or any of its parents scope",
			maker.Name,
			maker.Scope,
		)
	}

	return parent.makeInThisContext(maker)
}

func (ctx *Context) makeItem(maker Maker) (item interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.New("panic : " + fmt.Sprint(r))
		}
	}()

	item, err = maker.Make(ctx)
	return
}

// Make creates a new item.
// If the item can't be created, it returns nil.
func (ctx *Context) Make(name string) interface{} {
	item, _ := ctx.SafeMake(name)
	return item
}

// Fill creates a new item and copies it in a given pointer.
func (ctx *Context) Fill(name string, item interface{}) error {
	i, err := ctx.SafeMake(name)
	if err != nil {
		return err
	}
	return fill(i, item)
}

// Delete removes all the references to the items that has been made by this context.
// Before removing the references, it calls the Close method on these items.
// It will also call Delete on every child
// and remove its reference in the parent Context.
func (ctx *Context) Delete() {
	ctx.m.Lock()

	// check if already deleted
	if ctx.contextManager == nil {
		ctx.m.Unlock()
		return
	}

	// copy children, parent and items so they can be removed outside of the locked area
	children := make([]*Context, len(ctx.children))
	copy(children, ctx.children)

	parent := ctx.parent

	items := map[string]interface{}{}
	makers := map[string]Maker{}

	for name, item := range ctx.items {
		items[name] = item
		if maker, ok := ctx.contextManager.makers[name]; ok {
			makers[name] = maker
		}
	}

	// remove contextManager to mark this Context as closed
	ctx.contextManager = nil

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
	for name, item := range items {
		if maker, ok := makers[name]; ok {
			ctx.close(maker, item)
		}
	}

	// remove references
	ctx.m.Lock()
	ctx.parent = nil
	ctx.children = nil
	ctx.items = nil
	ctx.m.Unlock()
}

func (ctx *Context) close(maker Maker, item interface{}) {
	defer func() {
		recover()
	}()

	maker.Close(item)
	return
}
