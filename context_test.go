package di

import (
	"errors"
	"sync"
	"testing"
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

	if scope := app.Scope(); scope != "app" {
		t.Errorf("app should belong to the app scope instead of `%s`", scope)
	}
	if scope := subrequest.Scope(); scope != "subrequest" {
		t.Errorf("subrequest should belong to the subrequest scope instead of `%s`", scope)
	}
}

func TestContextParentScopes(t *testing.T) {
	cm, _ := NewContextManager("app", "request", "subrequest")
	app, _ := cm.Context("app")
	subrequest, _ := cm.Context("subrequest")

	if scopes := app.ParentScopes(); len(scopes) != 0 {
		t.Errorf("app should not have any parent scopes, has `%v`", scopes)
	}
	if scopes := subrequest.ParentScopes(); len(scopes) != 2 || scopes[0] != "app" || scopes[1] != "request" {
		t.Errorf("subrequest parent scopes are wrong, `%v`", scopes)
	}
}

func TestContextSubScopes(t *testing.T) {
	cm, _ := NewContextManager("app", "request", "subrequest")
	app, _ := cm.Context("app")
	subrequest, _ := cm.Context("subrequest")

	if scopes := app.SubScopes(); len(scopes) != 2 || scopes[0] != "request" || scopes[1] != "subrequest" {
		t.Errorf("app parent scopes are wrong, `%v`", scopes)
	}
	if scopes := subrequest.SubScopes(); len(scopes) != 0 {
		t.Errorf("subrequest should not have any subscopes, has `%v`", scopes)
	}
}

func TestContextHasSubScope(t *testing.T) {
	cm, _ := NewContextManager("app", "request", "subrequest")
	app, _ := cm.Context("app")
	subrequest, _ := cm.Context("subrequest")

	if app.HasSubScope("app") {
		t.Error("app should not have app as a subscope")
	}
	if !app.HasSubScope("request") {
		t.Error("app should have request as a subscope")
	}
	if !app.HasSubScope("subrequest") {
		t.Error("app should have subrequest as a subscope")
	}
	if app.HasSubScope("other") {
		t.Error("app should not have other as a subscope")
	}

	if subrequest.HasSubScope("app") {
		t.Error("subrequest should not have app as a subscope")
	}
	if subrequest.HasSubScope("request") {
		t.Error("subrequest should not have request as a subscope")
	}
	if subrequest.HasSubScope("subrequest") {
		t.Error("subrequest should not have subrequest as a subscope")
	}
	if subrequest.HasSubScope("other") {
		t.Error("subrequest should not have other as a subscope")
	}
}

func TestContextParentWithScope(t *testing.T) {
	cm, _ := NewContextManager("app", "request", "subrequest")
	app, _ := cm.Context("app")
	request, _ := app.SubContext("request")
	subrequest, _ := request.SubContext("subrequest")

	if context := request.ParentWithScope("app"); context != app {
		t.Errorf("wrong request Context retrieved, %+v", context)
	}
	if context := subrequest.ParentWithScope("app"); context != app {
		t.Errorf("wrong app Context retrieved, %+v", context)
	}
	if context := subrequest.ParentWithScope("request"); context != request {
		t.Errorf("wrong request Context retrieved, %+v", context)
	}

	if context := app.ParentWithScope("undefined"); context != nil {
		t.Errorf("should not be able to retrieve an undefined Context, %+v", context)
	}
	if context := app.ParentWithScope("request"); context != nil {
		t.Errorf("should not be able to retrieve request Context, %+v", context)
	}
}

func TestSubContextCreation(t *testing.T) {
	cm, _ := NewContextManager("app", "request", "subrequest")
	request, _ := cm.Context("request")

	if _, err := request.SubContext("app"); err == nil {
		t.Error("should not be able to create a subcontext with a parent scope")
	}
	if _, err := request.SubContext("request"); err == nil {
		t.Error("should not be able to create a subcontext with the same scope")
	}
	if _, err := request.SubContext("undefined"); err == nil {
		t.Error("should not be able to create a subcontext with an undefined scope")
	}

	subrequest, err := request.SubContext("subrequest")
	if err != nil {
		t.Errorf("should be able to create a subrequest Context, error = %s", err)
	}
	if subrequest.Scope() != "subrequest" || subrequest.Parent() != request {
		t.Errorf("the subrequest is not well defined, %+v", subrequest)
	}

	subrequest2, _ := request.SubContext("subrequest")
	if subrequest == subrequest2 {
		t.Error("should not create the same subrequest twice")
	}
}

