package clisync

import (
	"fmt"
	"github.com/leap-fish/necs/esync"
	"github.com/leap-fish/necs/router"
	log "github.com/sirupsen/logrus"
	"github.com/yohamta/donburi"
	"reflect"
)

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
	var ctypes = make([]donburi.IComponentType, 0)

	for _, componentData := range components {
		componentType := reflect.TypeOf(componentData)
		ctype, ok := esync.Registered(componentType)
		if !ok {
			log.Error("Missing esync registration for component: ", componentType)
			return
		}
		ctypes = append(ctypes, ctype)
	}

	entity := esync.FindByNetworkId(world, networkId)
	var entry *donburi.Entry
	if !world.Valid(entity) {
		entity = world.Create(ctypes...)
	}

	entry = world.Entry(entity)

	if entry != nil && world.Valid(entity) {
		for i := 0; i < len(components); i++ {
			data := components[i]
			if data == nil {
				panic("meow")
			}
			entry.SetComponent(ctypes[i], esync.ComponentFromVal(ctypes[i], data))
		}
	}
}

func RegisterClient(world donburi.World) {
	router.On[esync.WorldSnapshot](func(sender *router.NetworkClient, message esync.WorldSnapshot) {
		err := clientUpdateWorldState(world, message)
		if err != nil {
			log.
				WithError(err).
				Error("Could not deserialize component in world state")
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
