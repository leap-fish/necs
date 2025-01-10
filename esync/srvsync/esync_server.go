package srvsync

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
	"slices"
	"sync"
	"sync/atomic"

	"github.com/leap-fish/necs/esync"
	"github.com/leap-fish/necs/router"
	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/component"
	"golang.org/x/sync/errgroup"
)

var NetworkIdCounter = atomic.Uint64{}

var syncEntities = map[donburi.Entity][]component.IComponentType{}

var syncEntMtx = sync.RWMutex{}
var stateMtx = sync.RWMutex{}

var filterFuncs []func(client *router.NetworkClient, entry *donburi.Entry) bool

var world donburi.World

func init() {
	_ = esync.RegisterComponent(1, esync.NetworkId(0), esync.NetworkIdComponent)
}

// UseEsync is used to set the world instance to use for synchronization.
func UseEsync(w donburi.World) {
	world = w
}

// AddNetworkFilter accepts a callback that can be used to filter out entities that gets included in the snapshots
// sent to clients. By returning false in this filter function, the entity will be excluded.
func AddNetworkFilter(filter func(client *router.NetworkClient, entry *donburi.Entry) bool) {
	filterFuncs = append(filterFuncs, filter)
}

// NetworkSync marks an entity and a list for network synchronization.
// This means that the esync package will automatically try to send state updates to the connected clients.
// Note that donburi tags are not supported for synchronization, as they contain no data.
// This will return an error if the entity does not have all the components being synced.
func NetworkSync(world donburi.World, entity *donburi.Entity, components ...donburi.IComponentType) error {
	// Increments the Network ID counter to prevent reusing the ids
	NetworkIdCounter.Add(1)

	networkId := NetworkIdCounter.Load()

	entry := world.Entry(*entity)
	entry.AddComponent(esync.NetworkIdComponent)
	esync.NetworkIdComponent.SetValue(entry, esync.NetworkId(networkId))

	for _, listComponent := range components {
		if !entry.HasComponent(listComponent) {
			return fmt.Errorf("entity %d does not have the component %s", entry.Id(), listComponent.Name())
		}
	}

	components = append(components, esync.NetworkIdComponent)

	syncEntMtx.Lock()
	defer syncEntMtx.Unlock()
	syncEntities[*entity] = components

	return nil
}

var syncMutex sync.Mutex

// DoSync should be called by the server and will build world state and then attempt to network it out to all the peers.
// This is done by serializing all the components of the entity, and preparing a network bundle for the clients.
func DoSync() error {
	errs, _ := errgroup.WithContext(context.Background())

	syncMutex.Lock()
	defer syncMutex.Unlock()

	for _, client := range router.Peers() {
		snapshot := buildSnapshot(client, world)
		errs.Go(func() error {
			err := client.SendMessage(snapshot)
			return err
		})
	}

	return errs.Wait()
}

func buildEntityState(entry *donburi.Entry) (esync.EntityState, error) {
	s := donburi.GetComponents(entry)

	componentMap := make(esync.EntityState)
	for _, ecsComponent := range s {
		t := reflect.TypeOf(ecsComponent)

		// Skip any tags or non-identifiable types.
		if t == reflect.TypeOf(struct{}{}) {
			continue
		}

		syncEntMtx.RLock()
		// Skip components not in the actual list
		validList := syncEntities[entry.Entity()]
		syncEntMtx.RUnlock()

		contains := slices.ContainsFunc(validList, func(componentType component.IComponentType) bool {
			return componentType.Typ() == t
		})
		if !contains {
			continue
		}

		id := esync.Mapper.LookupId(t)
		serializedComponent, err := esync.Mapper.Serialize(ecsComponent)
		if err != nil {
			return nil, err
		}

		componentMap[esync.ComponentId(id)] = bytes.Clone(serializedComponent)
	}

	return componentMap, nil
}

func buildSnapshot(client *router.NetworkClient, world donburi.World) esync.WorldSnapshot {
	state := esync.WorldSnapshot([]esync.SerializedEntity{})

	stateMtx.Lock()
	defer stateMtx.Unlock()
	esync.NetworkEntityQuery.Each(world, func(entry *donburi.Entry) {
		// Used to filter out data
		if len(filterFuncs) > 0 {
			for _, f := range filterFuncs {
				if !f(client, entry) {
					return // Filtered
				}
			}
		}

		componentMap, err := buildEntityState(entry)
		if err != nil {
			return
		}

		entityNetworkId := esync.GetNetworkId(entry)
		if entityNetworkId == nil {
			return
		}
		state = append(state, esync.SerializedEntity{Id: *entityNetworkId, State: componentMap})
	})

	return state
}
