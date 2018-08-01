package di

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDelete(t *testing.T) {
	b, _ := NewBuilder()

	b.Add([]Def{
		{
			Name:  "obj1",
			Scope: App,
			Build: func(ctn Container) (interface{}, error) {
				return &mockObject{}, nil
			},
			Close: func(obj interface{}) error {
				obj.(*mockObject).Closed = true
				return nil
			},
		},
		{
			Name:  "obj2",
			Scope: Request,
			Build: func(ctn Container) (interface{}, error) {
				return &mockObject{}, nil
			},
		},
	}...)

	var err error

	app := b.Build()
	request, _ := app.SubContainer()
	obj1 := request.Get("obj1").(*mockObject)
	obj2 := request.Get("obj2").(*mockObject)

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
	b, _ := NewBuilder()

	b.Add([]Def{
		{
			Name:  "obj1",
			Scope: App,
			Build: func(ctn Container) (interface{}, error) {
				return &mockObject{}, nil
			},
			Close: func(obj interface{}) error {
				obj.(*mockObject).Closed = true
				return errors.New("close error")
			},
		},
		{
			Name:  "obj2",
			Scope: Request,
			Build: func(ctn Container) (interface{}, error) {
				return &mockObject{}, nil
			},
		},
	}...)

	var err error

	app := b.Build()
	request, _ := app.SubContainer()
	obj1 := request.Get("obj1").(*mockObject)
	obj2 := request.Get("obj2").(*mockObject)

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
	b, _ := NewBuilder()

	b.Add([]Def{
		{
			Name:  "obj1",
			Scope: App,
			Build: func(ctn Container) (interface{}, error) {
				return &mockObject{}, nil
			},
			Close: func(obj interface{}) error {
				obj.(*mockObject).Closed = true
				return nil
			},
		},
		{
			Name:  "obj2",
			Scope: Request,
			Build: func(ctn Container) (interface{}, error) {
				return &mockObject{}, nil
			},
			Close: func(obj interface{}) error {
				obj.(*mockObject).Closed = true
				return nil
			},
		},
		{
			Name:  "obj3",
			Scope: SubRequest,
			Build: func(ctn Container) (interface{}, error) {
				return &mockObject{}, nil
			},
			Close: func(obj interface{}) error {
				obj.(*mockObject).Closed = true
				return nil
			},
		},
		{
			Name:  "obj4",
			Scope: SubRequest,
			Build: func(ctn Container) (interface{}, error) {
				return &mockObject{}, nil
			},
		},
	}...)

	var err error

	app := b.Build()
	request, _ := app.SubContainer()
	subrequest, _ := request.SubContainer()

	require.False(t, app.IsClosed())
	require.False(t, request.IsClosed())
	require.False(t, subrequest.IsClosed())

	obj1 := app.Get("obj1").(*mockObject)
	obj2 := request.Get("obj2").(*mockObject)
	obj3 := subrequest.Get("obj3").(*mockObject)
	_ = subrequest.Get("obj4").(*mockObject)

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
	b, _ := NewBuilder()

	b.Add([]Def{
		{
			Name:  "obj1",
			Scope: App,
			Build: func(ctn Container) (interface{}, error) {
				return &mockObject{}, nil
			},
			Close: func(obj interface{}) error {
				obj.(*mockObject).Closed = true
				return nil
			},
		},
		{
			Name:  "obj2",
			Scope: Request,
			Build: func(ctn Container) (interface{}, error) {
				return &mockObject{}, nil
			},
			Close: func(obj interface{}) error {
				obj.(*mockObject).Closed = true
				return errors.New("close error")
			},
		},
	}...)

	var err error

	app := b.Build()
	request, _ := app.SubContainer()

	require.False(t, app.IsClosed())
	require.False(t, request.IsClosed())

	obj1 := request.Get("obj1").(*mockObject)
	obj2 := request.Get("obj2").(*mockObject)

	// delete request (it forces the request deletion)
	err = app.DeleteWithSubContainers()
	require.NotNil(t, err, "there should be an error while closing obj2")

	require.True(t, obj1.Closed)
	require.True(t, obj2.Closed)

	require.True(t, app.IsClosed())
	require.True(t, request.IsClosed())
}

func TestClosePanic(t *testing.T) {
	b, _ := NewBuilder()

	b.Add(Def{
		Name:  "object",
		Scope: App,
		Build: func(ctn Container) (interface{}, error) {
			return &mockObject{}, nil
		},
		Close: func(obj interface{}) error {
			panic("panic in Close function")
		},
	})

	app := b.Build()

	defer func() {
		require.Nil(t, recover(), "Close should not panic")
	}()

	_, err := app.SafeGet("object")
	require.Nil(t, err)

	err = app.Delete()
	require.NotNil(t, err)
}

func TestClean(t *testing.T) {
	b, _ := NewBuilder()

	b.Add([]Def{
		{
			Name:  "object",
			Scope: SubRequest,
			Build: func(ctn Container) (interface{}, error) {
				return &mockObject{}, nil
			},
			Close: func(obj interface{}) error {
				obj.(*mockObject).Closed = true
				return nil
			},
		},
		{
			Name:  "object-close-err",
			Scope: SubRequest,
			Build: func(ctn Container) (interface{}, error) {
				return &mockObject{}, nil
			},
			Close: func(obj interface{}) error {
				obj.(*mockObject).Closed = true
				return errors.New("close error")
			},
		},
	}...)

	app := b.Build()

	err := app.Clean()
	require.Nil(t, err, "should be able to call Clean even without children")

	var obj, objErr *mockObject

	obj = app.UnscopedGet("object").(*mockObject)
	require.False(t, obj.Closed, "the object should not be closed")

	err = app.Clean()
	require.Nil(t, err)
	require.True(t, obj.Closed, "the object should be closed")

	obj = app.UnscopedGet("object").(*mockObject)
	require.False(t, obj.Closed, "it is a new object, it should not be closed")

	objErr = app.UnscopedGet("object-close-err").(*mockObject)
	require.False(t, obj.Closed)

	err = app.Clean()
	require.True(t, obj.Closed)
	require.True(t, objErr.Closed)
	require.NotNil(t, err, "there should be an error because of object-close-err")
}
