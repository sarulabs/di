package di

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContainerDefinition(t *testing.T) {
	b, _ := NewBuilder()

	def1 := Definition{
		Name: "o1",
		Build: func(ctn Container) (interface{}, error) {
			return &mockObject{}, nil
		},
	}

	def2 := Definition{
		Name: "o2",
		Build: func(ctn Container) (interface{}, error) {
			return &mockObject{}, nil
		},
	}

	b.AddDefinition(def1)
	b.AddDefinition(def2)

	app := b.Build()
	defs := app.Definitions()

	assert.Len(t, defs, 2)
	assert.Equal(t, "o1", defs["o1"].Name)
	assert.Equal(t, "o2", defs["o2"].Name)
}

func TestContainerScope(t *testing.T) {
	b, _ := NewBuilder()
	app := b.Build()
	request, _ := app.SubContainer()
	subrequest, _ := request.SubContainer()

	assert.Equal(t, App, app.Scope())
	assert.Equal(t, Request, request.Scope())
	assert.Equal(t, SubRequest, subrequest.Scope())
}

func TestContainerParentScopes(t *testing.T) {
	b, _ := NewBuilder()
	app := b.Build()
	request, _ := app.SubContainer()
	subrequest, _ := request.SubContainer()

	assert.Empty(t, app.ParentScopes())
	assert.Equal(t, []string{App}, request.ParentScopes())
	assert.Equal(t, []string{App, Request}, subrequest.ParentScopes())
}

func TestContainerSubScopes(t *testing.T) {
	b, _ := NewBuilder()
	app := b.Build()
	request, _ := app.SubContainer()
	subrequest, _ := request.SubContainer()

	assert.Equal(t, []string{Request, SubRequest}, app.SubScopes())
	assert.Equal(t, []string{SubRequest}, request.SubScopes())
	assert.Empty(t, subrequest.SubScopes())
}
