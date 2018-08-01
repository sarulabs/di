package di

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

type mockObject struct {
	sync.Mutex
	Closed bool
}

type mockObjectWithDependency struct {
	Object *mockObject
}

func TestCycleError(t *testing.T) {
	b, _ := NewBuilder()

	b.Add([]Def{
		{
			Name: "o1",
			Build: func(ctn Container) (interface{}, error) {
				return &mockObjectWithDependency{
					Object: ctn.Get("o2").(*mockObjectWithDependency).Object,
				}, nil
			},
		},
		{
			Name: "o2",
			Build: func(ctn Container) (interface{}, error) {
				return &mockObjectWithDependency{
					Object: ctn.Get("o1").(*mockObjectWithDependency).Object,
				}, nil
			},
		},
	}...)

	app := b.Build()
	_, err := app.SafeGet("o1")
	require.NotNil(t, err)
}

func TestRace(t *testing.T) {
	b, _ := NewBuilder()

	b.Add([]Def{
		{
			Name:  "instance",
			Scope: App,
			Build: func(ctn Container) (interface{}, error) {
				return &mockObject{}, nil
			},
		},
		{
			Name:  "object",
			Scope: App,
			Build: func(ctn Container) (interface{}, error) {
				return &mockObject{}, nil
			},
			Close: func(obj interface{}) error {
				i := obj.(*mockObject)
				i.Lock()
				i.Closed = true
				i.Unlock()
				return nil
			},
		},
		{
			Name:  "object-with-dependency",
			Scope: Request,
			Build: func(ctn Container) (interface{}, error) {
				return &mockObjectWithDependency{
					Object: ctn.Get("object").(*mockObject),
				}, nil
			},
			Close: func(obj interface{}) error {
				o := obj.(*mockObjectWithDependency)
				o.Object.Lock()
				o.Object.Closed = true
				o.Object.Unlock()
				return nil
			},
		},
	}...)

	app := b.Build()

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
