package typemapper

import (
	"bytes"
	"fmt"
	"reflect"
	"sync"

	"github.com/hashicorp/go-msgpack/v2/codec"
)

// TypeMapper is used to map between registered IDs and components and
// translates between server and client components,
// Note: Unexported members are supported,
// however embedded members will not be populated if also unexported.
type TypeMapper struct {
	typeToId map[reflect.Type]uint
	idToType map[uint]reflect.Type

	mapMutex sync.Mutex

	handle *codec.MsgpackHandle
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

	return cdb
}

// RegisterType registers a mapping based on ID and reflect.Type.
func (db *TypeMapper) RegisterType(id uint, componentType reflect.Type) error {

	if db.idToType[id] != nil {
		return fmt.Errorf("cannot register mapping for component %s with id %d because it is reserved", componentType, id)
	}

	db.mapMutex.Lock()
	defer db.mapMutex.Unlock()

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

	db.mapMutex.Lock()
	defer db.mapMutex.Unlock()

	db.typeToId[typeof] = id
	db.idToType[id] = typeof

	return nil
}

// Lookup finds the Type based on a component ID.
func (db *TypeMapper) Lookup(id uint) reflect.Type {
	db.mapMutex.Lock()
	defer db.mapMutex.Unlock()

	return db.idToType[id]
}

// LookupId finds the component ID from a Type.
func (db *TypeMapper) LookupId(componentType reflect.Type) uint {
	db.mapMutex.Lock()
	defer db.mapMutex.Unlock()

	return db.typeToId[componentType]
}

// Serialize a component to bytes that can be networked.
func (db *TypeMapper) Serialize(component any) ([]byte, error) {
	componentType := reflect.TypeOf(component)
	id := db.LookupId(componentType)
	if id == 0 {
		return nil, fmt.Errorf("component ID not found for type %s; ensure it is registered with the component typemapper", componentType)
	}

	encodeBuf := &bytes.Buffer{}

	encoder := codec.NewEncoder(encodeBuf, db.handle)

	if err := encoder.Encode(id); err != nil {
		return nil, err
	}

	// if customEncoder, ok := component.(EncodeDecoder); ok {
	// 	encoded, err := customEncoder.Encode()
	// 	if err != nil {
	// 		return nil, err
	// 	}

	// 	// if _, err := encodeBuf.Write(encoded); err != nil {
	// 	// 	return nil, err
	// 	// }
	// 	if err := encoder.Encode(encoded); err != nil {
	// 		return nil, err
	// 	}

	// 	fmt.Printf("message: %#v\n", encodeBuf.Bytes())
	// } else {
	// }
	if err := encoder.Encode(component); err != nil {
		return nil, err
	}

	return encodeBuf.Bytes(), nil
}

// Deserialize a component by decoding its ID, and then the actual struct.
func (db *TypeMapper) Deserialize(data []byte) (any, error) {
	// buf := bytes.NewBuffer(data)
	// decoder := codec.NewDecoder(buf, db.handle)
	decoder := codec.NewDecoderBytes(data, db.handle)

	var id uint
	if err := decoder.Decode(&id); err != nil {
		return nil, err
	}

	component := db.Lookup(id)
	if component == nil {
		return nil, fmt.Errorf("component type not found for ID %d", id)
	}

	instanced := reflect.New(component).Interface()
	// if customDecoder, ok := instanced.(EncodeDecoder); ok {
	// 	var remaining []byte
	// 	if err := decoder.Decode(remaining); err != nil {
	// 		return nil, err
	// 	}
	// 	// remaining, err := io.ReadAll(buf)
	// 	// if err != nil {
	// 	// 	fmt.Printf("read all buf: %s\n", err)
	// 	// 	return nil, err
	// 	// }

	// 	if err := customDecoder.Decode(remaining); err != nil {
	// 		return nil, err
	// 	}

	// 	fmt.Printf("instance: %v\n", instanced)
	// } else {
	// }
	if err := decoder.Decode(instanced); err != nil {
		return nil, err
	}

	value := reflect.ValueOf(instanced).Elem().Interface()
	return value, nil
}
