package esync

import (
	"github.com/leap-fish/necs/typeid"
	"github.com/leap-fish/necs/typemapper"
	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/filter"
	"reflect"
	"unsafe"
)

type ComponentId uint
type NetworkId uint

type EntityState map[ComponentId][]byte
type SerializedEntity struct {
	Id    NetworkId
	State EntityState
}
type WorldSnapshot []SerializedEntity

var NetworkEntityQuery = donburi.NewQuery(filter.Contains(NetworkIdComponent))

var registered = map[reflect.Type]donburi.IComponentType{}

var (
	Mapper             = typemapper.NewMapper(map[uint]any{})
	NetworkIdComponent = donburi.NewComponentType[NetworkId]()
)

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

// AutoRegister registers a component by using typeID instead of manually defined IDs.
// This is experimental, and can produce duplicates for some types.
func AutoRegister(component any, ctype donburi.IComponentType) error {
	typ := reflect.TypeOf(component)
	id := typeid.GetTypeId(typ)
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
