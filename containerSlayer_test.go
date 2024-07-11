package di

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDelete(t *testing.T) {
	b, _ := NewEnhancedBuilder()

	b.Add(&Def{
		Name:  "obj1",
		Scope: App,
		Build: func(ctn Container) (interface{}, error) {
			return &mockD{}, nil
		},
		Close: func(obj interface{}) error {
			obj.(*mockD).Closed = true
			return nil
		},
	})
	b.Add(&Def{
		Name:  "obj2",
		Scope: Request,
		Build: func(ctn Container) (interface{}, error) {
			return &mockD{}, nil
		},
	})

	var err error

	app, _ := b.Build()
	request, _ := app.SubContainer()
	obj1 := request.Get("obj1").(*mockD)
	obj2 := request.Get("obj2").(*mockD)

	require.False(t, app.IsClosed())
	require.False(t, request.IsClosed())
	require.False(t, obj1.Closed)
	require.False(t, obj2.Closed)

	// close the app
	err = app.Delete()
	require.Nil(t, err)

	require.False(t, app.IsClosed(), "app should not be closed, it still has a request")
	require.False(t, request.IsClosed())
	require.False(t, obj1.Closed)
	require.False(t, obj2.Closed)

	// close the request
	err = request.Delete()
	require.Nil(t, err)

	require.True(t, app.IsClosed(), "app should be closed now that the request is closed")
	require.True(t, request.IsClosed())
	require.True(t, obj1.Closed)
	require.False(t, obj2.Closed, "obj2 should not be closed, it does not have a Close function")
}

func TestDeleteWithCloseError(t *testing.T) {
	b, _ := NewEnhancedBuilder()

	b.Add(&Def{
		Name:  "obj1",
		Scope: App,
		Build: func(ctn Container) (interface{}, error) {
			return &mockD{}, nil
		},
		Close: func(obj interface{}) error {
			obj.(*mockD).Closed = true
			return errors.New("close error")
		},
	})
	b.Add(&Def{
		Name:  "obj2",
		Scope: Request,
		Build: func(ctn Container) (interface{}, error) {
			return &mockD{}, nil
		},
	})

	var err error

	app, _ := b.Build()
	request, _ := app.SubContainer()
	obj1 := request.Get("obj1").(*mockD)
	obj2 := request.Get("obj2").(*mockD)

	require.False(t, app.IsClosed())
	require.False(t, request.IsClosed())
	require.False(t, obj1.Closed)
	require.False(t, obj2.Closed)

	// close the app
	err = app.Delete()
	require.Nil(t, err, "no error, app is not closed yet")

	require.False(t, app.IsClosed(), "app should not be closed, it still has a request")
	require.False(t, request.IsClosed())
	require.False(t, obj1.Closed)
	require.False(t, obj2.Closed)

	// close the request
	err = request.Delete()
	require.NotNil(t, err, "error because of obj1 Close function")

	require.True(t, app.IsClosed(), "app should be closed now that the request is closed")
	require.True(t, request.IsClosed())
	require.True(t, obj1.Closed)
	require.False(t, obj2.Closed, "obj2 should not be closed, it does not have a Close function")
}

