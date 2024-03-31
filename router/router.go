package router

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"github.com/leap-fish/necs/typeid"
	"github.com/leap-fish/necs/typemapper"
	"nhooyr.io/websocket"
	"reflect"
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

	connMap   = map[*websocket.Conn]string{}
	clientMap = map[*websocket.Conn]*NetworkClient{}
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
	if _, ok := clientMap[conn]; ok {
		return clientMap[conn]
	}

	clientMap[conn] = NewNetworkClient(context.Background(), conn)
	return clientMap[conn]
}

func Id(client *NetworkClient) string {

	if _, ok := connMap[client.Conn]; ok {
		return connMap[client.Conn]
	}

	bytes := make([]byte, 16)
	_, _ = rand.Read(bytes)
	id := fmt.Sprintf("%d:%x", len(connMap), bytes[:10])

	connMap[client.Conn] = id
	return id
}

func Peers() []*NetworkClient {
	peers := make([]*NetworkClient, 0, len(clientMap))
	for _, client := range clientMap {
		peers = append(peers, client)
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

	delete(connMap, sender)
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
}
