package di

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewEnhancedBuilderAndScopes(t *testing.T) {
	var b *EnhancedBuilder
	var err error

	_, err = NewEnhancedBuilder("app", "")
	require.NotNil(t, err, "should not be able to create a Builder with an empty scope")

	_, err = NewEnhancedBuilder("app", "request", "app", "subrequest")
	require.NotNil(t, err, "should not be able to create a Builder with two identical scopes")

	b, err = NewEnhancedBuilder("a", "b", "c")
	require.Nil(t, err)
	require.Equal(t, ScopeList{"a", "b", "c"}, b.Scopes())

	b, err = NewEnhancedBuilder()
	require.Nil(t, err)
	require.Equal(t, ScopeList{App, Request, SubRequest}, b.Scopes())

	require.Equal(t, ScopeList{}, (&EnhancedBuilder{}).Scopes())
}

func TestEnhancedBuilderDefinitions(t *testing.T) {
	b, _ := NewEnhancedBuilder()

	def1 := &Def{
		Name:  "o1",
		Build: func(ctn Container) (interface{}, error) { return nil, nil },
	}
	def2 := &Def{
		Name:  "o2",
		Build: func(ctn Container) (interface{}, error) { return nil, nil },
	}
	b.Add(def1)
	b.Add(def2)

	def1.Name = "updated"

	defs := b.Definitions()

	require.Len(t, defs, 2)
	require.Equal(t, "o1", defs["o1"].Name)
	require.Equal(t, "o2", defs["o2"].Name)

	require.Equal(t, DefMap{}, (&EnhancedBuilder{}).Definitions())
}

func TestEnhancedBuilderNameIsDefined(t *testing.T) {
	b, _ := NewEnhancedBuilder()

	b.Add(&Def{
		Name:  "name",
		Scope: App,
		Build: func(ctn Container) (interface{}, error) { return nil, nil },
	})

	require.True(t, b.NameIsDefined("name"))
	require.False(t, b.NameIsDefined("undefined"))

	require.Equal(t, false, (&EnhancedBuilder{}).NameIsDefined("name"))
}

func TestEnhancedBuilderServiceOverride(t *testing.T) {
	b, err := NewEnhancedBuilder()
	require.Nil(t, err)

	err = b.Add(&Def{
		Name: "name",
		Build: func(ctn Container) (interface{}, error) {
			return "first", nil
		},
	})
	require.Nil(t, err)

	err = b.Add(&Def{
		Name: "name",
		Build: func(ctn Container) (interface{}, error) {
			return "second", nil
		},
	})
	require.Nil(t, err)

	ctn, err := b.Build()
	require.Nil(t, err)

	require.Equal(t, "second", ctn.Get("name").(string))
}

func TestEnhancedBuilderAdd(t *testing.T) {
	b, err := NewEnhancedBuilder()
	require.Nil(t, err)

	buildFunc := func(ctn Container) (interface{}, error) { return nil, nil }

	err = b.Add(NewDef(buildFunc).SetScope(App))
	require.Nil(t, err)

	defNoName := NewDef(buildFunc)
	err = b.Add(defNoName)
	require.Nil(t, err)
	require.Equal(t, "", defNoName.Name, "the definition name is updated only after Build is called")

	err = b.Add(nil)
	require.NotNil(t, err, "should not be able to add a nil *Def")

	err = b.Add(NewDef(buildFunc).SetScope("undefined"))
	require.NotNil(t, err, "should not be able to add a Def in an undefined scope")

	err = b.Add(NewDef(nil).SetScope(App))
	require.NotNil(t, err, "should not be able to add a Def if Build is empty")

	err = b.Add(NewDef(buildFunc).SetName("_di_generated_XXX"))
	require.NotNil(t, err, "should not be able to add a Def if the name start by _di_generated_")

	defCheckIs := &Def{Name: "checkIs", Build: buildFunc, Is: []reflect.Type{}}
	err = b.Add(defCheckIs)
	require.Nil(t, err)
	defCheckIs.Is = append(defCheckIs.Is, reflect.TypeOf(""))
	require.Equal(t, 0, len(b.Definitions()["checkIs"].Is))

	err = (&EnhancedBuilder{}).Add(NewDef(buildFunc))
	require.NotNil(t, err, "can not add definition on a not properly created builder")
}

func TestEnhancedBuilderBuild(t *testing.T) {
	ctn, err := (&EnhancedBuilder{}).Build()
	require.NotNil(t, err)
	require.True(t, ctn.core.closed, "should have at least one scope to use Build")

	b, err := NewEnhancedBuilder()
	require.Nil(t, err)

	buildFunc := func(ctn Container) (interface{}, error) { return nil, nil }

	err = b.Add(&Def{
		Name:  "o1",
		Build: buildFunc,
	})
	require.Nil(t, err)

	err = b.Add(&Def{
		Name:  "o2",
		Scope: Request,
		Build: buildFunc,
	})
	require.Nil(t, err)

	app, err := b.Build()
	require.Nil(t, err)
	require.Equal(t, App, app.Scope())
	require.Len(t, app.Definitions(), 2)
	require.Equal(t, App, app.Definitions()["o1"].Scope)
	require.Equal(t, Request, app.Definitions()["o2"].Scope)

	// Check if the definition is updated.
	b, err = NewEnhancedBuilder()
	require.Nil(t, err)

	defA := &Def{
		Build:    buildFunc,
		Close:    func(obj interface{}) error { return nil },
		Name:     "A",
		Scope:    Request,
		Unshared: true,
		Is:       []reflect.Type{reflect.TypeOf("")},
		Tags:     []Tag{{Name: "tag"}},
	}

	err = b.Add(defA)
	require.Nil(t, err)
	require.Equal(t, -1, defA.Index())

	defB := NewDef(buildFunc)

	b.Add(defB)
	require.Nil(t, err)
	require.Equal(t, -1, defB.Index())

	defB.Build = nil
	defB.Close = func(obj interface{}) error { return nil }
	defB.Name = "redefined"
	defB.Scope = "redefined"
	defB.Unshared = true
	defB.Is = []reflect.Type{reflect.TypeOf("")}
	defB.Tags = nil

	_, err = b.Build()
	require.Nil(t, err)

	require.NotNil(t, defA.Build)
	require.NotNil(t, defA.Close)
	require.Equal(t, "A", defA.Name)
	require.Equal(t, Request, defA.Scope)
	require.Equal(t, true, defA.Unshared)
	require.Equal(t, []reflect.Type{reflect.TypeOf("")}, defA.Is)
	require.Equal(t, []Tag{{Name: "tag"}}, defA.Tags)
	require.Equal(t, 0, defA.Index())
	require.Equal(t, 0, defA.builderIndex)

	require.NotNil(t, defB.Build)
	require.Nil(t, defB.Close)
	require.Equal(t, "_di_generated_1", defB.Name)
	require.Equal(t, App, defB.Scope)
	require.Equal(t, false, defB.Unshared)
	require.Equal(t, []reflect.Type(nil), defB.Is)
	require.Nil(t, defB.Tags)
	require.Equal(t, 1, defB.Index())
	require.Equal(t, 1, defB.builderIndex)

	// Can not bind the same def two two different containers.
	def := NewDef(buildFunc).SetName("def")

	b, err = NewEnhancedBuilder()
	require.Nil(t, err)
	err = b.Add(def)
	require.Nil(t, err)
	_, err = b.Build()
	require.Nil(t, err)
	_, err = b.Build()
	require.NotNil(t, err, "can not build the same definition twice")
}
