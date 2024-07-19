package main

import (
	"log"

	"github.com/leap-fish/necs/examples/shared"
	"github.com/leap-fish/necs/router"
	"github.com/leap-fish/necs/transports"
)

func main() {
	router.OnConnect(func(sender *router.NetworkClient) {
		log.Println("Client has connected to the server!")

		_ = sender.SendMessage(shared.TestMessage{Message: "Hello from server"})
	})

	router.OnDisconnect(func(sender *router.NetworkClient, err error) {
		log.Println("Client has disconnected!")
	})

	router.On(func(sender *router.NetworkClient, message shared.TestMessage) {
		log.Println("Testmessage: ", message)
	})

	router.OnError(func(sender *router.NetworkClient, err error) {
		log.Printf("Message Error: %s", err.Error())
	})

	server := transports.NewWsServerTransport(7373, "", nil)
	err := server.Start()
	if err != nil {
		log.Fatalf("Unable to dial: %s", err)
	}
}
