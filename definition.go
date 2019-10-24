package di

// Def contains information to build and close an object inside a Container.
type Def struct {
	Build    func(ctn Container) (interface{}, error)
	Close    func(obj interface{}) error
	Name     string
	Scope    string
	Tags     []Tag
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
