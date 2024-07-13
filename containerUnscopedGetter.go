package di

import (
	"errors"
	"fmt"
	"reflect"
)

// UnscopedSafeGet retrieves an object from the Container, like SafeGet.
// The difference is that the object can be retrieved
// even if it belongs to a more specific scope.
// To do so, UnscopedSafeGet creates a sub-container.
// When the created object is no longer needed,
// it is important to use the Clean method to delete this sub-container.
//
// /!\ Do not use unscope functions inside a `Build` function.
// In this case, circular definitions are not detected. If you do this,
// you take the risk of having an infinite loop in your code when building an object.
func (ctn Container) UnscopedSafeGet(in interface{}) (interface{}, error) {
	var index int

	switch v := in.(type) {
	case int:
		index = v
	case Def:
		index = v.Index()
	case *Def:
		index = v.Index()
	case string:
		var ok bool
		index, ok = ctn.core.indexesByName[v]
		if !ok {
			return nil, fmt.Errorf("could not get `%s` because the definition does not exist", v)
		}
	case reflect.Type:
		indexes := ctn.core.indexesByType[v]
		if len(indexes) == 0 {
			return nil, fmt.Errorf("could not get type `%s` because it is not defined", v)
		}
		index = indexes[len(indexes)-1]
	}

	if index < 0 || index >= len(ctn.core.definitionScopeLevels) {
		return nil, fmt.Errorf("could not get index `%d` because it does not exist", index)
	}

	if ctn.core.definitionScopeLevels[index] <= ctn.core.scopeLevel {
		return ctn.SafeGet(index) // There was no need to call UnscopedSafeGet, SafeGet was enough.
	}

	child, err := ctn.getUnscopedChild()
	if err != nil {
		return nil, fmt.Errorf("could not get `%s` because %+v", ctn.core.definitions[index].Name, err)
	}

	return child.UnscopedSafeGet(index)
}

// UnscopedGet is similar to UnscopedSafeGet but it does not return the error.
// Instead it panics.
func (ctn Container) UnscopedGet(in interface{}) interface{} {
	obj, err := ctn.UnscopedSafeGet(in)
	if err != nil {
		panic(err)
	}

	return obj
}

// UnscopedFill is similar to UnscopedSafeGet but copies the object in dst instead of returning it.
func (ctn Container) UnscopedFill(in interface{}, dst interface{}) error {
	obj, err := ctn.UnscopedSafeGet(in)
	if err != nil {
		return err
	}

	return fill(obj, dst)
}

func (ctn Container) getUnscopedChild() (Container, error) {
	ctn.core.m.Lock()
	unscopedChild := ctn.core.unscopedChild
	ctn.core.m.Unlock()

	if unscopedChild == nil {
		return ctn.addUnscopedChild()
	}

	return Container{
		core:      unscopedChild,
		builtList: make([]int, 0, 10),
	}, nil
}

func (ctn Container) addUnscopedChild() (Container, error) {
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

	ctn.core.unscopedChild = child.core

	ctn.core.m.Unlock()

	return child, nil
}
