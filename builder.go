package di

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
)

// Builder can be used to create a Container.
// The Builder should be created with NewBuilder.
// Then you can add definitions with the Add method,
// and finally build the Container with the Build method.
//
// Consider using the EnhancedBuilder that provides more features.
type Builder struct {
	definitions    DefMap
	scopes         ScopeList
	insertionOrder map[string]int
	numAdded       int
}

// NewBuilder is the only way to create a working Builder.
// It initializes a Builder with a list of scopes.
// The scopes are ordered from the most generic to the most specific.
// If no scope is provided, the default scopes are used:
// [App, Request, SubRequest]
// It can return an error if the scopes are not valid.
func NewBuilder(scopes ...string) (*Builder, error) {
	if len(scopes) == 0 {
		scopes = []string{App, Request, SubRequest}
	}

	if err := checkScopes(scopes); err != nil {
		return nil, err
	}

	return &Builder{
		definitions:    DefMap{},
		scopes:         scopes,
		insertionOrder: map[string]int{},
		numAdded:       0,
	}, nil
}

func checkScopes(scopes []string) error {
	if len(scopes) == 0 {
		return errors.New("at least one scope is required")
	}

	for i, scope := range scopes {
		if scope == "" {
			return errors.New("a scope can not be an empty string")
		}
		if ScopeList(scopes[i+1:]).Contains(scope) {
			return fmt.Errorf("at least two scopes are identical")
		}
	}

	return nil
}

// Scopes returns the list of available scopes.
func (b *Builder) Scopes() ScopeList {
	return ScopeList(b.scopes).Copy()
}

// Definitions returns a map with the all the objects definitions
// registered with the Add method.
// The key of the map is the name of the Definition.
func (b *Builder) Definitions() DefMap {
	return b.definitions.Copy()
}

// IsDefined returns true if there is a definition with the given name.
func (b *Builder) IsDefined(name string) bool {
	_, ok := b.definitions[name]
	return ok
}

// Add adds one or more definitions in the Builder.
// It returns an error if a definition can not be added.
// If a definition with the same name has already been added,
// it will be replaced by the new one, as if the first one never existed.
func (b *Builder) Add(defs ...Def) error {
	for _, def := range defs {
		if err := b.add(def); err != nil {
			return err
		}
	}

	return nil
}

func (b *Builder) add(def Def) error {
	if def.Name == "" {
		return errors.New("name can not be empty")
	}

	// note that an empty scope is allowed
	// it will be replaced in the Build method by the most generic scope
	if def.Scope != "" && !b.scopes.Contains(def.Scope) {
		return fmt.Errorf("scope `%s` is not allowed", def.Scope)
	}

	if def.Build == nil {
		return errors.New("Build can not be nil")
	}

	b.definitions[def.Name] = def
	b.insertionOrder[def.Name] = b.numAdded
	b.numAdded++

	return nil
}

// Set is a shortcut to add a definition for an already built object.
func (b *Builder) Set(name string, obj interface{}) error {
	return b.add(Def{
		Name: name,
		Build: func(ctn Container) (interface{}, error) {
			return obj, nil
		},
	})
}

// Build creates a Container in the most generic scope
// with all the definitions registered in the Builder.
func (b *Builder) Build() Container {
	if err := checkScopes(b.scopes); err != nil {
		return newClosedContainer()
	}

	// Update definition scopes.
	for name, def := range b.definitions {
		if def.Scope == "" {
			def.Scope = b.scopes[0]
		}
		b.definitions[name] = def
	}

	// Put definitions in a slice and sort them by insertion order.
	definitions := []Def{}

	for _, def := range b.definitions {
		definitions = append(definitions, def)
	}

	sort.Slice(definitions, func(i, j int) bool {
		return b.insertionOrder[definitions[i].Name] < b.insertionOrder[definitions[j].Name]
	})

	// Generate the indexes based on the definitions.
	indexesByName := make(map[string]int, len(definitions))
	indexesByType := map[reflect.Type][]int{}
	definitionScopeLevels := make([]int, len(definitions))

	for index, def := range definitions {
		// Update the definition bound fields.
		def.builderBound = true
		def.builderIndex = index
		definitions[index] = def

		// Update indexes and definitionScopeLevels slices.
		indexesByName[def.Name] = index
		for _, defType := range def.Is {
			indexesByType[defType] = append(indexesByType[defType], index)
		}
		for i, s := range b.scopes {
			if s == def.Scope {
				definitionScopeLevels[index] = i
				break
			}
		}
	}

	return Container{
		core: &containerCore{
			closed: false,

			scopes:     b.scopes,
			scopeLevel: 0,

			parent:          nil,
			children:        map[*containerCore]struct{}{},
			unscopedChild:   nil,
			deleteIfNoChild: false,

			indexesByName:         indexesByName,
			indexesByType:         indexesByType,
			definitions:           definitions,
			objects:               make([]interface{}, len(indexesByName)),
			definitionScopeLevels: definitionScopeLevels,
			isBuilt:               make([]int32, len(indexesByName)),
			building:              make([]*buildingChan, len(indexesByName)),

			unshared:      []interface{}{},
			unsharedIndex: []int{},

			dependencies: newGraph(),
		},
		builtList: make([]int, 0, 10),
	}
}
