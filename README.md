# DI

[![Build Status](https://travis-ci.org/sarulabs/di.svg?branch=master)](https://travis-ci.org/sarulabs/di)
[![GoDoc](https://godoc.org/github.com/sarulabs/di?status.svg)](http://godoc.org/github.com/sarulabs/di)
[![Coverage](http://gocover.io/_badge/github.com/sarulabs/di)](http://gocover.io/github.com/sarulabs/di)
[![codebeat](https://codebeat.co/badges/d6095401-7dcf-4f63-ab75-7fac5c6aa898)](https://codebeat.co/projects/github-com-sarulabs-di)
[![goreport](https://goreportcard.com/badge/github.com/sarulabs/di)](https://goreportcard.com/report/github.com/sarulabs/di)



Dependency injection container for golang.

If you don't know what a dependency injection container is, you may want to read this article before :

http://fabien.potencier.org/do-you-need-a-dependency-injection-container.html


## Basic usage

### Object definition

A Definition contains at least the `Name` of the object and a `Build` function to create the object.

```go
di.Definition{
    Name: "my-object",
    Build: func(ctx di.Context) (interface{}, error) {
        return &MyObject{}, nil
    },
}
```

The definition can be added to a Builder with the `AddDefinition` method :

```go
builder := di.NewBuilder()

builder.AddDefinition(di.Definition{
    Name: "my-object",
    Build: func(ctx di.Context) (interface{}, error) {
        return &MyObject{}, nil
    },
})
```


### Object retrieval

Once the definitions have been added to a Builder, the Builder can generate a Context. This Context will provide the objects defined in the Builder.

```go
ctx := builder.Build()
obj := ctx.Get("my-object").(*MyObject)
```

The objects in a Context are singletons. You will retrieve the exact same object every time you call the `Get` method on the same Context. The `Build` function will only be called once.


### Nested definition

The `Build` function can also call the `Get` method of the Context. That allows to build objects that depend on other objects defined in the Context.

```go
di.Definition{
    Name: "nested-object",
    Build: func(ctx di.Context) (interface{}, error) {
        return &MyNestedObject{
            Object: ctx.Get("my-object").(*MyObject),
        }, nil
    },
}
```

You can not create a cycle in the definitions (A needs B and B needs A). If that happens, an error will be returned at the time of the creation of the object.


## Scopes

Definitions can also have a scope :

```go
di.Definition{
    Name: "my-object",
    Scope: di.Request,
    Build: func(ctx di.Context) (interface{}, error) {
        return &MyObject{}, nil
    },
}
```

The scopes are defined when the Builder is created :

```go
builder, err := di.NewBuilder("app", "request")
```

Scopes are defined from the wider to the narrower. If no scope is given to `NewBuilder`, it is created with the three default scopes : `di.App`, `di.Request` and `di.SubRequest`. These scopes should be enough almost all the time.

Contexts created by the Builder belongs to one of these scopes. A Context may have a parent with a wider scope and children with a narrower scope. A Context is only able to build objects from its own scope, but it can retrieve objects with a wider scope from its parent Context.

If a Definition does not have a scope, the wider scope will be used.

```go
// Create a Builder with the default scopes.
builder, _ := di.NewBuilder()

// Define an object in the App scope.
builder.AddDefinition(di.Definition{
    Name: "app-object",
    Scope: di.App,
    Build: func(ctx di.Context) (interface{}, error) {
        return &MyObject{}, nil
    },
})

// Define an object in the Request scope.
builder.AddDefinition(di.Definition{
    Name: "request-object",
    Scope: di.Request,
    Build: func(ctx di.Context) (interface{}, error) {
        return &MyObject{}, nil
    },
})

// Build creates a Context in the wider scope.
app := builder.Build()

// app Context can get children in the Request scope.
req1, _ := app.SubContext()
req2, _ := app.SubContext()

// app-object can be retrieved from the three contexts.
// The retrieved objects are the same : o1 == o2 == o3.
// The object is stored in the app context.
o1 := app.Get("app-object").(*MyObject)
o2 := req1.Get("app-object").(*MyObject)
o3 := req2.Get("app-object").(*MyObject)

// request-object can only be retrieved from req1 and req2.
// The retrieved objects are not the same : o4 != o5.
o4 := req1.Get("request-object").(*MyObject)
o5 := req2.Get("request-object").(*MyObject)
```


## Context deletion

A definition can also have a `Close` function.

```go
di.Definition{
    Name: "my-object",
    Scope: di.App,
    Build: func(ctx di.Context) (interface{}, error) {
        return &MyObject{}, nil
    },
    Close: func(obj interface{}) {
        obj.(*MyObject).Close()
    }
}
```

This function is called when the `Delete` method is called on a Context.

```go
// Create the context.
app := builder.Build()

// Retrieve an object.
obj := app.Get("my-object").(*MyObject)

// Delete the Context, the Close function will be called on obj.
app.Delete()
```

Delete closes all the objects stored in the Context. This means objects with the same scope created by this Context or one of its children. It will also call the Delete method on all the Context children. Once the Delete method has been called, the Context becomes unusable.

The `database example` at the end of this documentation is a good example of how you can use Delete.


## Define an already built object

The Builder `Set` method is a shortcut to define an already built object in the wider scope.

```go
builder.Set("my-object", object)
```

is the same as :

```go
builder.AddDefinition(di.Definition{
    Name: "my-object",
    Scope: di.App,
    Build: func(ctx di.Context) (interface{}, error) {
        return object, nil
    },
})
```

It can be useful to define you application parameters.


## Methods to retrieve an object

### Get

Get returns an interface that can be cast afterward. If the item can't be created, nil is returned.

```go
// It can be used safely.
objectInterface := ctx.Get("my-object")
object, ok := objectInterface.(*MyObject)

// Or if you don't care about panicking...
object := ctx.Get("my-object").(*MyObject)
```


### SafeGet

Get is fine to retrieve an object, but it does not give you any information if something goes wrong. SafeGet works like Get but also returns an error. It can be used to find why an object could not be created.

```go
objectInterface, err := ctx.SafeGet("my-object")
```


### Fill

The third method to retrieve an object is Fill. It returns an error if something goes wrong like SafeGet, but it may be more practical in certain situations.

```go
var object *MyObject
err = ctx.Fill("my-object", &MyObject)
```


## Nasty retrieval

The previous methods can retrieve an object defined in the same scope or a wider one. If you need an object that is defined in a narrower scope, you need to create a sub-context to retrieve it. It is logical but not always very practical.

`NastyGet`, `NastySafeGet` and `NastyFill` work like `Get`, `SafeGet`and `Fill` but can retrieve objects defined in a narrower scope. To do so they generate sub-contexts that are not accessible. To remove these contexts, you have to call the `Clean` method on the Context.

```go
builder, _ := di.NewBuilder()

builder.AddDefinition(di.Definition{
    Name: "request-object",
    Scope: di.Request,
    Build: func(ctx di.Context) (interface{}, error) {
        return &MyObject{}, nil
    },
    Close: func(obj interface{}) {
        obj.(*MyObject).Close()
    }
})

app := builder.Build()

// app can retrieve a request-object with nasty methods.
obj := req1.NastyGet("request-object").(*MyObject)

// Once the objects created with nasty methods are no longer used,
// you can call the Clean method. In this case, the Close function
// will be called on the object.
app.Clean()
```


## Logger

If a Logger is set in the Builder when the Context is created, it will be used to log errors that might happen when an object is retrieved or closed. It is particularly useful if you use the `Get` retrieval method that does not return an error.

```go
builder, _ := di.NewBuilder()
builder.Logger = di.BasicLogger{}
```


## Panic in Build and Close functions

Panic in Build and Close functions of a Definition are recovered. In particular that allows you to use the `Get` method in a Build function.


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
    app := createApp()

    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        // Create a request and delete it once it has been handled.
        // Deleting the request will close the database connection.
        request, _ := app.SubContext()
        defer request.Delete()
        handler(w, r, request)
    })

    http.ListenAndServe(":8080", nil)
}

func createApp() di.Context {
    builder, _ := di.NewBuilder()

    // Define the database configuration.
    builder.Set("dsn", "user:password@/dbname")

    // Define the connection in the Request scope.
    // Each request will use a different connection.
    builder.AddDefinition(di.Definition{
        Name:  "mysql",
        Scope: di.Request,
        Build: func(ctx di.Context) (interface{}, error) {
            dsn := ctx.Get("dsn").(string)
            return sql.Open("mysql", dsn)
        },
        Close: func(obj interface{}) {
            obj.(*sql.DB).Close()
        },
    })

    // Returns the app Context.
    return builder.Build()
}

func handler(w http.ResponseWriter, r *http.Request, ctx di.Context) {
    // Retrieve the connection.
    db := ctx.Get("mysql").(*sql.DB)

    var variable, value string

    row := db.QueryRow("SHOW STATUS WHERE `variable_name` = 'Threads_connected'")
    row.Scan(&variable, &value)

    // Display how many connection are opened.
    // As the connection is closed when the request is deleted,
    // the number should not increase after each request.
    w.Write([]byte(variable + " : " + value))
}
```
