package di

import (
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockItem struct {
	sync.Mutex
	Closed bool
}

type nestedMockItem struct {
	Item *mockItem
}

func TestContextScope(t *testing.T) {
	cm, _ := NewContextManager("app", "request", "subrequest")
	app, _ := cm.Context("app")
	subrequest, _ := cm.Context("subrequest")

	assert.Equal(t, "app", app.Scope())
	assert.Equal(t, "subrequest", subrequest.Scope())
}

func TestContextParentScopes(t *testing.T) {
	cm, _ := NewContextManager("app", "request", "subrequest")
	app, _ := cm.Context("app")
	subrequest, _ := cm.Context("subrequest")

	assert.Empty(t, app.ParentScopes())
	assert.Equal(t, []string{"app", "request"}, subrequest.ParentScopes())
}

func TestContextSubScopes(t *testing.T) {
	cm, _ := NewContextManager("app", "request", "subrequest")
	app, _ := cm.Context("app")
	subrequest, _ := cm.Context("subrequest")

	assert.Equal(t, []string{"request", "subrequest"}, app.SubScopes())
	assert.Empty(t, subrequest.SubScopes())
}

func TestContextHasSubScope(t *testing.T) {
	cm, _ := NewContextManager("app", "request", "subrequest")
	app, _ := cm.Context("app")
	subrequest, _ := cm.Context("subrequest")

	assert.False(t, app.HasSubScope("app"))
	assert.True(t, app.HasSubScope("request"))
	assert.True(t, app.HasSubScope("subrequest"))
	assert.False(t, app.HasSubScope("other"))

	assert.False(t, subrequest.HasSubScope("app"))
	assert.False(t, subrequest.HasSubScope("request"))
	assert.False(t, subrequest.HasSubScope("subrequest"))
	assert.False(t, subrequest.HasSubScope("other"))
}

func TestContextParentWithScope(t *testing.T) {
	cm, _ := NewContextManager("app", "request", "subrequest")
	app, _ := cm.Context("app")
	request, _ := app.SubContext("request")
	subrequest, _ := request.SubContext("subrequest")

	assert.True(t, app == request.ParentWithScope("app"))
	assert.True(t, app == subrequest.ParentWithScope("app"))
	assert.True(t, request == subrequest.ParentWithScope("request"))

	assert.Nil(t, app.ParentWithScope("undefined"))
	assert.Nil(t, app.ParentWithScope("request"))
}

func TestSubContextCreation(t *testing.T) {
	cm, _ := NewContextManager("app", "request", "subrequest")
	request, _ := cm.Context("request")

	var err error

	_, err = request.SubContext("app")
	assert.NotNil(t, err, "should not be able to create a subcontext with a parent scope")

	_, err = request.SubContext("request")
	assert.NotNil(t, err, "should not be able to create a subcontext with the same scope")

	_, err = request.SubContext("undefined")
	assert.NotNil(t, err, "should not be able to create a subcontext with an undefined scope")

	subrequest, err := request.SubContext("subrequest")
	assert.Nil(t, err, "should be able to create a subrequest Context")
	assert.Equal(t, "subrequest", subrequest.Scope())
	assert.True(t, request == subrequest.Parent())

	subrequest2, _ := request.SubContext("subrequest")
	assert.True(t, subrequest != subrequest2, "should not create the same subrequest twice")
}

func TestInstanceSafeGet(t *testing.T) {
	cm, _ := NewContextManager("app", "request", "subrequest")

	one := &mockItem{}
	two := &mockItem{}

	cm.Instance(Instance{Name: "i1", Aliases: []string{"a1"}, Item: one})
	cm.Instance(Instance{Name: "i2", Aliases: []string{"a2"}, Item: two})

	app, _ := cm.Context("app")
	request, _ := cm.Context("request")

	_, err := app.SafeGet("undefined")
	assert.NotNil(t, err, "should not be able to create an undefined instance")

	// SafeGet should work from tha app Context
	item1, err := app.SafeGet("i1")
	assert.Nil(t, err)
	assert.True(t, one == item1.(*mockItem))

	// SafeGet should also work from the request Context and with an alias
	item2, err := request.SafeGet("a2")
	assert.Nil(t, err)
	assert.True(t, two == item2.(*mockItem))
}

func TestMakerSafeGet(t *testing.T) {
	cm, _ := NewContextManager("app", "request", "subrequest")

	cm.Maker(Maker{
		Name:    "item",
		Aliases: []string{"i"},
		Scope:   "request",
		Make: func(ctx *Context) (interface{}, error) {
			return &mockItem{}, nil
		},
	})

	cm.Maker(Maker{
		Name:  "unmakable",
		Scope: "request",
		Make: func(ctx *Context) (interface{}, error) {
			return nil, errors.New("error")
		},
	})

	app, _ := cm.Context("app")
	request, _ := app.SubContext("request")
	subrequest, _ := request.SubContext("subrequest")

	var item, item2 interface{}
	var err error

	_, err = app.SafeGet("item")
	assert.NotNil(t, err, "should not be able to create the item from the app scope")

	_, err = request.SafeGet("undefined")
	assert.NotNil(t, err, "should not be able to create an undefined item")

	_, err = request.SafeGet("unmakable")
	assert.NotNil(t, err, "should not be able to create an item if there is an error in the Make function")

	// should be able to create the item from the request scope
	item, err = request.SafeGet("item")
	assert.Nil(t, err)
	assert.Equal(t, &mockItem{}, item.(*mockItem))

	// should retrieve the same item every time
	item2, err = request.SafeGet("item")
	assert.Nil(t, err)
	assert.Equal(t, &mockItem{}, item2.(*mockItem))
	assert.True(t, item == item2)

	// should work with an alias
	item, err = request.SafeGet("i")
	assert.Nil(t, err)
	assert.Equal(t, &mockItem{}, item.(*mockItem))

	// should be able to create an item from a subcontext
	item, err = subrequest.SafeGet("item")
	assert.Nil(t, err)
	assert.Equal(t, &mockItem{}, item.(*mockItem))
	assert.True(t, item == item2)
}

func TestMakePanic(t *testing.T) {
	cm, _ := NewContextManager("app")

	cm.Maker(Maker{
		Name:  "item",
		Scope: "app",
		Make: func(ctx *Context) (interface{}, error) {
			panic("panic in Make function")
		},
	})

	app, _ := cm.Context("app")

	defer func() {
		assert.Nil(t, recover(), "SafeGet should not panic")
	}()

	_, err := app.SafeGet("item")
	assert.NotNil(t, err, "should not panic but not be able to create the item either")
}

func TestNestedDependencies(t *testing.T) {
	cm, _ := NewContextManager("app", "request", "subrequest")

	appItem := &mockItem{}

	cm.Set("appItem", appItem)

	cm.Maker(Maker{
		Name:  "requestItem",
		Scope: "request",
		Make: func(ctx *Context) (interface{}, error) {
			return &nestedMockItem{ctx.Get("appItem").(*mockItem)}, nil
		},
	})

	request, _ := cm.Context("request")

	nestedItem := request.Get("requestItem").(*nestedMockItem)
	assert.True(t, appItem == nestedItem.Item)
}

func TestMake(t *testing.T) {
	cm, _ := NewContextManager("app", "request", "subrequest")

	cm.Maker(Maker{
		Name:  "item",
		Scope: "request",
		Make: func(ctx *Context) (interface{}, error) {
			return 10, nil
		},
	})

	request, _ := cm.Context("request")

	item := request.Get("item").(int)
	assert.Equal(t, 10, item)
}

func TestFill(t *testing.T) {
	cm, _ := NewContextManager("app")

	cm.Maker(Maker{
		Name:  "item",
		Scope: "app",
		Make: func(ctx *Context) (interface{}, error) {
			return 10, nil
		},
	})

	app, _ := cm.Context("app")

	var err error
	var item int
	var wrongType string

	err = app.Fill("item", &wrongType)
	assert.NotNil(t, err, "should have failed to fill an item with the wrong type")

	err = app.Fill("item", &item)
	assert.Nil(t, err)
	assert.Equal(t, 10, item)
}

func TestDelete(t *testing.T) {
	cm, _ := NewContextManager("app", "request", "subrequest")

	cm.Maker(Maker{
		Name:  "i1",
		Scope: "app",
		Make: func(ctx *Context) (interface{}, error) {
			return &mockItem{}, nil
		},
		Close: func(item interface{}) {
			i := item.(*mockItem)
			i.Lock()
			i.Closed = true
			i.Unlock()
		},
	})

	cm.Maker(Maker{
		Name:  "i2",
		Scope: "request",
		Make: func(ctx *Context) (interface{}, error) {
			return &mockItem{}, nil
		},
		Close: func(item interface{}) {
			i := item.(*mockItem)
			i.Lock()
			i.Closed = true
			i.Unlock()
		},
	})

	cm.Maker(Maker{
		Name:  "i3",
		Scope: "subrequest",
		Make: func(ctx *Context) (interface{}, error) {
			return &mockItem{}, nil
		},
		Close: func(item interface{}) {
			i := item.(*mockItem)
			i.Lock()
			i.Closed = true
			i.Unlock()
		},
	})

	cm.Maker(Maker{
		Name:  "i4",
		Scope: "subrequest",
		Make: func(ctx *Context) (interface{}, error) {
			return &mockItem{}, nil
		},
	})

	app, _ := cm.Context("app")
	request, _ := app.SubContext("request")
	subrequest, _ := request.SubContext("subrequest")

	var err error

	i1 := app.Get("i1").(*mockItem)
	i2 := request.Get("i2").(*mockItem)
	i3 := subrequest.Get("i3").(*mockItem)
	_ = subrequest.Get("i4").(*mockItem)

	request.Delete()

	assert.False(t, i1.Closed)
	assert.True(t, i2.Closed)
	assert.True(t, i3.Closed)

	assert.Nil(t, request.Parent(), "should have removed request parent")
	assert.Nil(t, subrequest.Parent(), "should have removed subrequest parent")

	_, err = app.SafeGet("i1")
	assert.Nil(t, err, "should still be able to create item from the app context")

	_, err = request.SafeGet("i2")
	assert.NotNil(t, err, "should not be able to create item from the closed request context")

	_, err = subrequest.SafeGet("i3")
	assert.NotNil(t, err, "should not be able to create item from the closed subrequest context")

	_, err = request.SubContext("subrequest")
	assert.NotNil(t, err, "should not be able to create a subcontext from a closed context")

	app.Delete()

	assert.True(t, i1.Closed)
}

func TestClosePanic(t *testing.T) {
	cm, _ := NewContextManager("app")

	cm.Maker(Maker{
		Name:  "item",
		Scope: "app",
		Make: func(ctx *Context) (interface{}, error) {
			return &mockItem{}, nil
		},
		Close: func(item interface{}) {
			panic("panic in Close function")
		},
	})

	app, _ := cm.Context("app")

	defer func() {
		assert.Nil(t, recover(), "Close should not panic")
	}()

	_, err := app.SafeGet("item")
	assert.Nil(t, err)

	app.Delete()
}

func TestRace(t *testing.T) {
	cm, _ := NewContextManager("app", "request", "subrequest")

	cm.Set("instance", &mockItem{})

	cm.Maker(Maker{
		Name:  "item",
		Scope: "app",
		Make: func(ctx *Context) (interface{}, error) {
			return &mockItem{}, nil
		},
		Close: func(item interface{}) {
			i := item.(*mockItem)
			i.Lock()
			i.Closed = true
			i.Unlock()
		},
	})

	cm.Maker(Maker{
		Name:  "nested",
		Scope: "request",
		Make: func(ctx *Context) (interface{}, error) {
			return &nestedMockItem{ctx.Get("item").(*mockItem)}, nil
		},
		Close: func(item interface{}) {
			i := item.(*nestedMockItem)
			i.Item.Lock()
			i.Item.Closed = true
			i.Item.Unlock()
		},
	})

	app, _ := cm.Context("app")

	for i := 0; i < 1000; i++ {
		go func() {
			request, _ := app.SubContext("request")
			defer request.Delete()

			request.Get("item")
			request.Get("instance")
			request.Get("nested")

			go func() {
				subrequest, _ := app.SubContext("subrequest")
				defer subrequest.Delete()

				subrequest.Get("item")
				subrequest.Get("instance")
				subrequest.Get("nested")
				subrequest.Get("item")
				subrequest.Get("instance")
				subrequest.Get("nested")
			}()

			request.Get("item")
			request.Get("instance")
			request.Get("nested")
		}()
	}
}
