package di

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuiltList(t *testing.T) {
	var list builtList

	last, hasLast := list.LastElement()

	require.Equal(t, 0, len(list.OrderedList()))
	require.False(t, list.Has("key"))
	require.Equal(t, "", last)
	require.False(t, hasLast)

	list = list.Add("a")
	newList := list.Add("b")
	newList = newList.Add("c")

	last, hasLast = list.LastElement()

	require.Equal(t, []string{"a"}, list.OrderedList())
	require.True(t, list.Has("a"))
	require.False(t, list.Has("b"))
	require.False(t, list.Has("c"))
	require.False(t, list.Has("d"))
	require.Equal(t, "a", last)
	require.True(t, hasLast)

	last, hasLast = newList.LastElement()

	require.Equal(t, []string{"a", "b", "c"}, newList.OrderedList())
	require.True(t, newList.Has("a"))
	require.True(t, newList.Has("b"))
	require.True(t, newList.Has("c"))
	require.False(t, newList.Has("d"))
	require.Equal(t, "c", last)
	require.True(t, hasLast)
}

func TestGraph(t *testing.T) {
	tests := []struct {
		descr       string
		vertices    []string
		edges       [][]string
		expected    []string
		expectedErr bool
	}{
		{
			descr:    "test dag 1",
			vertices: []string{},
			edges: [][]string{
				[]string{"X", "A"},
				[]string{"X", "B"},
				[]string{"Y", "A"},
				[]string{"Y", "B"},
				[]string{"A", "C"},
			},
			expected:    []string{"Y", "X", "B", "A", "C"},
			expectedErr: false,
		},
		{
			descr:    "test dag 2",
			vertices: []string{"NOT LINKED"},
			edges: [][]string{
				[]string{"X", "A"},
				[]string{"X", "B"},
				[]string{"Y", "A"},
				[]string{"Y", "B"},
				[]string{"A", "C"},
				[]string{"X", "A"}, // redeclared
			},
			expected:    []string{"Y", "X", "B", "A", "C", "NOT LINKED"},
			expectedErr: false,
		},
		{
			descr: "test dag 3",
			edges: [][]string{
				[]string{"5", "11"},
				[]string{"7", "11"},
				[]string{"7", "8"},
				[]string{"3", "8"},
				[]string{"3", "10"},
				[]string{"11", "2"},
				[]string{"11", "9"},
				[]string{"11", "10"},
				[]string{"8", "9"},
			},
			expected:    []string{"3", "7", "8", "5", "11", "10", "9", "2"},
			expectedErr: false,
		},
		{
			descr:    "test dag cycle",
			vertices: []string{},
			edges: [][]string{
				[]string{"X", "A"},
				[]string{"X", "B"},
				[]string{"Y", "A"},
				[]string{"Y", "B"},
				[]string{"A", "C"},
				[]string{"C", "X"},
			},
			expected:    []string{"X", "A", "B", "Y", "C"},
			expectedErr: true,
		},
	}

	for _, test := range tests {
		g := newGraph()

		for _, v := range test.vertices {
			g.AddVertex(v)
		}

		for _, e := range test.edges {
			g.AddEdge(e[0], e[1])
		}

		l, err := g.TopologicalOrdering()

		if test.expectedErr {
			require.NotNil(t, err, test.descr)
		} else {
			require.Nil(t, err, test.descr)
		}

		require.Equal(t, test.expected, l, test.descr)
	}
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
