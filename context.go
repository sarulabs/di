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
	singletons     map[string]interface{}
	items          map[interface{}]Maker
}

// Scope returns the name of the context scope.
func (c *Context) Scope() string {
	return c.scope
}

// ParentScopes returns the list of this context parent scopes.
func (c *Context) ParentScopes() []string {
	if manager := c.ContextManager(); manager != nil {
		for i, s := range manager.scopes {
			if s == c.scope {
				return manager.scopes[:i]
			}
		}
	}
	return []string{}
}

// SubScopes returns the list of this context subscopes.
func (c *Context) SubScopes() []string {
	if manager := c.ContextManager(); manager != nil {
		for i, s := range manager.scopes {
			if s == c.scope {
				return manager.scopes[i+1:]
			}
		}
	}
	return []string{}
}

// HasSubScope returns true if scope is one of this context subscopes.
func (c *Context) HasSubScope(scope string) bool {
	return stringSliceContains(c.SubScopes(), scope)
}

// ContextManager returns the ContextManager that has generated this Context.
func (c *Context) ContextManager() *ContextManager {
	c.m.Lock()
	defer c.m.Unlock()
	return c.contextManager
}

// Parent returns the parent Context.
func (c *Context) Parent() *Context {
	c.m.Lock()
	defer c.m.Unlock()
	return c.parent
}

// ParentWithScope looks over the parents to find one with the given scope.
func (c *Context) ParentWithScope(scope string) *Context {
	parent := c.Parent()

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
func (c *Context) SubContext(scope string) (*Context, error) {
	if !c.HasSubScope(scope) {
		return nil, fmt.Errorf("you need to call SubContext with a subscope of `%s` and `%s` is not", c.scope, scope)
	}

	return c.subContext(scope, c.SubScopes())
}

func (c *Context) subContext(scope string, subscopes []string) (*Context, error) {
	c.m.Lock()

	if c.contextManager == nil {
		c.m.Unlock()
		return nil, fmt.Errorf("could not create sub-context of closed `%s` context", c.scope)
	}

	child := &Context{
		scope:          subscopes[0],
		contextManager: c.contextManager,
		parent:         c,
		children:       []*Context{},
		singletons:     map[string]interface{}{},
		items:          map[interface{}]Maker{},
	}

	c.children = append(c.children, child)

	c.m.Unlock()

	if child.scope == scope {
		return child, nil
	}

	return child.subContext(scope, subscopes[1:])
}

// SafeMake creates a new item.
// If the item can't be created, it returns an error.
func (c *Context) SafeMake(name string, params ...interface{}) (interface{}, error) {
	manager := c.ContextManager()
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
	if c.scope != maker.Scope {
		return c.makeInParent(maker, params...)
	}

	// it's the suitable Maker in the right scope, provide the item
	return c.makeInThisContext(maker, params...)
}

func (c *Context) makeInThisContext(maker Maker, params ...interface{}) (interface{}, error) {
	// if it's a singleton, try to reuse an already made item
	if maker.Singleton {
		c.m.Lock()
		item, ok := c.singletons[maker.Name]
		c.m.Unlock()

		if ok {
			return item, nil
		}
	}

	// the item has not been made yet, so create it
	item, err := c.makeItem(maker, params...)
	if err != nil {
		return nil, err
	}

	c.m.Lock()
	defer c.m.Unlock()

	// ensure the Context is not closed before adding anything to it
	if c.contextManager == nil {
		return nil, errors.New("context has been deleted")
	}

	// if it is a singleton, save it to reuse it later on
	if maker.Singleton {
		c.singletons[maker.Name] = item
	}

	// save the item so you can close it later on
	// close does not work with items that are not hashable
	if isHashable(item) {
		c.items[item] = maker
	}

	return item, nil
}

func (c *Context) makeInParent(maker Maker, params ...interface{}) (interface{}, error) {
	parent := c.ParentWithScope(maker.Scope)
	if parent == nil {
		return nil, fmt.Errorf(
			"Maker for `%s` requires `%s` scope which does not match this Context scope or any of its parents scope",
			maker.Name,
			maker.Scope,
		)
	}

	item, err := parent.makeInThisContext(maker, params...)
	if err != nil {
		return item, err
	}

	c.m.Lock()
	defer c.m.Unlock()

	if c.contextManager == nil {
		// the item was created and saved in the parent Context, but this Context was closed in the meantime
		// close the item and remove the reference from the parent
		parent.m.Lock()
		parent.closeItem(maker, item)
		delete(parent.items, item)
		parent.m.Unlock()

		return nil, errors.New("context has been deleted")
	}

	// if the item is not a singleton, the item should be saved in this Context and not in the parent
	// that is because you want the item to be close as soon as this Context is deleted
	if !maker.Singleton {
		parent.m.Lock()
		delete(parent.items, item)
		parent.m.Unlock()

		c.items[item] = maker
	}

	return item, nil
}

func (c *Context) makeItem(maker Maker, params ...interface{}) (item interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.New("panic : " + fmt.Sprint(r))
		}
	}()

	item, err = maker.Make(c, params...)
	return
}

