# DI Container V1

A new version is available on master.

Dependency injection container for golang.

You can store all the things you want in a container, like you would do with a map. But the container is not a simple map. You can also register functions to create items. The items will only be created when you try to retrieve them.


## Examples

You can use the container like a map:

~~~go
type Item struct {
	Color string
}

c := di.NewContainer()
i := &Item{"red"}

// item registration
c.Instance("red-item", i)

// later in your code you can get it back
var redItem *Item
c.Make("red-item", &redItem)
~~~

You can also build the item only when `Make` is called.

~~~go
c.Factory("blue-item", func() interface{} {
	return &Item{"blue"}
})

// at this point no blue item has been created
var blueItem *Item
c.Make("blue-item", &blueItem)
~~~

If you register your item with `Factory`, a new item will be created each time `Make` is called. Sometimes you want to retrieve the exact same item as it was the case in the first example. To do that you should register your item with `Singleton`:

~~~go
c.Singleton("green-item", func() interface{} {
	return &Item{"green"}
})

// the item will be created when Make is called for the first time
// greenItem1 and greenItem2 are exactly identical
var greenItem1, greenItem2 *Item
c.Make("green-item", &greenItem1)
c.Make("green-item", &greenItem2)
~~~

You can also use aliases if you want:

~~~go
c.Alias("best-item", "green-item")

var greenItem *Item
c.Make("best-item", &greenItem)
~~~
