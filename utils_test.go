package di

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGraph(t *testing.T) {
	tests := []struct {
		descr       string
		vertices    []int
		edges       [][]int
		expected    []int
		expectedErr bool
	}{
		{
			descr:    "test dag 1",
			vertices: []int{},
			edges: [][]int{
				{1, 2},
				{1, 3},
				{4, 2},
				{4, 3},
				{2, 5},
			},
			expected:    []int{4, 1, 3, 2, 5},
			expectedErr: false,
		},
		{
			descr:    "test dag 2",
			vertices: []int{9999},
			edges: [][]int{
				{1, 2},
				{1, 3},
				{4, 2},
				{4, 3},
				{2, 5},
				{1, 2}, // redeclared
			},
			expected:    []int{4, 1, 3, 2, 5, 9999},
			expectedErr: false,
		},
		{
			descr: "test dag 3",
			edges: [][]int{
				{5, 11},
				{7, 11},
				{7, 8},
				{3, 8},
				{3, 10},
				{11, 2},
				{11, 9},
				{11, 10},
				{8, 9},
			},
			expected:    []int{3, 7, 8, 5, 11, 10, 9, 2},
			expectedErr: false,
		},
		{
			descr:    "test dag cycle",
			vertices: []int{},
			edges: [][]int{
				{1, 2},
				{1, 3},
				{4, 2},
				{4, 3},
				{2, 5},
				{5, 1},
			},
			expected:    []int{1, 2, 3, 4, 5},
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
