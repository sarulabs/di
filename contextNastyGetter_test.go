package di

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNastySafeGet(t *testing.T) {
	b, _ := NewBuilder()

	b.AddDefinition(Definition{
		Name:  "request-object",
		Scope: Request,
		Build: func(ctx Context) (interface{}, error) {
			return &mockObject{Closed: true}, nil
		},
	})

	b.AddDefinition(Definition{
		Name:  "subrequest-object",
		Scope: SubRequest,
		Build: func(ctx Context) (interface{}, error) {
			return &nestedMockObject{
				Object: ctx.Get("request-object").(*mockObject),
			}, nil
		},
	})

	app := b.Build()

	var obj, objBis interface{}
	var err error

	_, err = app.SafeGet("subrequest-object")
	assert.NotNil(t, err)

	obj, err = app.NastySafeGet("subrequest-object")
	assert.Nil(t, err)
	assert.True(t, obj.(*nestedMockObject).Object.Closed)

	objBis, err = app.NastySafeGet("request-object")
	assert.Nil(t, err)
	assert.True(t, objBis.(*mockObject).Closed)

	obj.(*nestedMockObject).Object.Closed = false
	assert.False(t, objBis.(*mockObject).Closed)
}

func TestNastyGet(t *testing.T) {
	b, _ := NewBuilder()

	b.AddDefinition(Definition{
		Name:  "object",
		Scope: Request,
		Build: func(ctx Context) (interface{}, error) {
			return 10, nil
		},
	})

	app := b.Build()

	object := app.NastyGet("object").(int)
	assert.Equal(t, 10, object)
}

func TestNastyFill(t *testing.T) {
	b, _ := NewBuilder()

	b.AddDefinition(Definition{
		Name:  "object",
		Scope: Request,
		Build: func(ctx Context) (interface{}, error) {
			return 10, nil
		},
	})

	app := b.Build()

	var object int

	err := app.NastyFill("object", &object)
	assert.Nil(t, err)
	assert.Equal(t, 10, object)
}
