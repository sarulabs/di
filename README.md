![DI](https://raw.githubusercontent.com/sarulabs/assets/master/di/logo.png)

Dependency injection container library for go programs (golang).

DI handles the life cycle of the objects in your application. It creates them when they are needed, resolves their dependencies and closes them properly when they are no longer used.

If you do not know if DI could help improving your application, learn more about dependency injection and dependency injection containers:

- [What is a dependency injection container and why use one ?](https://www.sarulabs.com/post/2/2018-06-12/what-is-a-dependency-injection-container-and-why-use-one.html)

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
	* [Nested definition](#nested-definition)
- [Scopes](#scopes)
- [Define an already built object](#define-an-already-built-object)
- [Methods to retrieve an object](#methods-to-retrieve-an-object)
	* [Get](#get)
	* [SafeGet](#safeget)
	* [Fill](#fill)
- [Unscoped retrieval](#unscoped-retrieval)
- [Logger](#logger)
- [Panic in Build and Close functions](#panic-in-build-and-close-functions)
- [Database example](#database-example)


# Basic usage

## Object definition

A Definition contains at least the `Name` of the object and a `Build` function to create the object.

```go
di.Definition{
    Name: "my-object",
    Build: func(ctn di.Container) (interface{}, error) {
        return &MyObject{}, nil
    },
}
```

The definition can be added to a Builder with the `AddDefinition` method:

```go
builder, _ := di.NewBuilder()

builder.AddDefinition(di.Definition{
    Name: "my-object",
    Build: func(ctn di.Container) (interface{}, error) {
        return &MyObject{}, nil
    },
})
```


## Object retrieval

Once the definitions have been added to a Builder, the Builder can generate a Container. This Container will provide the objects defined in the Builder.

```go
ctn := builder.Build()
obj := ctn.Get("my-object").(*MyObject)
```

The objects in a Container are singletons. You will retrieve the exact same object every time you call the `Get` method on the same Container. The `Build` function will only be called once.


## Nested definition

The `Build` function can also call the `Get` method of the Container. That allows to build objects that depend on other objects defined in the Container.

```go
di.Definition{
    Name: "nested-object",
    Build: func(ctn di.Container) (interface{}, error) {
        return &MyNestedObject{
            Object: ctn.Get("my-object").(*MyObject),
        }, nil
    },
}
```

You can not create a cycle in the definitions (A needs B and B needs A). If that happens, an error will be returned at the time of the creation of the object.


# Scopes

Definitions can also have a scope. They can be useful in request based applications (like a web application).

```go
di.Definition{
    Name: "my-object",
    Scope: di.Request,
    Build: func(ctn di.Container) (interface{}, error) {
        return &MyObject{}, nil
    },
}
```

The scopes are defined when the Builder is created:

```go
builder, err := di.NewBuilder(di.App, di.Request)
```

Scopes are defined from the widest to the narrowest. If no scope is given to `NewBuilder`, it is created with the three default scopes: `di.App`, `di.Request` and `di.SubRequest`. These scopes should be enough almost all the time.

Containers created by the Builder belongs to one of these scopes. A Container may have a parent with a wider scope and children with a narrower scope. A Container is only able to build objects from its own scope, but it can retrieve objects with a wider scope from its parent Container.

If a Definition does not have a scope, the widest scope will be used.

```go
// Create a Builder with the default scopes.
builder, _ := di.NewBuilder()

// Define an object in the App scope.
builder.AddDefinition(di.Definition{
    Name: "app-object",
    Scope: di.App,
    Build: func(ctn di.Container) (interface{}, error) {
        return &MyObject{}, nil
    },
})

// Define an object in the Request scope.
builder.AddDefinition(di.Definition{
    Name: "request-object",
    Scope: di.Request,
    Build: func(ctn di.Container) (interface{}, error) {
        return &MyObject{}, nil
    },
})

// Build creates a Container in the widest scope (App).
app := builder.Build()

// The App Container can create sub-containers in the Request scope.
req1, _ := app.SubContainer()
req2, _ := app.SubContainer()

// app-object can be retrieved from the three containers.
// The retrieved objects are the same: o1 == o2 == o3.
// The object is stored in the App Container.
o1 := app.Get("app-object").(*MyObject)
o2 := req1.Get("app-object").(*MyObject)
o3 := req2.Get("app-object").(*MyObject)

// request-object can only be retrieved from req1 and req2.
// The retrieved objects are not the same: o4 != o5.
o4 := req1.Get("request-object").(*MyObject)
o5 := req2.Get("request-object").(*MyObject)
```


## Container deletion

A definition can also have a `Close` function.

```go
di.Definition{
    Name: "my-object",
    Scope: di.App,
    Build: func(ctn di.Container) (interface{}, error) {
        return &MyObject{}, nil
    },
    Close: func(obj interface{}) {
        obj.(*MyObject).Close()
    }
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

There are actually two delete methods: `Delete` and `DeleteWithSubContainers`

`DeleteWithSubContainers` deletes the Container children and then the Container right away. `Delete` is a softer approach. It does not delete the Container children. Actually it does not delete the Container as long as it still has a child alive. So you have to call `Delete` on all the children to delete the Container.

You probably want to use `Delete` and close the children manually. `DeleteWithSubContainers` can cause errors if the parent is deleted while its children are still used.

The database example at the end of this documentation is a good example of how you can use Delete.


# Define an already built object

The Builder `Set` method is a shortcut to define an already built object in widest scope.

```go
builder.Set("my-object", object)
```

is the same as:

```go
builder.AddDefinition(di.Definition{
    Name: "my-object",
    Scope: di.App,
    Build: func(ctn di.Container) (interface{}, error) {
        return object, nil
    },
})
```


# Methods to retrieve an object

## Get

Get returns an interface that can be cast afterward. If the item can not be created, nil is returned.

```go
// It can be used safely.
objectInterface := ctn.Get("my-object")
object, ok := objectInterface.(*MyObject)

// Or if you do not care about panicking...
object := ctn.Get("my-object").(*MyObject)
```


## SafeGet

Get is fine to retrieve an object, but it does not give you any information if something goes wrong. SafeGet works like Get but also returns an error. It can be used to find why an object could not be created.

```go
objectInterface, err := ctn.SafeGet("my-object")
```


## Fill

The third method to retrieve an object is Fill. It returns an error if something goes wrong like SafeGet, but it may be more practical in some situations. It uses reflection to fill the given object, so it is slower than SafeGet

```go
var object *MyObject
err = ctn.Fill("my-object", &MyObject)
```


# Unscoped retrieval

The previous methods can retrieve an object defined in the same scope or a wider one. If you need an object defined in a narrower scope, you need to create a sub-container to retrieve it. It is logical but not always very practical.

`UnscopedGet`, `UnscopedSafeGet` and `UnscopedFill` work like `Get`, `SafeGet` and `Fill` but can retrieve objects defined in a narrower scope. To do so they generate sub-containers that can only be accessed by these three methods. To remove these containers without deleting the current container, you can call the `Clean` method.

```go
builder, _ := di.NewBuilder()

builder.AddDefinition(di.Definition{
    Name: "request-object",
    Scope: di.Request,
    Build: func(ctn di.Container) (interface{}, error) {
        return &MyObject{}, nil
    },
    Close: func(obj interface{}) {
        obj.(*MyObject).Close()
    }
})

app := builder.Build()

// app can retrieve a request-object with unscoped methods.
obj := app.UnscopedGet("request-object").(*MyObject)

// Once the objects created with unscoped methods are no longer used,
// you can call the Clean method. In this case, the Close function
// will be called on the object.
app.Clean()
```


# Logger

If a Logger is set in the Builder when the Container is created, it will be used to log errors that might happen when an object is retrieved or closed. It is particularly useful if you use the `Get` method that does not return an error.

```go
builder, _ := di.NewBuilder()
builder.Logger = di.BasicLogger{}
```


# Panic in Build and Close functions

Panic in Build and Close functions of a Definition are recovered. In particular that allows you to use the `Get` method in a Build function.

Using `Get` in a Build function instead of `SafeGet` is way more practical. But it also can make debugging a nightmare. Be sure to define a Logger in the Builder if you want to be able to trace the errors.


# Database example

Here is an example that shows how DI can be used to get a database connection in your application.

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

	// Use a logger or you will lose the errors
	// that can happen during the creation of the objects.
	builder.Logger = &di.BasicLogger{}

	// Define the connection pool in the App scope.
	// There will be one for the whole application.
	builder.AddDefinition(di.Definition{
		Name:  "mysql-pool",
		Scope: di.App,
		Build: func(ctn di.Container) (interface{}, error) {
			db, err := sql.Open("mysql", "user:password@/")
			db.SetMaxOpenConns(1)
			return db, err
		},
		Close: func(obj interface{}) {
			obj.(*sql.DB).Close()
		},
	})

	// Define the connection in the Request scope.
	// Each request will use its own connection.
	builder.AddDefinition(di.Definition{
		Name:  "mysql",
		Scope: di.Request,
		Build: func(ctn di.Container) (interface{}, error) {
			pool := ctn.Get("mysql-pool").(*sql.DB)
			return pool.Conn(context.Background())
		},
		Close: func(obj interface{}) {
			obj.(*sql.Conn).Close()
		},
	})

	// Returns the app Container.
	return builder.Build()
}

func handler(w http.ResponseWriter, r *http.Request, ctn di.Container) {
	// Retrieve the connection.
	conn := ctn.Get("mysql").(*sql.Conn)

	var variable, value string

	row := conn.QueryRowContainer(context.Background(), "SHOW STATUS WHERE `variable_name` = 'Threads_connected'")
	row.Scan(&variable, &value)

	// Display how many connections are opened.
	// As the connection is closed when the request is deleted,
	// the value should not be be higher than the number set with db.SetMaxOpenConns(1).
	w.Write([]byte(variable + ": " + value))
}
```