func TestDeleteWithSubContainers(t *testing.T) {
	b, _ := NewEnhancedBuilder()

	b.Add(&Def{
		Name:  "obj1",
		Scope: App,
		Build: func(ctn Container) (interface{}, error) {
			return &mockD{}, nil
		},
		Close: func(obj interface{}) error {
			obj.(*mockD).Closed = true
			return nil
		},
	})
	b.Add(&Def{
		Name:  "obj2",
		Scope: Request,
		Build: func(ctn Container) (interface{}, error) {
			return &mockD{}, nil
		},
		Close: func(obj interface{}) error {
			obj.(*mockD).Closed = true
			return nil
		},
	})
	b.Add(&Def{
		Name:  "obj3",
		Scope: SubRequest,
		Build: func(ctn Container) (interface{}, error) {
			return &mockD{}, nil
		},
		Close: func(obj interface{}) error {
			obj.(*mockD).Closed = true
			return nil
		},
	})
	b.Add(&Def{
		Name:  "obj4",
		Scope: SubRequest,
		Build: func(ctn Container) (interface{}, error) {
			return &mockD{}, nil
		},
	})

	var err error

	app, _ := b.Build()
	request, _ := app.SubContainer()
	subrequest, _ := request.SubContainer()

	require.False(t, app.IsClosed())
	require.False(t, request.IsClosed())
	require.False(t, subrequest.IsClosed())

	obj1 := app.Get("obj1").(*mockD)
	obj2 := request.Get("obj2").(*mockD)
	obj3 := subrequest.Get("obj3").(*mockD)
	_ = subrequest.Get("obj4").(*mockD)

	// delete request (it forces the subrequest deletion)
	err = request.DeleteWithSubContainers()
	require.Nil(t, err)

	require.False(t, obj1.Closed)
	require.True(t, obj2.Closed)
	require.True(t, obj3.Closed)

	require.False(t, app.IsClosed())
	require.True(t, request.IsClosed())
	require.True(t, subrequest.IsClosed())

	_, err = app.SafeGet("obj1")
	require.Nil(t, err, "should still be able to create object from the app context")

	_, err = request.SafeGet("obj2")
	require.NotNil(t, err, "should not be able to create object from the closed request context")

	_, err = subrequest.SafeGet("obj3")
	require.NotNil(t, err, "should not be able to create object from the closed subrequest context")

	_, err = request.SubContainer()
	require.NotNil(t, err, "should not be able to create a subcontext from a closed context")

	err = app.DeleteWithSubContainers()
	require.Nil(t, err)

	require.True(t, obj1.Closed)

	require.True(t, app.IsClosed())
	require.True(t, request.IsClosed())
	require.True(t, subrequest.IsClosed())
}

func TestDeleteWithSubContainersWithError(t *testing.T) {
	b, _ := NewEnhancedBuilder()

	b.Add(&Def{
		Name:  "obj1",
		Scope: App,
		Build: func(ctn Container) (interface{}, error) {
			return &mockD{}, nil
		},
		Close: func(obj interface{}) error {
			obj.(*mockD).Closed = true
			return nil
		},
	})
	b.Add(&Def{
		Name:  "obj2",
		Scope: Request,
		Build: func(ctn Container) (interface{}, error) {
			return &mockD{}, nil
		},
		Close: func(obj interface{}) error {
			obj.(*mockD).Closed = true
			return errors.New("close error")
		},
	})

	var err error

	app, _ := b.Build()
	request, _ := app.SubContainer()

	require.False(t, app.IsClosed())
	require.False(t, request.IsClosed())

	obj1 := request.Get("obj1").(*mockD)
	obj2 := request.Get("obj2").(*mockD)

	// delete request (it forces the request deletion)
	err = app.DeleteWithSubContainers()
	require.NotNil(t, err, "there should be an error while closing obj2")

	require.True(t, obj1.Closed)
	require.True(t, obj2.Closed)

	require.True(t, app.IsClosed())
	require.True(t, request.IsClosed())
}

func TestClosePanic(t *testing.T) {
	b, _ := NewEnhancedBuilder()

	b.Add(&Def{
		Name:  "object",
		Scope: App,
		Build: func(ctn Container) (interface{}, error) {
			return &mockD{}, nil
		},
		Close: func(obj interface{}) error {
			panic("panic in Close function")
		},
	})

	app, _ := b.Build()

	defer func() {
		require.Nil(t, recover(), "Close should not panic")
	}()

	_, err := app.SafeGet("object")
	require.Nil(t, err)

	err = app.Delete()
	require.NotNil(t, err)
}

func TestClean(t *testing.T) {
	b, _ := NewEnhancedBuilder()

	b.Add(&Def{
		Name:  "object",
		Scope: SubRequest,
		Build: func(ctn Container) (interface{}, error) {
			return &mockD{}, nil
		},
		Close: func(obj interface{}) error {
			obj.(*mockD).Closed = true
			return nil
		},
	})
	b.Add(&Def{
		Name:  "object-close-err",
		Scope: SubRequest,
		Build: func(ctn Container) (interface{}, error) {
			return &mockD{}, nil
		},
		Close: func(obj interface{}) error {
			obj.(*mockD).Closed = true
			return errors.New("close error")
		},
	})

	app, _ := b.Build()

	err := app.Clean()
	require.Nil(t, err, "should be able to call Clean even without children")

	var obj, objErr *mockD

	obj = app.UnscopedGet("object").(*mockD)
	require.False(t, obj.Closed, "the object should not be closed")

	err = app.Clean()
	require.Nil(t, err)
	require.True(t, obj.Closed, "the object should be closed")

	obj = app.UnscopedGet("object").(*mockD)
	require.False(t, obj.Closed, "it is a new object, it should not be closed")

	objErr = app.UnscopedGet("object-close-err").(*mockD)
	require.False(t, obj.Closed)

	err = app.Clean()
	require.True(t, obj.Closed)
	require.True(t, objErr.Closed)
	require.NotNil(t, err, "there should be an error because of object-close-err")
}

