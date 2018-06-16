package di

// Context represents a dependency injection container.
// A Context has a scope and may have a parent with a wider scope
// and children with a narrower scope.
// Objects can be retrieved from the Context.
// If the desired object does not already exist in the Context,
// it is built thanks to the object Definition.
// The following attempts to get this object will return the same object.
type Context interface {
	// Definition returns the map of the available Definitions ordered by Definition name.
	// These Definitions represent all the objects that this Context can build.
	Definitions() map[string]Definition

	// Scope returns the Context scope.
	Scope() string

	// Scopes returns the list of available scopes.
	Scopes() []string

	// ParentScopes returns the list of scopes wider than the Context scope.
	ParentScopes() []string

	// SubScopes returns the list of scopes narrower than the Context scope.
	SubScopes() []string

	// Parent returns the parent Context.
	Parent() Context

	// SubContext creates a new Context in the next subscope
	// that will have this Container as parent.
	SubContext() (Context, error)

	// SafeGet retrieves an object from the Context.
	// The object has to belong to this scope or a wider one.
	// If the object does not already exist, it is created and saved in the Context.
	// If the object can't be created, it returns an error.
	SafeGet(name string) (interface{}, error)

	// Get is similar to SafeGet but it does not return the error.
	Get(name string) interface{}

	// Fill is similar to SafeGet but it does not return the object.
	// Instead it fills the provided object with the value returned by SafeGet.
	// The provided object must be a pointer to the value returned by SafeGet.
	Fill(name string, dst interface{}) error

	// UnscopedSafeGet retrieves an object from the Context, like SafeGet.
	// The difference is that the object can be retrieved
	// even if it belongs to a narrower scope.
	// To do so UnscopedSafeGet creates a subcontext.
	// When the created object is no longer needed,
	// it is important to use the Clean method to Delete this subcontext.
	UnscopedSafeGet(name string) (interface{}, error)

	// UnscopedGet is similar to UnscopedSafeGet but it does not return the error.
	UnscopedGet(name string) interface{}

	// UnscopedFill is similar to UnscopedSafeGet but copies the object in dst instead of returning it.
	UnscopedFill(name string, dst interface{}) error

	// NastySafeGet is the previous name of the UnscopedSafeGet method.
	NastySafeGet(name string) (interface{}, error)

	// NastyGet is the previous name of the UnscopedGet method.
	NastyGet(name string) interface{}

	// NastyFill is the previous name of the UnscopedFill method.
	NastyFill(name string, dst interface{}) error

	// Clean deletes the subcontext created by UnscopedSafeGet, UnscopedGet or UnscopedFill.
	Clean()

	// DeleteWithSubContexts takes all the objects saved in this Context
	// and calls the Close function of their Definition on them.
	// It will also call DeleteWithSubContexts on each child and remove its reference in the parent Context.
	// After deletion, the Context can no longer be used.
	DeleteWithSubContexts()

	// Delete works like DeleteWithSubContexts if the Context does not have any child.
	// But if the Context has subcontexts, it will not be deleted right away.
	// The deletion only occurs when all the subcontexts have been deleted.
	// So you have to call Delete or DeleteWithSubContexts on all the subcontexts.
	Delete()

	// IsClosed returns true if the Context has been deleted.
	IsClosed() bool
}
