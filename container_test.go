package di

import "testing"

type Item struct {
	Attr string
}

func TestFactories(t *testing.T) {
	c := NewContainer()

	noCall := 0

	c.Factory("item", func() interface{} {
		noCall++
		return Item{"new"}
	})

	var item1, item2 Item

	c.Make("item", &item1)
	c.Make("item", &item2)

	item1.Attr = "updated"

	if item1.Attr != "updated" || item2.Attr != "new" {
		t.Error("factories do not return distinct items")
	}

	if noCall != 2 {
		t.Error("factory is not called each time")
	}
}

func TestSingletons(t *testing.T) {
	c := NewContainer()

	noCall := 0

	c.Singleton("item", func() interface{} {
		noCall++
		return &Item{"new"}
	})

	var item1, item2 *Item

	c.Make("item", &item1)
	c.Make("item", &item2)

	item1.Attr = "updated"

	if item1.Attr != "updated" || item2.Attr != "updated" {
		t.Error("singletons do not return the same item")
	}

	if noCall != 1 {
		t.Error("factory is not called only once")
	}
}

func TestInstances(t *testing.T) {
	c := NewContainer()

	item := &Item{"new"}

	c.Instance("item", item)

	var item1, item2 *Item

	c.Make("item", &item1)
	c.Make("item", &item2)

	item1.Attr = "updated"

	if item1.Attr != "updated" || item2.Attr != "updated" {
		t.Error("instances do not return the same item")
	}
}

func TestResolveNames(t *testing.T) {
	c := NewContainer()

	c.Instance("item", &Item{"new"})
	c.Alias("alias1", "item")
	c.Alias("alias2", "item")
	c.Alias("alias3", "alias1")

	if c.ResolveName("item") != "item" ||
		c.ResolveName("alias1") != "item" ||
		c.ResolveName("alias2") != "item" ||
		c.ResolveName("alias3") != "item" {
		t.Error("could not resolve existing entry")
	}

	if c.ResolveName("notSet") != "" {
		t.Error("resolved a not existing entry")
	}
}

func TestAliases(t *testing.T) {
	c := NewContainer()

	c.Instance("item", &Item{"new"})
	c.Alias("alias1", "item")
	c.Alias("alias2", "item")
	c.Alias("alias3", "alias1")

	var item1, item2, item3, item4 *Item

	c.Make("item", &item1)
	c.Make("alias1", &item2)
	c.Make("alias2", &item3)
	c.Make("alias3", &item4)

	item1.Attr = "updated"

	if item1.Attr != "updated" || item2.Attr != "updated" || item3.Attr != "updated" || item4.Attr != "updated" {
		t.Error("aliases do not return the same item")
	}
}

func TestMixed(t *testing.T) {
	c := NewContainer()

	c.Factory("string", func() interface{} {
		return "A"
	})
	c.Singleton("int", func() interface{} {
		return 1
	})
	c.Instance("item", Item{"new"})
	c.Alias("alias", "string")

	var (
		s1, s2 string
		i      int
		item   Item
	)

	c.Make("string", &s1)
	c.Make("int", &i)
	c.Make("item", &item)
	c.Make("alias", &s2)

	if s1 != "A" || i != 1 || item.Attr != "new" || s2 != "A" {
		t.Error("mixed test do not return the right items")
	}
}

func TestBindingError(t *testing.T) {
	var (
		err  error
		item string
	)

	c := NewContainer()
	c.Instance("item", "new")
	c.Instance("itemB", "newB")

	// Factory
	err = c.Factory("item", func() interface{} {
		return "updated"
	})

	if err == nil {
		t.Error("Factory should return an error if an entry already exist")
	}

	c.Make("item", &item)

	if item != "new" {
		t.Error("Factory should not modify the container if an error is returned")
	}

	// Singleton
	err = c.Singleton("item", func() interface{} {
		return "updated"
	})

	if err == nil {
		t.Error("Singleton should return an error if an entry already exist")
	}

	c.Make("item", &item)

	if item != "new" {
		t.Error("Singleton should not modify the container if an error is returned")
	}

	// Instance
	err = c.Instance("item", "updated")

	if err == nil {
		t.Error("Instance should return an error if an entry already exist")
	}

	c.Make("item", &item)

	if item != "new" {
		t.Error("Instance should not modify the container if an error is returned")
	}

	// Alias
	err = c.Alias("item", "itemB")

	if err == nil {
		t.Error("Alias should return an error if an entry already exist")
	}

	c.Make("item", &item)

	if item != "new" {
		t.Error("Instance should not modify the container if an error is returned")
	}
}

func TestMakeErrors(t *testing.T) {
	var (
		item      Item
		itemPtr   *Item
		itemPtr2  *Item
		container *Container
		s         string
	)

	defer func() {
		if r := recover(); r != nil {
			t.Error("Make should not panic")
		}
	}()

	c := NewContainer()
	c.Factory("item", func() interface{} {
		return &Item{"factory"}
	})

	err := c.Make("notSet", &item)
	if err == nil {
		t.Error("Make should return an error if you ask for something that does not exist")
	}

	typeError := "if destination is not a pointer to the entry type, Make should return an error and not modify the destination"

	// Here entry type is *Item, Make should receive **Item : ok
	if c.Make("item", &itemPtr) != nil || itemPtr.Attr != "factory" {
		t.Error("Make does not work")
	}

	// Misuse
	item = Item{"new"}
	itemPtr = &Item{"new"}
	container = NewContainer()
	s = "string"

	if c.Make("item", item) == nil || item.Attr != "new" {
		t.Error(typeError)
	}
	if c.Make("item", &item) == nil || item.Attr != "new" {
		t.Error(typeError)
	}
	if c.Make("item", itemPtr) == nil || itemPtr.Attr != "new" {
		t.Error(typeError)
	}
	if c.Make("item", itemPtr2) == nil {
		t.Error(typeError)
	}
	if c.Make("item", &s) == nil || s != "string" {
		t.Error(typeError)
	}
	if c.Make("item", &container) == nil {
		t.Error(typeError)
	}
	if c.Make("item", nil) == nil {
		t.Error(typeError)
	}
	if c.Make("item", "item") == nil {
		t.Error(typeError)
	}
}