func TestInstanceSafeMake(t *testing.T) {
	cm, _ := NewContextManager("app", "request", "subrequest")

	one := &mockItem{}
	two := &mockItem{}

	cm.Instance(Instance{Name: "i1", Aliases: []string{"a1"}, Item: one})
	cm.Instance(Instance{Name: "i2", Aliases: []string{"a2"}, Item: two})

	app, _ := cm.Context("app")
	request, _ := cm.Context("request")

	if _, err := app.SafeMake("undefined"); err == nil {
		t.Error("should not be able to create an undefined instance")
	}

	// SafeMake should work from tha app Context
	item1, err := app.SafeMake("i1")
	if err != nil {
		t.Errorf("error while retrieving i1 from app, error = `%s`", err)
	}
	if item1.(*mockItem) != one {
		t.Errorf("item i1 was not retrieved correctly, %+v is not %+v", item1.(*mockItem), one)
	}

	// SafeMake should also work from the request Context and with an alias
	item2, err := request.SafeMake("a2")
	if err != nil {
		t.Errorf("error while retrieving i2 from request, error = `%s`", err)
	}
	if item2.(*mockItem) != two {
		t.Errorf("item i2 was not retrieved correctly, %+v is not %+v", item2.(*mockItem), two)
	}
}

func TestMakerSafeMake(t *testing.T) {
	cm, _ := NewContextManager("app", "request", "subrequest")

	cm.Maker(Maker{
		Name:    "item",
		Aliases: []string{"i"},
		Scope:   "request",
		Make: func(c *Context, params ...interface{}) (interface{}, error) {
			if len(params) == 0 {
				return nil, errors.New("could not create the item")
			}
			return params[0].(int), nil
		},
	})

	app, _ := cm.Context("app")
	request, _ := app.SubContext("request")
	subrequest, _ := request.SubContext("subrequest")

	if _, err := app.SafeMake("item", 0); err == nil {
		t.Error("should not be able to create the item from the app scope")
	}
	if _, err := request.SafeMake("undefined"); err == nil {
		t.Error("should not be able to create an undefined item")
	}
	if _, err := request.SafeMake("item"); err == nil {
		t.Error("should get the error from the Make function because SafeMake was called without any parameter")
	}

	var item interface{}
	var err error

	// should be able to create the item from the request scope
	item, err = request.SafeMake("item", 10)
	if err != nil {
		t.Errorf("could not create the item, error = %s", err)
	}
	if item.(int) != 10 {
		t.Errorf("wrong item retrieved, %v", item)
	}

	// should retrieve a different item every time, it is not a singleton
	item, err = request.SafeMake("item", 20)
	if err != nil {
		t.Errorf("could not create the item, error = %s", err)
	}
	if item.(int) != 20 {
		t.Errorf("wrong item retrieved, %v", item)
	}

	// should work with an alias
	item, err = request.SafeMake("i", 30)
	if err != nil {
		t.Errorf("could not create the item, error = %s", err)
	}
	if item.(int) != 30 {
		t.Errorf("wrong item retrieved, %v", item)
	}

	// should be able to create an item from a subcontext
	item, err = subrequest.SafeMake("item", 40)
	if err != nil {
		t.Errorf("could not create the item, error = %s", err)
	}
	if item.(int) != 40 {
		t.Errorf("wrong item retrieved, %v", item)
	}
}

func TestMakePanic(t *testing.T) {
	cm, _ := NewContextManager("app")

	cm.Maker(Maker{
		Name:  "item",
		Scope: "app",
		Make: func(c *Context, params ...interface{}) (interface{}, error) {
			panic("panic in Make function")
		},
	})

	app, _ := cm.Context("app")

	defer func() {
		if r := recover(); r != nil {
			t.Error("SafeMake should not panic")
		}
	}()

	if _, err := app.SafeMake("item"); err == nil {
		t.Error("should not panic but not be able to create the item either")
	}
}

