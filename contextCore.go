package di

import "sync"

// contextCore contains a Context data.
// But it can not build objects on its own.
// It should be used inside a context.
type contextCore struct {
	m               sync.Mutex
	closed          bool
	scope           string
	scopes          ScopeList
	definitions     DefinitionMap
	parent          *contextCore
	children        []*contextCore
	unscopedChild   *contextCore
	objects         map[string]interface{}
	deleteIfNoChild bool
}

func (ctx *contextCore) Definitions() map[string]Definition {
	return ctx.definitions.Copy()
}

func (ctx *contextCore) Scope() string {
	return ctx.scope
}

func (ctx *contextCore) Scopes() []string {
	return ctx.scopes.Copy()
}

func (ctx *contextCore) ParentScopes() []string {
	return ctx.scopes.ParentScopes(ctx.scope)
}

func (ctx *contextCore) SubScopes() []string {
	return ctx.scopes.SubScopes(ctx.scope)
}
