package di

// container is the implementation of the Container interface.
type container struct {
	// containerCore contains the container data.
	// Several containers can share the same containerCore.
	// In this case these containers represent the same entity,
	// but at a different stage of an object construction.
	*containerCore

	// built contains the name of the Definition being built by this container.
	// It is used to avoid cycles in object Definitions.
	// Each time a Container is passed in parameter of the Build function
	// of a definition, this is in fact a new container.
	// This container is created with a built field
	// updated with the name of the Definition.
	built []string

	*containerLineage
	*containerSlayer
	*containerGetter
	*containerUnscopedGetter

	logger Logger
}

func (ctn *container) Parent() Container {
	return ctn.containerLineage.Parent(ctn)
}

func (ctn *container) SubContainer() (Container, error) {
	return ctn.containerLineage.SubContainer(ctn)
}

// DEPRECATED
func (ctn *container) SubContext() (Container, error) {
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

// DEPRECATED
func (ctn *container) NastySafeGet(name string) (interface{}, error) {
	return ctn.containerUnscopedGetter.UnscopedSafeGet(ctn, name)
}

// DEPRECATED
func (ctn *container) NastyGet(name string) interface{} {
	return ctn.containerUnscopedGetter.UnscopedGet(ctn, name)
}

// DEPRECATED
func (ctn *container) NastyFill(name string, dst interface{}) error {
	return ctn.containerUnscopedGetter.UnscopedFill(ctn, name, dst)
}

func (ctn *container) Delete() {
	ctn.containerSlayer.Delete(ctn.logger, ctn.containerCore)
}

func (ctn *container) DeleteWithSubContainers() {
	ctn.containerSlayer.DeleteWithSubContainers(ctn.logger, ctn.containerCore)
}

// DEPRECATED
func (ctn *container) DeleteWithSubContexts() {
	ctn.containerSlayer.DeleteWithSubContainers(ctn.logger, ctn.containerCore)
}

func (ctn *container) IsClosed() bool {
	return ctn.containerSlayer.IsClosed(ctn.containerCore)
}

func (ctn *container) Clean() {
	ctn.containerSlayer.Clean(ctn.logger, ctn.containerCore)
}
