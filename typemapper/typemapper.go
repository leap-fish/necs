package typemapper

import (
	"bytes"
	"fmt"
	"github.com/hashicorp/go-msgpack/codec"
	"reflect"
	"sync"
)

// TypeMapper is used to map between registered IDs and components and
// translates between server and client components,
// Note: Unexported members are supported,
// however embedded members will not be populated if also unexported.
type TypeMapper struct {
	typeToId map[reflect.Type]uint
	idToType map[uint]reflect.Type

	handle *codec.MsgpackHandle

	dec       *codec.Decoder
	decodeBuf *bytes.Buffer
	decLock   sync.Mutex

	enc       *codec.Encoder
	encodeBuf *bytes.Buffer
	encLock   sync.Mutex
}

// NewMapper initializes a type mapper.
// This is responsible for serialization/deserialization.
func NewMapper(components map[uint]any) TypeMapper {
	componentLen := len(components)
	typeToId := make(map[reflect.Type]uint, componentLen)
	idToType := make(map[uint]reflect.Type, componentLen)

	for id, instance := range components {
		typeof := reflect.TypeOf(instance)
		typeToId[typeof] = id
		idToType[id] = typeof
	}

	cdb := TypeMapper{
		typeToId: typeToId,
		idToType: idToType,
		handle:   &codec.MsgpackHandle{},
	}

	cdb.encodeBuf = &bytes.Buffer{}
	cdb.decodeBuf = &bytes.Buffer{}
	cdb.dec = codec.NewDecoder(cdb.decodeBuf, cdb.handle)
	cdb.enc = codec.NewEncoder(cdb.encodeBuf, cdb.handle)

	return cdb
}

// RegisterType registers a mapping based on ID and reflect.Type.
func (db *TypeMapper) RegisterType(id uint, componentType reflect.Type) error {

	if db.idToType[id] != nil {
		return fmt.Errorf("cannot register mapping for component %s with id %d because it is reserved", componentType, id)
	}

	db.typeToId[componentType] = id
	db.idToType[id] = componentType

	return nil
}

// Register registers a mapping based on ID and an instance of the type.
func (db *TypeMapper) Register(id uint, component any) error {
	typeof := reflect.TypeOf(component)

	if db.idToType[id] != nil {
		return fmt.Errorf("cannot register mapping for component %s with id %d because it is reserved", typeof, id)
	}

	db.typeToId[typeof] = id
	db.idToType[id] = typeof

	return nil
}

// Lookup finds the Type based on a component ID.
func (db *TypeMapper) Lookup(id uint) reflect.Type {
	return db.idToType[id]
}

// LookupId finds the component ID from a Type.
func (db *TypeMapper) LookupId(componentType reflect.Type) uint {
	return db.typeToId[componentType]
}

// Serialize a component to bytes that can be networked.
func (db *TypeMapper) Serialize(component any) ([]byte, error) {
	db.encLock.Lock()
	defer db.encLock.Unlock()

	componentType := reflect.TypeOf(component)
	id := db.LookupId(componentType)
	if id == 0 {
		return nil, fmt.Errorf("component ID not found for type %s; ensure it is registered with the component typemapper", componentType)
	}

	db.encodeBuf.Reset()

	if err := db.enc.Encode(id); err != nil {
		return nil, err
	}

	if err := db.enc.Encode(component); err != nil {
		return nil, err
	}

	return db.encodeBuf.Bytes(), nil
}

// Deserialize a component by decoding its ID, and then the actual struct.
func (db *TypeMapper) Deserialize(data []byte) (any, error) {
	db.decLock.Lock()
	defer db.decLock.Unlock()

	db.decodeBuf.Reset()
	db.decodeBuf.Write(data)

	var id uint
	if err := db.dec.Decode(&id); err != nil {
		return nil, err
	}

	component := db.Lookup(id)
	if component == nil {
		return nil, fmt.Errorf("component type not found for ID %d", id)
	}

	instanced := reflect.New(component).Interface()
	if err := db.dec.Decode(instanced); err != nil {
		return nil, err
	}

	value := reflect.ValueOf(instanced).Elem().Interface()
	return value, nil
}
