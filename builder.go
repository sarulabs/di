package di

import (
	"errors"
	"fmt"
)

// Builder is the only way to create a working Container.
// The scopes and object definitions are set in the Builder
// that can create a Container based on this information.
type Builder struct {
	Logger      Logger
	definitions map[string]Definition
	scopes      []string
}

// NewBuilder is the only way to create a working Builder.
// It initializes the Builder with a list of scopes.
// The scope are ordered from the wider to the narrower.
// If no scope is provided, the default scopes are used:
// [App, Request, SubRequest]
func NewBuilder(scopes ...string) (*Builder, error) {
	if len(scopes) == 0 {
		scopes = []string{App, Request, SubRequest}
	}

	if err := checkScopes(scopes); err != nil {
		return nil, err
	}

	return &Builder{
		definitions: map[string]Definition{},
		scopes:      scopes,
	}, nil
}

func checkScopes(scopes []string) error {
	if len(scopes) == 0 {
		return errors.New("at least one scope is required")
	}

	for i, scope := range scopes {
		if scope == "" {
			return errors.New("a scope can't be an empty string")
		}
		if stringSliceContains(scopes[i+1:], scope) {
			return fmt.Errorf("at least two scopes are identical")
		}
	}

	return nil
}

// Scopes returns the list of available scopes.
func (b *Builder) Scopes() []string {
	scopes := make([]string, len(b.scopes))
	copy(scopes, b.scopes)
	return scopes
}

// Definitions returns a map with the objects definitions added with the AddDefinition method.
// The key of the map is the name of the Definition.
func (b *Builder) Definitions() map[string]Definition {
	defs := map[string]Definition{}

	for name, def := range b.definitions {
		defs[name] = def
	}

	return defs
}

// IsDefined returns true if there is a definition with the given name.
func (b *Builder) IsDefined(name string) bool {
	_, ok := b.definitions[name]
	return ok
}

// AddDefinition adds an object Definition in the Builder.
// It returns an error if the Definition can't be added.
func (b *Builder) AddDefinition(def Definition) error {
	if err := b.checkName(def.Name); err != nil {
		return err
	}

	if def.Scope != "" && !stringSliceContains(b.scopes, def.Scope) {
		return fmt.Errorf("scope `%s` is not defined", def.Scope)
	}

	if def.Build == nil {
		return errors.New("Build can't be nil")
	}

	b.definitions[def.Name] = def

	return nil
}

func (b *Builder) checkName(name string) error {
	if name == "" {
		return errors.New("name can't be empty")
	}

	if b.IsDefined(name) {
		return fmt.Errorf("name `%s` is already used", name)
	}

	return nil
}

// Set adds a definition for an already build object.
// The scope used as the Definition scope is the Builder wider scope.
func (b *Builder) Set(name string, obj interface{}) error {
	if err := checkScopes(b.scopes); err != nil {
		return err
	}

	return b.AddDefinition(Definition{
		Name:  name,
		Scope: b.scopes[0],
		Build: func(ctn Container) (interface{}, error) {
			return obj, nil
		},
	})
}

// Build creates a Container in the wider scope
// with all the current scopes and definitions.
func (b *Builder) Build() Container {
	if err := checkScopes(b.scopes); err != nil {
		return nil
	}

	defs := b.Definitions()

	for name, def := range defs {
		if def.Scope == "" {
			def.Scope = b.scopes[0]
			defs[name] = def
		}
	}

	logger := b.Logger

	if logger == nil {
		logger = &MuteLogger{}
	}

	return &container{
		containerCore: &containerCore{
			scopes:      b.scopes,
			scope:       b.scopes[0],
			definitions: defs,
			parent:      nil,
			children:    []*containerCore{},
			objects:     map[string]interface{}{},
		},
		logger: logger,
	}
}
