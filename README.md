# DI

Dependency injection container for golang.

If you don't know what a dependency injection container is, you may want to read this article before :

http://fabien.potencier.org/do-you-need-a-dependency-injection-container.html


## Main components

First let's define the main components of this library :

**Context** : a Context is a dependency injection container. Contexts should be used to retrieve items. They contain all previously built items.

**ContextManager** : a ContextManager is used to store the definitions of the items you want to create.

**Scope** : contexts are not  necessarily independent. For example, a context can be attached to your whole application whereas others can be attached to an http request. In this case `application` and `request` are two scopes. The `request` contexts are sub-contexts of the `application` context. They are isolated at their level but share the same `application` context.


## Item definitions

### Instance

Instances are the simplest way to register an item. It works like a map. An Instance has a name (and may have one or more aliases). This name corresponds to an item and will be used to retrieve it later on.

You can register the Instance in a ContextManager :

```go
// app is the only scope of the ContextManager
cm, _ := NewContextManager("app")

// register &MyItem{} under the name item-name
cm.Instance(di.Instance{
    Name: "item-name",
    Aliases: []string{"item-alias"},
    Item: &MyItem{},
})
```

And then you can retrieve it from any Context created with this ContextManager :

```go
// get an app Context to retrieve the Instance
context, _ := cm.Context("app")
item := context.Make("item-name").(*MyItem)
```

You will retrieve the exact same item every time you call the `Make` method.


## Maker

In an Instance, you directly register an item. In a Maker you define how to build and close an item.

```go
cm, _ := NewContextManager("app", "request")

cm.Maker(di.Maker{
    Name: "item-name",
    Aliases: []string{"item-alias"},
    Scope: "request",
    Singleton: false,
    // define how to make the item
    Make: func(c *Context, params ...interface{}) (interface{}, error) {
        return &MyItem{}, nil
    },
    // define how to close it
    Close: func(item interface{}) {
        item.(*MyItem).Close()
    },
})

context, _ := cm.Context("app")
item := context.Make("item-name").(*MyItem)
```

Makers belong to a scope and can only be created from a context with this scope or a sub-scope. This means an `app` context can not create an item registered in the `request` scope. To get this item you need to create a `request` context from the `app` context.

Makers can be singletons. In this case the `Make` function defined in the Maker will only be called the first time you try to retrieve the item. The created item is then stored in the context and will be returned each time the `Make` method of the context is called.


## Create and close items

### Create an item

There are two functions to create an item.

```go
// Make returns an interface that can be cast afterward.
// If the item can not be made, nil is returned.
itemInterface := context.Make("my-item")
item := itemInterface.(*MyItem)

// SafeMake returns an interface, but also an error if something went wrong.
// It can be used to find why an item could not be made.
itemInterface, err := context.SafeMake("my-item")
```

You can pass parameters to the `Make` function :

```go
cm, _ := NewContextManager("app", "request")
cm.Maker(di.Maker{
    Name: "item-name",
    Scope: "request",
    Make: func(c *Context, params ...interface{}) (interface{}, error) {
        if len(params) == 0 {
          return nil, errors.New("require a parameter")
        }
        return &MyItem{params[0].(string)}, nil
    },
})
context, _ := cm.Context("app")
item := context.Make("item-name", "my-parameter").(*MyItem)
```

Be careful with parameters and singletons. The parameters of the first call will be used every time.


### Close an item

To close an item you can use the `Close` method :

```go
item := context.Make("my-item").(*ItemThatMustBeClosed)
// and then later
context.Close(item)
```

But you can also close all the items created with a context by deleting it when you have finished to use it :

```go
item1 := context.Make("item1").(*ItemThatMustBeClosed)
item2 := context.Make("item2").(*ItemThatMustBeClosed)
// and then later
context.Delete()
```

Contexts have a reference to all their children, so it's really important to call the `Delete` method once you've finished using a Context to free its memory. That's why almost every time, you won't need to close each item individually. You just have to delete the Context :

```go
cm, _ := di.NewContextManager("app")
app, _ := cm.Context("app")
defer app.Delete()
// and here you can use the context
```


## Scopes

The scopes are defined when the ContextManager is created. Then you can create contexts from this ContextManager.

```go
// subrequest is a sub-scope of request that is a sub-scope of app
cm, _ := di.NewContextManager("app", "request", "subrequest")

// you can create an app context from the ContextManager
app, _ := cm.Context("app")

// but you can also directly create a request context.
// This will create another app context. So request is not a sub-context of the app context above.
request, _ := cm.Context("request")
request.Parent() == app // false

// you can create a sub-context from a context
subrequest, _ := request.SubContext("subrequest")
subrequest.Parent() == request // true
```

## Database example

Here is an example that shows how DI can be used to get a database connection in your application.

```go
package main

import (
    "database/sql"
    "net/http"

    "github.com/sarulabs/di"

    _ "github.com/go-sql-driver/mysql"
)

func main() {
    cm := createContextManager()

    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        // Create a request and delete it once it has been handled.
        // It's very important to delete the request to free its memory.
        request, _ := cm.Context("request")
        defer request.Delete()
        handler(w, r, request)
    })

    http.ListenAndServe(":8080", nil)
}

func createContextManager() *di.ContextManager {
    cm, _ := di.NewContextManager("app", "request")

    // register the database configuration
    cm.Instance(di.Instance{
        Name: "dsn",
        Item: "user:password@/dbname",
    })

    // register the connection
    cm.Maker(di.Maker{
        Name:  "mysql",
        Scope: "request",

        Make: func(c *di.Context, params ...interface{}) (interface{}, error) {
            dsn := c.Make("dsn").(string)
            return sql.Open("mysql", dsn)
        },

        Close: func(item interface{}) {
            item.(*sql.DB).Close()
        },
    })

    return cm
}

func handler(w http.ResponseWriter, r *http.Request, c *di.Context) {
    // Retrieve the conection
    db := c.Make("mysql").(*sql.DB)

    var variable, value string

    row := db.QueryRow("SHOW STATUS WHERE `variable_name` = 'Threads_connected'")
    row.Scan(&variable, &value)

    // Display how many connection are opened.
    // As the connection is closed when the request is deleted,
    // the number should not increase after each request.
    w.Write([]byte(variable + " : " + value))
}
```
