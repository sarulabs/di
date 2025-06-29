package di

import (
	"fmt"
	"reflect"
	"sync/atomic"
)

// Get retrieves an object from the Container.
// The object has to belong to the Container or one of its parents.
// If the object does not already exist, it is created and saved in the Container.
// If the object can not be created, it panics.
//
// There are different ways to retrieve an object.
//   - From its name: ctn.Get("object-name")
//   - From its definition: ctn.Get(objectDef) or ctn.Get(objectDefPtr) - only with the EnhancedBuilder
//   - From its index: ctn.Get(objectDef.Index()) - only with the EnhancedBuilder
//   - From its type: ctn.Get(reflect.typeOf(MyObject{})) - only if objectDef.Is includes the given type
//     In case there are more than one definition matching the given type,
//     the chosen one is the last definition inserted in the builder.
func (ctn Container) Get(in interface{}) interface{} {
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
			panic(fmt.Errorf("could not get `%s` because the definition does not exist", v))
		}
	case reflect.Type:
		indexes := ctn.core.indexesByType[v]
		if len(indexes) == 0 {
			panic(fmt.Errorf("could not get type `%s` because it is not defined", v))
		} else {
			index = indexes[len(indexes)-1]
		}
	default:
		panic(fmt.Errorf("could not get `%#v` because the argument is not valid (int, Def, *Def, string or reflect.Type allowed)", in))
	}

	if index < 0 || index >= len(ctn.core.definitionScopeLevels) {
		panic(fmt.Errorf("could not get index `%d` because it does not exist", index))
	}

	// Finding the right core.
	inputCore := ctn.core
	core := ctn.core

	if core.definitionScopeLevels[index] != core.scopeLevel {
		for core.definitionScopeLevels[index] != core.scopeLevel {
			core = core.parent

			if core == nil {
				panic(fmt.Errorf(
					"could not get `%s` because it requires `%s` scope which does not match this container scope or any of its parents scope",
					inputCore.definitions[index].Name,
					inputCore.definitions[index].Scope,
				))
			}
		}
	}

	if atomic.LoadInt32(&core.isBuilt[index]) == 1 {
		return core.objects[index] // Try to fetch an already built object as quickly as possible.
	}

	if inputCore != core {
		ctn.core = core
		ctn.builtList = make([]int, 0, 10) // Reset the builtList if the scope changed.
	}

	// Retrieve the definition.
	def := core.definitions[index]

	// Cycle detection.
	if len(ctn.builtList) > 0 {
		for _, builtIndex := range ctn.builtList {
			if builtIndex == index {
				panic(formatCycleError(ctn, def))
			}
		}
	}

	// Handle unshared objects.
	if def.Unshared {
		obj, err := buildObject(def.Build, ctn, index, def.Name)

		if err != nil {
			panic(fmt.Errorf("could not build `%s`: %+v", def.Name, err))
		}

		if def.Close == nil {
			return obj
		}

		core.m.Lock()
		if core.closed {
			core.m.Unlock()
			err := closeObject(obj, def.Close, def.Name)
			panic(formatBuiltOnClosedContainerError(def, err))
		}
		core.unshared = append(core.unshared, obj)
		core.unsharedIndex = append(core.unsharedIndex, index)
		if len(ctn.builtList) == 0 {
			core.dependencies.AddVertex(-len(core.unshared))
		} else {
			core.dependencies.AddEdge(ctn.builtList[len(ctn.builtList)-1], -len(core.unshared))
		}
		core.m.Unlock()

		return obj
	}

	// Handle shared objects.
	core.m.Lock()
	if core.closed {
		core.m.Unlock()
		panic(fmt.Errorf(
			"could not get `%s` because the container has been deleted", def.Name,
		))
	}

	if atomic.LoadInt32(&core.isBuilt[index]) == 1 { // Check again if the object was created, with the lock this time.
		core.m.Unlock()
		return core.objects[index]
	}

	if building := core.building[index]; building != nil {
		core.m.Unlock()
		<-(*building)         // Wait for the object to be created by another call to Get.
		return ctn.Get(index) // Can not get the object without calling Get again as its creation may have failed.
	}

	building := make(buildingChan)
	core.building[index] = &building // Mark the object as building.
	core.m.Unlock()                  // And release the lock as it can take a while to create the object.

	// Building the shared object.
	obj, err := buildObject(def.Build, ctn, index, def.Name)

	core.m.Lock()

	if err != nil {
		// The object could not be created. Remove the building channel from the container
		// and close it to allow the object to be created again.
		core.building[index] = nil
		core.m.Unlock()
		close(building)
		panic(err)
	}

	if core.closed {
		// The container has been deleted while the object was being built.
		// The newly created object needs to be closed, and it will not be returned.
		core.m.Unlock()
		close(building)
		err = closeObject(obj, def.Close, def.Name)
		panic(formatBuiltOnClosedContainerError(def, err))
	}

	if len(ctn.builtList) == 0 {
		core.dependencies.AddVertex(index)
	} else {
		core.dependencies.AddEdge(ctn.builtList[len(ctn.builtList)-1], index)
	}
	core.objects[index] = obj
	atomic.StoreInt32(&core.isBuilt[index], 1)
	core.m.Unlock()
	close(building)

	return obj
}
