package di

import (
	"errors"
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
			err = errors.New("destination should be a pointer to the source type")
		}
	}()

	reflect.ValueOf(dest).Elem().Set(reflect.ValueOf(src))
	return
}

// isHashable checks if an interface can be used as the key of a map.
func isHashable(item interface{}) (hashable bool) {
	defer func() {
		if r := recover(); r != nil {
			hashable = false
		}
	}()
	hashable = true
	_ = map[interface{}]interface{}{item: item}
	return
}
