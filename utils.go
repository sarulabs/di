package di

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// graph is a Directed Acyclic Graph.
// It is used to store the dependencies inside a container.
// These dependencies are then used to determine the order
// that should be used to close the objects.
type graph struct {
	verticeSlice []int
	vertices     map[int]*graphVertex
}

// graphVertex contains the vertex data.
type graphVertex struct {
	// numIn in the number of incoming edges.
	numIn int
	// numInTmp is used by the TopologicalOrdering to avoid messing with numIn
	numInTmp int
	// out contains the name the outgoing edges.
	out []int
	// outMap is the same as "out", but in a map
	// to quickly check if a vertex is in the outgoing edges.
	outMap map[int]struct{}
}

// newGraph creates a new graph.
func newGraph() *graph {
	return &graph{
		verticeSlice: []int{},
		vertices:     map[int]*graphVertex{},
	}
}

// AddVertex adds a vertex to the graph.
func (g *graph) AddVertex(v int) {
	_, ok := g.vertices[v]
	if ok {
		return
	}

	g.verticeSlice = append(g.verticeSlice, v)

	g.vertices[v] = &graphVertex{
		numIn:  0,
		out:    []int{},
		outMap: map[int]struct{}{},
	}
}

// AddEdge adds an edge to the graph.
func (g *graph) AddEdge(from, to int) {
	g.AddVertex(from)
	g.AddVertex(to)

	// check if the edge is already registered
	if _, ok := g.vertices[from].outMap[to]; ok {
		return
	}

	// update the vertices
	g.vertices[from].out = append(g.vertices[from].out, to)
	g.vertices[from].outMap[to] = struct{}{}
	g.vertices[to].numIn++
}

// TopologicalOrdering returns a valid topological sort.
// It implements Kahn's algorithm.
// If there is a cycle in the graph, an error is returned.
// The list of vertices is also returned even if it is not ordered.
func (g *graph) TopologicalOrdering() ([]int, error) {
	l := []int{}
	q := []int{}

	for _, v := range g.verticeSlice {
		if g.vertices[v].numIn == 0 {
			q = append(q, v)
		}
		g.vertices[v].numInTmp = g.vertices[v].numIn
	}

	for len(q) > 0 {
		n := q[len(q)-1]
		q = q[:len(q)-1]
		l = append(l, n)

		for _, m := range g.vertices[n].out {
			g.vertices[m].numInTmp--
			if g.vertices[m].numInTmp == 0 {
				q = append(q, m)
			}
		}
	}

	if len(l) != len(g.verticeSlice) {
		return append([]int{}, g.verticeSlice...), errors.New("a cycle has been found in the dependencies")
	}

	return l, nil
}

// multiErrBuilder can accumulate errors.
type multiErrBuilder struct {
	errs []error
}

// Add adds an error in the multiErrBuilder.
func (b *multiErrBuilder) Add(err error) {
	if err != nil {
		b.errs = append(b.errs, err)
	}
}

// Build returns an errors containing all the messages
// of the accumulated errors. If there is no error
// in the builder, it returns nil.
func (b *multiErrBuilder) Build() error {
	if len(b.errs) == 0 {
		return nil
	}

	msgs := make([]string, len(b.errs))

	for i, err := range b.errs {
		msgs[i] = err.Error()
	}

	return errors.New(strings.Join(msgs, " AND "))
}

// fill copies src in dest. dest should be a pointer to src type.
func fill(src, dest interface{}) (err error) {
	defer func() {
		if r := recover(); r != nil {
			d := reflect.TypeOf(dest)
			s := reflect.TypeOf(src)
			err = fmt.Errorf("the fill destination should be a pointer to a `%s`, but you used a `%s`", s, d)
		}
	}()

	reflect.ValueOf(dest).Elem().Set(reflect.ValueOf(src))

	return err
}
