package syncx_test

import (
	"github.com/leap-fish/necs/internal/syncx"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMap(t *testing.T) {
	m := &syncx.Map[int, string]{}
	// Test LoadAndStore
	actual, loaded := m.LoadOrStore(1, "value1")
	assert.False(t, loaded, "Expected loaded=false")
	assert.Equal(t, "value1", actual, "Expected actual value 'value1'")

	// Test Load
	actualValue, ok := m.Load(1)
	assert.True(t, ok, "Expected ok=true")
	assert.Equal(t, "value1", actualValue, "Expected actual value 'value1'")

	// Test Store
	m.Store(2, "value2")
	actualValue, ok = m.Load(2)
	assert.True(t, ok, "Expected ok=true")
	assert.Equal(t, "value2", actualValue, "Expected actual value 'value2'")

	// Test Delete
	m.Delete(1)
	_, ok = m.Load(1)
	assert.False(t, ok, "Expected ok=false for key 1 after deletion")

	// Test LoadAndDelete
	actualValue, loaded = m.LoadAndDelete(2)
	assert.True(t, loaded, "Expected loaded=true")
	assert.Equal(t, "value2", actualValue, "Expected actual value 'value2'")

	// Test Range
	m.Store(3, "value3")
	m.Store(4, "value4")
	var count int
	m.Range(func(key int, value string) bool {
		count++
		return true
	})
	assert.Equal(t, 2, count, "Expected 2 iterations")
}
