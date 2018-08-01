package di

// container is the implementation of the Container interface.
type container struct {
	// containerCore contains the container data.
	// Several Container can share the same containerCore.
	// In this case these Container represent the same entity,
	// but at a different stage of an object construction.
	// They differ by their built field.
	*containerCore

	// builtList contains the name of the definitions already built by this Container.
	// It is used to avoid cycles in object definitions.
	// Each time a Container is passed as parameter of the Build function
	// of a definition, this is in fact a new Container.
	// Is has the same core but an updated built field.
	builtList builtList

	*containerLineage
	*containerSlayer
	*containerGetter
	*containerUnscopedGetter
}

func (ctn *container) Parent() Container {
	return ctn.containerLineage.Parent(ctn)
}

func (ctn *container) SubContainer() (Container, error) {
	return ctn.containerLineage.SubContainer(ctn)
}

func (ctn *container) SafeGet(name string) (interface{}, error) {
	return ctn.containerGetter.SafeGet(ctn, name)
}

func (ctn *container) Get(name string) interface{} {
	return ctn.containerGetter.Get(ctn, name)
}

func (ctn *container) Fill(name string, dst interface{}) error {
	return ctn.containerGetter.Fill(ctn, name, dst)
}

func (ctn *container) UnscopedSafeGet(name string) (interface{}, error) {
	return ctn.containerUnscopedGetter.UnscopedSafeGet(ctn, name)
}

func (ctn *container) UnscopedGet(name string) interface{} {
	return ctn.containerUnscopedGetter.UnscopedGet(ctn, name)
}

func (ctn *container) UnscopedFill(name string, dst interface{}) error {
	return ctn.containerUnscopedGetter.UnscopedFill(ctn, name, dst)
}

func (ctn *container) Clean() error {
	return ctn.containerSlayer.Clean(ctn.containerCore)
}

func (ctn *container) Delete() error {
	return ctn.containerSlayer.Delete(ctn.containerCore)
}

func (ctn *container) DeleteWithSubContainers() error {
	return ctn.containerSlayer.DeleteWithSubContainers(ctn.containerCore)
}

func (ctn *container) IsClosed() bool {
	return ctn.containerSlayer.IsClosed(ctn.containerCore)
}
