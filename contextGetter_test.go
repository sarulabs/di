package di

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSafeGet(t *testing.T) {
	b, _ := NewBuilder()

	b.AddDefinition(Definition{
		Name:  "object",
		Scope: Request,
		Build: func(ctx Context) (interface{}, error) {
			return &mockObject{}, nil
		},
	})

	b.AddDefinition(Definition{
		Name:  "unmakable",
		Scope: Request,
		Build: func(ctx Context) (interface{}, error) {
			return nil, errors.New("error")
		},
	})

	app := b.Build()
	request, _ := app.SubContext()
	subrequest, _ := request.SubContext()

	var obj, objBis interface{}
	var err error

	_, err = app.SafeGet("object")
	assert.NotNil(t, err, "should not be able to create the object from the app scope")

	_, err = request.SafeGet("undefined")
	assert.NotNil(t, err, "should not be able to create an undefined object")

	_, err = request.SafeGet("unmakable")
	assert.NotNil(t, err, "should not be able to create an object if there is an error in the Build function")

	// should be able to create the object from the request scope
	obj, err = request.SafeGet("object")
	assert.Nil(t, err)
	assert.Equal(t, &mockObject{}, obj.(*mockObject))

	// should retrieve the same object every time
	objBis, err = request.SafeGet("object")
	assert.Nil(t, err)
	assert.Equal(t, &mockObject{}, objBis.(*mockObject))
	assert.True(t, obj == objBis)

	// should be able to create an object from a subcontext
	obj, err = subrequest.SafeGet("object")
	assert.Nil(t, err)
	assert.Equal(t, &mockObject{}, obj.(*mockObject))
	assert.True(t, obj == objBis)
}

func TestBuildPanic(t *testing.T) {
	b, _ := NewBuilder()

	b.AddDefinition(Definition{
		Name:  "object",
		Scope: App,
		Build: func(ctx Context) (interface{}, error) {
			panic("panic in Build function")
		},
	})

	app := b.Build()

	defer func() {
		assert.Nil(t, recover(), "SafeGet should not panic")
	}()

	_, err := app.SafeGet("object")
	assert.NotNil(t, err, "should not panic but not be able to create the object either")
}

func TestNestedDependencies(t *testing.T) {
	b, _ := NewBuilder()

	appObject := &mockObject{}

	b.Set("appObject", appObject)

	b.AddDefinition(Definition{
		Name:  "nestedObject",
		Scope: Request,
		Build: func(ctx Context) (interface{}, error) {
			return &nestedMockObject{
				Object: ctx.Get("appObject").(*mockObject),
			}, nil
		},
	})

	app := b.Build()
	request, _ := app.SubContext()

	nestedObject := request.Get("nestedObject").(*nestedMockObject)
	assert.True(t, appObject == nestedObject.Object)
}

func TestGet(t *testing.T) {
	b, _ := NewBuilder()

	b.AddDefinition(Definition{
		Name:  "object",
		Scope: Request,
		Build: func(ctx Context) (interface{}, error) {
			return 10, nil
		},
	})

	app := b.Build()
	request, _ := app.SubContext()

	object := request.Get("object").(int)
	assert.Equal(t, 10, object)
}

func TestFill(t *testing.T) {
	b, _ := NewBuilder()

	b.AddDefinition(Definition{
		Name:  "object",
		Scope: App,
		Build: func(ctx Context) (interface{}, error) {
			return 10, nil
		},
	})

	app := b.Build()

	var err error
	var object int
	var wrongType string

	err = app.Fill("object", &wrongType)
	assert.NotNil(t, err, "should have failed to fill an object with the wrong type")

	err = app.Fill("object", &object)
	assert.Nil(t, err)
	assert.Equal(t, 10, object)
}
