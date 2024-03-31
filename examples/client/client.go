package main

import (
	"github.com/leap-fish/necs/router"
	"github.com/leap-fish/necs/transports"
	"log"
	"nhooyr.io/websocket"
)

type TestMessage struct {
	Message string
}

func main() {
	client := transports.NewWsClientTransport("ws://localhost:7373")

	router.OnConnect(func(sender *router.NetworkClient) {
		log.Println("Connected to the server!")
	})

	router.On[TestMessage](func(sender *router.NetworkClient, message TestMessage) {
		log.Println("Testmessage: ", message)
		sender.SendMessage(TestMessage{"This is from the client"})
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
