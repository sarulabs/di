package di

import (
	"errors"
	"fmt"
)

// containerLineage contains all the functions that are useful
// to retrieve or create the parent and children of a container.
type containerLineage struct{}

func (l *containerLineage) Parent(ctn *container) Container {
	return l.parent(ctn)
}

func (l *containerLineage) parent(ctn *container) *container {
	ctn.m.RLock()
	parent := ctn.containerCore.parent
	ctn.m.RUnlock()

	return &container{
		containerCore: parent,
	}
}

func (l *containerLineage) SubContainer(ctn *container) (Container, error) {
	child, err := l.createChild(ctn)
	if err != nil {
		return nil, err
	}

	ctn.m.Lock()

	if ctn.closed {
		ctn.m.Unlock()
		return nil, errors.New("the container is closed")
	}

	ctn.children[child.containerCore] = struct{}{}

	ctn.m.Unlock()

	return child, nil
}

func (l *containerLineage) createChild(ctn *container) (*container, error) {
	subscopes := ctn.SubScopes()

	if len(subscopes) == 0 {
		return nil, fmt.Errorf("there is no more specific scope than `%s`", ctn.scope)
	}

	return &container{
		containerCore: &containerCore{
			scope:         subscopes[0],
			scopes:        ctn.scopes,
			definitions:   ctn.definitions,
			parent:        ctn.containerCore,
			children:      map[*containerCore]struct{}{},
			unscopedChild: nil,
			objects:       map[string]interface{}{},
		},
	}, nil
}
