package di

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewBuilder(t *testing.T) {
	var b *Builder
	var err error

	_, err = NewBuilder("app", "")
	assert.NotNil(t, err, "should not be able to create a ContextManager with an empty scope")

	_, err = NewBuilder("app", "request", "app", "subrequest")
	assert.NotNil(t, err, "should not be able to create a ContextManager with two identical scopes")

	b, err = NewBuilder("a", "b", "c")
	assert.Nil(t, err)
	assert.Equal(t, []string{"a", "b", "c"}, b.Scopes())
}

func TestDefinitions(t *testing.T) {
	b, _ := NewBuilder()

	def1 := Definition{
		Name:  "o1",
		Build: func(ctx Context) (interface{}, error) { return nil, nil },
	}

	def2 := Definition{
		Name:  "o2",
		Build: func(ctx Context) (interface{}, error) { return nil, nil },
	}

	b.AddDefinition(def1)
	b.AddDefinition(def2)
	defs := b.Definitions()

	assert.Len(t, defs, 2)
	assert.Equal(t, "o1", defs["o1"].Name)
	assert.Equal(t, "o2", defs["o2"].Name)
}

func TestIsDefined(t *testing.T) {
	b, _ := NewBuilder()

	b.AddDefinition(Definition{
		Name:  "name",
		Scope: App,
		Build: func(ctx Context) (interface{}, error) { return nil, nil },
	})

	assert.True(t, b.IsDefined("name"))
	assert.False(t, b.IsDefined("undefined"))
}

func TestAddDefinitionErrors(t *testing.T) {
	b, _ := NewBuilder()

	var err error

	buildFunc := func(ctx Context) (interface{}, error) { return nil, nil }

	err = b.AddDefinition(Definition{Name: "name", Scope: App, Build: buildFunc})
	assert.Nil(t, err)

	err = b.AddDefinition(Definition{Name: "object", Scope: "undefined", Build: buildFunc})
	assert.NotNil(t, err, "should not be able to add a Definition in an undefined scope")

	err = b.AddDefinition(Definition{Name: "name", Scope: App, Build: buildFunc})
	assert.NotNil(t, err, "should not be able to add a Definition if the name is already used")

	err = b.AddDefinition(Definition{Name: "", Scope: App, Build: buildFunc})
	assert.NotNil(t, err, "should not be able to add a Definition if the name is empty")

	err = b.AddDefinition(Definition{Name: "object", Scope: App})
	assert.NotNil(t, err, "should not be able to add a Definition if Build is empty")

	err = b.AddDefinition(Definition{Name: "object", Scope: App, Build: buildFunc})
	assert.Nil(t, err)
}

func TestSet(t *testing.T) {
	var err error

	b := &Builder{}

	err = b.Set("name", nil)
	assert.NotNil(t, err, "should have at least one scope to use Set")

	b, _ = NewBuilder()

	err = b.Set("name", nil)
	assert.Nil(t, err)

	err = b.Set("name", nil)
	assert.NotNil(t, err, "should not be able to set an object if the name is already used")

	err = b.Set("", nil)
	assert.NotNil(t, err, "should not be able to set an object if the name is empty")
}

func TestBuild(t *testing.T) {
	ctx := (&Builder{}).Build()
	assert.Nil(t, ctx, "should have at least one scope to use Build")

	b, _ := NewBuilder()

	buildFn := func(ctx Context) (interface{}, error) { return nil, nil }

	def1 := Definition{
		Name:  "o1",
		Build: buildFn,
	}

	def2 := Definition{
		Name:  "o2",
		Scope: Request,
		Build: buildFn,
	}

	b.AddDefinition(def1)
	b.AddDefinition(def2)

	app := b.Build()
	assert.Equal(t, App, app.Scope())
	assert.Len(t, app.Definitions(), 2)
	assert.Equal(t, App, app.Definitions()["o1"].Scope)
}
