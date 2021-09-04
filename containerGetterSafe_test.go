package di

import (
	"errors"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGetterSafeGet(t *testing.T) {
	builder, _ := NewEnhancedBuilder()

	var defA, defB, defC, defBuildErr, defBuildPanic *Def
	var err error

	defA = &Def{
		Name:  "defA",
		Scope: SubRequest,
		Build: func(ctn Container) (interface{}, error) {
			b, _ := ctn.SafeGet(defB)
			c, _ := ctn.SafeGet(defC)
			return &mockA{
				BField: b.(*mockB),
				CField: c.(mockC),
				SField: "a",
			}, nil
		},
		Is: NewIs(&mockA{}),
	}
	err = builder.Add(defA)
	require.Nil(t, err)

	defB = &Def{
		Name:  "defB",
		Scope: Request,
		Build: func(ctn Container) (interface{}, error) {
			c, _ := ctn.SafeGet(defC)
			return &mockB{
				CField: c.(mockC),
			}, nil
		},
		Is: NewIs(&mockB{}),
	}
	err = builder.Add(defB)
	require.Nil(t, err)

	defC = &Def{
		Name:  "defC",
		Scope: App,
		Build: func(ctn Container) (interface{}, error) {
			return mockC{
				SField: "ok",
			}, nil
		},
		Is: NewIs(mockC{}),
	}
	err = builder.Add(defC)
	require.Nil(t, err)

	defBuildErr = &Def{
		Build: func(ctn Container) (interface{}, error) {
			return nil, errors.New("error in Build function")
		},
	}
	err = builder.Add(defC)
	require.Nil(t, err)

	defBuildPanic = &Def{
		Build: func(ctn Container) (interface{}, error) {
			panic("panic in Build function")
		},
	}
	err = builder.Add(defC)
	require.Nil(t, err)

	app, _ := builder.Build()
	request, _ := app.SubContainer()
	subrequest, _ := request.SubContainer()

	var a interface{}

	a, err = subrequest.SafeGet(defA)
	require.Nil(t, err)
	require.Equal(t, "ok", a.(*mockA).BField.CField.SField)
	a, err = subrequest.SafeGet(*defA)
	require.Nil(t, err)
	require.Equal(t, "ok", a.(*mockA).BField.CField.SField)
	a, err = subrequest.SafeGet(defA.Index())
	require.Nil(t, err)
	require.Equal(t, "ok", a.(*mockA).BField.CField.SField)
	a, err = subrequest.SafeGet("defA")
	require.Nil(t, err)
	require.Equal(t, "ok", a.(*mockA).BField.CField.SField)
	a, err = subrequest.SafeGet(reflect.TypeOf(&mockA{}))
	require.Nil(t, err)
	require.Equal(t, "ok", a.(*mockA).BField.CField.SField)

	var b interface{}

	b, err = subrequest.SafeGet(defB)
	require.Nil(t, err)
	require.Equal(t, "ok", b.(*mockB).CField.SField)
	b, err = subrequest.SafeGet(*defB)
	require.Nil(t, err)
	require.Equal(t, "ok", b.(*mockB).CField.SField)
	require.Nil(t, err)
	b, err = subrequest.SafeGet(defB.Index())
	require.Equal(t, "ok", b.(*mockB).CField.SField)
	require.Nil(t, err)
	b, err = subrequest.SafeGet("defB")
	require.Nil(t, err)
	require.Equal(t, "ok", b.(*mockB).CField.SField)
	b, err = subrequest.SafeGet(reflect.TypeOf(&mockB{}))
	require.Nil(t, err)
	require.Equal(t, "ok", b.(*mockB).CField.SField)

	var c interface{}

	c, err = subrequest.SafeGet(defC)
	require.Nil(t, err)
	require.Equal(t, "ok", c.(mockC).SField)
	c, err = subrequest.SafeGet(*defC)
	require.Nil(t, err)
	require.Equal(t, "ok", c.(mockC).SField)
	c, err = subrequest.SafeGet(defC.Index())
	require.Nil(t, err)
	require.Equal(t, "ok", c.(mockC).SField)
	c, err = subrequest.SafeGet("defC")
	require.Nil(t, err)
	require.Equal(t, "ok", c.(mockC).SField)
	c, err = subrequest.SafeGet(reflect.TypeOf(mockC{}))
	require.Nil(t, err)
	require.Equal(t, "ok", c.(mockC).SField)

	// same object retrieved every time
	var a2, b2, c2 interface{}
	a, err = subrequest.SafeGet(defA)
	require.Nil(t, err)
	a2, err = subrequest.SafeGet(defA)
	require.Nil(t, err)
	b, err = subrequest.SafeGet(defB)
	require.Nil(t, err)
	b2, err = subrequest.SafeGet(defB)
	require.Nil(t, err)
	c, err = subrequest.SafeGet(defC)
	require.Nil(t, err)
	c2, err = subrequest.SafeGet(defC)
	require.Nil(t, err)
	require.True(t, a.(*mockA) == a2.(*mockA))
	require.True(t, b.(*mockB) == b2.(*mockB))
	require.True(t, c.(mockC) == c2.(mockC))

	// unknown definitions
	_, err = app.SafeGet("unknown-name")
	require.NotNil(t, err)
	_, err = app.SafeGet(Def{})
	require.NotNil(t, err)
	_, err = app.SafeGet(&Def{})
	require.NotNil(t, err)
	_, err = app.SafeGet(-1)
	require.NotNil(t, err)
	_, err = app.SafeGet(1000)
	require.NotNil(t, err)
	_, err = app.SafeGet(reflect.TypeOf(mockD{}))
	require.NotNil(t, err)

	// build errors
	_, err = app.SafeGet(defBuildErr)
	require.NotNil(t, err)
	_, err = app.SafeGet(defBuildPanic)
	require.NotNil(t, err)

	// scope errors
	_, err = app.SafeGet(defA)
	require.NotNil(t, err)
	_, err = app.SafeGet(defB)
	require.NotNil(t, err)
	c, err = app.SafeGet(defC)
	require.Nil(t, err)
	require.Equal(t, "ok", c.(mockC).SField)

	// after deletion only already built objects can be retrieved
	subrequest.Delete()
	request.Delete()
	app.Delete()
	_, err = app.SafeGet(defC)
	require.NotNil(t, err)

}

func TestGetterSafeGetUnshared(t *testing.T) {
	b, _ := NewEnhancedBuilder()

	b.Add(&Def{
		Name: "unshared",
		Build: func(ctn Container) (interface{}, error) {
			return &mockA{}, nil
		},
		Unshared: true,
	})

	b.Add(&Def{
		Name: "unshared-close",
		Build: func(ctn Container) (interface{}, error) {
			return &mockA{}, nil
		},
		Close: func(obj interface{}) error {
			return nil
		},
		Unshared: true,
	})

	b.Add(&Def{
		Name: "unshared-error",
		Build: func(ctn Container) (interface{}, error) {
			return nil, errors.New("build-error")
		},
		Unshared: true,
	})

	var app, _ = b.Build()

	// should retrieve different object every time
	obj1, err := app.SafeGet("unshared")
	require.Nil(t, err)
	obj2, err := app.SafeGet("unshared")
	require.Nil(t, err)

	require.False(t, obj1 == obj2)

	// build error
	_, err = app.SafeGet("unshared-error")
	require.NotNil(t, err)

	// after deletion it is only possible to build definitions without a close function
	_, err = app.SafeGet("unshared")
	require.Nil(t, err)
	_, err = app.SafeGet("unshared-close")
	require.Nil(t, err)

	app.Delete()

	_, err = app.SafeGet("unshared")
	require.Nil(t, err)
	_, err = app.SafeGet("unshared-close")
	require.NotNil(t, err)
}

func TestGetterSafeDeleteDuringBuild(t *testing.T) {
	built := false
	closed := false

	b, _ := NewEnhancedBuilder()

	b.Add(&Def{
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

	app, _ := b.Build()

	_, err := app.SafeGet("object")
	require.NotNil(t, err)
	require.True(t, app.IsClosed())
	require.True(t, built)
	require.True(t, closed)
}

func TestGetterSafeDeleteDuringBuildWithCloseError(t *testing.T) {
	built := false
	closed := false

	b, _ := NewEnhancedBuilder()

	b.Add(&Def{
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

	app, _ := b.Build()

	_, err := app.SafeGet("object")
	require.NotNil(t, err)
	require.True(t, app.IsClosed())
	require.True(t, built)
	require.True(t, closed)
}

func TestGetterSafeCycleError(t *testing.T) {
	b, _ := NewEnhancedBuilder()

	b.Add(&Def{
		Name: "o1",
		Build: func(ctn Container) (interface{}, error) {
			return ctn.SafeGet("o2")
		},
	})
	b.Add(&Def{
		Name: "o2",
		Build: func(ctn Container) (interface{}, error) {
			return ctn.SafeGet("o3")
		},
	})
	b.Add(&Def{
		Name: "o3",
		Build: func(ctn Container) (interface{}, error) {
			return ctn.SafeGet("o1")
		},
	})

	app, _ := b.Build()

	_, err := app.SafeGet("o1")
	require.NotNil(t, err)
}

func TestGetterSafeConcurrentBuild(t *testing.T) {
	var numBuild uint64
	var numClose uint64

	b, _ := NewEnhancedBuilder()

	b.Add(&Def{
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

	app, _ := b.Build()

	var wg sync.WaitGroup

	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			req, _ := app.SubContainer()
			req.SafeGet("object")
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
