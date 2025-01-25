package esync

import (
	"reflect"
	"unsafe"

	"github.com/leap-fish/necs/typemapper"
	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"
)

var InterpComponent = donburi.NewComponentType[InterpData]()

type InterpData struct {
	Components []uint
}

func NewInterpData(components ...donburi.IComponentType) *InterpData {
	ids := []uint{}
	for i := range components {
		// typeof := reflect.TypeOf()
		ids = append(ids, interpolated.LookupId(components[i].Typ()))
	}

	return &InterpData{
		Components: ids,
	}
}

func (i *InterpData) ComponentKeys() []uint {
	return i.Components
}

type ComponentId uint
type NetworkId uint

type EntityState map[ComponentId][]byte
type SerializedEntity struct {
	Id    NetworkId
	State EntityState
}
type WorldSnapshot []SerializedEntity

// type LerpFn[T any] func(entry *donburi.Entry, from T, to T, delta float64)
type LerpFn[T any] func(from T, to T, delta float64) *T

var NetworkEntityQuery = donburi.NewQuery(filter.Contains(NetworkIdComponent))

var interpolated = typemapper.NewComponentMapper()
var registered = map[reflect.Type]donburi.IComponentType{}

var (
	Mapper             = typemapper.NewMapper(map[uint]any{})
	NetworkIdComponent = donburi.NewComponentType[NetworkId]()
)

func RegisterInterpolated[T any](id uint, comp *donburi.ComponentType[T], lerp ...LerpFn[T]) error {
	if len(lerp) == 0 {
		return interpolated.RegisterInterpolatedComponent(id, comp, nil)
	} else {
		return interpolated.RegisterInterpolatedComponent(id, comp, lerp[0])
	}
}

func LookupInterpId(typ reflect.Type) uint {
	return interpolated.LookupId(typ)
}

func LookupInterpType(id uint) reflect.Type {
	return interpolated.LookupType(id)
}

func LookupInterpSetter(id uint) any {
	return interpolated.LookupSetter(id)
}

func RegisteredInterpId(id uint) bool {
	return interpolated.RegisteredId(id)
}

func RegisteredInterpType(typ reflect.Type) bool {
	return interpolated.RegisteredType(typ)
}

func Registered(componentType reflect.Type) (donburi.IComponentType, bool) {
	ctype, ok := registered[componentType]
	return ctype, ok
}

// RegisterComponent registers a component for use with esync. Make sure the client and server have the same definition of components.
// Note that ID 1 is reserved for the NetworkId component used by esync.
func RegisterComponent(id uint, component any, ctype donburi.IComponentType) error {
	typ := reflect.TypeOf(component)
	err := Mapper.RegisterType(id, typ)
	if err != nil {
		return err
	}
	registered[typ] = ctype

	return nil
}

// FindByNetworkId performs an "Each" query over network entities to find one with a matching ID.
func FindByNetworkId(world donburi.World, networkId NetworkId) donburi.Entity {
	var found donburi.Entity
	NetworkEntityQuery.Each(world, func(entry *donburi.Entry) {
		id := GetNetworkId(entry)
		if id == nil || *id != networkId {
			return
		}

		found = entry.Entity()
	})

	return found
}

func GetNetworkId(entry *donburi.Entry) *NetworkId {
	if entry == nil {
		return nil
	}

	if !entry.Valid() {
		return nil
	}

	if !entry.HasComponent(NetworkIdComponent) {
		return nil
	}

	nid := NetworkIdComponent.Get(entry)
	return nid
}

func ComponentFromVal(ctype donburi.IComponentType, value interface{}) unsafe.Pointer {
	if reflect.TypeOf(value) != ctype.Typ() {
		panic("Type assertion failed")
	}
	newVal := reflect.New(ctype.Typ()).Elem()
	newVal.Set(reflect.ValueOf(value))
	ptr := unsafe.Pointer(newVal.UnsafeAddr())

	return ptr
}
