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

func TestInstanceSafeMake(t *testing.T) {
	cm, _ := NewContextManager("app", "request", "subrequest")

	one := &mockItem{}
	two := &mockItem{}

	cm.Instance(Instance{Name: "i1", Aliases: []string{"a1"}, Item: one})
	cm.Instance(Instance{Name: "i2", Aliases: []string{"a2"}, Item: two})

	app, _ := cm.Context("app")
	request, _ := cm.Context("request")

	_, err := app.SafeMake("undefined")
	assert.NotNil(t, err, "should not be able to create an undefined instance")

	// SafeMake should work from tha app Context
	item1, err := app.SafeMake("i1")
	assert.Nil(t, err)
	assert.True(t, one == item1.(*mockItem))

	// SafeMake should also work from the request Context and with an alias
	item2, err := request.SafeMake("a2")
	assert.Nil(t, err)
	assert.True(t, two == item2.(*mockItem))
}

func TestMakerSafeMake(t *testing.T) {
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

	_, err = app.SafeMake("item")
	assert.NotNil(t, err, "should not be able to create the item from the app scope")

	_, err = request.SafeMake("undefined")
	assert.NotNil(t, err, "should not be able to create an undefined item")

	_, err = request.SafeMake("unmakable")
	assert.NotNil(t, err, "should not be able to create an item if there is an error in the Make function")

	// should be able to create the item from the request scope
	item, err = request.SafeMake("item")
	assert.Nil(t, err)
	assert.Equal(t, &mockItem{}, item.(*mockItem))

	// should retrieve a different item every time, it is not a singleton
	item.(*mockItem).Closed = true

	item2, err = request.SafeMake("item")
	assert.Nil(t, err)
	assert.Equal(t, &mockItem{}, item2.(*mockItem))
	assert.True(t, item != item2)

	// should work with an alias
	item, err = request.SafeMake("i")
	assert.Nil(t, err)
	assert.Equal(t, &mockItem{}, item.(*mockItem))

	// should be able to create an item from a subcontext
	item, err = subrequest.SafeMake("item")
	assert.Nil(t, err)
	assert.Equal(t, &mockItem{}, item.(*mockItem))
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
		assert.Nil(t, recover(), "SafeMake should not panic")
	}()

	_, err := app.SafeMake("item")
	assert.NotNil(t, err, "should not panic but not be able to create the item either")
}

func TestSingletonSafeMake(t *testing.T) {
	cm, _ := NewContextManager("app", "request", "subrequest")

	cm.Maker(Maker{
		Name:      "item",
		Scope:     "request",
		Singleton: true,
		Make: func(ctx *Context) (interface{}, error) {
			return &mockItem{}, nil
		},
	})

	app, _ := cm.Context("app")
	request, _ := app.SubContext("request")
	subrequest, _ := request.SubContext("subrequest")

	var item, item2 interface{}
	var err error

	// should be able to create the item from the request scope
	item, err = request.SafeMake("item")
	assert.Nil(t, err)
	assert.Equal(t, &mockItem{}, item.(*mockItem))

	item.(*mockItem).Closed = true

	// should retrieve the item every time, even with different parameters
	item2, err = request.SafeMake("item")
	assert.Nil(t, err)
	assert.True(t, item == item2)

	// should be able to retrieve the same item from a subcontext
	item2, err = subrequest.SafeMake("item")
	assert.Nil(t, err)
	assert.True(t, item == item2)
}

