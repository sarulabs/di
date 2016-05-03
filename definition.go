package di

// App is the name of the application scope.
const App = "app"

// Request is the name of the request scope.
const Request = "request"

// SubRequest is the name of the subrequest scope.
const SubRequest = "subrequest"

// Definition contains information to build and close an object inside a Context.
type Definition struct {
	Name  string
	Scope string
	Build func(ctx Context) (interface{}, error)
	Close func(obj interface{})
	Tags  []Tag
}

// Tag can contain more specific information about a Definition.
// It is useful to find a Definition thanks to its tags instead of its name.
type Tag struct {
	Name string
	Args map[string]string
}
