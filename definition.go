package di

// Def contains information to build and close an object inside a Container.
type Def struct {
	Name  string
	Scope string
	Build func(ctn Container) (interface{}, error)
	Close func(obj interface{}) error
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