func TestNestedDependencies(t *testing.T) {
	cm, _ := NewContextManager("app", "request", "subrequest")

	appItem := &mockItem{}

	cm.Instance(Instance{Name: "appItem", Item: appItem})

	cm.Maker(Maker{
		Name:  "requestItem",
		Scope: "request",
		Make: func(ctx *Context) (interface{}, error) {
			return &nestedMockItem{ctx.Make("appItem").(*mockItem)}, nil
		},
	})

	request, _ := cm.Context("request")

	nestedItem := request.Make("requestItem").(*nestedMockItem)
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

	item := request.Make("item").(int)
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

func TestClose(t *testing.T) {
	cm, _ := NewContextManager("app", "request", "subrequest")

	cm.Maker(Maker{
		Name:  "item",
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

	request, _ := cm.Context("request")

	i1 := request.Make("item").(*mockItem)
	i2 := request.Make("item").(*mockItem)

	assert.False(t, i1.Closed)
	assert.False(t, i2.Closed)

	request.Close(i1)

	assert.True(t, i1.Closed)
	assert.False(t, i2.Closed)
}

func TestCloseFromParent(t *testing.T) {
	cm, _ := NewContextManager("app", "request", "subrequest")

	cm.Maker(Maker{
		Name:  "item",
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

	app, _ := cm.Context("app")
	request, _ := app.SubContext("request")

	i1 := request.Make("item").(*mockItem)
	i2 := request.Make("item").(*mockItem)

	assert.False(t, i1.Closed)
	assert.False(t, i2.Closed)

	app.Close(i1)

	assert.True(t, i1.Closed)
	assert.False(t, i2.Closed)
}

func TestCloseFromChild(t *testing.T) {
	cm, _ := NewContextManager("app", "request", "subrequest")

	cm.Maker(Maker{
		Name:  "item",
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

	subrequest, _ := cm.Context("subrequest")

	i1 := subrequest.Make("item").(*mockItem)
	i2 := subrequest.Make("item").(*mockItem)

	assert.False(t, i1.Closed)
	assert.False(t, i2.Closed)

	subrequest.Close(i1)

	assert.True(t, i1.Closed)
	assert.False(t, i2.Closed)
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

	item, _ := app.SafeMake("item")
	app.Close(item)
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

	app, _ := cm.Context("app")
	request, _ := app.SubContext("request")
	subrequest, _ := request.SubContext("subrequest")

	var err error

	i1 := app.Make("i1").(*mockItem)
	i2 := request.Make("i2").(*mockItem)
	i3 := subrequest.Make("i3").(*mockItem)

	request.Delete()

	assert.False(t, i1.Closed)
	assert.True(t, i2.Closed)
	assert.True(t, i3.Closed)

	assert.Nil(t, request.Parent(), "should have removed request parent")
	assert.Nil(t, subrequest.Parent(), "should have removed subrequest parent")

	_, err = app.SafeMake("i1")
	assert.Nil(t, err, "should still be able to create item from the app context")

	_, err = request.SafeMake("i2")
	assert.NotNil(t, err, "should not be able to create item from the closed request context")

	_, err = subrequest.SafeMake("i3")
	assert.NotNil(t, err, "should not be able to create item from the closed subrequest context")

	_, err = request.SubContext("subrequest")
	assert.NotNil(t, err, "should not be able to create a subcontext from a closed context")
}

func TestIfDeleteRemovesSingletonsCorrectly(t *testing.T) {
	cm, _ := NewContextManager("app", "request")

	cm.Maker(Maker{
		Name:      "item",
		Scope:     "app",
		Singleton: true,
		Make: func(ctx *Context) (interface{}, error) {
			return &mockItem{}, nil
		},
		Close: func(item interface{}) {
			item.(*mockItem).Closed = true
		},
	})

	app, _ := cm.Context("app")
	request, _ := app.SubContext("request")

	item := request.Make("item").(*mockItem)

	assert.Len(t, app.items, 1, "singleton should be saved in app")
	assert.Len(t, request.items, 0, "singleton should be saved in request")

	request.Delete()

	assert.False(t, item.Closed)
	assert.Len(t, app.items, 1, "singleton should still exist in app")

	app.Delete()

	assert.True(t, item.Closed)
	assert.Len(t, app.items, 0, "singleton should not exist in app anymore")
}

func TestIfDeleteRemovesOneShotItemsCorrectly(t *testing.T) {
	cm, _ := NewContextManager("app", "request")

	cm.Maker(Maker{
		Name:      "item",
		Scope:     "app",
		Singleton: false,
		Make: func(ctx *Context) (interface{}, error) {
			return &mockItem{}, nil
		},
		Close: func(item interface{}) {
			item.(*mockItem).Closed = true
		},
	})

	app, _ := cm.Context("app")
	request, _ := app.SubContext("request")

	item := request.Make("item").(*mockItem)

	assert.Len(t, app.items, 0, "item should not be saved in app")
	assert.Len(t, request.items, 1, "item should be saved in request")

	request.Delete()

	assert.True(t, item.Closed)
	assert.Len(t, request.items, 0, "item should not exist in request anymore")
}

func TestRace(t *testing.T) {
	cm, _ := NewContextManager("app", "request", "subrequest")

	cm.Instance(Instance{
		Name: "instance",
		Item: &mockItem{},
	})

	cm.Maker(Maker{
		Name:      "singleton",
		Scope:     "app",
		Singleton: true,
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
			return &nestedMockItem{ctx.Make("item").(*mockItem)}, nil
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

			request.Make("singleton")
			request.Make("item")
			request.Make("instance")
			request.Make("nested")

			go func() {
				subrequest, _ := app.SubContext("subrequest")
				defer subrequest.Delete()

				subrequest.Make("singleton")
				subrequest.Make("item")
				subrequest.Make("instance")
				subrequest.Make("nested")
				subrequest.Make("singleton")
				subrequest.Make("item")
				subrequest.Make("instance")
				subrequest.Make("nested")
			}()

			request.Make("singleton")
			request.Make("item")
			request.Make("instance")
			request.Make("nested")
		}()
	}
}

func TestUnhashableItem(t *testing.T) {
	cm, _ := NewContextManager("app")

	cm.Maker(Maker{
		Name:  "item",
		Scope: "app",
		Make: func(ctx *Context) (interface{}, error) {
			return map[string]string{}, nil
		},
	})

	app, _ := cm.Context("app")

	m, ok := app.Make("item").(map[string]string)
	assert.True(t, ok)
	assert.NotNil(t, m)
}
