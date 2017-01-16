package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOpenQfile(t *testing.T) {
	assert := assert.New(t)
	file, err := OpenQfile("testdata/qfile")

	// Loading successful?
	assert.NoError(err)
	assert.NotNil(file)
	assert.Len(file.params, 71)

	// Existing value
	assert.Equal("04012345678", file.GetFirst("number"))
	assert.EqualValues([]string{"04012345678"}, file.GetAll("number"))

	// Non-existing value
	assert.Nil(file.GetAll("non-existing"))
	assert.Equal("", file.GetFirst("non-existing"))

	// Set new value
	assert.EqualError(file.Set("foo", "bar"), "Tag not found")
	file.Add("foo", "bar")
	assert.Equal("bar", file.GetFirst("foo"))
	assert.NoError(file.Set("foo", "baz"))
	assert.Equal("baz", file.GetFirst("foo"))

	// Close
	assert.NoError(file.Close())
}
