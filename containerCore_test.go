package di

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestContainerDefinitions(t *testing.T) {
	b, _ := NewBuilder()

	b.Add([]Def{
		{
			Name: "o1",
			Build: func(ctn Container) (interface{}, error) {
				return &mockObject{}, nil
			},
		},
		{
			Name: "o2",
			Build: func(ctn Container) (interface{}, error) {
				return &mockObject{}, nil
			},
		},
	}...)

	app := b.Build()
	defs := app.Definitions()

	require.Len(t, defs, 2)
	require.Equal(t, "o1", defs["o1"].Name)
	require.Equal(t, "o2", defs["o2"].Name)
}

func TestContainerScope(t *testing.T) {
	b, _ := NewBuilder()
	app := b.Build()
	request, _ := app.SubContainer()
	subrequest, _ := request.SubContainer()

	require.Equal(t, App, app.Scope())
	require.Equal(t, Request, request.Scope())
	require.Equal(t, SubRequest, subrequest.Scope())
}

func TestContainerScopes(t *testing.T) {
	b, _ := NewBuilder()
	app := b.Build()
	request, _ := app.SubContainer()
	subrequest, _ := request.SubContainer()

	list := []string{App, Request, SubRequest}

	require.Equal(t, list, app.Scopes())
	require.Equal(t, list, request.Scopes())
	require.Equal(t, list, subrequest.Scopes())
}

func TestContainerParentScopes(t *testing.T) {
	b, _ := NewBuilder()
	app := b.Build()
	request, _ := app.SubContainer()
	subrequest, _ := request.SubContainer()

	require.Empty(t, app.ParentScopes())
	require.Equal(t, []string{App}, request.ParentScopes())
	require.Equal(t, []string{App, Request}, subrequest.ParentScopes())
}

func TestContainerSubScopes(t *testing.T) {
	b, _ := NewBuilder()
	app := b.Build()
	request, _ := app.SubContainer()
	subrequest, _ := request.SubContainer()

	require.Equal(t, []string{Request, SubRequest}, app.SubScopes())
	require.Equal(t, []string{SubRequest}, request.SubScopes())
	require.Empty(t, subrequest.SubScopes())
}
