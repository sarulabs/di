package di

import (
	"reflect"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

type mockA struct {
	BField *mockB
	CField mockC
	SField string
}
type mockB struct{ CField mockC }
type mockC struct{ SField string }
type mockD struct {
	sync.Mutex
	Closed bool
}
type mockE struct {
	D *mockD
}

func TestRace(t *testing.T) {
	b, _ := NewEnhancedBuilder()

	b.Add(&Def{
		Name:  "instance",
		Scope: App,
		Build: func(ctn Container) (interface{}, error) {
			return &mockD{}, nil
		},
	})
	b.Add(&Def{
		Name:  "object",
		Scope: App,
		Build: func(ctn Container) (interface{}, error) {
			return &mockD{}, nil
		},
		Close: func(obj interface{}) error {
			i := obj.(*mockD)
			i.Lock()
			i.Closed = true
			i.Unlock()
			return nil
		},
	})
	b.Add(&Def{
		Name:  "object-with-dependency",
		Scope: Request,
		Build: func(ctn Container) (interface{}, error) {
			return &mockE{
				D: ctn.Get("object").(*mockD),
			}, nil
		},
		Close: func(obj interface{}) error {
			o := obj.(*mockE)
			o.D.Lock()
			o.D.Closed = true
			o.D.Unlock()
			return nil
		},
	})

	app, _ := b.Build()

	var wgApp sync.WaitGroup

	for i := 0; i < 100; i++ {
		wgApp.Add(1)

		go func() {
			defer wgApp.Done()

			request, _ := app.SubContainer()
			defer request.Delete()

			request.Get("instance")
			request.Get("object")
			request.Get("object-with-dependency")

			var wgReq sync.WaitGroup

			for j := 0; j < 10; j++ {
				wgReq.Add(1)

				go func() {
					defer wgReq.Done()

					subrequest, _ := request.SubContainer()
					defer subrequest.Delete()

					subrequest.Get("instance")
					subrequest.Get("object")
					subrequest.Get("object-with-dependency")
					subrequest.Get("instance")
					subrequest.Get("object")
					subrequest.Get("object-with-dependency")
				}()
			}

			wgReq.Wait()

			request.Get("instance")
			request.Get("object")
			request.Get("object-with-dependency")
		}()
	}

	wgApp.Wait()
}

func TestContainerDefinitions(t *testing.T) {
	b, _ := NewEnhancedBuilder()

	b.Add(&Def{
		Name: "o1",
		Build: func(ctn Container) (interface{}, error) {
			return nil, nil
		},
	})
	b.Add(&Def{
		Name: "o2",
		Build: func(ctn Container) (interface{}, error) {
			return nil, nil
		},
	})

	app, _ := b.Build()
	defs := app.Definitions()

	require.Len(t, defs, 2)
	require.Equal(t, "o1", defs["o1"].Name)
	require.Equal(t, "o2", defs["o2"].Name)
}

func TestContainerNameIsDefined(t *testing.T) {
	b, _ := NewEnhancedBuilder()

	b.Add(&Def{
		Name: "o1",
		Build: func(ctn Container) (interface{}, error) {
			return nil, nil
		},
	})

	app, _ := b.Build()

	require.True(t, app.NameIsDefined("o1"))
	require.False(t, app.NameIsDefined("o2"))
}

func TestContainerTypeIsDefined(t *testing.T) {
	b, _ := NewEnhancedBuilder()

	b.Add(&Def{
		Name: "o1",
		Build: func(ctn Container) (interface{}, error) {
			return nil, nil
		},
		Is: NewIs(&mockA{}, ""),
	})

	app, _ := b.Build()

	require.True(t, app.TypeIsDefined(reflect.TypeOf(&mockA{})))
	require.True(t, app.TypeIsDefined(reflect.TypeOf("")))
	require.False(t, app.TypeIsDefined(reflect.TypeOf(mockA{})))
	require.False(t, app.TypeIsDefined(reflect.TypeOf(1)))
}

func TestContainerDefinitionsForType(t *testing.T) {
	b, _ := NewEnhancedBuilder()

	def1 := &Def{
		Name: "o1",
		Build: func(ctn Container) (interface{}, error) {
			return nil, nil
		},
		Is: NewIs(&mockA{}, ""),
	}

	b.Add(def1)

	def2 := &Def{
		Name: "o2",
		Build: func(ctn Container) (interface{}, error) {
			return nil, nil
		},
		Is: NewIs(mockA{}, ""),
	}

	b.Add(def2)

	app, _ := b.Build()

	strTypes := app.DefinitionsForType(reflect.TypeOf(""))
	structTypes := app.DefinitionsForType(reflect.TypeOf(mockA{}))
	ptrTypes := app.DefinitionsForType(reflect.TypeOf(&mockA{}))
	require.Equal(t, 2, len(strTypes))
	require.Equal(t, def1.Name, strTypes[0].Name)
	require.Equal(t, def2.Name, strTypes[1].Name)
	require.Equal(t, 1, len(structTypes))
	require.Equal(t, def2.Name, structTypes[0].Name)
	require.Equal(t, 1, len(ptrTypes))
	require.Equal(t, def1.Name, ptrTypes[0].Name)
}

func TestContainerScope(t *testing.T) {
	b, _ := NewEnhancedBuilder()
	app, _ := b.Build()
	request, _ := app.SubContainer()
	subrequest, _ := request.SubContainer()

	require.Equal(t, App, app.Scope())
	require.Equal(t, Request, request.Scope())
	require.Equal(t, SubRequest, subrequest.Scope())
}

func TestContainerScopes(t *testing.T) {
	b, _ := NewEnhancedBuilder()
	app, _ := b.Build()
	request, _ := app.SubContainer()
	subrequest, _ := request.SubContainer()

	list := []string{App, Request, SubRequest}

	require.Equal(t, list, app.Scopes())
	require.Equal(t, list, request.Scopes())
	require.Equal(t, list, subrequest.Scopes())
}

func TestContainerParentScopes(t *testing.T) {
	b, _ := NewEnhancedBuilder()
	app, _ := b.Build()
	request, _ := app.SubContainer()
	subrequest, _ := request.SubContainer()

	require.Empty(t, app.ParentScopes())
	require.Equal(t, []string{App}, request.ParentScopes())
	require.Equal(t, []string{App, Request}, subrequest.ParentScopes())
}

func TestContainerSubScopes(t *testing.T) {
	b, _ := NewEnhancedBuilder()
	app, _ := b.Build()
	request, _ := app.SubContainer()
	subrequest, _ := request.SubContainer()

	require.Equal(t, []string{Request, SubRequest}, app.SubScopes())
	require.Equal(t, []string{SubRequest}, request.SubScopes())
	require.Empty(t, subrequest.SubScopes())
}
