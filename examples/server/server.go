package main

import (
	"github.com/leap-fish/necs/router"
	"github.com/leap-fish/necs/transports"
	"log"
)

type TestMessage struct {
	Message string
}

func main() {
	router.OnConnect(func(sender *router.NetworkClient) {
		log.Println("Client has connected to the server!")

		_ = sender.SendMessage(TestMessage{Message: "Hello from server"})
	})

	router.OnDisconnect(func(sender *router.NetworkClient, err error) {
		log.Println("Client has disconnected!")
	})

	router.On[TestMessage](func(sender *router.NetworkClient, message TestMessage) {
		log.Println("Testmessage: ", message)
	})

	server := transports.NewWsServerTransport(7373, "", nil)
	err := server.Start()
	if err != nil {
		log.Fatalf("Unable to dial: %s", err)
	}

}
