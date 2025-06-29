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

func TestGetterGet(t *testing.T) {
	builder, _ := NewEnhancedBuilder()

	var defA, defB, defC, defBuildErr, defBuildPanic *Def
	var err error

	defA = &Def{
		Name:  "defA",
		Scope: SubRequest,
		Build: func(ctn Container) (interface{}, error) {
			return &mockA{
				BField: ctn.Get(defB).(*mockB),
				CField: ctn.Get(defC).(mockC),
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
			return &mockB{
				CField: ctn.Get(defC).(mockC),
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

	var a *mockA

	a = subrequest.Get(defA).(*mockA)
	require.Equal(t, "ok", a.BField.CField.SField)
	a = subrequest.Get(*defA).(*mockA)
	require.Equal(t, "ok", a.BField.CField.SField)
	a = subrequest.Get(defA.Index()).(*mockA)
	require.Equal(t, "ok", a.BField.CField.SField)
	a = subrequest.Get("defA").(*mockA)
	require.Equal(t, "ok", a.BField.CField.SField)
	a = subrequest.Get(reflect.TypeOf(&mockA{})).(*mockA)
	require.Equal(t, "ok", a.BField.CField.SField)

	var b *mockB

	b = subrequest.Get(defB).(*mockB)
	require.Equal(t, "ok", b.CField.SField)
	b = subrequest.Get(*defB).(*mockB)
	require.Equal(t, "ok", b.CField.SField)
	b = subrequest.Get(defB.Index()).(*mockB)
	require.Equal(t, "ok", b.CField.SField)
	b = subrequest.Get("defB").(*mockB)
	require.Equal(t, "ok", b.CField.SField)
	b = subrequest.Get(reflect.TypeOf(&mockB{})).(*mockB)
	require.Equal(t, "ok", b.CField.SField)

	var c mockC

	c = subrequest.Get(defC).(mockC)
	require.Equal(t, "ok", c.SField)
	c = subrequest.Get(*defC).(mockC)
	require.Equal(t, "ok", c.SField)
	c = subrequest.Get(defC.Index()).(mockC)
	require.Equal(t, "ok", c.SField)
	c = subrequest.Get("defC").(mockC)
	require.Equal(t, "ok", c.SField)
	c = subrequest.Get(reflect.TypeOf(mockC{})).(mockC)
	require.Equal(t, "ok", c.SField)

	// same object retrieved every time
	oA1 := subrequest.Get(defA).(*mockA)
	oA2 := subrequest.Get(defA).(*mockA)
	require.True(t, oA1 == oA2)
	oB1 := subrequest.Get(defB).(*mockB)
	oB2 := subrequest.Get(defB).(*mockB)
	require.True(t, oB1 == oB2)
	oC1 := subrequest.Get(defC).(mockC)
	oC2 := subrequest.Get(defC).(mockC)
	require.True(t, oC1 == oC2)

	// unknown definitions
	require.Panics(t, func() {
		app.Get("unknown-name")
	})
	require.Panics(t, func() {
		app.Get(Def{})
	})
	require.Panics(t, func() {
		app.Get(&Def{})
	})
	require.Panics(t, func() {
		app.Get(-1)
	})
	require.Panics(t, func() {
		app.Get(1000)
	})
	require.Panics(t, func() {
		app.Get(reflect.TypeOf(mockD{}))
	})

	// build errors
	require.Panics(t, func() {
		app.Get(defBuildErr)
	})
	require.Panics(t, func() {
		app.Get(defBuildPanic)
	})

	// scope errors
	require.Panics(t, func() {
		app.Get(defA)
	})
	require.Panics(t, func() {
		app.Get(defB)
	})
	require.Equal(t, "ok", app.Get(defC).(mockC).SField)

	// after deletion only already built objects can be retrieved
	subrequest.Delete()
	request.Delete()
	app.Delete()
	require.Panics(t, func() {
		app.Get(defC)
	})
}

func TestGetterGetWithInvalidType(t *testing.T) {
	b, _ := NewEnhancedBuilder()

	def := &Def{
		Name: "mockA",
		Is:   []reflect.Type{reflect.TypeOf(mockA{})},
		Build: func(ctn Container) (interface{}, error) {
			return mockA{}, nil
		},
	}

	b.Add(def)

	var app, _ = b.Build()

	app.Get(def)
	app.Get(def.Index())
	app.Get("mockA")
	app.Get(reflect.TypeOf(mockA{}))

	require.Panics(t, func() {
		app.Get(mockA{})
	})
}

func TestGetterGetUnshared(t *testing.T) {
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
	obj1 := app.Get("unshared")
	obj2 := app.Get("unshared")

	require.False(t, obj1 == obj2)

	// build error
	require.Panics(t, func() {
		app.Get("unshared-error")
	})

	// after deletion it is only possible to build definitions without a close function
	app.Get("unshared")
	app.Get("unshared-close")

	app.Delete()

	app.Get("unshared")
	require.Panics(t, func() {
		app.Get("unshared-close")
	})
}

func TestGetterDeleteDuringBuild(t *testing.T) {
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

	require.Panics(t, func() {
		app.Get("object")
	})
	require.True(t, app.IsClosed())
	require.True(t, built)
	require.True(t, closed)
}

func TestGetterDeleteDuringBuildWithCloseError(t *testing.T) {
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

	require.Panics(t, func() {
		app.Get("object")
	})
	require.True(t, app.IsClosed())
	require.True(t, built)
	require.True(t, closed)
}

func TestGetterCycleError(t *testing.T) {
	b, _ := NewEnhancedBuilder()

	b.Add(&Def{
		Name: "o1",
		Build: func(ctn Container) (interface{}, error) {
			ctn.Get("o2")
			return nil, nil
		},
	})
	b.Add(&Def{
		Name: "o2",
		Build: func(ctn Container) (interface{}, error) {
			ctn.Get("o3")
			return nil, nil
		},
	})
	b.Add(&Def{
		Name: "o3",
		Build: func(ctn Container) (interface{}, error) {
			ctn.Get("o1")
			return nil, nil
		},
	})

	app, _ := b.Build()

	require.Panics(t, func() {
		app.Get("o1")
	})
}

func TestGetterConcurrentBuild(t *testing.T) {
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
