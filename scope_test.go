package di

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestScopeList(t *testing.T) {
	list := ScopeList{"a", "b", "c"}

	copy := list.Copy()
	require.Equal(t, list, copy)

	require.True(t, list.Contains("a"))
	require.True(t, list.Contains("b"))
	require.True(t, list.Contains("c"))
	require.False(t, list.Contains("d"))

	require.Equal(t, ScopeList{}, list.ParentScopes("a"))
	require.Equal(t, ScopeList{"a"}, list.ParentScopes("b"))
	require.Equal(t, ScopeList{"a", "b"}, list.ParentScopes("c"))
	require.Equal(t, ScopeList{}, list.ParentScopes("x"))

	require.Equal(t, ScopeList{"b", "c"}, list.SubScopes("a"))
	require.Equal(t, ScopeList{"c"}, list.SubScopes("b"))
	require.Equal(t, ScopeList{}, list.SubScopes("c"))
	require.Equal(t, ScopeList{}, list.SubScopes("x"))
}
