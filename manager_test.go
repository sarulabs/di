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

func TestIsDefined(t *testing.T) {
	b, _ := NewBuilder()

	b.AddDefinition(Definition{
		Name:  "name",
		Scope: App,
		Build: func(ctx *Context) (interface{}, error) {
			return nil, nil
		},
	})

	assert.True(t, b.IsDefined("name"))
	assert.False(t, b.IsDefined("undefined"))
}

func TestAddDefinitionErrors(t *testing.T) {
	b, _ := NewBuilder()

	var err error

	buildFunc := func(ctx *Context) (interface{}, error) {
		return nil, nil
	}

	err = b.AddDefinition(Definition{Name: "name", Scope: App, Build: buildFunc})
	assert.Nil(t, err)

	err = b.AddDefinition(Definition{Name: "object", Scope: "undefined", Build: buildFunc})
	assert.NotNil(t, err, "should not be able to add a Definition in an undefined scope")

	err = b.AddDefinition(Definition{Name: "name", Scope: App, Build: buildFunc})
	assert.NotNil(t, err, "should not be able to add a Definition if the name is already used")

	err = b.AddDefinition(Definition{Name: "", Scope: App, Build: buildFunc})
	assert.NotNil(t, err, "should not be able to add a Definition if the name is empty")

	err = b.AddDefinition(Definition{Name: "object", Scope: App, Build: buildFunc})
	assert.Nil(t, err)
}

func TestSet(t *testing.T) {
	b, _ := NewBuilder()

	var err error

	err = b.Set("name", nil)
	assert.Nil(t, err)

	err = b.Set("name", nil)
	assert.NotNil(t, err, "should not be able to set an object if the name is already used")

	err = b.Set("", nil)
	assert.NotNil(t, err, "should not be able to set an object if the name is empty")
}

func TestBuild(t *testing.T) {
	b, _ := NewBuilder()

	app, err := b.Build()
	assert.Nil(t, err)
	assert.Equal(t, App, app.Scope())
}
