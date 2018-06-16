package di

import (
	"fmt"
	"runtime/debug"
)

// contextSlayer contains all the functions that are useful
// to delete a context.
type contextSlayer struct{}

func (s *contextSlayer) Delete(logger Logger, ctx *contextCore) {
	ctx.m.Lock()

	if len(ctx.children) > 0 {
		ctx.deleteIfNoChild = true
		ctx.m.Unlock()
		return
	}

	ctx.m.Unlock()

	s.DeleteWithSubContexts(logger, ctx)
}

func (s *contextSlayer) DeleteWithSubContexts(logger Logger, ctx *contextCore) {
	ctx.m.Lock()

	c := &contextCore{
		children:      make([]*contextCore, len(ctx.children)),
		unscopedChild: ctx.unscopedChild,
		parent:        ctx.parent,
		objects:       map[string]interface{}{},
	}

	copy(c.children, ctx.children)

	for name, obj := range ctx.objects {
		c.objects[name] = obj
	}

	ctx.children = nil
	ctx.unscopedChild = nil
	ctx.parent = nil
	ctx.objects = nil
	ctx.closed = true

	ctx.m.Unlock()

	s.deleteClone(logger, ctx, c)
}

func (s *contextSlayer) deleteClone(logger Logger, ctx *contextCore, c *contextCore) {
	for _, child := range c.children {
		s.DeleteWithSubContexts(logger, child)
	}

	if c.unscopedChild != nil {
		s.DeleteWithSubContexts(logger, c.unscopedChild)
	}

	if c.parent != nil {
		s.removeChild(logger, c.parent, ctx)
	}

	for name, obj := range c.objects {
		s.closeObject(logger, ctx, obj, ctx.definitions[name])
	}
}

func (s *contextSlayer) removeChild(logger Logger, ctx *contextCore, child *contextCore) {
	ctx.m.Lock()

	for i, c := range ctx.children {
		if c == child {
			ctx.children = append(ctx.children[:i], ctx.children[i+1:]...)
			break
		}
	}

	if !ctx.deleteIfNoChild || len(ctx.children) > 0 {
		ctx.m.Unlock()
		return
	}

	ctx.m.Unlock()

	s.DeleteWithSubContexts(logger, ctx)
}

func (s *contextSlayer) closeObject(logger Logger, ctx *contextCore, obj interface{}, def Definition) {
	defer func() {
		if r := recover(); r != nil {
			msg := fmt.Sprintf("could not close `%s` err=%s stack=%s", def.Name, r, debug.Stack())
			logger.Error(msg)
		}
	}()

	if def.Close != nil {
		def.Close(obj)
	}

	return
}

func (s *contextSlayer) IsClosed(ctx *contextCore) bool {
	ctx.m.Lock()
	defer ctx.m.Unlock()
	return ctx.closed
}

func (s *contextSlayer) Clean(logger Logger, ctx *contextCore) {
	ctx.m.Lock()
	child := ctx.unscopedChild
	ctx.unscopedChild = nil
	ctx.m.Unlock()

	if child != nil {
		s.Delete(logger, child)
	}
}
