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

func TestContextDefinition(t *testing.T) {
	b, _ := NewBuilder()

	def1 := Definition{
		Name: "o1",
		Build: func(ctx Context) (interface{}, error) {
			return &mockObject{}, nil
		},
	}

	def2 := Definition{
		Name: "o2",
		Build: func(ctx Context) (interface{}, error) {
			return &mockObject{}, nil
		},
	}

	b.AddDefinition(def1)
	b.AddDefinition(def2)

	app, _ := b.Build()
	defs := app.Definitions()

	assert.Len(t, defs, 2)
	assert.Equal(t, "o1", defs["o1"].Name)
	assert.Equal(t, "o2", defs["o2"].Name)
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
		Build: func(ctx Context) (interface{}, error) {
			return &nestedMockObject{
				Object: ctx.Get("appObject").(*mockObject),
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
		Build: func(ctx Context) (interface{}, error) {
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
		Build: func(ctx Context) (interface{}, error) {
			return 10, nil
		},
	})

	app, _ := b.Build()

	var err error
	var object int
	var wrongType string

	err = app.Fill("object", &wrongType)
	assert.NotNil(t, err, "should have failed to fill an object with the wrong type")

	err = app.Fill("object", &object)
	assert.Nil(t, err)
	assert.Equal(t, 10, object)
}

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

	app, _ := b.Build()

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

	app, _ := b.Build()

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

	app, _ := b.Build()

	var object int

	err := app.NastyFill("object", &object)
	assert.Nil(t, err)
	assert.Equal(t, 10, object)
}

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

	app, _ := b.Build()

	var obj *mockObject

	obj = app.NastyGet("object").(*mockObject)
	assert.False(t, obj.Closed)

	app.Clean()
	assert.True(t, obj.Closed)

	obj = app.NastyGet("object").(*mockObject)
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

	app, _ := b.Build()
	request, _ := app.SubContext()
	subrequest, _ := request.SubContext()

	assert.False(t, request.IsClosed())
	assert.False(t, subrequest.IsClosed())

	var err error

	obj1 := app.Get("obj1").(*mockObject)
	obj2 := request.Get("obj2").(*mockObject)
	obj3 := subrequest.Get("obj3").(*mockObject)
	_ = subrequest.Get("obj4").(*mockObject)

	request.Delete()

	assert.False(t, obj1.Closed)
	assert.True(t, obj2.Closed)
	assert.True(t, obj3.Closed)

	assert.True(t, request.IsClosed())
	assert.True(t, subrequest.IsClosed())

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
		Build: func(ctx Context) (interface{}, error) {
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

func TestCycleError(t *testing.T) {
	b, _ := NewBuilder()

	b.AddDefinition(Definition{
		Name: "o1",
		Build: func(ctx Context) (interface{}, error) {
			return &nestedMockObject{
				Object: ctx.Get("o2").(*nestedMockObject).Object,
			}, nil
		},
	})

	b.AddDefinition(Definition{
		Name: "o2",
		Build: func(ctx Context) (interface{}, error) {
			return &nestedMockObject{
				Object: ctx.Get("o1").(*nestedMockObject).Object,
			}, nil
		},
	})

	app, _ := b.Build()
	_, err := app.SafeGet("o1")
	assert.NotNil(t, err)
}

func TestRace(t *testing.T) {
	b, _ := NewBuilder()

	b.Set("instance", &mockObject{})

	b.AddDefinition(Definition{
		Name:  "object",
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
		Name:  "nested",
		Scope: Request,
		Build: func(ctx Context) (interface{}, error) {
			return &nestedMockObject{
				Object: ctx.Get("object").(*mockObject),
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
