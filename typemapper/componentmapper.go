package typemapper

import (
	"reflect"
	"sync"

	"github.com/yohamta/donburi"
)

type interpolatedComponentData struct {
	typ    donburi.IComponentType
	setter any
}

type ComponentMapper struct {
	lock sync.Mutex

	typeToId      map[reflect.Type]uint
	idToComponent map[uint]interpolatedComponentData
}

func NewComponentMapper() *ComponentMapper {
	return &ComponentMapper{
		typeToId:      make(map[reflect.Type]uint),
		idToComponent: make(map[uint]interpolatedComponentData),
	}
}

// RegisterInterpolatedComponent registers the given component and setter with
// the provided ID, note that these IDs don't interfere with the normal esync.Register
func (c *ComponentMapper) RegisterInterpolatedComponent(id uint, comp donburi.IComponentType, lerp any) error {
	// if lerp == nil {
	// 	return fmt.Errorf("invalid lerp function")
	// }

	c.lock.Lock()
	defer c.lock.Unlock()

	c.idToComponent[id] = interpolatedComponentData{
		typ:    comp,
		setter: lerp,
	}
	c.typeToId[comp.Typ()] = id

	return nil
}

func (c *ComponentMapper) LookupSetter(id uint) any {
	c.lock.Lock()
	defer c.lock.Unlock()

	return c.idToComponent[id].setter
}

func (c *ComponentMapper) RegisteredType(typ reflect.Type) bool {
	c.lock.Lock()
	defer c.lock.Unlock()

	_, ok := c.typeToId[typ]
	return ok
}

func (c *ComponentMapper) RegisteredId(id uint) bool {
	c.lock.Lock()
	defer c.lock.Unlock()

	_, ok := c.idToComponent[id]
	return ok
}

func (c *ComponentMapper) LookupType(id uint) reflect.Type {
	c.lock.Lock()
	defer c.lock.Unlock()

	return c.idToComponent[id].typ.Typ()
}

func (c *ComponentMapper) LookupId(typ reflect.Type) uint {
	c.lock.Lock()
	defer c.lock.Unlock()

	return c.typeToId[typ]
}
