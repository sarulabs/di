package di

import (
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockObject struct {
	sync.Mutex
	Closed bool
}

type nestedMockObject struct {
	Object *mockObject
}

func TestContextScope(t *testing.T) {
	b, _ := NewBuilder()
	app, _ := b.Build()
	request, _ := app.SubContext()
	subrequest, _ := request.SubContext()

	assert.Equal(t, App, app.Scope())
	assert.Equal(t, Request, request.Scope())
	assert.Equal(t, SubRequest, subrequest.Scope())
}

func TestContextParentScopes(t *testing.T) {
	b, _ := NewBuilder()
	app, _ := b.Build()
	request, _ := app.SubContext()
	subrequest, _ := request.SubContext()

	assert.Empty(t, app.ParentScopes())
	assert.Equal(t, []string{App}, request.ParentScopes())
	assert.Equal(t, []string{App, Request}, subrequest.ParentScopes())
}

func TestContextSubScopes(t *testing.T) {
	b, _ := NewBuilder()
	app, _ := b.Build()
	request, _ := app.SubContext()
	subrequest, _ := request.SubContext()

	assert.Equal(t, []string{Request, SubRequest}, app.SubScopes())
	assert.Equal(t, []string{SubRequest}, request.SubScopes())
	assert.Empty(t, subrequest.SubScopes())
}

func TestContextHasSubScope(t *testing.T) {
	b, _ := NewBuilder()
	app, _ := b.Build()
	request, _ := app.SubContext()
	subrequest, _ := request.SubContext()

	assert.False(t, app.HasSubScope(App))
	assert.True(t, app.HasSubScope(Request))
	assert.True(t, app.HasSubScope(SubRequest))
	assert.False(t, app.HasSubScope("other"))

	assert.False(t, subrequest.HasSubScope(App))
	assert.False(t, subrequest.HasSubScope(Request))
	assert.False(t, subrequest.HasSubScope(SubRequest))
	assert.False(t, subrequest.HasSubScope("other"))
}

func TestContextParentWithScope(t *testing.T) {
	b, _ := NewBuilder()
	app, _ := b.Build()
	request, _ := app.SubContext()
	subrequest, _ := request.SubContext()

	assert.True(t, app == request.ParentWithScope(App))
	assert.True(t, app == subrequest.ParentWithScope(App))
	assert.True(t, request == subrequest.ParentWithScope(Request))

	assert.Nil(t, app.ParentWithScope("undefined"))
	assert.Nil(t, app.ParentWithScope(Request))
}

func TestSubContextCreation(t *testing.T) {
	var err error
	b, _ := NewBuilder()

	app, err := b.Build()
	assert.Nil(t, err)

	request, err := app.SubContext()
	assert.Nil(t, err)

	subrequest, err := request.SubContext()
	assert.Nil(t, err)

	_, err = subrequest.SubContext()
	assert.NotNil(t, err, "subrequest does not have any subcontext")
}

func TestSafeGet(t *testing.T) {
	b, _ := NewBuilder()

	b.AddDefinition(Definition{
		Name:  "object",
		Scope: Request,
		Build: func(ctx *Context) (interface{}, error) {
			return &mockObject{}, nil
		},
	})

	b.AddDefinition(Definition{
		Name:  "unmakable",
		Scope: Request,
		Build: func(ctx *Context) (interface{}, error) {
			return nil, errors.New("error")
		},
	})

	app, _ := b.Build()
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

	// should be able to create the item from the request scope
	obj, err = request.SafeGet("object")
	assert.Nil(t, err)
	assert.Equal(t, &mockObject{}, obj.(*mockObject))

	// should retrieve the same item every time
	objBis, err = request.SafeGet("object")
	assert.Nil(t, err)
	assert.Equal(t, &mockObject{}, objBis.(*mockObject))
	assert.True(t, obj == objBis)

	// should be able to create an item from a subcontext
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
		Build: func(ctx *Context) (interface{}, error) {
			panic("panic in Build function")
		},
	})

	app, _ := b.Build()

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
		Build: func(ctx *Context) (interface{}, error) {
			return &nestedMockObject{
				ctx.Get("appObject").(*mockObject),
			}, nil
		},
	})

	app, _ := b.Build()
	request, _ := app.SubContext()

	nestedObject := request.Get("nestedObject").(*nestedMockObject)
	assert.True(t, appObject == nestedObject.Object)
}

