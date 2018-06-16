package di

// Definition contains information to build and close an object inside a Container.
type Definition struct {
	Name  string
	Scope string
	Build func(ctn Container) (interface{}, error)
	Close func(obj interface{})
	Tags  []Tag
}

// Tag can contain more specific information about a Definition.
// It is useful to find a Definition thanks to its tags instead of its name.
type Tag struct {
	Name string
	Args map[string]string
}

// DefinitionMap is a collection of Definition ordered by name.
type DefinitionMap map[string]Definition

// Copy returns a copy of the DefinitionMap.
func (m DefinitionMap) Copy() map[string]Definition {
	defs := map[string]Definition{}

	for name, def := range m {
		defs[name] = def
	}

	return defs
}
