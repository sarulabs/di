package di

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewBuilder(t *testing.T) {
	var b *Builder
	var err error

	_, err = NewBuilder("app", "")
	require.NotNil(t, err, "should not be able to create a Builder with an empty scope")

	_, err = NewBuilder("app", "request", "app", "subrequest")
	require.NotNil(t, err, "should not be able to create a Builder with two identical scopes")

	b, err = NewBuilder("a", "b", "c")
	require.Nil(t, err)
	require.Equal(t, ScopeList{"a", "b", "c"}, b.Scopes())

	b, err = NewBuilder()
	require.Nil(t, err)
	require.Equal(t, ScopeList{App, Request, SubRequest}, b.Scopes())
}

func TestBuilderDefinitions(t *testing.T) {
	b, _ := NewBuilder()

	b.Add([]Def{
		{
			Name:  "o1",
			Build: func(ctn Container) (interface{}, error) { return nil, nil },
		},
		{
			Name:  "o2",
			Build: func(ctn Container) (interface{}, error) { return nil, nil },
		},
	}...)

	defs := b.Definitions()

	require.Len(t, defs, 2)
	require.Equal(t, "o1", defs["o1"].Name)
	require.Equal(t, "o2", defs["o2"].Name)
}

func TestBuilderIsDefined(t *testing.T) {
	b, _ := NewBuilder()

	b.Add(Def{
		Name:  "name",
		Scope: App,
		Build: func(ctn Container) (interface{}, error) { return nil, nil },
	})

	require.True(t, b.IsDefined("name"))
	require.False(t, b.IsDefined("undefined"))
}

func TestBuilderServiceOverride(t *testing.T) {
	b, _ := NewBuilder()

	var err error

	err = b.Add(Def{
		Name: "name",
		Build: func(ctn Container) (interface{}, error) {
			return "first", nil
		},
	})
	require.Nil(t, err)

	err = b.Add(Def{
		Name: "name",
		Build: func(ctn Container) (interface{}, error) {
			return "second", nil
		},
	})
	require.Nil(t, err)

	require.Equal(t, "second", b.Build().Get("name").(string))
}

func TestBuilderAddErrors(t *testing.T) {
	b, _ := NewBuilder()

	var err error

	buildFunc := func(ctn Container) (interface{}, error) { return nil, nil }

	err = b.Add(Def{Name: "name", Scope: App, Build: buildFunc})
	require.Nil(t, err)

	err = b.Add(Def{Name: "object", Scope: "undefined", Build: buildFunc})
	require.NotNil(t, err, "should not be able to add a Def in an undefined scope")

	err = b.Add(Def{Name: "", Scope: App, Build: buildFunc})
	require.NotNil(t, err, "should not be able to add a Def if the name is empty")

	err = b.Add(Def{Name: "object", Scope: App})
	require.NotNil(t, err, "should not be able to add a Def if Build is empty")

	err = b.Add(Def{Name: "object", Scope: App, Build: buildFunc})
	require.Nil(t, err)
}

func TestBuilderSet(t *testing.T) {
	b, _ := NewBuilder()

	var err error

	err = b.Set("", "error")
	require.NotNil(t, err, "should not be able to set an object without a name")

	err = b.Set("key", "value")
	require.Nil(t, err)

	ctn := b.Build()
	require.Equal(t, "value", ctn.Get("key").(string))
}

func TestBuilderBuild(t *testing.T) {
	ctn := (&Builder{}).Build()
	require.True(t, ctn.core.closed, "should have at least one scope to use Build")

	b, _ := NewBuilder()

	b.Add([]Def{
		{
			Name:  "o1",
			Build: func(ctn Container) (interface{}, error) { return nil, nil },
		},
		{
			Name:  "o2",
			Scope: Request,
			Build: func(ctn Container) (interface{}, error) { return nil, nil },
		},
	}...)

	app := b.Build()
	require.Equal(t, App, app.Scope())
	require.Len(t, app.Definitions(), 2)
	require.Equal(t, App, app.Definitions()["o1"].Scope)
	require.Equal(t, Request, app.Definitions()["o2"].Scope)
}

func TestBuilderGet(t *testing.T) {
	var err error

	b, _ := NewBuilder()

	err = b.Add(*NewDefForType(mockA{}).SetName("nameA").SetIs(mockA{}))
	require.Nil(t, err)
	err = b.Add(*NewDefForType(&mockB{}).SetName("nameB").SetIs(&mockB{}))
	require.Nil(t, err)
	err = b.Add(*NewDefFor(mockC{SField: "ok"}).SetName("nameC").SetIs(mockC{}))
	require.Nil(t, err)

	app := b.Build()
	a := app.Get(reflect.TypeOf(mockA{})).(mockA)
	require.Equal(t, "ok", a.BField.CField.SField)
}
