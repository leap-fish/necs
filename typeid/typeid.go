package typeid

import (
	"hash/fnv"
	"reflect"
)

// GetTypeId returns a hash based on the reflected type.
// This is used to ensure consistent mappings across binaries.
func GetTypeId(t reflect.Type) uint {
	h := fnv.New64a()
	_, _ = h.Write([]byte(t.String()))
	return uint(h.Sum64())
}
