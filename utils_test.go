package di

import "testing"

func TestStringSliceContains(t *testing.T) {
	if !stringSliceContains([]string{"1", "2", "3"}, "2") {
		t.Error("slice should contain 3")
	}

	if stringSliceContains([]string{"1", "2", "3"}, "0") {
		t.Error("slice should not contain 0")
	}
}

func TestFillUtil(t *testing.T) {
	var err error

	var i int
	err = fill(100, &i)
	if err != nil || i != 100 {
		t.Errorf("i should have been initialized but is %d (err=%s)", i, err)
	}

	err = fill(100, i)
	if err == nil {
		t.Error("i should not have been initialized")
	}
}

func TestIsHashable(t *testing.T) {
	if !isHashable("string") {
		t.Error("string are hashable")
	}

	if !isHashable(33) {
		t.Error("int are hashable")
	}

	if !isHashable(struct{}{}) {
		t.Error("structs are hashable")
	}

	if !isHashable(&struct{}{}) {
		t.Error("pointers are hashable")
	}

	if isHashable([]interface{}{}) {
		t.Error("slices are not hashable")
	}

	if isHashable(map[interface{}]interface{}{}) {
		t.Error("maps are not hashable")
	}
}
