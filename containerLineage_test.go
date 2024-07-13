package di

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParentContainer(t *testing.T) {
	var err error
	b, _ := NewEnhancedBuilder()

	app, _ := b.Build()

	_, err = app.ParentContainer()
	require.NotNil(t, err)

	req, _ := app.SubContainer()
	parent, err := req.ParentContainer()
	require.Nil(t, err)
	require.Equal(t, parent, app)
}

func TestSubContainerCreation(t *testing.T) {
	var err error
	b, _ := NewEnhancedBuilder()

	app, _ := b.Build()

	request, err := app.SubContainer()
	require.Nil(t, err)

	subrequest, err := request.SubContainer()
	require.Nil(t, err)

	_, err = subrequest.SubContainer()
	require.NotNil(t, err, "sub-request does not have any sub-container")

	require.Equal(t, request, subrequest.Parent())
	require.Equal(t, app, request.Parent())
}

func TestLineageGetterCycleError(t *testing.T) {
	var err error

	b, _ := NewEnhancedBuilder()

	b.Add(&Def{
		Name:  "o-app",
		Scope: App,
		Build: func(ctn Container) (interface{}, error) {
			ctn.Get("o-req")
			return nil, nil
		},
	})
	b.Add(&Def{
		Name: "o-req",
		Build: func(ctn Container) (interface{}, error) {
			ctn.Get("o-app")
			return nil, nil
		},
	})

	app, _ := b.Build()
	req, _ := app.SubContainer()

	_, err = req.SafeGet("o-req")
	require.NotNil(t, err)
}
