package di

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewContextManagerErrors(t *testing.T) {
	var err error

	_, err = NewContextManager()
	assert.NotNil(t, err, "should not be able to create a ContextManager without any scope")

	_, err = NewContextManager("app", "")
	assert.NotNil(t, err, "should not be able to create a ContextManager with an empty scope")

	_, err = NewContextManager("app", "request", "app", "subrequest")
	assert.NotNil(t, err, "should not be able to create a ContextManager with two identical scopes")
}

func TestResolveName(t *testing.T) {
	cm, _ := NewContextManager("app", "request", "subrequest")

	cm.Instance(Instance{
		Name:    "instance",
		Aliases: []string{"instance1", "i"},
	})

	cm.Maker(Maker{
		Name:    "maker",
		Aliases: []string{"maker1", "m"},
		Scope:   "request",
	})

	var name string
	var err error

	name, err = cm.ResolveName("instance")
	assert.Nil(t, err)
	assert.Equal(t, "instance", name)

	name, err = cm.ResolveName("instance1")
	assert.Nil(t, err)
	assert.Equal(t, "instance", name)

	name, err = cm.ResolveName("i")
	assert.Nil(t, err)
	assert.Equal(t, "instance", name)

	name, err = cm.ResolveName("maker")
	assert.Nil(t, err)
	assert.Equal(t, "maker", name)

	name, err = cm.ResolveName("maker1")
	assert.Nil(t, err)
	assert.Equal(t, "maker", name)

	name, err = cm.ResolveName("m")
	assert.Nil(t, err)
	assert.Equal(t, "maker", name)

	name, err = cm.ResolveName("MAKER")
	assert.NotNil(t, err)
	assert.Equal(t, "", name)
}

func TestNameIsUsed(t *testing.T) {
	cm, _ := NewContextManager("app", "request", "subrequest")

	cm.Instance(Instance{
		Name:    "instance",
		Aliases: []string{"instance1", "i"},
	})

	cm.Maker(Maker{
		Name:    "maker",
		Aliases: []string{"maker1", "m"},
		Scope:   "request",
	})

	assert.True(t, cm.NameIsUsed("instance"))
	assert.True(t, cm.NameIsUsed("instance1"))
	assert.True(t, cm.NameIsUsed("i"))
	assert.True(t, cm.NameIsUsed("maker"))
	assert.True(t, cm.NameIsUsed("maker1"))
	assert.True(t, cm.NameIsUsed("m"))
	assert.False(t, cm.NameIsUsed("MAKER"))
}

func TestMakerRegistrationErrors(t *testing.T) {
	cm, _ := NewContextManager("app", "request", "subrequest")

	var err error

	err = cm.Maker(Maker{Name: "maker", Aliases: []string{"m"}, Scope: "request"})
	assert.Nil(t, err)

	err = cm.Maker(Maker{Name: "maker2", Scope: "undefined"})
	assert.NotNil(t, err, "should not be able to register a Maker in an undefined scope")

	err = cm.Maker(Maker{Name: "maker", Scope: "subrequest"})
	assert.NotNil(t, err, "should not be able to register a Maker if the name is already used")

	err = cm.Maker(Maker{Name: "", Scope: "subrequest"})
	assert.NotNil(t, err, "should not be able to register a Maker if the name is empty")

	err = cm.Maker(Maker{Name: "maker2", Aliases: []string{""}, Scope: "subrequest"})
	assert.NotNil(t, err, "should not be able to register a Maker if an alias is empty")

	err = cm.Maker(Maker{Name: "maker2", Aliases: []string{"maker2"}, Scope: "subrequest"})
	assert.NotNil(t, err, "should not be able to register a Maker if an alias is equal to the name")

	err = cm.Maker(Maker{Name: "maker2", Aliases: []string{"a", "a"}, Scope: "subrequest"})
	assert.NotNil(t, err, "should not be able to register a Maker if two aliases are identical")

	err = cm.Maker(Maker{Name: "maker2", Aliases: []string{"maker"}, Scope: "subrequest"})
	assert.NotNil(t, err, "should not be able to register a Maker if an alias is already used")

	err = cm.Maker(Maker{Name: "maker2", Aliases: []string{"m2"}, Scope: "request"})
	assert.Nil(t, err)
}

func TestInstanceRegistrationErrors(t *testing.T) {
	cm, _ := NewContextManager("app", "request", "subrequest")

	var err error

	err = cm.Instance(Instance{Name: "instance", Aliases: []string{"i"}})
	assert.Nil(t, err)

	err = cm.Instance(Instance{Name: "instance"})
	assert.NotNil(t, err, "should not be able to register a Instance if the name is already used")

	err = cm.Instance(Instance{Name: ""})
	assert.NotNil(t, err, "should not be able to register a Instance if the name is empty")

	err = cm.Instance(Instance{Name: "instance2", Aliases: []string{"instance"}})
	assert.NotNil(t, err, "should not be able to register a Instance if an alias is already used")

	err = cm.Instance(Instance{Name: "instance2", Aliases: []string{"i2"}})
	assert.Nil(t, err)
}

func TestContextCeation(t *testing.T) {
	cm, _ := NewContextManager("app", "request", "subrequest")

	_, err := cm.Context("undefined")
	assert.NotNil(t, err, "should not be able to create a context in an undefined scope")

	app, err := cm.Context("app")
	assert.Nil(t, err)
	assert.Equal(t, cm, app.ContextManager())
	assert.Equal(t, "app", app.Scope())

	subrequest, err := cm.Context("subrequest")
	assert.Nil(t, err)
	assert.True(t, cm == subrequest.ContextManager())
	assert.Equal(t, "subrequest", subrequest.Scope())
	assert.Equal(t, "request", subrequest.Parent().Scope())
	assert.Equal(t, "app", subrequest.Parent().Parent().Scope())
	assert.True(t, app != subrequest.Parent().Parent())
}

func TestThatManagerIsClosedAfterTheFirstContextIsCreated(t *testing.T) {
	cm, _ := NewContextManager("app")
	cm.Context("app")

	var err error

	err = cm.Instance(Instance{
		Name: "instance",
		Item: "value",
	})

	assert.NotNil(t, err, "it should not be possible to register an instance at this point")

	err = cm.Maker(Maker{
		Name:  "maker",
		Scope: "app",
	})

	assert.NotNil(t, err, "it should not be possible to register an maker at this point")
}
