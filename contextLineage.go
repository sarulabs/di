package di

import (
	"errors"
	"fmt"
)

// contextLineage contains all the functions that are useful
// to retrieve or create the parent and children of a context.
type contextLineage struct{}

func (l *contextLineage) getParent(ctx *contextCore) *contextCore {
	ctx.m.Lock()
	defer ctx.m.Unlock()
	return ctx.parent
}

func (l *contextLineage) Parent(ctx *context) Context {
	return &context{
		contextCore: l.getParent(ctx.contextCore),
		built:       ctx.built,
		logger:      ctx.logger,
	}
}

func (l *contextLineage) SubContext(ctx *context) (Context, error) {
	child, err := l.createChild(ctx)
	if err != nil {
		return nil, err
	}

	ctx.m.Lock()

	if ctx.closed {
		ctx.m.Unlock()
		return nil, errors.New("the Context is closed")
	}

	ctx.children = append(ctx.children, child.contextCore)

	ctx.m.Unlock()

	return child, nil
}

func (l *contextLineage) createChild(ctx *context) (*context, error) {
	subscopes := ctx.SubScopes()

	if len(subscopes) == 0 {
		return nil, fmt.Errorf("there is no narrower scope than `%s`", ctx.scope)
	}

	return &context{
		contextCore: &contextCore{
			scope:         subscopes[0],
			scopes:        ctx.scopes,
			definitions:   ctx.definitions,
			parent:        ctx.contextCore,
			children:      []*contextCore{},
			unscopedChild: nil,
			objects:       map[string]interface{}{},
		},
		built:  ctx.built,
		logger: ctx.logger,
	}, nil
}
