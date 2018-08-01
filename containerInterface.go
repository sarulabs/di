package di

// Container represents a dependency injection container.
// To create a Container, you should use a Builder or another Container.
//
// A Container has a scope and may have a parent in a more generic scope
// and children in a more specific scope.
// Objects can be retrieved from the Container.
// If the requested object does not already exist in the Container,
// it is built thanks to the object definition.
// The following attempts to get this object will return the same object.
type Container interface {
	// Definition returns the map of the available definitions ordered by name.
	// These definitions represent all the objects that this Container can build.
	Definitions() map[string]Def

	// Scope returns the Container scope.
	Scope() string

	// Scopes returns the list of available scopes.
	Scopes() []string

	// ParentScopes returns the list of scopes that are more generic than the Container scope.
	ParentScopes() []string

	// SubScopes returns the list of scopes that are more specific than the Container scope.
	SubScopes() []string

	// Parent returns the parent Container.
	Parent() Container

	// SubContainer creates a new Container in the next sub-scope
	// that will have this Container as parent.
	SubContainer() (Container, error)

	// SafeGet retrieves an object from the Container.
	// The object has to belong to this scope or a more generic one.
	// If the object does not already exist, it is created and saved in the Container.
	// If the object can not be created, it returns an error.
	SafeGet(name string) (interface{}, error)

	// Get is similar to SafeGet but it does not return the error.
	// Instead it panics.
	Get(name string) interface{}

	// Fill is similar to SafeGet but it does not return the object.
	// Instead it fills the provided object with the value returned by SafeGet.
	// The provided object must be a pointer to the value returned by SafeGet.
	Fill(name string, dst interface{}) error

	// UnscopedSafeGet retrieves an object from the Container, like SafeGet.
	// The difference is that the object can be retrieved
	// even if it belongs to a more specific scope.
	// To do so, UnscopedSafeGet creates a sub-container.
	// When the created object is no longer needed,
	// it is important to use the Clean method to delete this sub-container.
	UnscopedSafeGet(name string) (interface{}, error)

	// UnscopedGet is similar to UnscopedSafeGet but it does not return the error.
	// Instead it panics.
	UnscopedGet(name string) interface{}

	// UnscopedFill is similar to UnscopedSafeGet but copies the object in dst instead of returning it.
	UnscopedFill(name string, dst interface{}) error

	// Clean deletes the sub-container created by UnscopedSafeGet, UnscopedGet or UnscopedFill.
	Clean() error

	// DeleteWithSubContainers takes all the objects saved in this Container
	// and calls the Close function of their Definition on them.
	// It will also call DeleteWithSubContainers on each child and remove its reference in the parent Container.
	// After deletion, the Container can no longer be used.
	// The sub-containers are deleted even if they are still used in other goroutines.
	// It can cause errors. You may want to use the Delete method instead.
	DeleteWithSubContainers() error

	// Delete works like DeleteWithSubContainers if the Container does not have any child.
	// But if the Container has sub-containers, it will not be deleted right away.
	// The deletion only occurs when all the sub-containers have been deleted manually.
	// So you have to call Delete or DeleteWithSubContainers on all the sub-containers.
	Delete() error

	// IsClosed returns true if the Container has been deleted.
	IsClosed() bool
}
