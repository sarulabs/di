package di

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSubContainerCreation(t *testing.T) {
	var err error
	b, _ := NewBuilder()

	app := b.Build()

	request, err := app.SubContainer()
	assert.Nil(t, err)

	subrequest, err := request.SubContainer()
	assert.Nil(t, err)

	_, err = subrequest.SubContainer()
	assert.NotNil(t, err, "sub-request does not have any sub-container")
}
