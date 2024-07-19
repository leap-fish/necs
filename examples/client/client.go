package main

import (
	"log"

	"github.com/leap-fish/necs/examples/shared"
	"github.com/leap-fish/necs/router"
	"github.com/leap-fish/necs/transports"
	"nhooyr.io/websocket"
)

func main() {
	client := transports.NewWsClientTransport("ws://localhost:7373")

	router.OnConnect(func(sender *router.NetworkClient) {
		log.Println("Connected to the server!")
	})

	router.On(func(sender *router.NetworkClient, message shared.TestMessage) {
		log.Println("Testmessage: ", message)
		sender.SendMessage(shared.TestMessage{Message: "This is from the client"})
	})

	router.OnError(func(sender *router.NetworkClient, err error) {
		log.Printf("Message Error: %s", err.Error())
	})

	err := client.Start(func(conn *websocket.Conn) {
		// If you want to use the connection for other purposes, this is where you might want to
		// store it for later use.
		log.Println("Conn?")
	})
	if err != nil {
		log.Fatalf("Unable to dial: %s", err)
	}

}
