package di

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSubContainerCreation(t *testing.T) {
	var err error
	b, _ := NewBuilder()

	app := b.Build()

	request, err := app.SubContainer()
	require.Nil(t, err)

	subrequest, err := request.SubContainer()
	require.Nil(t, err)

	_, err = subrequest.SubContainer()
	require.NotNil(t, err, "sub-request does not have any sub-container")

	require.Equal(t, request, subrequest.Parent())
	require.Equal(t, app, request.Parent())
}
