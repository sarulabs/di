package di

// context is the implementation of the Context interface
type context struct {
	// contextCore contains the context data.
	// Several contexts can share the same contextCore.
	// In this case these contexts represent the same entity,
	// but at a different stage of an object construction.
	*contextCore

	// built contains the name of the Definition being built by this context.
	// It is used to avoid cycles in object Definitions.
	// Each time a Context is passed in parameter of the Build function
	// of a definition, this is in fact a new context.
	// This context is created with a built field
	// updated with the name of the Definition.
	built []string

	*contextLineage
	*contextSlayer
	*contextGetter
	*contextUnscopedGetter

	logger Logger
}

func (ctx *context) Parent() Context {
	return ctx.contextLineage.Parent(ctx)
}

func (ctx *context) SubContext() (Context, error) {
	return ctx.contextLineage.SubContext(ctx)
}

func (ctx *context) SafeGet(name string) (interface{}, error) {
	return ctx.contextGetter.SafeGet(ctx, name)
}

func (ctx *context) Get(name string) interface{} {
	return ctx.contextGetter.Get(ctx, name)
}

func (ctx *context) Fill(name string, dst interface{}) error {
	return ctx.contextGetter.Fill(ctx, name, dst)
}

func (ctx *context) UnscopedSafeGet(name string) (interface{}, error) {
	return ctx.contextUnscopedGetter.UnscopedSafeGet(ctx, name)
}

func (ctx *context) UnscopedGet(name string) interface{} {
	return ctx.contextUnscopedGetter.UnscopedGet(ctx, name)
}

func (ctx *context) UnscopedFill(name string, dst interface{}) error {
	return ctx.contextUnscopedGetter.UnscopedFill(ctx, name, dst)
}

func (ctx *context) NastySafeGet(name string) (interface{}, error) {
	return ctx.contextUnscopedGetter.UnscopedSafeGet(ctx, name)
}

func (ctx *context) NastyGet(name string) interface{} {
	return ctx.contextUnscopedGetter.UnscopedGet(ctx, name)
}

func (ctx *context) NastyFill(name string, dst interface{}) error {
	return ctx.contextUnscopedGetter.UnscopedFill(ctx, name, dst)
}

func (ctx *context) Delete() {
	ctx.contextSlayer.Delete(ctx.logger, ctx.contextCore)
}

func (ctx *context) DeleteWithSubContexts() {
	ctx.contextSlayer.DeleteWithSubContexts(ctx.logger, ctx.contextCore)
}

func (ctx *context) IsClosed() bool {
	return ctx.contextSlayer.IsClosed(ctx.contextCore)
}

func (ctx *context) Clean() {
	ctx.contextSlayer.Clean(ctx.logger, ctx.contextCore)
}
