package di

import (
	"fmt"
	"runtime/debug"
	"sync"
)

// contextCore contains a Context data.
// But it can not build objects on its own.
// It should be used inside a context.
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

func (ctx *contextCore) Delete() {
	ctx.m.Lock()

	c := &contextCore{
		children:   make([]*contextCore, len(ctx.children)),
		nastyChild: ctx.nastyChild,
		parent:     ctx.parent,
		objects:    map[string]interface{}{},
	}

	copy(c.children, ctx.children)

	for name, obj := range ctx.objects {
		c.objects[name] = obj
	}

	ctx.children = nil
	ctx.nastyChild = nil
	ctx.parent = nil
	ctx.objects = nil
	ctx.closed = true

	ctx.m.Unlock()

	ctx.deleteClone(c)
}

func (ctx *contextCore) deleteClone(c *contextCore) {
	for _, child := range c.children {
		child.Delete()
	}

	if c.nastyChild != nil {
		c.nastyChild.Delete()
	}

	if c.parent != nil {
		c.parent.removeChild(ctx)
	}

	for name, obj := range c.objects {
		ctx.closeObject(obj, ctx.definitions[name])
	}
}

func (ctx *contextCore) removeChild(child *contextCore) {
	ctx.m.Lock()
	defer ctx.m.Unlock()

	for i, c := range ctx.children {
		if c == child {
			ctx.children = append(ctx.children[:i], ctx.children[i+1:]...)
			return
		}
	}
}

func (ctx *contextCore) closeObject(obj interface{}, def Definition) {
	defer func() {
		if r := recover(); r != nil {
			msg := fmt.Sprintf("could not close `%s` err=%s stack=%s", def.Name, r, debug.Stack())
			ctx.logger.Error(msg)
		}
	}()

	if def.Close != nil {
		def.Close(obj)
	}

	return
}

func (ctx *contextCore) IsClosed() bool {
	ctx.m.Lock()
	defer ctx.m.Unlock()
	return ctx.closed
}

func (ctx *contextCore) Clean() {
	ctx.m.Lock()
	child := ctx.nastyChild
	ctx.nastyChild = nil
	ctx.m.Unlock()

	if child != nil {
		child.Delete()
	}
}
