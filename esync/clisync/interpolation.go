package clisync

import (
	"fmt"
	"math"
	"reflect"
	"time"
	"unsafe"

	"github.com/leap-fish/necs/esync"
	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/ecs"
	"github.com/yohamta/donburi/filter"
)

var (
	timeCacheComponent = donburi.NewComponentType[timeCacheData]()
)

type componentTimeData struct {
	value any
	ts    time.Time
}

// timeCacheData contains a map which the key matches an interpolation
// component key and contains a list of historic values for that type.
type timeCacheData struct {
	// map[component key]historic values
	history [math.MaxUint8][]componentTimeData
}

var (
	requests     int64
	totalLatency int64
	avgLatency   float64
	delay        int
	lastSnapshot time.Time = time.Now()
)

// calculateDelay sets the delay which is an index based on the average latency in seconds
func calculateDelay(now time.Time) {
	requests++

	totalLatency += int64(time.Since(lastSnapshot))
	avgLatency = float64(totalLatency) / float64(requests)

	delay = int(math.Floor(avgLatency / float64(time.Second)))
	lastSnapshot = now
}

// NewInterpolateSystem returns an ecs system that should be registered if you
// have any client-side interpolating components.
func NewInterpolateSystem() ecs.System {
	query := donburi.NewQuery(filter.Contains(
		esync.NetworkIdComponent,
		esync.InterpComponent,
		timeCacheComponent,
	))

	return func(ecs *ecs.ECS) {
		now := time.Now()

		for e := range query.Iter(ecs.World) {
			if !e.Valid() {
				continue
			}

			multiHistory := timeCacheComponent.Get(e)
			interpolated := esync.InterpComponent.Get(e)

			// Loop through each of this entry's interpolated components and
			// interpolate them using their lerp functions.
			for _, key := range interpolated.ComponentKeys() {
				compType := esync.LookupInterpType(key)
				comp, ok := esync.Registered(compType)
				if !ok {
					panic(fmt.Sprintf("unregistered component %T", compType))
				}
				if !e.HasComponent(comp) {
					continue
				}

				var (
					prev, next, delayed *componentTimeData
				)

				// Get the historic buffer for this component type.
				buf := multiHistory.history[key]
				if len(buf) <= 1 {
					continue // to fix a rare panic we skip this
				}
				for i := len(buf) - 1; i >= 0; i-- {
					if buf[i].ts.Compare(now) <= 0 {
						if len(buf) <= i {
							continue
						}

						prev = &buf[i]

						if i > 0 {
							next = &buf[i-1]
							break
						}
					}
				}
				// delayed should be our latest component value given our average
				// latency delay (in seconds).
				delayed = &buf[max(0, len(buf)-1-delay)]

				if prev == nil {
					e.SetComponent(comp, unsafe.Pointer(&buf[0].value))
					continue
				}
				if next == nil {
					e.SetComponent(comp, unsafe.Pointer(&buf[len(buf)-1].value))
					continue
				}

				// Get the `t` value for our lerp function by getting the difference in
				// our prev position and average it by our average latency's position
				// compared to our next position.
				t := float64(now.Sub(prev.ts)) / float64(delayed.ts.Sub(next.ts))

				setter := esync.LookupInterpSetter(key)
				v := reflect.ValueOf(setter)
				values := v.Call([]reflect.Value{
					reflect.ValueOf(next.value),
					reflect.ValueOf(delayed.value),
					reflect.ValueOf(t),
				})

				// Return value from the setter should be the interpolated value
				// now set the component.
				got := values[0].UnsafePointer()
				e.SetComponent(comp, got)
			}
		}
	}
}
