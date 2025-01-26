package typemapper

import (
	"errors"
	"fmt"
	"reflect"
	"sync"

	"github.com/yohamta/donburi"
)

var (
	ErrNilLerpFunction       = errors.New("lerp function nil")
	ErrMalformedLerpFunction = errors.New("malformed lerp function")
)

type interpolatedComponentData struct {
	typ    donburi.IComponentType
	setter any
}

type ComponentMapper struct {
	lock sync.Mutex

	typeToId      map[reflect.Type]uint8
	idToComponent map[uint8]interpolatedComponentData
}

func NewComponentMapper() *ComponentMapper {
	return &ComponentMapper{
		typeToId:      make(map[reflect.Type]uint8),
		idToComponent: make(map[uint8]interpolatedComponentData),
	}
}

// RegisterInterpolatedComponent registers the given component and setter with
// the provided ID, note that these IDs don't interfere with the normal esync.Register
func (c *ComponentMapper) RegisterInterpolatedComponent(id uint8, comp donburi.IComponentType, lerp any) error {
	if lerp == nil {
		return fmt.Errorf("must provide lerp function: %w", ErrNilLerpFunction)
	}

	typ := reflect.TypeOf(lerp)
	if typ.Kind() != reflect.Func {
		return fmt.Errorf("lerp must be a function: %w", ErrMalformedLerpFunction)
	}
	if typ.NumIn() != 3 {
		return fmt.Errorf("lerp function must have 3 arguments: %w", ErrMalformedLerpFunction)
	}

	c.lock.Lock()
	defer c.lock.Unlock()

	c.idToComponent[id] = interpolatedComponentData{
		typ:    comp,
		setter: lerp,
	}
	c.typeToId[comp.Typ()] = id

	return nil
}

func (c *ComponentMapper) LookupSetter(id uint8) any {
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

func (c *ComponentMapper) RegisteredId(id uint8) bool {
	c.lock.Lock()
	defer c.lock.Unlock()

	_, ok := c.idToComponent[id]
	return ok
}

func (c *ComponentMapper) LookupType(id uint8) reflect.Type {
	c.lock.Lock()
	defer c.lock.Unlock()

	return c.idToComponent[id].typ.Typ()
}

func (c *ComponentMapper) LookupId(typ reflect.Type) uint8 {
	c.lock.Lock()
	defer c.lock.Unlock()

	return c.typeToId[typ]
}
