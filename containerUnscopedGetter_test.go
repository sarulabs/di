package di

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnscopedSafeGet(t *testing.T) {
	b, _ := NewBuilder()

	b.AddDefinition(Definition{
		Name:  "request-object",
		Scope: Request,
		Build: func(ctn Container) (interface{}, error) {
			return &mockObject{Closed: true}, nil
		},
	})

	b.AddDefinition(Definition{
		Name:  "subrequest-object",
		Scope: SubRequest,
		Build: func(ctn Container) (interface{}, error) {
			return &nestedMockObject{
				Object: ctn.Get("request-object").(*mockObject),
			}, nil
		},
	})

	app := b.Build()

	var obj, objBis interface{}
	var err error

	_, err = app.SafeGet("subrequest-object")
	assert.NotNil(t, err)

	obj, err = app.UnscopedSafeGet("subrequest-object")
	assert.Nil(t, err)
	assert.True(t, obj.(*nestedMockObject).Object.Closed)

	objBis, err = app.UnscopedSafeGet("request-object")
	assert.Nil(t, err)
	assert.True(t, objBis.(*mockObject).Closed)

	obj.(*nestedMockObject).Object.Closed = false
	assert.False(t, objBis.(*mockObject).Closed)
}

func TestUnscopedGet(t *testing.T) {
	b, _ := NewBuilder()

	b.AddDefinition(Definition{
		Name:  "object",
		Scope: Request,
		Build: func(ctn Container) (interface{}, error) {
			return 10, nil
		},
	})

	app := b.Build()

	object := app.UnscopedGet("object").(int)
	assert.Equal(t, 10, object)
}

func TestUnscopedFill(t *testing.T) {
	b, _ := NewBuilder()

	b.AddDefinition(Definition{
		Name:  "object",
		Scope: Request,
		Build: func(ctn Container) (interface{}, error) {
			return 10, nil
		},
	})

	app := b.Build()

	var object int

	err := app.UnscopedFill("object", &object)
	assert.Nil(t, err)
	assert.Equal(t, 10, object)
}
