package di

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStringSliceContains(t *testing.T) {
	assert.True(t, stringSliceContains([]string{"1", "2", "3"}, "2"))
	assert.False(t, stringSliceContains([]string{"1", "2", "3"}, "0"))
}

func TestFillUtil(t *testing.T) {
	var err error

	var i int
	err = fill(100, &i)
	assert.Nil(t, err)
	assert.Equal(t, 100, i)

	err = fill(100, i)
	assert.NotNil(t, err)
}
