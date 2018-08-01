![DI](https://raw.githubusercontent.com/sarulabs/assets/master/di/logo.png)

Dependency injection framework for go programs (golang).

DI handles the life cycle of the objects in your application. It creates them when they are needed, resolves their dependencies and closes them properly when they are no longer used.

If you do not know if DI could help improving your application, learn more about dependency injection and dependency injection containers:

- [What is a dependency injection container and why use one ?](https://www.sarulabs.com/post/2/2018-06-12/what-is-a-dependency-injection-container-and-why-use-one.html)

There is also an [Examples](#examples) section at the end of the documentation.

DI is focused on performance. It does not rely on reflection.


# Table of contents

[![Build Status](https://travis-ci.org/sarulabs/di.svg?branch=master)](https://travis-ci.org/sarulabs/di)
[![GoDoc](https://godoc.org/github.com/sarulabs/di?status.svg)](http://godoc.org/github.com/sarulabs/di)
[![Test Coverage](https://api.codeclimate.com/v1/badges/5af97cbfd6e4fe7257e3/test_coverage)](https://codeclimate.com/github/sarulabs/di/test_coverage)
[![Maintainability](https://api.codeclimate.com/v1/badges/5af97cbfd6e4fe7257e3/maintainability)](https://codeclimate.com/github/sarulabs/di/maintainability)
[![codebeat](https://codebeat.co/badges/d6095401-7dcf-4f63-ab75-7fac5c6aa898)](https://codebeat.co/projects/github-com-sarulabs-di)
[![goreport](https://goreportcard.com/badge/github.com/sarulabs/di)](https://goreportcard.com/report/github.com/sarulabs/di)

- [Basic usage](#basic-usage)
	* [Object definition](#object-definition)
	* [Object retrieval](#object-retrieval)
	* [Definitions and dependencies](#definitions-and-dependencies)
- [Scopes](#scopes)
    * [The principle](#the-principle)
    * [Scopes in practice](#scopes-in-practice)
    * [Scopes and dependencies](#scopes-and-dependencies)
- [Container deletion](#container-deletion)
- [Methods to retrieve an object](#methods-to-retrieve-an-object)
	* [Get](#get)
	* [SafeGet](#safeget)
	* [Fill](#fill)
- [Unscoped retrieval](#unscoped-retrieval)
- [Panic in Build and Close functions](#panic-in-build-and-close-functions)
- [HTTP helpers](#http-helpers)
- [Examples](#examples)
- [Migration from v1](#migration-from-v1)


# Basic usage

## Object definition

A Definition contains at least the `Name` of the object and a `Build` function to create the object.

```go
di.Def{
    Name: "my-object",
    Build: func(ctn di.Container) (interface{}, error) {
        return &MyObject{}, nil
    },
}
```

The definition can be added to a Builder with the `Add` method:

```go
builder, _ := di.NewBuilder()

builder.Add(di.Def{
    Name: "my-object",
    Build: func(ctn di.Container) (interface{}, error) {
        return &MyObject{}, nil
    },
})
```


## Object retrieval

Once the definitions have been added to a Builder, the Builder can generate a `Container`. This Container will provide the objects defined in the Builder.

```go
ctn := builder.Build() // create the container
obj := ctn.Get("my-object").(*MyObject) // retrieve the object
```

The `Get` method returns an `interface{}`. You need to cast the interface before using the object.

The objects are stored as singletons in the Container. You will retrieve the exact same object every time you call the `Get` method on the same Container. The `Build` function will only be called once.


## Definitions and dependencies

The `Build` function can also use the `Get` method of the Container. That allows to build objects that depend on other objects defined in the Container.

```go
di.Def{
    Name: "object-with-dependency",
    Build: func(ctn di.Container) (interface{}, error) {
        return &MyObjectWithDependency{
            Object: ctn.Get("my-object").(*MyObject),
        }, nil
    },
}
```

You can not create a cycle in the definitions (A needs B and B needs A). If that happens, an error will be returned at the time of the creation of the object.


# Scopes

## The principle

Definitions can also have a scope. They can be useful in request based applications, like a web application.

```go
di.Def{
    Name: "my-object",
    Scope: di.Request,
    Build: func(ctn di.Container) (interface{}, error) {
        return &MyObject{}, nil
    },
}
```

The available scopes are defined when the Builder is created:

```go
builder, err := di.NewBuilder(di.App, di.Request)
```

Scopes are defined from the more generic to the more specific (eg: `App` ≻ `Request` ≻ `SubRequest`). If no scope is given to `NewBuilder`, the Builder is created with the three default scopes: `di.App`, `di.Request` and `di.SubRequest`. These scopes should be enough almost all the time.

The containers belong to one of these scopes. A container may have a parent in a more generic scope and children in a more specific scope. The Builder generates a Container in the most generic scope. Then the Container can generate children in the next scope thanks to the `SubContainer` method.

A container is only able to build objects defined in its own scope, but it can retrieve objects in a more generic scope thanks to its parent. For example a `Request` container can retrieve an `App` object, but an `App` container can not retrieve a `Request` object.

If a Definition does not have a scope, the most generic scope will be used.

## Scopes in practice

```go
// Create a Builder with the default scopes (App, Request, SubRequest).
builder, _ := di.NewBuilder()

// Define an object in the App scope.
builder.Add(di.Def{
    Name: "app-object",
    Scope: di.App, // this line is optional, di.App is the default scope
    Build: func(ctn di.Container) (interface{}, error) {
        return &MyObject{}, nil
    },
})

// Define an object in the Request scope.
builder.Add(di.Def{
    Name: "request-object",
    Scope: di.Request,
    Build: func(ctn di.Container) (interface{}, error) {
        return &MyObject{}, nil
    },
})

// Build creates a Container in the most generic scope (App).
app := builder.Build()

// The App Container can create sub-containers in the Request scope.
req1, _ := app.SubContainer()
req2, _ := app.SubContainer()

// app-object can be retrieved from the three containers.
// The retrieved objects are the same: o1 == o2 == o3.
// The object is stored in app.
o1 := app.Get("app-object").(*MyObject)
o2 := req1.Get("app-object").(*MyObject)
o3 := req2.Get("app-object").(*MyObject)

// request-object can only be retrieved from req1 and req2.
// The retrieved objects are not the same: o4 != o5.
// o4 is stored in req1, and o5 is stored in req2.
o4 := req1.Get("request-object").(*MyObject)
o5 := req2.Get("request-object").(*MyObject)
```

More graphically, the containers could be represented like this:

<img src="https://raw.githubusercontent.com/sarulabs/assets/master/di/scopes.jpg" width="500" height="451">

The `App` container can only get the `App` object. A `Request` container or a `SubRequest` container can get either the `App` object or the `Request` object, possibly by using their parent. The objects are built and stored in containers that have the same scope. They are only created when they are requested.

## Scopes and dependencies

If an object depends on other objects defined in the container, the scopes of the dependencies must be either equal or more generic compared to the object scope.

For example the following definitions are not valid:

```go
di.Def{
    Name: "request-object",
    Scope: di.Request,
    Build: func(ctn di.Container) (interface{}, error) {
        return &MyObject{}, nil
    },
}

di.Def{
    Name: "object-with-dependency",
    Scope: di.App, // NOT ALLOWED !!! should be di.Request or di.SubRequest
    Build: func(ctn di.Container) (interface{}, error) {
        return &MyObjectWithDependency{
            Object: ctn.Get("request-object").(*MyObject),
        }, nil
    },
}
```


# Container deletion

A definition can also have a `Close` function.

```go
di.Def{
    Name: "my-object",
    Scope: di.App,
    Build: func(ctn di.Container) (interface{}, error) {
        return &MyObject{}, nil
    },
    Close: func(obj interface{}) error {
        // assuming that MyObject has a Close method that returns an error
        return obj.(*MyObject).Close() 
    },
}
```

This function is called when the `Delete` method is called on a Container.

```go
// Create the Container.
app := builder.Build()

// Retrieve an object.
obj := app.Get("my-object").(*MyObject)

// Delete the Container, the Close function will be called on obj.
app.Delete()
```

Delete closes all the objects stored in the Container. Once a Container has been deleted, it becomes unusable.

It is important to always use `Delete` even if the objects definitions do not have a `Close` function. It allows to free the memory taken by the Container.

There are actually two delete methods: `Delete` and `DeleteWithSubContainers`

`DeleteWithSubContainers` deletes the children of the Container and then the Container. It does this right away. `Delete` is a softer approach. It does not delete the children of the Container. Actually it does not delete the Container as long as it still has a child alive. So you have to call `Delete` on all the children. The parent Container will be deleted when `Delete` is called on the last child.

You probably want to use `Delete` and close the children manually. `DeleteWithSubContainers` can cause errors if the parent is deleted while its children are still used.


# Methods to retrieve an object

When a container is asked to retrieve an object, it starts by checking if the object has already been created. If it has, the container returns the already built instance of the object. Otherwise it uses the Build function of the associated definition to create the object. It returns the object, but also keeps a reference to be able to return the same instance if the object is requested again.

A container can only build objects defined in the same scope. If the container is asked to retrieve an object that belongs to a different scope. It forwards the request to its parent.

There are three methods to retrieve an object: `Get`, `SafeGet` and `Fill`.


## Get

`Get` returns an interface that can be cast afterwards. If the object can not be created, the `Get` function panics.

```go
obj := ctn.Get("my-object").(*MyObject)
```


## SafeGet

`Get` is an easy way to retrieve an object. The problem is that it can panic. If it is a problem for you, you can use `SafeGet`. Instead of panicking, it returns an error.

```go
objectInterface, err := ctn.SafeGet("my-object")
object, ok := objectInterface.(*MyObject)
```


## Fill

The third and last method to retrieve an object is `Fill`. It returns an error if something goes wrong like `SafeGet`, but it may be more practical in some situations. It uses reflection to fill the given object. Using reflection makes it is slower than `SafeGet`.

```go
var object *MyObject
err := ctn.Fill("my-object", &object)
```


# Unscoped retrieval

The previous methods can retrieve an object defined in the same scope or a more generic one. If you need an object defined in a more specific scope, you need to create a sub-container to retrieve it. For example, an `App` container can not create a `Request` object. A `Request` container should be created to retrieve the `Request` object. It is logical but not always very practical.

`UnscopedGet`, `UnscopedSafeGet` and `UnscopedFill` work like `Get`, `SafeGet` and `Fill` but can retrieve objects defined in a more generic scope. To do so, they generate sub-containers that can only be accessed internally by these three methods. To remove these containers without deleting the current container, you can call the `Clean` method.

```go
builder, _ := di.NewBuilder()

builder.Add(di.Def{
    Name: "request-object",
    Scope: di.Request,
    Build: func(ctn di.Container) (interface{}, error) {
        return &MyObject{}, nil
    },
    Close: func(obj interface{}) error {
        return obj.(*MyObject).Close()
    },
})

app := builder.Build()

// app can retrieve a request-object with unscoped methods.
obj := app.UnscopedGet("request-object").(*MyObject)

// Once the objects created with unscoped methods are no longer used,
// you can call the Clean method. In this case, the Close function
// will be called on the object.
app.Clean()
```


# Panic in Build and Close functions

Panics in `Build` and `Close` functions of a definition are recovered and converted into errors. In particular that allows you to use the `Get` method in a `Build` function.


# HTTP helpers

DI includes some elements to ease its integration in a web application.

The `HTTPMiddleware` function can be used to inject a container in an `http.Request`.

```go
// create an App container
builder, _ := NewBuilder()
builder.Add(/* some definitions */)
app := builder.Build()

handlerWithDiMiddleware := di.HTTPMiddleware(handler, app, func(msg string) {
    logger.Error(msg) // use your own logger here, it is used to log container deletion errors
})
```

For each `http.Request`, a sub-container of the `app` container is created. It is deleted at the end of the http request.

The container can be used in the handler:

```go
handler := func(w http.ResponseWriter, r *http.Request) {
    // retrieve the Request container with the C function
    ctn := di.C(r)
    obj := ctn.Get("object").(*MyObject)

    // there is a shortcut to do that
    obj := di.Get(r, "object").(*MyObject)
}
```

The handler and the middleware can panic. Do not forget to use another middleware to recover from the panic and log the errors.


# Examples

The [sarulabs/di-example](https://github.com/sarulabs/di-example) repository is a good example to understand how DI can be used in a web application.

More explanations about this repository can be found in this blog post:

- [How to write a REST API in Go with DI](https://www.sarulabs.com/post/3/2018-08-02/how-to-write-a-rest-api-in-go-with-di.html)

If you do not have time to check this repository, here is a shorter example that does not use the HTTP helpers. It does not handle the errors to be more concise.

```go
package main

import (
    "context"
    "database/sql"
    "net/http"

    "github.com/sarulabs/di"

    _ "github.com/go-sql-driver/mysql"
)

func main() {
    app := createApp()
    defer app.Delete()

    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        // Create a request and delete it once it has been handled.
        // Deleting the request will close the connection.
        request, _ := app.SubContainer()
        defer request.Delete()

        handler(w, r, request)
    })

    http.ListenAndServe(":8080", nil)
}

func createApp() di.Container {
    builder, _ := di.NewBuilder()

    builder.Add([]di.Def{
        {
            // Define the connection pool in the App scope.
            // There will be one for the whole application.
            Name:  "mysql-pool",
            Scope: di.App,
            Build: func(ctn di.Container) (interface{}, error) {
                db, err := sql.Open("mysql", "user:password@/")
                db.SetMaxOpenConns(1)
                return db, err
            },
            Close: func(obj interface{}) error {
                return obj.(*sql.DB).Close()
            },
        },
        {
            // Define the connection in the Request scope.
            // Each request will use its own connection.
            Name:  "mysql",
            Scope: di.Request,
            Build: func(ctn di.Container) (interface{}, error) {
                pool := ctn.Get("mysql-pool").(*sql.DB)
                return pool.Conn(context.Background())
            },
            Close: func(obj interface{}) error {
                return obj.(*sql.Conn).Close()
            },
        },
    }...)

    // Returns the app Container.
    return builder.Build()
}

func handler(w http.ResponseWriter, r *http.Request, ctn di.Container) {
    // Retrieve the connection.
    conn := ctn.Get("mysql").(*sql.Conn)

    var variable, value string

    row := conn.QueryRowContext(context.Background(), "SHOW STATUS WHERE `variable_name` = 'Threads_connected'")
    row.Scan(&variable, &value)

    // Display how many connections are opened.
    // As the connection is closed when the request is deleted,
    // the value should not be be higher than the number set with db.SetMaxOpenConns(1).
    w.Write([]byte(variable + ": " + value))
}
```


# Migration from v1

DI `v2` improves error handling. It should also be faster. Migrating to `v2` is highly recommended and should not be too difficult. There should not be any more changes in the API for a long time.

### Renamed elements

Some elements have been renamed.

A `Context` is now a `Container`.

The Context methods `SubContext`, `NastySafeGet`, `NastyGet`, `NastyFill` have been renamed. Their new names are `SubContainer`, `UnscopedSafeGet`, `UnscopedGet`, and `UnscopedFill`.

`Definition` is now `Def`. The `AddDefinition` of the Builder is now `Add` and can take more than one definition as parameter. Definition `Tags` have been removed.

### Errors

The `Close` function in a definition now returns an `error`.

The Container methods `Clean`, `Delete` and `DeleteWithSubContainers` also return an `error`.

### Get

The `Get` method used to return `nil` if it could not retrieve the object. Now it panics with the error.

### Logger

The `Logger` does not exist anymore. The errors are now directly handled by the retrieval functions.

```go
// remove this line if you have it
builder.Logger = ...
```

### Builder.Set

The `Set` method of the builder does not exist anymore. You should use the `Add` method and a `Def`.