func TestGet(t *testing.T) {
	b, _ := NewBuilder()

	b.AddDefinition(Definition{
		Name:  "object",
		Scope: Request,
		Build: func(ctx *Context) (interface{}, error) {
			return 10, nil
		},
	})

	app, _ := b.Build()
	request, _ := app.SubContext()

	object := request.Get("object").(int)
	assert.Equal(t, 10, object)
}

func TestFill(t *testing.T) {
	b, _ := NewBuilder()

	b.AddDefinition(Definition{
		Name:  "object",
		Scope: App,
		Build: func(ctx *Context) (interface{}, error) {
			return 10, nil
		},
	})

	app, _ := b.Build()

	var err error
	var object int
	var wrongType string

	err = app.Fill("object", &wrongType)
	assert.NotNil(t, err, "should have failed to fill an item with the wrong type")

	err = app.Fill("object", &object)
	assert.Nil(t, err)
	assert.Equal(t, 10, object)
}

func TestDelete(t *testing.T) {
	b, _ := NewBuilder()

	b.AddDefinition(Definition{
		Name:  "obj1",
		Scope: App,
		Build: func(ctx *Context) (interface{}, error) {
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
		Build: func(ctx *Context) (interface{}, error) {
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
		Build: func(ctx *Context) (interface{}, error) {
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
		Build: func(ctx *Context) (interface{}, error) {
			return &mockObject{}, nil
		},
	})

	app, _ := b.Build()
	request, _ := app.SubContext()
	subrequest, _ := request.SubContext()

	var err error

	obj1 := app.Get("obj1").(*mockObject)
	obj2 := request.Get("obj2").(*mockObject)
	obj3 := subrequest.Get("obj3").(*mockObject)
	_ = subrequest.Get("obj4").(*mockObject)

	request.Delete()

	assert.False(t, obj1.Closed)
	assert.True(t, obj2.Closed)
	assert.True(t, obj3.Closed)

	assert.Nil(t, request.Parent(), "should have removed request parent")
	assert.Nil(t, subrequest.Parent(), "should have removed subrequest parent")

	_, err = app.SafeGet("obj1")
	assert.Nil(t, err, "should still be able to create object from the app context")

	_, err = request.SafeGet("obj2")
	assert.NotNil(t, err, "should not be able to create object from the closed request context")

	_, err = subrequest.SafeGet("obj3")
	assert.NotNil(t, err, "should not be able to create object from the closed subrequest context")

	_, err = request.SubContext()
	assert.NotNil(t, err, "should not be able to create a subcontext from a closed context")

	app.Delete()

	assert.True(t, obj1.Closed)
}

func TestClosePanic(t *testing.T) {
	b, _ := NewBuilder()

	b.AddDefinition(Definition{
		Name:  "object",
		Scope: App,
		Build: func(ctx *Context) (interface{}, error) {
			return &mockObject{}, nil
		},
		Close: func(obj interface{}) {
			panic("panic in Close function")
		},
	})

	app, _ := b.Build()

	defer func() {
		assert.Nil(t, recover(), "Close should not panic")
	}()

	_, err := app.SafeGet("object")
	assert.Nil(t, err)

	app.Delete()
}

func TestRace(t *testing.T) {
	b, _ := NewBuilder()

	b.Set("instance", &mockObject{})

	b.AddDefinition(Definition{
		Name:  "object",
		Scope: App,
		Build: func(ctx *Context) (interface{}, error) {
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
		Name:  "nested",
		Scope: Request,
		Build: func(ctx *Context) (interface{}, error) {
			return &nestedMockObject{
				ctx.Get("object").(*mockObject),
			}, nil
		},
		Close: func(obj interface{}) {
			o := obj.(*nestedMockObject)
			o.Object.Lock()
			o.Object.Closed = true
			o.Object.Unlock()
		},
	})

	app, _ := b.Build()

	cApp := make(chan struct{}, 100)

	for i := 0; i < 100; i++ {
		go func() {
			request, _ := app.SubContext()
			defer request.Delete()

			request.Get("instance")
			request.Get("object")
			request.Get("nested")

			cReq := make(chan struct{}, 10)

			for j := 0; j < 10; j++ {
				go func() {
					subrequest, _ := request.SubContext()
					defer subrequest.Delete()

					subrequest.Get("instance")
					subrequest.Get("object")
					subrequest.Get("nested")
					subrequest.Get("instance")
					subrequest.Get("object")
					subrequest.Get("nested")

					cReq <- struct{}{}
				}()
			}

			for j := 0; j < 10; j++ {
				<-cReq
			}

			request.Get("instance")
			request.Get("object")
			request.Get("nested")

			cApp <- struct{}{}
		}()
	}

	for j := 0; j < 100; j++ {
		<-cApp
	}
}