// Make creates a new item.
// If the item can't be created, it returns nil.
func (c *Context) Make(name string, params ...interface{}) interface{} {
	item, _ := c.SafeMake(name, params...)
	return item
}

// Fill creates a new item and copies it in a given pointer.
func (c *Context) Fill(item interface{}, name string, params ...interface{}) error {
	i, err := c.SafeMake(name, params...)
	if err != nil {
		return err
	}
	return fill(i, item)
}

// Close apply the Close method defined in a Maker
// on an item build with the Make method of the Maker
// and retrived with the Make method of this Context.
func (c *Context) Close(item interface{}) {
	c.close(item, true, true)
}

func (c *Context) close(item interface{}, closeParent, closeChildren bool) bool {
	// try to find the item in the context
	c.m.Lock()
	maker, ok := c.items[item]
	c.m.Unlock()

	if ok && maker.Close != nil {
		c.closeItem(maker, item)

		c.m.Lock()
		delete(c.items, item)
		c.m.Unlock()

		return true
	}

	// the item was not in this context, try to find it in its children
	if closeChildren {
		c.m.Lock()

		children := make([]*Context, len(c.children))
		copy(children, c.children)

		c.m.Unlock()

		for _, child := range children {
			if child.close(item, false, true) {
				return true
			}
		}
	}

	// the item was not in the children, try to remove it from the parent.
	parent := c.Parent()

	if closeParent && parent != nil {
		return parent.close(item, true, false)
	}

	return false
}

func (c *Context) closeItem(maker Maker, item interface{}) {
	defer func() {
		recover()
	}()

	maker.Close(item)
	return
}

// Delete removes all the references to the items that has been made by this context.
// Before removing the references, it calls the Close method on these items.
// It will also call Delete on every child
// and remove its reference in the parent Context.
func (c *Context) Delete() {
	c.m.Lock()

	// copy children, parent and items so they can be removed outside of the locked area
	children := make([]*Context, len(c.children))
	copy(children, c.children)

	parent := c.parent

	items := map[interface{}]Maker{}
	for item, maker := range c.items {
		items[item] = maker
	}

	// remove contextManager to mark this Context as closed
	c.contextManager = nil

	c.m.Unlock()

	// delete children
	for _, child := range children {
		child.Delete()
	}

	// remove reference from parent
	if parent != nil {
		parent.m.Lock()
		for i, child := range parent.children {
			if c == child {
				parent.children = append(parent.children[:i], parent.children[i+1:]...)
				break
			}
		}
		parent.m.Unlock()
	}

	// close items
	for item := range items {
		c.Close(item)
	}

	// remove references
	c.m.Lock()
	c.parent = nil
	c.children = nil
	c.singletons = nil
	c.items = nil
	c.m.Unlock()
}
