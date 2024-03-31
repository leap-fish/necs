package typeid_test

import (
	"github.com/leap-fish/necs/typeid"
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

type HealthComponent struct {
	Current uint8
	Max     uint8
}

type ComplexComponent struct {
	HealthComponent

	CustomData map[string]int
	Colliders  []ColliderComponent
}

type ColliderComponent struct {
	Size uint
}

type SimpleOne uint
type SimpleTwo uint

type Test struct {
}

type testTuple struct {
	instance any
	expected uint
}

func TestGetTypeId_IsConsistent(t *testing.T) {
	checkTypes := []testTuple{
		{Test{}, 16851497653430128169},
		{Test{}, 16851497653430128169},
		{ColliderComponent{}, 13490843505570961692},
		{ComplexComponent{}, 4386901356493381958},
		{HealthComponent{}, 10689379905657179838},
		{SimpleOne(444), 0xa09fb4b3f816b311},
		{SimpleTwo(5659), 0xe8446db38ef98a1b},
	}

	type equalCheck struct {
		a any
		b any
	}
	
	preventEqual := []equalCheck{
		{a: SimpleOne(22), b: SimpleTwo(5)},
	}

	for _, checkType := range preventEqual {
		idA := typeid.GetTypeId(reflect.TypeOf(checkType.a))
		idB := typeid.GetTypeId(reflect.TypeOf(checkType.b))
		assert.NotEqual(t, idA, idB)
	}

	for _, checkType := range checkTypes {
		id := typeid.GetTypeId(reflect.TypeOf(checkType.instance))
		assert.Equal(t, checkType.expected, id)
	}

}