func TestSingletonSafeMake(t *testing.T) {
	cm, _ := NewContextManager("app", "request", "subrequest")

	cm.Maker(Maker{
		Name:      "item",
		Scope:     "request",
		Singleton: true,
		Make: func(c *Context, params ...interface{}) (interface{}, error) {
			if len(params) == 0 {
				return nil, errors.New("could not create the item")
			}
			return params[0].(int), nil
		},
	})

	app, _ := cm.Context("app")
	request, _ := app.SubContext("request")
	subrequest, _ := request.SubContext("subrequest")

	var item interface{}
	var err error

	if _, err := app.SafeMake("item", 0); err == nil {
		t.Error("should not be able to create the item from the app scope")
	}
	if _, err := request.SafeMake("undefined"); err == nil {
		t.Error("should not be able to create an undefined item")
	}
	if _, err := request.SafeMake("item"); err == nil {
		t.Error("should get the error from the Make function because SafeMake was called without any parameter")
	}

	// should be able to create the item from the request scope
	item, err = request.SafeMake("item", 10)
	if err != nil {
		t.Errorf("could not create the item, error = %s", err)
	}
	if item.(int) != 10 {
		t.Errorf("wrong item retrieved, %v", item)
	}

	// should retrieve the item every time, even with different parameters
	item, err = request.SafeMake("item", 20)
	if err != nil {
		t.Errorf("could not create the item, error = %s", err)
	}
	if item.(int) != 10 {
		t.Errorf("wrong item retrieved, %v", item)
	}

	// should be able to retrieve the same item from a subcontext
	item, err = subrequest.SafeMake("item", 20)
	if err != nil {
		t.Errorf("could not create the item, error = %s", err)
	}
	if item.(int) != 10 {
		t.Errorf("wrong item retrieved, %v", item)
	}
}

func TestNestedDependencies(t *testing.T) {
	cm, _ := NewContextManager("app", "request", "subrequest")

	appItem := &mockItem{}

	cm.Instance(Instance{Name: "appItem", Item: appItem})

	cm.Maker(Maker{
		Name:  "requestItem",
		Scope: "request",
		Make: func(c *Context, params ...interface{}) (interface{}, error) {
			return &nestedMockItem{c.Make("appItem").(*mockItem)}, nil
		},
	})

	request, _ := cm.Context("request")

	nestedItem := request.Make("requestItem").(*nestedMockItem)

	if nestedItem.Item != appItem {
		t.Errorf("nested item is not well defined %+v instead of %+v", nestedItem.Item, appItem)
	}
}

func TestMake(t *testing.T) {
	cm, _ := NewContextManager("app", "request", "subrequest")

	cm.Maker(Maker{
		Name:  "item",
		Scope: "request",
		Make: func(c *Context, params ...interface{}) (interface{}, error) {
			return 10, nil
		},
	})

	request, _ := cm.Context("request")

	if item := request.Make("item").(int); item != 10 {
		t.Errorf("wrong item retrieved, %d", item)
	}
}

func TestFill(t *testing.T) {
	cm, _ := NewContextManager("app")

	cm.Maker(Maker{
		Name:  "item",
		Scope: "app",
		Make: func(c *Context, params ...interface{}) (interface{}, error) {
			return params[0], nil
		},
	})

	app, _ := cm.Context("app")

	var item int
	var wrongType string

	if err := app.Fill(&wrongType, "item", 10); err == nil {
		t.Error("should have failed to fill an item with the wrong type")
	}

	if err := app.Fill(&item, "item"); err == nil {
		t.Error("should have required one parameter")
	}

	if err := app.Fill(&item, "item", 10); err != nil {
		t.Errorf("should have filled the item : %d", err)
	}

	if item != 10 {
		t.Errorf("wrong item retrieved, %d", item)
	}
}

