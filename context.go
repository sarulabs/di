package di

import (
	"errors"
	"fmt"
)

// Context can build items thanks to their definition contained in a ContextManager.
type Context struct {
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
	for i, s := range c.contextManager.scopes {
		if s == c.scope {
			return c.contextManager.scopes[:i]
		}
	}
	return []string{}
}

// SubScopes returns the list of this context subscopes.
func (c *Context) SubScopes() []string {
	for i, s := range c.contextManager.scopes {
		if s == c.scope {
			return c.contextManager.scopes[i+1:]
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
	return c.contextManager
}

// Parent returns the parent Context.
func (c *Context) Parent() *Context {
	return c.parent
}

// ParentWithScope looks over the parents to find one with the given scope.
func (c *Context) ParentWithScope(scope string) *Context {
	parent := c.parent

	for parent != nil {
		if parent.scope == scope {
			return parent
		}
		parent = parent.parent
	}

	return nil
}

// SubContext creates a new Context in a subscope
// that will have this Container as parent.
func (c *Context) SubContext(scope string) (*Context, error) {
	if !c.HasSubScope(scope) {
		return nil, fmt.Errorf("you need to call SubContext with a subscope of `%s` and `%s` is not", c.scope, scope)
	}

	return c.subContext(scope, c.SubScopes()), nil
}

func (c *Context) subContext(scope string, subscopes []string) *Context {
	child := &Context{
		scope:          subscopes[0],
		contextManager: c.contextManager,
		parent:         c,
		children:       []*Context{},
		singletons:     map[string]interface{}{},
		items:          map[interface{}]Maker{},
	}

	c.children = append(c.children, child)

	if child.scope == scope {
		return child
	}

	return child.subContext(scope, subscopes[1:])
}

// SafeMake creates a new item.
// If the item can't be created, it returns an error.
func (c *Context) SafeMake(name string, params ...interface{}) (interface{}, error) {
	// an empty scope means that the context has been deleted
	if c.scope == "" {
		return nil, errors.New("context has been deleted")
	}

	n, err := c.contextManager.ResolveName(name)
	if err != nil {
		return nil, err
	}

	// name is registered, check if it matches an Instance in the ContextManager
	if instance, ok := c.contextManager.instances[n]; ok {
		return instance, nil
	}

	// it's not an Instance, so it's a Maker
	// try to find the Maker in the ContextManager
	maker, ok := c.contextManager.makers[n]
	if !ok {
		return nil, fmt.Errorf("could not find Maker for `%s` in the ContextManager", name)
	}

	// if the Maker scope doesn't math this Context scope
	// try to make the item in a parent Context with the Maker scope
	if c.Scope() != maker.Scope {
		if parent := c.ParentWithScope(maker.Scope); parent != nil {
			return parent.SafeMake(name, params...)
		}
		return nil, fmt.Errorf(
			"Maker for `%s` requires `%s` scope which does not match this Context scope or any of its parents scope",
			name,
			maker.Scope,
		)
	}

	// it's the right scope, try to create the item
	// if it's a singleton, try to reuse an already made item
	if maker.Singleton {
		if item, ok := c.singletons[maker.Name]; ok {
			return item, nil
		}
	}

	item, err := c.makeItem(maker, params...)
	if err != nil {
		return nil, err
	}
	if maker.Singleton {
		c.singletons[maker.Name] = item
	}
	c.items[item] = maker
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

// Close apply the Close method defined in a Maker
// on an item build with the Make method of the Maker
// and retrived with the Make method of this Context.
func (c *Context) Close(item interface{}) {
	c.close(item, true, true)
}

func (c *Context) close(item interface{}, closeParent, closeChildren bool) bool {
	// an empty scope means that the context has been deleted
	if c.scope == "" {
		return false
	}

	// try to find the item in the context
	if maker, ok := c.items[item]; ok && maker.Close != nil {
		c.closeItem(maker, item)
		delete(c.items, item)
		return true
	}

	if closeChildren {
		// the item was not in the context, try to find it in the children
		for _, child := range c.children {
			if child.close(item, false, true) {
				return true
			}
		}
	}

	if closeParent && c.parent != nil {
		// the item was not in the children, try to find it in parent contexts
		if c.parent.close(item, true, false) {
			return true
		}
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
	// an empty scope means that the context has aleady been deleted
	if c.scope == "" {
		return
	}

	// delete children
	for _, child := range c.children {
		child.Delete()
	}

	// remove reference from parent
	if c.parent != nil {
		for i, child := range c.parent.children {
			if child == c {
				c.parent.children = append(c.parent.children[:i], c.parent.children[i+1:]...)
				break
			}
		}
	}

	// close items
	for item := range c.items {
		c.Close(item)
	}

	// remove references
	c.scope = ""
	c.parent = nil
	c.children = nil
	c.singletons = nil
	c.items = nil
}
