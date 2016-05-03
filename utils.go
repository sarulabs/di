package di

import (
	"fmt"
	"reflect"
)

// stringSliceContains checks if a slice of string contains a given element
func stringSliceContains(arr []string, s string) bool {
	for _, elt := range arr {
		if s == elt {
			return true
		}
	}

	return false
}

// fill copies src in dest. dest should be a pointer to src type.
func fill(src, dest interface{}) (err error) {
	defer func() {
		if r := recover(); r != nil {
			d := reflect.TypeOf(dest)
			s := reflect.TypeOf(src)
			err = fmt.Errorf("destination is `%s` but should be a pointer to the source type `%s`", d, s)
		}
	}()

	reflect.ValueOf(dest).Elem().Set(reflect.ValueOf(src))
	return
}