func TestCloseOrder(t *testing.T) {
	var (
		index  int
		closed = []string{}
	)

	b, _ := NewEnhancedBuilder()

	b.Add(&Def{
		Name:  "app-1",
		Scope: App,
		Build: func(ctn Container) (interface{}, error) {
			ctn.Get("app-2")
			return nil, nil
		},
		Close: func(obj interface{}) error {
			closed = append(closed, "app-1")
			return nil
		},
	})
	b.Add(&Def{
		Name:  "app-2",
		Scope: App,
		Build: func(ctn Container) (interface{}, error) {
			return nil, nil
		},
		Close: func(obj interface{}) error {
			closed = append(closed, "app-2")
			return nil
		},
	})
	b.Add(&Def{
		Name:  "req-1",
		Scope: Request,
		Build: func(ctn Container) (interface{}, error) {
			ctn.Get("app-1")
			return nil, nil
		},
		Close: func(obj interface{}) error {
			closed = append(closed, "req-1")
			return nil
		},
	})
	b.Add(&Def{
		Name:  "req-2",
		Scope: Request,
		Build: func(ctn Container) (interface{}, error) {
			ctn.Get("req-1")
			ctn.Get("req-3")
			ctn.Get("req-4")
			ctn.Get("app-1")
			return nil, nil
		},
		Close: func(obj interface{}) error {
			closed = append(closed, "req-2")
			return nil
		},
	})
	b.Add(&Def{
		Name:  "req-3",
		Scope: Request,
		Build: func(ctn Container) (interface{}, error) {
			ctn.Get("req-1")
			ctn.Get("app-2")
			return nil, nil
		},
		Close: func(obj interface{}) error {
			closed = append(closed, "req-3")
			return nil
		},
	})
	b.Add(&Def{
		Name:  "req-4",
		Scope: Request,
		Build: func(ctn Container) (interface{}, error) {
			ctn.Get("req-3")
			ctn.Get("app-1")
			ctn.Get("req-5")
			return nil, nil
		},
		Close: func(obj interface{}) error {
			closed = append(closed, "req-4")
			return nil
		},
	})
	b.Add(&Def{
		Name:  "req-5",
		Scope: Request,
		Build: func(ctn Container) (interface{}, error) {
			ctn.Get("app-1")
			ctn.Get("req-1")

			index++
			return index, nil
		},
		Close: func(obj interface{}) error {
			closed = append(closed, fmt.Sprintf("req-5#%d", obj.(int)))
			return nil
		},
		Unshared: true,
	})

	app, _ := b.Build()

	index = 0
	r1, _ := app.SubContainer()
	r1.Get("req-1")
	r1.Get("req-2")
	r1.Get("req-3")
	r1.Get("req-4")
	r1.Get("req-5")
	r1.Get("app-1")
	r1.Get("app-2")

	index = 0
	r2, _ := app.SubContainer()
	r2.Get("app-2")
	r2.Get("app-1")
	r2.Get("req-4")
	r2.Get("req-3")
	r2.Get("req-1")
	r2.Get("req-5")

	var err error

	err = r1.Delete()
	require.Nil(t, err)
	require.Equal(t, []string{"req-5#2", "req-2", "req-4", "req-5#1", "req-3", "req-1"}, closed)

	err = r2.Delete()
	require.Nil(t, err)
	require.Equal(t, []string{"req-5#2", "req-2", "req-4", "req-5#1", "req-3", "req-1", "req-5#2", "req-4", "req-5#1", "req-3", "req-1"}, closed)

	err = app.Delete()
	require.Nil(t, err)
	require.Equal(t, []string{"req-5#2", "req-2", "req-4", "req-5#1", "req-3", "req-1", "req-5#2", "req-4", "req-5#1", "req-3", "req-1", "app-1", "app-2"}, closed)
}

