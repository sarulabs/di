package di

import "sync"

// containerCore contains the data of a Container.
// But it can not build objects on its own.
// It should be used inside a container.
type containerCore struct {
	m               sync.RWMutex
	closed          bool
	scope           string
	scopes          ScopeList
	definitions     DefMap
	parent          *containerCore
	children        map[*containerCore]struct{}
	unscopedChild   *containerCore
	objects         map[string]interface{}
	deleteIfNoChild bool
}

func (ctn *containerCore) Definitions() map[string]Def {
	return ctn.definitions.Copy()
}

func (ctn *containerCore) Scope() string {
	return ctn.scope
}

func (ctn *containerCore) Scopes() []string {
	return ctn.scopes.Copy()
}

func (ctn *containerCore) ParentScopes() []string {
	return ctn.scopes.ParentScopes(ctn.scope)
}

func (ctn *containerCore) SubScopes() []string {
	return ctn.scopes.SubScopes(ctn.scope)
}
