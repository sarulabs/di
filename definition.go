package di

// Def contains information to build and close an object inside a Container.
type Def struct {
	// Build is the function that is used to create the object.
	Build func(ctn Container) (interface{}, error)
	// Close is the function that is used to clean the object when the container is deleted.
	// It can be nil if nothing needs to be done to close the object.
	Close func(obj interface{}) error
	// Name is the key that is used to retrieve the object from the container.
	Name string
	// Scope determines in which container the object is stored.
	// Typical scopes are "app" and "request".
	Scope string
	// Tags are not used inside this library. But they can be useful to sort your definitions.
	Tags []Tag
	// Unshared is false by default. That means that the object is only created once in a given container.
	// They are singleton and the same instance will be returned each time "Get", "SafeGet" or "Fill" is called.
	// If you want to retrieve a new object every time, "Unshared" needs to be set to true.
	Unshared bool
}

// Tag can contain more specific information about a Definition.
// It is useful to find a Definition thanks to its tags instead of its name.
type Tag struct {
	Name string
	Args map[string]string
}

// DefMap is a collection of Def ordered by name.
type DefMap map[string]Def

// Copy returns a copy of the DefMap.
func (m DefMap) Copy() DefMap {
	defs := DefMap{}

	for name, def := range m {
		defs[name] = def
	}

	return defs
}