func TestClose(t *testing.T) {
	cm, _ := NewContextManager("app", "request", "subrequest")

	cm.Maker(Maker{
		Name:  "item",
		Scope: "request",
		Make: func(c *Context, params ...interface{}) (interface{}, error) {
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

	if i1.Closed || i2.Closed {
		t.Errorf("items should not be closed when they are created `%t` `%t`", i1.Closed, i2.Closed)
	}

	request.Close(i1)

	if !i1.Closed {
		t.Error("should have closed i1")
	}
	if i2.Closed {
		t.Error("should not have closed i2")
	}
}

func TestCloseFromParent(t *testing.T) {
	cm, _ := NewContextManager("app", "request", "subrequest")

	cm.Maker(Maker{
		Name:  "item",
		Scope: "request",
		Make: func(c *Context, params ...interface{}) (interface{}, error) {
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

	if i1.Closed || i2.Closed {
		t.Errorf("items should not be closed when they are created `%t` `%t`", i1.Closed, i2.Closed)
	}

	app.Close(i1)

	if !i1.Closed {
		t.Error("should have closed i1 from a parent context")
	}
	if i2.Closed {
		t.Error("should not have closed i2 from a parent context")
	}
}

func TestCloseFromChild(t *testing.T) {
	cm, _ := NewContextManager("app", "request", "subrequest")

	cm.Maker(Maker{
		Name:  "item",
		Scope: "request",
		Make: func(c *Context, params ...interface{}) (interface{}, error) {
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

	if i1.Closed || i2.Closed {
		t.Errorf("items should not be closed when they are created `%t` `%t`", i1.Closed, i2.Closed)
	}

	subrequest.Close(i1)

	if !i1.Closed {
		t.Error("should have closed i1 from a parent context")
	}
	if i2.Closed {
		t.Error("should not have closed i2 from a parent context")
	}
}

func TestClosePanic(t *testing.T) {
	cm, _ := NewContextManager("app")

	cm.Maker(Maker{
		Name:  "item",
		Scope: "app",
		Make: func(c *Context, params ...interface{}) (interface{}, error) {
			return &mockItem{}, nil
		},
		Close: func(item interface{}) {
			panic("panic in Close function")
		},
	})

	app, _ := cm.Context("app")

	defer func() {
		if r := recover(); r != nil {
			t.Error("Close should not panic")
		}
	}()

	item, _ := app.SafeMake("item")
	app.Close(item)
}

func TestDelete(t *testing.T) {
	cm, _ := NewContextManager("app", "request", "subrequest")

	cm.Maker(Maker{
		Name:  "i1",
		Scope: "app",
		Make: func(c *Context, params ...interface{}) (interface{}, error) {
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
		Make: func(c *Context, params ...interface{}) (interface{}, error) {
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
		Make: func(c *Context, params ...interface{}) (interface{}, error) {
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

	i1 := app.Make("i1").(*mockItem)
	i2 := request.Make("i2").(*mockItem)
	i3 := subrequest.Make("i3").(*mockItem)

	request.Delete()

	if i1.Closed {
		t.Errorf("should not have closed i1")
	}
	if !i2.Closed || !i3.Closed {
		t.Errorf("should have closed i2 and i3, `%t` `%t`", i2.Closed, i3.Closed)
	}

	if request.Parent() != nil {
		t.Errorf("should have removed request parent %+v", request.Parent())
	}
	if subrequest.Parent() != nil {
		t.Errorf("should have removed subrequest parent %+v", subrequest.Parent())
	}

	if _, err := app.SafeMake("i1"); err != nil {
		t.Errorf("should still be able to create item from the app context, error = %s", err)
	}
	if _, err := request.SafeMake("i2"); err == nil {
		t.Error("should not be able to create item from the closed request context")
	}
	if _, err := subrequest.SafeMake("i3"); err == nil {
		t.Error("should not be able to create item from the closed subrequest context")
	}

	if _, err := request.SubContext("subrequest"); err == nil {
		t.Error("should not be able to create a subcontext from a closed context")
	}
}

func TestIfDeleteRemovesSingletonsCorrectly(t *testing.T) {
	cm, _ := NewContextManager("app", "request")

	cm.Maker(Maker{
		Name:      "item",
		Scope:     "app",
		Singleton: true,
		Make: func(c *Context, params ...interface{}) (interface{}, error) {
			return &mockItem{}, nil
		},
		Close: func(item interface{}) {
			item.(*mockItem).Closed = true
		},
	})

	app, _ := cm.Context("app")
	request, _ := app.SubContext("request")

	item := request.Make("item").(*mockItem)

	if len(app.items) != 1 {
		t.Error("singleton should be saved in app")
	}
	if len(request.items) != 0 {
		t.Error("singleton should not be saved in request")
	}

	request.Delete()

	if item.Closed {
		t.Error("should not have closed the singleton")
	}
	if len(app.items) != 1 {
		t.Error("singleton should still exist in app")
	}

	app.Delete()

	if !item.Closed {
		t.Error("should have closed the singleton")
	}
	if len(app.items) != 0 {
		t.Error("singleton should not exist in app anymore")
	}
}

func TestIfDeleteRemovesOneShotItemsCorrectly(t *testing.T) {
	cm, _ := NewContextManager("app", "request")

	cm.Maker(Maker{
		Name:      "item",
		Scope:     "app",
		Singleton: false,
		Make: func(c *Context, params ...interface{}) (interface{}, error) {
			return &mockItem{}, nil
		},
		Close: func(item interface{}) {
			item.(*mockItem).Closed = true
		},
	})

	app, _ := cm.Context("app")
	request, _ := app.SubContext("request")

	item := request.Make("item").(*mockItem)

	if len(app.items) != 0 {
		t.Error("item should not be saved in app")
	}
	if len(request.items) != 1 {
		t.Error("item should be saved in request")
	}

	request.Delete()

	if !item.Closed {
		t.Error("should have closed item")
	}
	if len(request.items) != 0 {
		t.Error("item should not exist in request anymore")
	}
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
		Make: func(c *Context, params ...interface{}) (interface{}, error) {
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
		Make: func(c *Context, params ...interface{}) (interface{}, error) {
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
		Make: func(c *Context, params ...interface{}) (interface{}, error) {
			return &nestedMockItem{c.Make("item").(*mockItem)}, nil
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
