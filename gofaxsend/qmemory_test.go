package gofaxsend

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewQmemory(t *testing.T) {
	assert := assert.New(t)

	values := map[string][]string{}
	var mem1 Qfiler = NewQmemory(values)
	assert.NotNil(mem1)

	var mem2 Qfiler = NewQmemory(nil)
	assert.NotNil(mem2)

	// Make sure we can use the internal map, that it is not nil and therefore does not panic
	mem2.Set("foo", "bar")
}

func TestQmemory(t *testing.T) {
	assert := assert.New(t)
	file := NewQmemory(map[string][]string{
		"number": {"04012345678"},
		"priority": {"127"},
		"sender": {"Max Mustermann"},
	})

	// Loading successful?
	assert.NotNil(file)

	// Existing string value
	assert.Equal("04012345678", file.GetString("number"))
	assert.EqualValues([]string{"04012345678"}, file.GetAll("number"))

	// Non-existing string value
	assert.Empty(file.GetAll("non-existing"))
	assert.Equal("", file.GetString("non-existing"))

	// Existing int value
	i, err := file.GetInt("priority")
	assert.Equal(127, i)
	assert.NoError(err)

	// Existing non-int value
	i, err = file.GetInt("sender")
	assert.Equal(0, i)
	assert.Error(err)

	// Non-existing int value
	i, err = file.GetInt("xxxxxx")
	assert.Equal(0, i)
	assert.EqualError(err, "tag not found")

	// Set new value
	file.Add("foo", "bar")
	assert.Equal("bar", file.GetString("foo"))
	file.Set("foo", "baz")
	assert.Equal("baz", file.GetString("foo"))

	file.Set("baz", "foo")
	assert.Equal("baz", file.GetString("foo"))

	// Write
	assert.NoError(file.Write())
}
