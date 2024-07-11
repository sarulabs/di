package di

import (
	"reflect"
	"sync"
)

// Container represents a dependency injection container.
// To create a Container, you should use a Builder, an EnhancedBuilder or another Container.
//
// A Container has a scope and may have a parent in a more generic scope
// and children in a more specific scope.
// Objects can be retrieved from the Container.
// If the requested object does not already exist in the Container,
// it is built thanks to the object definition.
type Container struct {
	// containerCore contains the container data.
	// Several Container can share the same containerCore.
	// In this case these Container represent the same entity,
	// but at a different stage of an object construction.
	// They differ by their builtList field.
	core *containerCore

	// builtList contains the indexes of the definitions that have already been built by this Container.
	// It is used to avoid cycles in object definitions.
	// Each time a Container is passed as parameter of the Build function
	// of a definition, this is in fact a new Container.
	// Is has the same core but an updated builtList field.
	builtList []int
}

// containerCore contains the data of a Container.
// But it can not build objects on its own.
// It should be used inside a container.
type containerCore struct {
	m      sync.RWMutex
	closed bool

	// scopes
	scopes     ScopeList
	scopeLevel int

	// lineage
	parent          *containerCore
	children        map[*containerCore]struct{}
	unscopedChild   *containerCore
	deleteIfNoChild bool

	// definitions and objects
	indexesByName         map[string]int
	indexesByType         map[reflect.Type][]int
	definitions           []Def
	definitionScopeLevels []int
	objects               []interface{}
	isBuilt               []int32
	building              []*buildingChan

	// unshared objects are stored separately in unshared.
	// The index of their definition is stored in unsharedIndex at the same position as in unshared.
	unshared      []interface{}
	unsharedIndex []int

	// dependencies is a graph that allows to determine
	// in which order the definitions should be closed.
	// Each vertice is an index. If >= 0 it is the index of a shared object.
	// If < 0, it is the opposite of the index in unshared minus 1.
	// For example the first object in unshared is at position 0, so its vertice is -0-1=-1.
	dependencies *graph
}

// Definitions returns the map of the available definitions ordered by name.
// These definitions represent all the objects that this Container can build.
func (ctn Container) Definitions() map[string]Def {
	defs := make(map[string]Def, len(ctn.core.definitions))

	for _, def := range ctn.core.definitions {
		defs[def.Name] = def
	}

	return defs
}

// NameIsDefined returns true if there is a definition for the given name.
func (ctn Container) NameIsDefined(name string) bool {
	_, ok := ctn.core.indexesByName[name]
	return ok
}

// TypeIsDefined returns true if there is a definition for the given type.
// Types are declared in the Is field of a definition.
func (ctn Container) TypeIsDefined(typ reflect.Type) bool {
	_, ok := ctn.core.indexesByType[typ]
	return ok
}

// DefinitionsForType returns the list of the definitions matching the given type.
// Types are declared in the Is field of a definition.
func (ctn Container) DefinitionsForType(typ reflect.Type) []Def {
	indexes := ctn.core.indexesByType[typ]
	defs := make([]Def, 0, len(indexes))

	for _, index := range indexes {
		defs = append(defs, ctn.core.definitions[index])
	}

	return defs
}

// Scope returns the Container scope.
func (ctn Container) Scope() string {
	return ctn.core.scopes[ctn.core.scopeLevel]
}

// Scopes returns the list of available scopes.
func (ctn Container) Scopes() []string {
	return ctn.core.scopes.Copy()
}

// ParentScopes returns the list of scopes that are more generic than the Container scope.
func (ctn Container) ParentScopes() []string {
	return ctn.core.scopes.ParentScopes(ctn.Scope())
}

// SubScopes returns the list of scopes that are more specific than the Container scope.
func (ctn Container) SubScopes() []string {
	return ctn.core.scopes.SubScopes(ctn.Scope())
}

// newClosedContainer returns a closed container. It is not usable and is returned when there is an error.
func newClosedContainer() Container {
	return Container{
		core: &containerCore{
			closed: true,

			scopes:     []string{},
			scopeLevel: 0,

			parent:          nil,
			children:        map[*containerCore]struct{}{},
			unscopedChild:   nil,
			deleteIfNoChild: false,

			indexesByName:         map[string]int{},
			indexesByType:         map[reflect.Type][]int{},
			definitions:           []Def{},
			objects:               []interface{}{},
			definitionScopeLevels: []int{},
			isBuilt:               []int32{},
			building:              []*buildingChan{},

			unshared:      []interface{}{},
			unsharedIndex: []int{},

			dependencies: newGraph(),
		},
		builtList: make([]int, 0, 10),
	}
}
