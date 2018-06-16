package di

import (
	"fmt"
	"runtime/debug"
)

// containerSlayer contains all the functions that are useful
// to delete a container.
type containerSlayer struct{}

func (s *containerSlayer) Delete(logger Logger, ctn *containerCore) {
	ctn.m.Lock()

	if len(ctn.children) > 0 {
		ctn.deleteIfNoChild = true
		ctn.m.Unlock()
		return
	}

	ctn.m.Unlock()

	s.DeleteWithSubContainers(logger, ctn)
}

func (s *containerSlayer) DeleteWithSubContainers(logger Logger, ctn *containerCore) {
	ctn.m.Lock()

	c := &containerCore{
		children:      make([]*containerCore, len(ctn.children)),
		unscopedChild: ctn.unscopedChild,
		parent:        ctn.parent,
		objects:       map[string]interface{}{},
	}

	copy(c.children, ctn.children)

	for name, obj := range ctn.objects {
		c.objects[name] = obj
	}

	ctn.children = nil
	ctn.unscopedChild = nil
	ctn.parent = nil
	ctn.objects = nil
	ctn.closed = true

	ctn.m.Unlock()

	s.deleteClone(logger, ctn, c)
}

func (s *containerSlayer) deleteClone(logger Logger, ctn *containerCore, c *containerCore) {
	for _, child := range c.children {
		s.DeleteWithSubContainers(logger, child)
	}

	if c.unscopedChild != nil {
		s.DeleteWithSubContainers(logger, c.unscopedChild)
	}

	if c.parent != nil {
		s.removeChild(logger, c.parent, ctn)
	}

	for name, obj := range c.objects {
		s.closeObject(logger, ctn, obj, ctn.definitions[name])
	}
}

func (s *containerSlayer) removeChild(logger Logger, ctn *containerCore, child *containerCore) {
	ctn.m.Lock()

	for i, c := range ctn.children {
		if c == child {
			ctn.children = append(ctn.children[:i], ctn.children[i+1:]...)
			break
		}
	}

	if !ctn.deleteIfNoChild || len(ctn.children) > 0 {
		ctn.m.Unlock()
		return
	}

	ctn.m.Unlock()

	s.DeleteWithSubContainers(logger, ctn)
}

func (s *containerSlayer) closeObject(logger Logger, ctn *containerCore, obj interface{}, def Definition) {
	defer func() {
		if r := recover(); r != nil {
			msg := fmt.Sprintf("could not close `%s` err=%s stack=%s", def.Name, r, debug.Stack())
			logger.Error(msg)
		}
	}()

	if def.Close != nil {
		def.Close(obj)
	}

	return
}

func (s *containerSlayer) IsClosed(ctn *containerCore) bool {
	ctn.m.Lock()
	defer ctn.m.Unlock()
	return ctn.closed
}

func (s *containerSlayer) Clean(logger Logger, ctn *containerCore) {
	ctn.m.Lock()
	child := ctn.unscopedChild
	ctn.unscopedChild = nil
	ctn.m.Unlock()

	if child != nil {
		s.Delete(logger, child)
	}
}