func TestCloseOrderSafeGet(t *testing.T) {
	var (
		index  int
		closed = []string{}
	)

	b, _ := NewEnhancedBuilder()

	b.Add(&Def{
		Name:  "app-1",
		Scope: App,
		Build: func(ctn Container) (interface{}, error) {
			ctn.SafeGet("app-2")
			return nil, nil
		},
		Close: func(obj interface{}) error {
			closed = append(closed, "app-1")
			return nil
		},
	})
	b.Add(&Def{
		Name:  "app-2",
		Scope: App,
		Build: func(ctn Container) (interface{}, error) {
			return nil, nil
		},
		Close: func(obj interface{}) error {
			closed = append(closed, "app-2")
			return nil
		},
	})
	b.Add(&Def{
		Name:  "req-1",
		Scope: Request,
		Build: func(ctn Container) (interface{}, error) {
			ctn.SafeGet("app-1")
			return nil, nil
		},
		Close: func(obj interface{}) error {
			closed = append(closed, "req-1")
			return nil
		},
	})
	b.Add(&Def{
		Name:  "req-2",
		Scope: Request,
		Build: func(ctn Container) (interface{}, error) {
			ctn.SafeGet("req-1")
			ctn.SafeGet("req-3")
			ctn.SafeGet("req-4")
			ctn.SafeGet("app-1")
			return nil, nil
		},
		Close: func(obj interface{}) error {
			closed = append(closed, "req-2")
			return nil
		},
	})
	b.Add(&Def{
		Name:  "req-3",
		Scope: Request,
		Build: func(ctn Container) (interface{}, error) {
			ctn.SafeGet("req-1")
			ctn.SafeGet("app-2")
			return nil, nil
		},
		Close: func(obj interface{}) error {
			closed = append(closed, "req-3")
			return nil
		},
	})
	b.Add(&Def{
		Name:  "req-4",
		Scope: Request,
		Build: func(ctn Container) (interface{}, error) {
			ctn.SafeGet("req-3")
			ctn.SafeGet("app-1")
			ctn.SafeGet("req-5")
			return nil, nil
		},
		Close: func(obj interface{}) error {
			closed = append(closed, "req-4")
			return nil
		},
	})
	b.Add(&Def{
		Name:  "req-5",
		Scope: Request,
		Build: func(ctn Container) (interface{}, error) {
			ctn.SafeGet("app-1")
			ctn.SafeGet("req-1")

			index++
			return index, nil
		},
		Close: func(obj interface{}) error {
			closed = append(closed, fmt.Sprintf("req-5#%d", obj.(int)))
			return nil
		},
		Unshared: true,
	})

	app, _ := b.Build()

	index = 0
	r1, _ := app.SubContainer()
	r1.SafeGet("req-1")
	r1.SafeGet("req-2")
	r1.SafeGet("req-3")
	r1.SafeGet("req-4")
	r1.SafeGet("req-5")
	r1.SafeGet("app-1")
	r1.SafeGet("app-2")

	index = 0
	r2, _ := app.SubContainer()
	r2.SafeGet("app-2")
	r2.SafeGet("app-1")
	r2.SafeGet("req-4")
	r2.SafeGet("req-3")
	r2.SafeGet("req-1")
	r2.SafeGet("req-5")

	var err error

	err = r1.Delete()
	require.Nil(t, err)
	require.Equal(t, []string{"req-5#2", "req-2", "req-4", "req-5#1", "req-3", "req-1"}, closed)

	err = r2.Delete()
	require.Nil(t, err)
	require.Equal(t, []string{"req-5#2", "req-2", "req-4", "req-5#1", "req-3", "req-1", "req-5#2", "req-4", "req-5#1", "req-3", "req-1"}, closed)

	err = app.Delete()
	require.Nil(t, err)
	require.Equal(t, []string{"req-5#2", "req-2", "req-4", "req-5#1", "req-3", "req-1", "req-5#2", "req-4", "req-5#1", "req-3", "req-1", "app-1", "app-2"}, closed)
}
