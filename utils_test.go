package di

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuiltList(t *testing.T) {
	var list builtList

	require.Equal(t, 0, len(list.OrderedList()))
	require.False(t, list.Has("key"))

	newList := list.Add("a")
	newList = newList.Add("b")
	newList = newList.Add("c")

	require.Equal(t, []string{"a", "b", "c"}, newList.OrderedList())
	require.True(t, newList.Has("a"))
	require.True(t, newList.Has("b"))
	require.True(t, newList.Has("c"))
	require.False(t, newList.Has("d"))
}

func TestMultiErrBuilder(t *testing.T) {
	builder := &multiErrBuilder{}

	err := builder.Build()
	require.Nil(t, err)

	builder.Add(errors.New("a"))
	err = builder.Build()
	require.NotNil(t, err)
	require.Equal(t, "a", err.Error())

	builder.Add(errors.New("b"))
	err = builder.Build()
	require.NotNil(t, err)
	require.Equal(t, "a AND b", err.Error())

	builder.Add(errors.New("c"))
	err = builder.Build()
	require.NotNil(t, err)
	require.Equal(t, "a AND b AND c", err.Error())
}

func TestFillUtil(t *testing.T) {
	var err error

	var i int
	err = fill(100, &i)
	require.Nil(t, err)
	require.Equal(t, 100, i)

	err = fill(100, i)
	require.NotNil(t, err)
}
