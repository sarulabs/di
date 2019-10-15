package di

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSafeGet(t *testing.T) {
	b, _ := NewBuilder()

	b.Add([]Def{
		{
			Name:  "object",
			Scope: Request,
			Build: func(ctn Container) (interface{}, error) {
				return &mockObject{}, nil
			},
		},
		{
			Name:  "unmakable",
			Scope: Request,
			Build: func(ctn Container) (interface{}, error) {
				return nil, errors.New("error")
			},
		},
	}...)

	app := b.Build()
	request, _ := app.SubContainer()
	subrequest, _ := request.SubContainer()

	var obj, objBis interface{}
	var err error

	_, err = app.SafeGet("object")
	require.NotNil(t, err, "should not be able to create the object from the app scope")

	_, err = request.SafeGet("undefined")
	require.NotNil(t, err, "should not be able to create an undefined object")

	_, err = request.SafeGet("unmakable")
	require.NotNil(t, err, "should not be able to create an object if there is an error in the Build function")

	// should be able to create the object from the request scope
	obj, err = request.SafeGet("object")
	require.Nil(t, err)
	require.Equal(t, &mockObject{}, obj.(*mockObject))

	// should retrieve the same object every time
	objBis, err = request.SafeGet("object")
	require.Nil(t, err)
	require.Equal(t, &mockObject{}, objBis.(*mockObject))
	require.True(t, obj == objBis)

	// should be able to create an object from a sub-container
	obj, err = subrequest.SafeGet("object")
	require.Nil(t, err)
	require.Equal(t, &mockObject{}, obj.(*mockObject))
	require.True(t, obj == objBis)
}

func TestUnsharedObjects(t *testing.T) {
	b, _ := NewBuilder()

	b.Add(Def{
		Name: "unshared",
		Build: func(ctn Container) (interface{}, error) {
			return &mockObject{}, nil
		},
		Unshared: true,
	})

	var app = b.Build()

	obj1, err := app.SafeGet("unshared")
	require.Nil(t, err)

	obj2, err := app.SafeGet("unshared")
	require.Nil(t, err)

	// should retrieve different object every time
	require.False(t, obj1 == obj2)
}

func TestBuildPanic(t *testing.T) {
	b, _ := NewBuilder()

	b.Add(Def{
		Name:  "object",
		Scope: App,
		Build: func(ctn Container) (interface{}, error) {
			panic("panic in Build function")
		},
	})

	app := b.Build()

	defer func() {
		require.Nil(t, recover(), "SafeGet should not panic")
	}()

	_, err := app.SafeGet("object")
	require.NotNil(t, err, "should not panic but not be able to create the object either")
}

func TestDependencies(t *testing.T) {
	b, _ := NewBuilder()

	appObject := &mockObject{}

	b.Add([]Def{
		{
			Name:  "appObject",
			Scope: App,
			Build: func(ctn Container) (interface{}, error) {
				return appObject, nil
			},
		},
		{
			Name:  "objWithDependency",
			Scope: Request,
			Build: func(ctn Container) (interface{}, error) {
				return &mockObjectWithDependency{
					Object: ctn.Get("appObject").(*mockObject),
				}, nil
			},
		},
	}...)

	app := b.Build()
	request, _ := app.SubContainer()

	objWithDependency := request.Get("objWithDependency").(*mockObjectWithDependency)
	require.True(t, appObject == objWithDependency.Object)
}

func TestDependenciesError(t *testing.T) {
	b, _ := NewBuilder()

	b.Add([]Def{
		{
			Name:  "reqObject",
			Scope: Request,
			Build: func(ctn Container) (interface{}, error) {
				return &mockObject{}, nil
			},
		},
		{
			Name:  "objWithDependency",
			Scope: App,
			Build: func(ctn Container) (interface{}, error) {
				return &mockObjectWithDependency{
					Object: ctn.Get("reqObject").(*mockObject),
				}, nil
			},
		},
	}...)

	app := b.Build()
	request, _ := app.SubContainer()

	_, err := request.SafeGet("objWithDependency")
	require.NotNil(t, err, "an App object should not depends on a Request object")
}

func TestGet(t *testing.T) {
	b, _ := NewBuilder()

	b.Add(Def{
		Name:  "object",
		Scope: Request,
		Build: func(ctn Container) (interface{}, error) {
			return 10, nil
		},
	})

	app := b.Build()
	request, _ := app.SubContainer()

	object := request.Get("object").(int)
	require.Equal(t, 10, object)
}

func TestGetPanic(t *testing.T) {
	b, _ := NewBuilder()

	b.Add(Def{
		Name: "object",
		Build: func(ctn Container) (interface{}, error) {
			return 10, errors.New("build error")
		},
	})

	app := b.Build()

	require.Panics(t, func() {
		app.Get("object")
	})
}

func TestFill(t *testing.T) {
	b, _ := NewBuilder()

	b.Add(Def{
		Name:  "object",
		Scope: App,
		Build: func(ctn Container) (interface{}, error) {
			return 10, nil
		},
	})

	app := b.Build()

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

func TestDeleteDuringBuild(t *testing.T) {
	built := false
	closed := false

	b, _ := NewBuilder()

	b.Add(Def{
		Name: "object",
		Build: func(ctn Container) (interface{}, error) {
			ctn.Delete()
			built = true
			return 10, nil
		},
		Close: func(obj interface{}) error {
			closed = true
			return nil
		},
	})

	app := b.Build()

	_, err := app.SafeGet("object")
	require.NotNil(t, err)
	require.True(t, app.IsClosed())
	require.True(t, built)
	require.True(t, closed)
}

func TestDeleteDuringBuildWithCloseError(t *testing.T) {
	built := false
	closed := false

	b, _ := NewBuilder()

	b.Add(Def{
		Name: "object",
		Build: func(ctn Container) (interface{}, error) {
			ctn.Delete()
			built = true
			return 10, nil
		},
		Close: func(obj interface{}) error {
			closed = true
			return errors.New("could not close object")
		},
	})

	app := b.Build()

	_, err := app.SafeGet("object")
	require.NotNil(t, err)
	require.True(t, app.IsClosed())
	require.True(t, built)
	require.True(t, closed)
}

func TestConcurrentBuild(t *testing.T) {
	var numBuild uint64
	var numClose uint64

	b, _ := NewBuilder()

	b.Add(Def{
		Name: "object",
		Build: func(ctn Container) (interface{}, error) {
			time.Sleep(250 * time.Millisecond)
			atomic.AddUint64(&numBuild, 1)
			return nil, nil
		},
		Close: func(obj interface{}) error {
			atomic.AddUint64(&numClose, 1)
			return nil
		},
	})

	app := b.Build()

	var wg sync.WaitGroup

	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			req, _ := app.SubContainer()
			req.Get("object")
			req.Delete()
			wg.Done()
		}()
	}

	wg.Wait()

	require.Equal(t, uint64(1), atomic.LoadUint64(&numBuild))
	require.Equal(t, uint64(0), atomic.LoadUint64(&numClose))

	app.Delete()

	require.Equal(t, uint64(1), atomic.LoadUint64(&numClose))
}
