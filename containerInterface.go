package di

// Context is the previous name of the Container interface.
// DEPRECATED: will be removed in v2.
type Context = Container

// Container represents a dependency injection container.
// A Container has a scope and may have a parent with a wider scope
// and children with a narrower scope.
// Objects can be retrieved from the Container.
// If the desired object does not already exist in the Container,
// it is built thanks to the object Definition.
// The following attempts to get this object will return the same object.
type Container interface {
	// Definition returns the map of the available Definitions ordered by Definition name.
	// These Definitions represent all the objects that this Container can build.
	Definitions() map[string]Definition

	// Scope returns the Container scope.
	Scope() string

	// Scopes returns the list of available scopes.
	Scopes() []string

	// ParentScopes returns the list of scopes wider than the Container scope.
	ParentScopes() []string

	// SubScopes returns the list of scopes narrower than the Container scope.
	SubScopes() []string

	// Parent returns the parent Container.
	Parent() Container

	// SubContainer creates a new Container in the next subscope
	// that will have this Container as parent.
	SubContainer() (Container, error)

	// SubContext is the previous name of the SubContainer method.
	// DEPRECATED: will be removed in v2.
	SubContext() (Container, error)

	// SafeGet retrieves an object from the Container.
	// The object has to belong to this scope or a wider one.
	// If the object does not already exist, it is created and saved in the Container.
	// If the object can't be created, it returns an error.
	SafeGet(name string) (interface{}, error)

	// Get is similar to SafeGet but it does not return the error.
	Get(name string) interface{}

	// Fill is similar to SafeGet but it does not return the object.
	// Instead it fills the provided object with the value returned by SafeGet.
	// The provided object must be a pointer to the value returned by SafeGet.
	Fill(name string, dst interface{}) error

	// UnscopedSafeGet retrieves an object from the Container, like SafeGet.
	// The difference is that the object can be retrieved
	// even if it belongs to a narrower scope.
	// To do so UnscopedSafeGet creates a sub-container.
	// When the created object is no longer needed,
	// it is important to use the Clean method to Delete this sub-container.
	UnscopedSafeGet(name string) (interface{}, error)

	// UnscopedGet is similar to UnscopedSafeGet but it does not return the error.
	UnscopedGet(name string) interface{}

	// UnscopedFill is similar to UnscopedSafeGet but copies the object in dst instead of returning it.
	UnscopedFill(name string, dst interface{}) error

	// NastySafeGet is the previous name of the UnscopedSafeGet method.
	// DEPRECATED: will be removed in v2.
	NastySafeGet(name string) (interface{}, error)

	// NastyGet is the previous name of the UnscopedGet method.
	// DEPRECATED: will be removed in v2.
	NastyGet(name string) interface{}

	// NastyFill is the previous name of the UnscopedFill method.
	// DEPRECATED: will be removed in v2.
	NastyFill(name string, dst interface{}) error

	// Clean deletes the sub-container created by UnscopedSafeGet, UnscopedGet or UnscopedFill.
	Clean()

	// DeleteWithSubContainers takes all the objects saved in this Container
	// and calls the Close function of their Definition on them.
	// It will also call DeleteWithSubContainers on each child and remove its reference in the parent Container.
	// After deletion, the Container can no longer be used.
	DeleteWithSubContainers()

	// DeleteWithSubContexts is the previous name of the DeleteWithSubContainers method.
	// DEPRECATED: will be removed in v2.
	DeleteWithSubContexts()

	// Delete works like DeleteWithSubContainers if the Container does not have any child.
	// But if the Container has sub-containers, it will not be deleted right away.
	// The deletion only occurs when all the sub-containers have been deleted.
	// So you have to call Delete or DeleteWithSubContainers on all the sub-containers.
	Delete()

	// IsClosed returns true if the Container has been deleted.
	IsClosed() bool
}
