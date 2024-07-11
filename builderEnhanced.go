package di

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

const generatedNamePrefix = "_di_generated_"

// EnhancedBuilder can be used to create a Container.
// The EnhancedBuilder should be created with NewEnhancedBuilder.
// Then you can add definitions with the Add method.
// Once all the definitions have been added to the builder,
// you can generate the Container with the Build method.
//
// It works the same way as the basic Builder. But the definitions given to the Add method are pointers.
// The definitions are updated when the Build method is called.
// That allows to retrieve objects by their definitions which is faster than retrieving them by name.
type EnhancedBuilder struct {
	definitions    DefMap
	bindings       map[string]*Def
	insertionOrder map[string]int
	numAdded       int
	scopes         ScopeList
}

// NewEnhancedBuilder is the only way to create a working EnhancedBuilder.
// It initializes an EnhancedBuilder with a list of scopes.
// The scopes are ordered from the most generic to the most specific.
// If no scope is provided, the default scopes are used:
// [App, Request, SubRequest]
// It can return an error if the scopes are not valid.
func NewEnhancedBuilder(scopes ...string) (*EnhancedBuilder, error) {
	if len(scopes) == 0 {
		scopes = []string{App, Request, SubRequest}
	}

	if err := checkBuilderScopes(scopes); err != nil {
		return nil, err
	}

	return &EnhancedBuilder{
		definitions:    DefMap{},
		bindings:       map[string]*Def{},
		insertionOrder: map[string]int{},
		numAdded:       0,
		scopes:         scopes,
	}, nil
}

func checkBuilderScopes(scopes []string) error {
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
func (b *EnhancedBuilder) Scopes() ScopeList {
	return ScopeList(b.scopes).Copy()
}

// Definitions returns a map with the all objects definitions registered at this point.
// The key of the map is the name of the definition.
func (b *EnhancedBuilder) Definitions() DefMap {
	return b.definitions.Copy()
}

// NameIsDefined returns true if there is a definition registered with the given name.
func (b *EnhancedBuilder) NameIsDefined(name string) bool {
	_, ok := b.definitions[name]
	return ok
}

// Add adds one definition to the Builder.
// It returns an error if the definition can not be added.
//
// The name must be unique. If a definition with the same name has already been added,
// it will be replaced by the new one, as if the first one never was added.
// If an empty name is provided, a name starting with "_di_generated_" is generated.
// You can not add a definition with a name starting with "_di_generated_" as it is reserved for auto-genrated ones.
// Providing a name is recommended as it makes errors much easier to understand.
//
// The input definition is a pointer.
// It will be updated when the container is generated with the Build method.
// It binds the definition to the generated Container.
// That allows to build an object not only from its name
// but also from its definition which happens to be faster.
func (b *EnhancedBuilder) Add(def *Def) error {
	if def == nil {
		return errors.New("the definition can not be nil")
	}

	if len(b.scopes) == 0 {
		return errors.New("the builder was not created with NewEnhancedBuilder")
	}

	if def.Scope != "" && !b.scopes.Contains(def.Scope) {
		return fmt.Errorf("scope `%s` is not allowed", def.Scope)
	}

	if def.Build == nil {
		return errors.New("the Build function can not be nil")
	}

	if strings.HasPrefix(def.Name, generatedNamePrefix) {
		return errors.New("the definition name can not start by `" + generatedNamePrefix + "`")
	}

	defStruct := *def

	if defStruct.Name == "" {
		defStruct.Name = generatedNamePrefix + strconv.Itoa(b.numAdded)
	}

	if defStruct.Is != nil {
		defStruct.Is = make([]reflect.Type, len(def.Is))
		copy(defStruct.Is, def.Is)
	}

	b.definitions[defStruct.Name] = defStruct
	b.bindings[defStruct.Name] = def
	b.insertionOrder[defStruct.Name] = b.numAdded
	b.numAdded++

	return nil
}

// Build creates a Container in the most generic scope
// with all the definitions registered in the builder.
//
// The definition provided in the Add method
// are updated to match their state when they were added to the builder.
//
// A definition can only belong to one container.
// That means you can only call Build once.
func (b *EnhancedBuilder) Build() (Container, error) {
	if err := checkBuilderScopes(b.scopes); err != nil {
		return newClosedContainer(), err
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
		// Update the bound fields of the definition.
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

		// Update the bound definition.
		if b.bindings[def.Name].builderBound {
			return newClosedContainer(), errors.New("the definition `" + def.Name + "` was already added to another container")
		}
		b.bindings[def.Name].Build = def.Build
		b.bindings[def.Name].Close = def.Close
		b.bindings[def.Name].Name = def.Name
		b.bindings[def.Name].Scope = def.Scope
		b.bindings[def.Name].Unshared = def.Unshared
		b.bindings[def.Name].Is = def.Is
		b.bindings[def.Name].Tags = def.Tags
		b.bindings[def.Name].builderBound = true
		b.bindings[def.Name].builderIndex = def.builderIndex
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
	}, nil
}
