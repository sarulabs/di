package di

import (
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

	app := b.Build()
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

	app := b.Build()

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
