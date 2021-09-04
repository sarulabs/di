package di

import (
	"fmt"
	"sync/atomic"
)

// DeleteWithSubContainers takes all the objects saved in this Container
// and calls the Close function of their Definition on them.
// It will also call DeleteWithSubContainers on each child and remove its reference in the parent Container.
// After deletion, the Container can no longer be used.
// The sub-containers are deleted even if they are still used in other goroutines.
// It can cause errors. You may want to use the Delete method instead.
func (ctn Container) DeleteWithSubContainers() error {
	return deleteContainerCore(ctn.core)
}

// Delete works like DeleteWithSubContainers if the Container does not have any child.
// But if the Container has sub-containers, it will not be deleted right away.
// The deletion only occurs when all the sub-containers have been deleted manually.
// So you have to call Delete or DeleteWithSubContainers on all the sub-containers.
func (ctn Container) Delete() error {
	ctn.core.m.Lock()

	if len(ctn.core.children) > 0 {
		ctn.core.deleteIfNoChild = true
		ctn.core.m.Unlock()
		return nil
	}

	ctn.core.m.Unlock()

	return deleteContainerCore(ctn.core)
}

// Clean deletes the sub-container created by UnscopedSafeGet, UnscopedGet or UnscopedFill.
func (ctn Container) Clean() error {
	ctn.core.m.Lock()
	unscopedChild := ctn.core.unscopedChild
	ctn.core.unscopedChild = nil
	ctn.core.m.Unlock()

	if unscopedChild != nil {
		return deleteContainerCore(unscopedChild)
	}

	return nil
}

// IsClosed returns true if the Container has been deleted.
func (ctn Container) IsClosed() bool {
	ctn.core.m.RLock()
	closed := ctn.core.closed
	ctn.core.m.RUnlock()
	return closed
}

func deleteContainerCore(core *containerCore) error {
	core.m.Lock()
	clone := &containerCore{
		parent:        core.parent,
		children:      core.children,
		unscopedChild: core.unscopedChild,
		indexesByName: core.indexesByName,
		indexesByType: core.indexesByType,
		definitions:   core.definitions,
		objects:       core.objects,
		unshared:      core.unshared,
		unsharedIndex: core.unsharedIndex,
		dependencies:  core.dependencies,
	}
	core.closed = true
	core.m.Unlock()

	// Stop returning the already built objects.
	for i := 0; i < len(core.isBuilt); i++ {
		atomic.StoreInt32(&core.isBuilt[i], 0)
	}

	// Delete clone.
	errBuilder := &multiErrBuilder{}

	for child := range clone.children {
		errBuilder.Add(deleteContainerCore(child))
	}

	if clone.unscopedChild != nil {
		errBuilder.Add(deleteContainerCore(clone.unscopedChild))
	}

	if clone.parent != nil {
		// Remove from parent.
		clone.parent.m.Lock()

		delete(clone.parent.children, core)

		if !clone.parent.deleteIfNoChild || len(clone.parent.children) > 0 {
			clone.parent.m.Unlock()
		} else {
			clone.parent.m.Unlock()
			errBuilder.Add(deleteContainerCore(clone.parent))
		}
	}

	// Close objects in the right order.
	indexes, err := clone.dependencies.TopologicalOrdering()
	errBuilder.Add(err)

	for _, index := range indexes {
		if index >= 0 {
			errBuilder.Add(closeObject(
				clone.objects[index],
				clone.definitions[index].Close,
				clone.definitions[index].Name,
			))
		} else {
			errBuilder.Add(closeObject(
				clone.unshared[-index-1],
				clone.definitions[clone.unsharedIndex[-index-1]].Close,
				clone.definitions[clone.unsharedIndex[-index-1]].Name,
			))
		}

	}

	return errBuilder.Build()
}

func closeObject(obj interface{}, closeFunc func(interface{}) error, defName string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("could not close `%s`, Close function panicked: %+v", defName, r)
		}
	}()

	if closeFunc != nil {
		err = closeFunc(obj)
	}

	if err != nil {
		return fmt.Errorf("could not close `%s`: %+v", defName, err)
	}

	return err
}
