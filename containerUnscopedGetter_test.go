package di

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUnscopedSafeGet(t *testing.T) {
	b, _ := NewBuilder()

	b.Add([]Def{
		{
			Name:  "request-object",
			Scope: Request,
			Build: func(ctn Container) (interface{}, error) {
				return &mockObject{Closed: true}, nil
			},
		},
		{
			Name:  "subrequest-object",
			Scope: SubRequest,
			Build: func(ctn Container) (interface{}, error) {
				return &mockObjectWithDependency{
					Object: ctn.Get("request-object").(*mockObject),
				}, nil
			},
		},
	}...)

	app := b.Build()

	var obj, objBis interface{}
	var err error

	_, err = app.UnscopedSafeGet("unknown")
	require.NotNil(t, err, "should not be able to get an undefined object")

	_, err = app.SafeGet("subrequest-object")
	require.NotNil(t, err, "should use UnscopedSafeGet instead of SafeGet")

	obj, err = app.UnscopedSafeGet("subrequest-object")
	require.Nil(t, err)
	require.True(t, obj.(*mockObjectWithDependency).Object.Closed)

	objBis, err = app.UnscopedSafeGet("request-object")
	require.Nil(t, err)
	require.True(t, objBis.(*mockObject).Closed)

	obj.(*mockObjectWithDependency).Object.Closed = false
	require.False(t, objBis.(*mockObject).Closed)

	// can call UnscopedSafeGet on sub-request too
	req, _ := app.SubContainer()
	subReq, _ := req.SubContainer()

	_, err = subReq.UnscopedSafeGet("subrequest-object")
	require.Nil(t, err)

	// error if the container has been deleted
	err = req.DeleteWithSubContainers()
	require.Nil(t, err)
	_, err = req.UnscopedSafeGet("request-object")
	require.NotNil(t, err)
	_, err = req.UnscopedSafeGet("subrequest-object")
	require.NotNil(t, err)
}

func TestUnscopedGet(t *testing.T) {
	b, _ := NewBuilder()

	b.Add([]Def{
		{
			Name:  "object",
			Scope: Request,
			Build: func(ctn Container) (interface{}, error) {
				return 10, nil
			},
		},
		{
			Name:  "object-close-err",
			Scope: Request,
			Build: func(ctn Container) (interface{}, error) {
				return 10, errors.New("close error")
			},
		},
	}...)

	app := b.Build()

	object := app.UnscopedGet("object").(int)
	require.Equal(t, 10, object)

	require.Panics(t, func() {
		app.UnscopedGet("object-close-err")
	})
}

func TestUnscopedFill(t *testing.T) {
	b, _ := NewBuilder()

	b.Add([]Def{
		{
			Name:  "object",
			Scope: Request,
			Build: func(ctn Container) (interface{}, error) {
				return 10, nil
			},
		},
		{
			Name:  "object-close-err",
			Scope: Request,
			Build: func(ctn Container) (interface{}, error) {
				return 10, errors.New("close error")
			},
		},
	}...)

	app := b.Build()

	var object int
	var wrongType string

	err := app.UnscopedFill("object", &object)
	require.Nil(t, err)
	require.Equal(t, 10, object)

	err = app.UnscopedFill("object", &wrongType)
	require.NotNil(t, err)

	err = app.UnscopedFill("object-close-err", &object)
	require.NotNil(t, err)
}
