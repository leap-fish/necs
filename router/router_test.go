package router_test

import (
	"github.com/leap-fish/necs/router"
	"testing"

	"github.com/stretchr/testify/assert"
)

type ExampleChatMessage struct {
	message string
}

func Test_RouterOn(t *testing.T) {
	cm := ExampleChatMessage{message: "Goldroger - Perwoll"}

	var called bool
	router.On[ExampleChatMessage](func(sender *router.NetworkClient, message ExampleChatMessage) {
		called = true
	})

	serialized, err := router.Serialize(cm)
	assert.Nil(t, err)
	assert.NotNil(t, serialized)
	assert.Len(t, serialized, 10)

	assert.False(t, called)

	err = router.ProcessMessage(&router.NetworkClient{}, serialized)
	assert.Nil(t, err)
	// TODO: If ProcessMessage's Call() is done in a goroutine, this does not pass
	assert.True(t, called)
}

func BenchmarkRouter_ProcessMessage(b *testing.B) {
	router.ResetRouter()
	cm := ExampleChatMessage{message: "Goldroger - Perwoll"}
	for i := 0; i < 1; i++ {
		router.On[ExampleChatMessage](func(sender *router.NetworkClient, message ExampleChatMessage) {
		})
	}
	serialized, _ := router.Serialize(cm)

	for i := 0; i < b.N; i++ {
		_ = router.ProcessMessage(&router.NetworkClient{}, serialized)
	}
}
