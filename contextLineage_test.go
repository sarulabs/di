package di

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSubContextCreation(t *testing.T) {
	var err error
	b, _ := NewBuilder()

	app := b.Build()

	request, err := app.SubContext()
	assert.Nil(t, err)

	subrequest, err := request.SubContext()
	assert.Nil(t, err)

	_, err = subrequest.SubContext()
	assert.NotNil(t, err, "subrequest does not have any subcontext")
}
