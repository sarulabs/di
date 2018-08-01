package di

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// builtList is used to store the objects
// that a container has already built.
// The key is the name of the object,
// and the value is the number of elements
// in the map when the element is inserted.
type builtList map[string]int

// Add adds an element in the map.
func (l builtList) Add(name string) builtList {
	if l == nil {
		return builtList{name: 0}
	}
	l[name] = len(l)
	return l
}

// Has checks if the map contains the given element.
func (l builtList) Has(name string) bool {
	_, ok := l[name]
	return ok
}

// OrderedList returns the list of elements in the order
// they were inserted.
func (l builtList) OrderedList() []string {
	s := make([]string, len(l))

	for name, i := range l {
		s[i] = name
	}

	return s
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
