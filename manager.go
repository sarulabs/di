package di

import (
	"errors"
	"fmt"
)

// Instance is used to register a prebuilt item in a ContextManager.
type Instance struct {
	Name    string
	Aliases []string
	Item    interface{}
}

// Maker is used to define how to build and close an item in a ContextManager.
type Maker struct {
	Name    string
	Aliases []string
	Scope   string
	Make    func(ctx *Context) (interface{}, error)
	Close   func(item interface{})
}

// ContextManager contains the definition of every items.
// You must use the NewContextManager function to create a ContextManager.
type ContextManager struct {
	frozen    bool
	aliases   map[string]string
	instances map[string]interface{}
	makers    map[string]Maker
	scopes    []string
}

// NewContextManager creates a new ContextManager initialized with a list of scopes.
// If a > b then scope[a] is a sub-scope of scope[b].
// For example if scopes are ["app", "request"], then "request" is a sub-scope of "app".
func NewContextManager(scopes ...string) (*ContextManager, error) {
	if err := checkScopes(scopes); err != nil {
		return nil, err
	}

	return &ContextManager{
		aliases:   map[string]string{},
		instances: map[string]interface{}{},
		makers:    map[string]Maker{},
		scopes:    scopes,
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
		for j := i + 1; j < len(scopes); j++ {
			if scope == scopes[j] {
				return fmt.Errorf("at least two scopes are identical")
			}
		}
	}

	return nil
}

// ResolveName returns the real name of an entry by fowling aliases.
// If the name is not used, it returns an empty string and an error.
func (cm *ContextManager) ResolveName(name string) (string, error) {
	if n, ok := cm.aliases[name]; ok {
		return n, nil
	}
	if _, ok := cm.instances[name]; ok {
		return name, nil
	}
	if _, ok := cm.makers[name]; ok {
		return name, nil
	}
	return "", fmt.Errorf("could not resolve name `%s`", name)
}

// NameIsUsed returns true if the name is already used in the ContextManager.
// It can be used as an Instance name, a Maker name, or as an alias.
func (cm *ContextManager) NameIsUsed(name string) bool {
	_, err := cm.ResolveName(name)
	return err == nil
}

func (cm *ContextManager) checkAliases(name string, aliases []string) error {
	for i, alias := range aliases {
		if alias == "" {
			return errors.New("alias can't be empty")
		}
		if cm.NameIsUsed(alias) {
			return fmt.Errorf("alias `%s` is already used", alias)
		}
		if alias == name {
			return errors.New("one of the aliases is the same as the real name")
		}
		for j := i + 1; j < len(aliases); j++ {
			if aliases[j] == alias {
				return fmt.Errorf("there are two aliases named `%s`", alias)
			}
		}
	}

	return nil
}

// Maker adds a Maker to the ContextManager.
// It returns an error if the Maker is not well defined.
func (cm *ContextManager) Maker(maker Maker) error {
	if cm.frozen {
		return errors.New("the ContextManager is frozen because a Context has already been created")
	}

	// check if scope exists
	if !stringSliceContains(cm.scopes, maker.Scope) {
		return fmt.Errorf("scope `%s` is not defined", maker.Scope)
	}

	// check if name is valid and available
	if maker.Name == "" {
		return errors.New("Maker name can't be empty")
	}
	if cm.NameIsUsed(maker.Name) {
		return fmt.Errorf("name `%s` is already used", maker.Name)
	}

	// check if aliases are valid
	if err := cm.checkAliases(maker.Name, maker.Aliases); err != nil {
		return err
	}

	// everything is ok, add the maker
	cm.makers[maker.Name] = maker

	for _, alias := range maker.Aliases {
		cm.aliases[alias] = maker.Name
	}

	return nil
}

// Instance adds an Instance to the ContextManager.
// It returns an error if the name or the aliases are already used.
func (cm *ContextManager) Instance(instance Instance) error {
	if cm.frozen {
		return errors.New("the ContextManager is frozen because a Context has already been created")
	}

	// check if name is valid and available
	if instance.Name == "" {
		return errors.New("Instance name can't be empty")
	}
	if cm.NameIsUsed(instance.Name) {
		return fmt.Errorf("name `%s` is already used", instance.Name)
	}

	// check if aliases are valid
	if err := cm.checkAliases(instance.Name, instance.Aliases); err != nil {
		return err
	}

	// everything is ok, add the instance
	cm.instances[instance.Name] = instance.Item

	for _, alias := range instance.Aliases {
		cm.aliases[alias] = instance.Name
	}

	return nil
}

// Context returns a context for the desired scope.
// You can ask for any scope, not only the first one.
// But if you have two scopes ["app", "request"] and you ask
// for a "request" Context twice, it will create two different "app" Context.
func (cm *ContextManager) Context(scope string) (*Context, error) {
	cm.frozen = true

	ctx := &Context{
		scope:          cm.scopes[0],
		contextManager: cm,
		parent:         nil,
		children:       []*Context{},
		items:          map[string]interface{}{},
	}

	if scope == ctx.scope {
		return ctx, nil
	}

	return ctx.SubContext(scope)
}
