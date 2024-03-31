package typemapper_test

import (
	"github.com/leap-fish/necs/typemapper"
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

type SimpleValueOne uint
type SimpleValueTwo uint

type HealthComponent struct {
	Current uint8
	Max     uint8
}

type ComplexComponent struct {
	HealthComponent

	CustomData map[string]int
	Name       string
	Colliders  []ColliderComponent
}

type ColliderComponent struct {
	Size uint
}

var testComponentMapping = map[uint]any{
	1: HealthComponent{},
	2: ColliderComponent{},
	3: ComplexComponent{},
	4: SimpleValueOne(0),
	5: SimpleValueTwo(0),
}

func TestTypeMapper_Lookup(t *testing.T) {
	mapper := typemapper.NewMapper(testComponentMapping)

	health := HealthComponent{5, 10}
	collider := ColliderComponent{}

	lookup1 := mapper.Lookup(2)
	assert.Equal(t, lookup1, reflect.TypeOf(collider))

	lookup2 := mapper.Lookup(1)
	assert.Equal(t, lookup2, reflect.TypeOf(health))

	lookup3 := mapper.Lookup(99)
	assert.Equal(t, lookup3, nil)
}

func TestTypeMapper_SimpleObjectSerialization(t *testing.T) {
	mapper := typemapper.NewMapper(testComponentMapping)

	simple := SimpleValueOne(12)
	first, err := mapper.Serialize(simple)
	assert.Nil(t, err)
	assert.NotNil(t, first)

	firstDeserialized, err := mapper.Deserialize(first)
	assert.Nil(t, err)
	assert.NotNil(t, firstDeserialized)
	assert.IsType(t, simple, firstDeserialized)

	deserializedFirst := firstDeserialized.(SimpleValueOne)
	assert.Nil(t, err)
	assert.NotNil(t, deserializedFirst)
	assert.Equal(t, SimpleValueOne(12), deserializedFirst)

	simpleTwo := SimpleValueTwo(15)
	second, err := mapper.Serialize(simpleTwo)
	assert.Nil(t, err)
	assert.NotNil(t, second)

	secondDeserialized, err := mapper.Deserialize(second)
	assert.Nil(t, err)
	assert.NotNil(t, secondDeserialized)
	assert.IsType(t, simpleTwo, secondDeserialized)

	deserializedSecond := secondDeserialized.(SimpleValueTwo)
	assert.Nil(t, err)
	assert.NotNil(t, deserializedSecond)
	assert.Equal(t, SimpleValueTwo(15), deserializedSecond)
}

func TestTypeMapper_FullSerialization(t *testing.T) {
	mapper := typemapper.NewMapper(testComponentMapping)

	health := HealthComponent{5, 10}

	first, err := mapper.Serialize(health)
	assert.Nil(t, err)
	assert.NotNil(t, first)

	firstDeserialized, err := mapper.Deserialize(first)
	assert.Nil(t, err)
	assert.NotNil(t, firstDeserialized)
	assert.IsType(t, health, firstDeserialized)

	deserializedHealth := firstDeserialized.(HealthComponent)
	assert.Nil(t, err)
	assert.NotNil(t, deserializedHealth)
	assert.Equal(t, uint8(5), deserializedHealth.Current)
	assert.Equal(t, uint8(10), deserializedHealth.Max)
}

func TestTypeMapper_FullSerialization_ComplexObject(t *testing.T) {
	mapper := typemapper.NewMapper(testComponentMapping)

	complexComp := ComplexComponent{
		HealthComponent: HealthComponent{
			Current: 5,
			Max:     10,
		},
		Name:       "ichbingoldie",
		CustomData: make(map[string]int),
		Colliders: []ColliderComponent{
			{1},
			{5},
			{10},
		},
	}
	complexComp.CustomData["john"] = 199

	first, err := mapper.Serialize(complexComp)
	assert.Nil(t, err)
	assert.NotNil(t, first)

	firstDeserialized, err := mapper.Deserialize(first)
	assert.Nil(t, err)
	assert.NotNil(t, firstDeserialized)
	assert.IsType(t, complexComp, firstDeserialized)

	deserComplexComp := firstDeserialized.(ComplexComponent)
	assert.Nil(t, err)
	assert.NotNil(t, deserComplexComp)
	assert.Equal(t, uint8(5), deserComplexComp.Current)
	assert.Equal(t, uint8(10), deserComplexComp.Max)
	assert.Equal(t, "ichbingoldie", deserComplexComp.Name)
	assert.Equal(t, 199, deserComplexComp.CustomData["john"])
	assert.Len(t, deserComplexComp.Colliders, 3)
	assert.Equal(t, uint(1), deserComplexComp.Colliders[0].Size)
	assert.Equal(t, uint(5), deserComplexComp.Colliders[1].Size)
	assert.Equal(t, uint(10), deserComplexComp.Colliders[2].Size)
}

func BenchmarkTypeMapper_Deserialize(b *testing.B) {
	mapper := typemapper.NewMapper(testComponentMapping)
	health := HealthComponent{5, 10}
	first, _ := mapper.Serialize(health)

	for i := 0; i < b.N; i++ {
		_, _ = mapper.Deserialize(first)
	}
}

func BenchmarkTypeMapper_Serialize(b *testing.B) {
	mapper := typemapper.NewMapper(testComponentMapping)
	health := HealthComponent{5, 10}

	for i := 0; i < b.N; i++ {
		_, _ = mapper.Serialize(health)
	}
}
