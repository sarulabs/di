package di

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefConstructors(t *testing.T) {
	b, err := NewEnhancedBuilder()
	require.Nil(t, err)

	type C struct{ SField string }
	type B struct{ CField C }
	type A struct {
		BField B
		CField C
		SField string
	}

	defC := NewDefFor(C{SField: "value"}).SetIs(C{})
	err = b.Add(defC)
	require.Nil(t, err)

	defB := NewDef(func(ctn Container) (interface{}, error) {
		return B{CField: ctn.Get(defC).(C)}, nil
	}).SetIs(B{})
	err = b.Add(defB)
	require.Nil(t, err)

	defA := NewDefForType(A{}).SetIs(A{})
	err = b.Add(defA)
	require.Nil(t, err)

	defAPtr := NewDefForType(&A{}).SetIs(&A{})
	err = b.Add(defAPtr)
	require.Nil(t, err)

	defErrS := NewDefForType("")
	err = b.Add(defErrS)
	require.Nil(t, err)

	ctn, err := b.Build()
	require.Nil(t, err)

	a := ctn.Get(defA).(A)
	require.Equal(t, "", a.SField)
	require.Equal(t, "value", a.CField.SField)
	require.Equal(t, "value", a.BField.CField.SField)

	aPrt := ctn.Get(defAPtr).(*A)
	require.Equal(t, "", aPrt.SField)
	require.Equal(t, "value", aPrt.CField.SField)
	require.Equal(t, "value", aPrt.BField.CField.SField)

	_, err = ctn.SafeGet(defErrS)
	require.NotNil(t, err, "NewDefForType only works for structs and pointers to structs")
}

func TestDefSetters(t *testing.T) {
	def := NewDef(nil).
		SetBuild(func(ctn Container) (interface{}, error) { return nil, nil }).
		SetClose(func(obj interface{}) error { return nil }).
		SetName("name").
		SetScope(App).
		SetUnshared(true).
		SetIs("", Def{}, &Def{}).
		SetTags(Tag{Name: "tag1"}, Tag{Name: "tag2"})

	require.NotNil(t, def.Build)
	require.NotNil(t, def.Close)
	require.Equal(t, "name", def.Name)
	require.Equal(t, App, def.Scope)
	require.Equal(t, true, def.Unshared)
	require.Equal(t, []reflect.Type{reflect.TypeOf(""), reflect.TypeOf(Def{}), reflect.TypeOf(&Def{})}, def.Is)
	require.Equal(t, []Tag{{Name: "tag1"}, {Name: "tag2"}}, def.Tags)
}
