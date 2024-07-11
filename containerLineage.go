package di

import (
	"errors"
	"fmt"
)

// Parent returns the parent Container.
func (ctn Container) Parent() Container {
	return Container{
		core:      ctn.core.parent,
		builtList: ctn.builtList,
	}
}

// SubContainer creates a new Container in the next sub-scope
// that will have this Container as parent.
func (ctn Container) SubContainer() (Container, error) {
	if 1+ctn.core.scopeLevel >= len(ctn.core.scopes) {
		return Container{}, fmt.Errorf("there is no more specific scope than `%s`", ctn.core.scopes[ctn.core.scopeLevel])
	}

	child := Container{
		core: &containerCore{
			closed: false,

			scopes:     ctn.core.scopes,
			scopeLevel: ctn.core.scopeLevel + 1,

			parent:          ctn.core,
			children:        map[*containerCore]struct{}{},
			unscopedChild:   nil,
			deleteIfNoChild: false,

			indexesByName:         ctn.core.indexesByName,
			indexesByType:         ctn.core.indexesByType,
			definitions:           ctn.core.definitions,
			definitionScopeLevels: ctn.core.definitionScopeLevels,
			objects:               make([]interface{}, len(ctn.core.indexesByName)),
			isBuilt:               make([]int32, len(ctn.core.indexesByName)),
			building:              make([]*buildingChan, len(ctn.core.indexesByName)),
			unshared:              []interface{}{},
			unsharedIndex:         []int{},

			dependencies: newGraph(),
		},
		builtList: make([]int, 0, 10),
	}

	ctn.core.m.Lock()

	if ctn.core.closed {
		ctn.core.m.Unlock()
		return Container{}, errors.New("the container is closed")
	}

	ctn.core.children[child.core] = struct{}{}

	ctn.core.m.Unlock()

	return child, nil
}
