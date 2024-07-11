package di

import (
	"errors"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUnscopedSafeGet(t *testing.T) {
	b, _ := NewEnhancedBuilder()

	var defReq, defSub *Def

	defReq = &Def{
		Name:  "request-object",
		Scope: Request,
		Build: func(ctn Container) (interface{}, error) {
			return &mockD{Closed: true}, nil
		},
		Is: NewIs(mockD{}),
	}
	b.Add(defReq)

	defSub = &Def{
		Name:  "subrequest-object",
		Scope: SubRequest,
		Build: func(ctn Container) (interface{}, error) {
			return &mockE{
				D: ctn.Get("request-object").(*mockD),
			}, nil
		},
		Is: NewIs(mockE{}),
	}
	b.Add(defSub)

	app, _ := b.Build()

	var obj, objBis interface{}
	var err error

	// success subrequest
	obj, err = app.UnscopedSafeGet("subrequest-object")
	require.Nil(t, err)
	require.True(t, obj.(*mockE).D.Closed)

	obj, err = app.UnscopedSafeGet(defSub)
	require.Nil(t, err)
	require.True(t, obj.(*mockE).D.Closed)

	obj, err = app.UnscopedSafeGet(*defSub)
	require.Nil(t, err)
	require.True(t, obj.(*mockE).D.Closed)

	obj, err = app.UnscopedSafeGet(defSub.Index())
	require.Nil(t, err)
	require.True(t, obj.(*mockE).D.Closed)

	obj, err = app.UnscopedSafeGet(reflect.TypeOf(mockE{}))
	require.Nil(t, err)
	require.True(t, obj.(*mockE).D.Closed)

	// success request
	objBis, err = app.UnscopedSafeGet("request-object")
	require.Nil(t, err)
	require.True(t, objBis.(*mockD).Closed)

	// check link between objects
	obj.(*mockE).D.Closed = false
	require.False(t, objBis.(*mockD).Closed)

	// errors
	_, err = app.UnscopedSafeGet("unknown")
	require.NotNil(t, err, "should not be able to get an undefined object")

	_, err = app.UnscopedSafeGet(-1)
	require.NotNil(t, err)

	_, err = app.UnscopedSafeGet(1000)
	require.NotNil(t, err)

	_, err = app.UnscopedSafeGet(Def{})
	require.NotNil(t, err)

	_, err = app.UnscopedSafeGet(&Def{})
	require.NotNil(t, err)

	_, err = app.UnscopedSafeGet(reflect.TypeOf(mockA{}))
	require.NotNil(t, err)

	_, err = app.SafeGet("subrequest-object")
	require.NotNil(t, err, "should use UnscopedSafeGet instead of SafeGet")

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

func TestUnscoppedCreateChild(t *testing.T) {
	b, _ := NewEnhancedBuilder()
	app, _ := b.Build()

	req, err := app.addUnscopedChild()
	require.Nil(t, err)

	subReq, err := req.addUnscopedChild()
	require.Nil(t, err)

	_, err = subReq.addUnscopedChild()
	require.NotNil(t, err)
}

func TestUnscopedGet(t *testing.T) {
	b, _ := NewEnhancedBuilder()

	b.Add(&Def{
		Name:  "object",
		Scope: Request,
		Build: func(ctn Container) (interface{}, error) {
			return 10, nil
		},
	})
	b.Add(&Def{
		Name:  "object-close-err",
		Scope: Request,
		Build: func(ctn Container) (interface{}, error) {
			return 10, errors.New("close error")
		},
	})

	app, _ := b.Build()

	object := app.UnscopedGet("object").(int)
	require.Equal(t, 10, object)

	require.Panics(t, func() {
		app.UnscopedGet("object-close-err")
	})
}

func TestUnscopedFill(t *testing.T) {
	b, _ := NewEnhancedBuilder()

	b.Add(&Def{
		Name:  "object",
		Scope: Request,
		Build: func(ctn Container) (interface{}, error) {
			return 10, nil
		},
	})
	b.Add(&Def{
		Name:  "object-close-err",
		Scope: Request,
		Build: func(ctn Container) (interface{}, error) {
			return 10, errors.New("close error")
		},
	})

	app, _ := b.Build()

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
