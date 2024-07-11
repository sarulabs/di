package di

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFill(t *testing.T) {
	b, _ := NewEnhancedBuilder()

	b.Add(&Def{
		Name:  "object",
		Scope: App,
		Build: func(ctn Container) (interface{}, error) {
			return 10, nil
		},
	})

	app, _ := b.Build()

	var err error
	var object int
	var wrongType string

	err = app.Fill("unknown", &wrongType)
	require.NotNil(t, err)

	err = app.Fill("object", &wrongType)
	require.NotNil(t, err, "should have failed to fill an object with the wrong type")

	err = app.Fill("object", &object)
	require.Nil(t, err)
	require.Equal(t, 10, object)
}
