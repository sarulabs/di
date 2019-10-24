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
	require.False(t, list.HasDef("key"))
	require.Equal(t, objectKey{defName: ""}, last)
	require.False(t, hasLast)

	list = list.Add(objectKey{defName: "a"})
	newList := list.Add(objectKey{defName: "b"})
	newList = newList.Add(objectKey{defName: "c"})

	last, hasLast = list.LastElement()

	require.Equal(t, []string{"a"}, list.OrderedList())
	require.True(t, list.HasDef("a"))
	require.False(t, list.HasDef("b"))
	require.False(t, list.HasDef("c"))
	require.False(t, list.HasDef("d"))
	require.Equal(t, objectKey{defName: "a"}, last)
	require.True(t, hasLast)

	last, hasLast = newList.LastElement()

	require.Equal(t, []string{"a", "b", "c"}, newList.OrderedList())
	require.True(t, newList.HasDef("a"))
	require.True(t, newList.HasDef("b"))
	require.True(t, newList.HasDef("c"))
	require.False(t, newList.HasDef("d"))
	require.Equal(t, objectKey{defName: "c"}, last)
	require.True(t, hasLast)
}

func TestGraph(t *testing.T) {
	tests := []struct {
		descr       string
		vertices    []string
		edges       [][]string
		expected    []objectKey
		expectedErr bool
	}{
		{
			descr:    "test dag 1",
			vertices: []string{},
			edges: [][]string{
				{"X", "A"},
				{"X", "B"},
				{"Y", "A"},
				{"Y", "B"},
				{"A", "C"},
			},
			expected:    []objectKey{{defName: "Y"}, {defName: "X"}, {defName: "B"}, {defName: "A"}, {defName: "C"}},
			expectedErr: false,
		},
		{
			descr:    "test dag 2",
			vertices: []string{"NOT LINKED"},
			edges: [][]string{
				{"X", "A"},
				{"X", "B"},
				{"Y", "A"},
				{"Y", "B"},
				{"A", "C"},
				{"X", "A"}, // redeclared
			},
			expected:    []objectKey{{defName: "Y"}, {defName: "X"}, {defName: "B"}, {defName: "A"}, {defName: "C"}, {defName: "NOT LINKED"}},
			expectedErr: false,
		},
		{
			descr: "test dag 3",
			edges: [][]string{
				{"5", "11"},
				{"7", "11"},
				{"7", "8"},
				{"3", "8"},
				{"3", "10"},
				{"11", "2"},
				{"11", "9"},
				{"11", "10"},
				{"8", "9"},
			},
			expected:    []objectKey{{defName: "3"}, {defName: "7"}, {defName: "8"}, {defName: "5"}, {defName: "11"}, {defName: "10"}, {defName: "9"}, {defName: "2"}},
			expectedErr: false,
		},
		{
			descr:    "test dag cycle",
			vertices: []string{},
			edges: [][]string{
				{"X", "A"},
				{"X", "B"},
				{"Y", "A"},
				{"Y", "B"},
				{"A", "C"},
				{"C", "X"},
			},
			expected:    []objectKey{{defName: "X"}, {defName: "A"}, {defName: "B"}, {defName: "Y"}, {defName: "C"}},
			expectedErr: true,
		},
	}

	for _, test := range tests {
		g := newGraph()

		for _, v := range test.vertices {
			g.AddVertex(objectKey{defName: v})
		}

		for _, e := range test.edges {
			g.AddEdge(objectKey{defName: e[0]}, objectKey{defName: e[1]})
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
