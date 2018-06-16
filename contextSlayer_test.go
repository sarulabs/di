package di

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClean(t *testing.T) {
	b, _ := NewBuilder()

	b.AddDefinition(Definition{
		Name:  "object",
		Scope: SubRequest,
		Build: func(ctx Context) (interface{}, error) {
			return &mockObject{}, nil
		},
		Close: func(obj interface{}) {
			obj.(*mockObject).Closed = true
		},
	})

	app := b.Build()

	var obj *mockObject

	obj = app.UnscopedGet("object").(*mockObject)
	assert.False(t, obj.Closed)

	app.Clean()
	assert.True(t, obj.Closed)

	obj = app.UnscopedGet("object").(*mockObject)
	assert.False(t, obj.Closed)
}

func TestDelete(t *testing.T) {
	b, _ := NewBuilder()

	b.AddDefinition(Definition{
		Name:  "obj1",
		Scope: App,
		Build: func(ctx Context) (interface{}, error) {
			return &mockObject{}, nil
		},
	})

	b.AddDefinition(Definition{
		Name:  "obj2",
		Scope: Request,
		Build: func(ctx Context) (interface{}, error) {
			return &mockObject{}, nil
		},
	})

	app := b.Build()
	request, _ := app.SubContext()

	assert.False(t, app.IsClosed())
	assert.False(t, request.IsClosed())

	app.Delete()

	assert.False(t, app.IsClosed())
	assert.False(t, request.IsClosed())

	request.Delete()

	assert.True(t, app.IsClosed())
	assert.True(t, request.IsClosed())
}

func TestDeleteWithSubContexts(t *testing.T) {
	b, _ := NewBuilder()

	b.AddDefinition(Definition{
		Name:  "obj1",
		Scope: App,
		Build: func(ctx Context) (interface{}, error) {
			return &mockObject{}, nil
		},
		Close: func(obj interface{}) {
			i := obj.(*mockObject)
			i.Lock()
			i.Closed = true
			i.Unlock()
		},
	})

	b.AddDefinition(Definition{
		Name:  "obj2",
		Scope: Request,
		Build: func(ctx Context) (interface{}, error) {
			return &mockObject{}, nil
		},
		Close: func(obj interface{}) {
			i := obj.(*mockObject)
			i.Lock()
			i.Closed = true
			i.Unlock()
		},
	})

	b.AddDefinition(Definition{
		Name:  "obj3",
		Scope: SubRequest,
		Build: func(ctx Context) (interface{}, error) {
			return &mockObject{}, nil
		},
		Close: func(obj interface{}) {
			i := obj.(*mockObject)
			i.Lock()
			i.Closed = true
			i.Unlock()
		},
	})

	b.AddDefinition(Definition{
		Name:  "obj4",
		Scope: SubRequest,
		Build: func(ctx Context) (interface{}, error) {
			return &mockObject{}, nil
		},
	})

	app := b.Build()
	request, _ := app.SubContext()
	subrequest, _ := request.SubContext()

	assert.False(t, request.IsClosed())
	assert.False(t, subrequest.IsClosed())

	obj1 := app.Get("obj1").(*mockObject)
	obj2 := request.Get("obj2").(*mockObject)
	obj3 := subrequest.Get("obj3").(*mockObject)
	_ = subrequest.Get("obj4").(*mockObject)

	request.DeleteWithSubContexts()

	assert.False(t, obj1.Closed)
	assert.True(t, obj2.Closed)
	assert.True(t, obj3.Closed)

	assert.False(t, app.IsClosed())
	assert.True(t, request.IsClosed())
	assert.True(t, subrequest.IsClosed())

	var err error

	_, err = app.SafeGet("obj1")
	assert.Nil(t, err, "should still be able to create object from the app context")

	_, err = request.SafeGet("obj2")
	assert.NotNil(t, err, "should not be able to create object from the closed request context")

	_, err = subrequest.SafeGet("obj3")
	assert.NotNil(t, err, "should not be able to create object from the closed subrequest context")

	_, err = request.SubContext()
	assert.NotNil(t, err, "should not be able to create a subcontext from a closed context")

	app.DeleteWithSubContexts()

	assert.True(t, obj1.Closed)

	assert.True(t, app.IsClosed())
	assert.True(t, request.IsClosed())
	assert.True(t, subrequest.IsClosed())
}

func TestClosePanic(t *testing.T) {
	b, _ := NewBuilder()

	b.AddDefinition(Definition{
		Name:  "object",
		Scope: App,
		Build: func(ctx Context) (interface{}, error) {
			return &mockObject{}, nil
		},
		Close: func(obj interface{}) {
			panic("panic in Close function")
		},
	})

	app := b.Build()

	defer func() {
		assert.Nil(t, recover(), "Close should not panic")
	}()

	_, err := app.SafeGet("object")
	assert.Nil(t, err)

	app.Delete()
}
