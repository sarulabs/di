package di

import (
	"fmt"
)

// containerSlayer contains all the functions that are useful
// to delete a container.
type containerSlayer struct{}

func (s *containerSlayer) Delete(ctn *containerCore) error {
	ctn.m.Lock()

	if len(ctn.children) > 0 {
		ctn.deleteIfNoChild = true
		ctn.m.Unlock()
		return nil
	}

	ctn.m.Unlock()

	return s.DeleteWithSubContainers(ctn)
}

func (s *containerSlayer) DeleteWithSubContainers(ctn *containerCore) error {
	ctn.m.Lock()
	clone := &containerCore{
		definitions:   ctn.definitions,
		children:      ctn.children,
		unscopedChild: ctn.unscopedChild,
		parent:        ctn.parent,
		objects:       ctn.objects,
		dependencies:  ctn.dependencies,
	}
	ctn.closed = true
	ctn.m.Unlock()

	return s.deleteClone(ctn, clone)
}

func (s *containerSlayer) deleteClone(ctn *containerCore, clone *containerCore) error {
	errBuilder := &multiErrBuilder{}

	for child := range clone.children {
		err := s.DeleteWithSubContainers(child)
		errBuilder.Add(err)
	}

	if clone.unscopedChild != nil {
		err := s.DeleteWithSubContainers(clone.unscopedChild)
		errBuilder.Add(err)
	}

	if clone.parent != nil {
		err := s.removeChild(clone.parent, ctn)
		errBuilder.Add(err)
	}

	keys, err := clone.dependencies.TopologicalOrdering()
	errBuilder.Add(err)

	for _, key := range keys {
		obj, hasObj := clone.objects[key]
		def, hasDef := clone.definitions[key.defName]
		if hasObj && hasDef {
			err := s.closeObject(obj, def)
			errBuilder.Add(err)
		}
	}

	return errBuilder.Build()
}

func (s *containerSlayer) removeChild(ctn *containerCore, child *containerCore) error {
	ctn.m.Lock()

	delete(ctn.children, child)

	if !ctn.deleteIfNoChild || len(ctn.children) > 0 {
		ctn.m.Unlock()
		return nil
	}

	ctn.m.Unlock()

	return s.DeleteWithSubContainers(ctn)
}

func (s *containerSlayer) closeObject(obj interface{}, def Def) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("could not close `%s`, Close function panicked: %+v", def.Name, r)
		}
	}()

	if _, isBuilding := obj.(buildingChan); isBuilding {
		return nil
	}

	if def.Close != nil {
		err = def.Close(obj)
	}

	if err != nil {
		return fmt.Errorf("could not close `%s`: %+v", def.Name, err)
	}

	return err
}

func (s *containerSlayer) IsClosed(ctn *containerCore) bool {
	ctn.m.RLock()
	closed := ctn.closed
	ctn.m.RUnlock()
	return closed
}

func (s *containerSlayer) Clean(ctn *containerCore) error {
	ctn.m.Lock()
	unscopedChild := ctn.unscopedChild
	ctn.unscopedChild = nil
	ctn.m.Unlock()

	if unscopedChild != nil {
		return s.DeleteWithSubContainers(unscopedChild)
	}

	return nil
}
