package router

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"reflect"
	"sync"

	"github.com/coder/websocket"
	"github.com/leap-fish/necs/typeid"
	"github.com/leap-fish/necs/typemapper"
)

var (
	ErrCallbackNotRegistered = errors.New("callback type not registered")
	ErrMessageNotRegistered  = errors.New("message type is not registered")

	mapper = typemapper.NewMapper(map[uint]any{})

	// Connect and disconnect callback arrays are responsible for handling connect and disconnect events.
	// These are separate, because they do not take a dynamic type.
	connectCallbacks    []func(sender *NetworkClient)
	disconnectCallbacks []func(sender *NetworkClient, err error)
	errorCallbacks      []func(sender *NetworkClient, err error)

	callbacks = make(map[reflect.Type][]any)

	idMap          = make(map[*websocket.Conn]string)
	idMapMutex     sync.Mutex
	clientMap      = make(map[*websocket.Conn]*NetworkClient)
	clientMapMutex sync.Mutex
)

// On adds a callback to be called whenever the specified message type T is received.
// Note: sender will be nil in client callbacks.
// This can return an error if the type id is reserved or already in use.
func On[T any](callback func(sender *NetworkClient, message T)) {
	handlerType := reflect.TypeOf(callback).In(1)

	// Register the type in the type registry.
	id := typeid.GetTypeId(handlerType)

	// Error is ignored because it just means there is already a mapping with this type registered, so the mapper
	// does not want to register another one. Not an issue for this call.
	_ = mapper.RegisterType(id, handlerType)

	// Add the callback to the router.
	// So we can reference it when processing messages.
	callbacks[handlerType] = append(callbacks[handlerType], callback)
}

// OnConnect adds a callback to call whenever a session connects to the server.
// Note: sender will be nil in client callbacks.
func OnConnect(callback func(sender *NetworkClient)) {
	connectCallbacks = append(connectCallbacks, callback)
}

// OnDisconnect adds a callback to call whenever a session disconnects from the server.
// Note: sender will be nil in client callbacks.
func OnDisconnect(callback func(sender *NetworkClient, err error)) {
	disconnectCallbacks = append(disconnectCallbacks, callback)
}

// OnError adds a callback to call whenever a message error occurs.
// Note: sender will be nil in client callbacks.
func OnError(callback func(sender *NetworkClient, err error)) {
	errorCallbacks = append(errorCallbacks, callback)
}

// ProcessMessage deserializes a byte message and calls its registered callbacks.
func ProcessMessage(sender *NetworkClient, msg []byte) error {
	instance, err := mapper.Deserialize(msg)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrCallbackNotRegistered, err)
	}

	instanceType := reflect.TypeOf(instance)
	callbackList := callbacks[instanceType]

	if callbackList == nil {
		return fmt.Errorf("%w: %s", ErrMessageNotRegistered, instanceType)
	}

	arguments := []reflect.Value{reflect.ValueOf(sender), reflect.ValueOf(instance)}

	var localCallback reflect.Value
	for _, callback := range callbackList {
		localCallback = reflect.ValueOf(callback)
		// TODO: Make this a goroutine if we have enough handlers?
		localCallback.Call(arguments)
	}

	return nil
}

func Client(conn *websocket.Conn) *NetworkClient {
	clientMapMutex.Lock()
	defer clientMapMutex.Unlock()

	client, ok := clientMap[conn]
	if ok {
		return client
	}
	clientMap[conn] = NewNetworkClient(context.Background(), conn)
	return clientMap[conn]
}

func GetId(conn *websocket.Conn) string {
	idMapMutex.Lock()
	defer idMapMutex.Unlock()

	id, ok := idMap[conn]
	if ok {
		return id
	}

	bytes := make([]byte, 16)
	_, _ = rand.Read(bytes)
	id = fmt.Sprintf("%x", bytes[:10])

	idMap[conn] = id
	return id
}

// Peers returns a new slice of NetworkClient pointers from the underlying map.
// Use PeerMap if you are able to as this avoids this kind of duplication.
func Peers() []*NetworkClient {
	var peers []*NetworkClient

	clientMapMutex.Lock()
	defer clientMapMutex.Unlock()

	for _, v := range clientMap {
		peers = append(peers, v)
	}

	return peers
}

func Broadcast(msg any) error {
	payload, err := Serialize(msg)
	if err != nil {
		return err
	}

	for _, client := range Peers() {
		err := client.SendMessageBytes(payload)
		if err != nil {
			return err
		}
	}
	return nil
}

func Serialize(msg any) ([]byte, error) {
	msgType := reflect.TypeOf(msg)
	if mapper.LookupId(msgType) == 0 {
		id := typeid.GetTypeId(msgType)
		_ = mapper.RegisterType(id, msgType)
	}
	return mapper.Serialize(msg)
}

func CallProcessMessage(sender *websocket.Conn, msg []byte) error {
	return ProcessMessage(Client(sender), msg)
}

func CallConnect(sender *websocket.Conn) {
	client := Client(sender)
	for _, callback := range connectCallbacks {
		go callback(client)
	}
}

func CallDisconnect(sender *websocket.Conn, err error) {
	client := Client(sender)
	for _, callback := range disconnectCallbacks {
		go callback(client, err)
	}

	clientMapMutex.Lock()
	defer clientMapMutex.Unlock()

	delete(idMap, sender)
	delete(clientMap, sender)
}

func CallError(sender *websocket.Conn, err error) {
	client := Client(sender)
	for _, callback := range errorCallbacks {
		go callback(client, err)
	}
}

func ResetRouter() {
	mapper = typemapper.NewMapper(map[uint]any{})
	connectCallbacks = []func(sender *NetworkClient){}
	disconnectCallbacks = []func(sender *NetworkClient, err error){}
	errorCallbacks = []func(sender *NetworkClient, err error){}
	callbacks = make(map[reflect.Type][]any)

	clientMapMutex.Lock()
	defer clientMapMutex.Unlock()

	idMap = make(map[*websocket.Conn]string)
	clientMap = make(map[*websocket.Conn]*NetworkClient)
}
