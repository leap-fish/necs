package clisync

import (
	"fmt"
	"reflect"
	"time"

	"github.com/leap-fish/necs/esync"
	"github.com/leap-fish/necs/router"
	"github.com/yohamta/donburi"
)

const MaxHistorySize = 32

func clientUpdateWorldState(world donburi.World, state esync.WorldSnapshot) error {
	for _, ent := range state {
		var components []any
		for componentId, componentBytes := range ent.State {
			instance, err := esync.Mapper.Deserialize(componentBytes)
			if err != nil {
				return fmt.Errorf("unable to deserialize component id: %d: %w", componentId, err)
			}
			components = append(components, instance)
		}
		// For entities that are in the world snapshot:
		applyEntityDiff(world, ent.Id, components)
	}

	return nil
}

func applyEntityDiff(world donburi.World, networkId esync.NetworkId, components []any) {
	ctypes := make([]donburi.IComponentType, 0)
	refTypes := make([]reflect.Type, len(components))

	for i, componentData := range components {
		componentType := reflect.TypeOf(componentData)
		ctype, ok := esync.Registered(componentType)
		if !ok {
			// TODO: Add back erroring here
			continue
		}

		ctypes = append(ctypes, ctype)
		refTypes[i] = componentType
	}

	entity := esync.FindByNetworkId(world, networkId)
	var entry *donburi.Entry
	if !world.Valid(entity) {
		entity = world.Create(ctypes...)
	}

	entry = world.Entry(entity)
	now := time.Now()

	// calculate average latency and the delay index
	calculateDelay(now)

	if entry != nil && world.Valid(entity) {
		interpolated := entry.HasComponent(esync.InterpComponent)

		for i := 0; i < len(components); i++ {
			data := components[i]
			if data == nil {
				panic("meow")
			}

			ok := esync.RegisteredInterpType(refTypes[i])
			if !ok || !interpolated {
				entry.SetComponent(ctypes[i], esync.ComponentFromVal(ctypes[i], data))
				continue
			}

			key := esync.LookupInterpId(refTypes[i])
			// Add the base value for this component if it doesn't have one
			if !entry.HasComponent(ctypes[i]) {
				entry.SetComponent(ctypes[i], ctypes[i].New())
			}

			// Add a component cache to keep track of historic values for this
			// interpolated component
			if !entry.HasComponent(timeCacheComponent) {
				donburi.Add(entry, timeCacheComponent, &timeCacheData{
					history: make(map[uint][]componentTimeData),
				})
			}

			// Append the new value to our historic cache with its associated
			// timestamp of when we received this
			multHistory := timeCacheComponent.Get(entry)
			multHistory.history[key] = append(multHistory.history[key], componentTimeData{
				value: data,
				ts:    now,
			})

			// Shift the positions if we've reached the limit
			if len(multHistory.history[key]) > MaxHistorySize {
				multHistory.history[key] = multHistory.history[key][1:]
			}
		}
	}
}

func RegisterClient(world donburi.World) {
	router.On[esync.WorldSnapshot](func(sender *router.NetworkClient, message esync.WorldSnapshot) {
		err := clientUpdateWorldState(world, message)
		if err != nil {
			panic(err)
			// TODO: Add back error handling here
		}

		// Removal of old entities
		esync.NetworkEntityQuery.Each(world, func(entry *donburi.Entry) {
			id := esync.GetNetworkId(entry)
			if id == nil {
				return
			}

			var found bool
			for _, entity := range message {
				if entity.Id == *id {
					found = true
				}
			}

			if !found {
				entry.Remove()
			}
		})
	})
}
