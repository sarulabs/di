package di

// context is the implementation of the Context interface
type context struct {
	// contextCore contains the context data.
	// Serveral contexts can share the same contextCore.
	// In this case these contexts represent the same entity,
	// but at a different stage in an object construction.
	*contextCore

	// built contains the name of the Definition being built by this context.
	// It is used to avoid cycles in object Definitions.
	// Each time a Context is passed in parameter of the Build function
	// of a definition, this is in fact a new context.
	// This context is created with a built attribute
	// updated with the name of the Definition.
	built []string

	logger Logger

	lineage     *contextLineage
	slayer      *contextSlayer
	getter      *contextGetter
	nastyGetter *contextNastyGetter
}

func (ctx context) Parent() Context {
	return ctx.lineage.Parent(ctx)
}

func (ctx context) SubContext() (Context, error) {
	return ctx.lineage.SubContext(ctx)
}

func (ctx context) SafeGet(name string) (interface{}, error) {
	return ctx.getter.SafeGet(ctx, name)
}

func (ctx context) Get(name string) interface{} {
	return ctx.getter.Get(ctx, name)
}

func (ctx context) Fill(name string, dst interface{}) error {
	return ctx.getter.Fill(ctx, name, dst)
}

func (ctx context) NastySafeGet(name string) (interface{}, error) {
	return ctx.nastyGetter.NastySafeGet(ctx, name)
}

func (ctx context) NastyGet(name string) interface{} {
	return ctx.nastyGetter.NastyGet(ctx, name)
}

func (ctx context) NastyFill(name string, dst interface{}) error {
	return ctx.nastyGetter.NastyFill(ctx, name, dst)
}

func (ctx context) Delete() {
	ctx.slayer.Delete(ctx.logger, ctx.contextCore)
}

func (ctx context) DeleteWithSubContexts() {
	ctx.slayer.DeleteWithSubContexts(ctx.logger, ctx.contextCore)
}

func (ctx context) IsClosed() bool {
	return ctx.slayer.IsClosed(ctx.contextCore)
}

func (ctx context) Clean() {
	ctx.slayer.Clean(ctx.logger, ctx.contextCore)
}
