package di

import (
	"errors"
	"fmt"
)

// containerLineage contains all the functions that are useful
// to retrieve or create the parent and children of a container.
type containerLineage struct{}

func (l *containerLineage) getParent(ctn *containerCore) *containerCore {
	ctn.m.Lock()
	defer ctn.m.Unlock()
	return ctn.parent
}

func (l *containerLineage) Parent(ctn *container) Container {
	return &container{
		containerCore: l.getParent(ctn.containerCore),
		built:         ctn.built,
		logger:        ctn.logger,
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
		return nil, errors.New("the Container is closed")
	}

	ctn.children = append(ctn.children, child.containerCore)

	ctn.m.Unlock()

	return child, nil
}

func (l *containerLineage) createChild(ctn *container) (*container, error) {
	subscopes := ctn.SubScopes()

	if len(subscopes) == 0 {
		return nil, fmt.Errorf("there is no narrower scope than `%s`", ctn.scope)
	}

	return &container{
		containerCore: &containerCore{
			scope:         subscopes[0],
			scopes:        ctn.scopes,
			definitions:   ctn.definitions,
			parent:        ctn.containerCore,
			children:      []*containerCore{},
			unscopedChild: nil,
			objects:       map[string]interface{}{},
		},
		built:  ctn.built,
		logger: ctn.logger,
	}, nil
}
