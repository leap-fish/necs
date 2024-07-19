package shared

import "github.com/yohamta/donburi"

type TestMessage struct {
	Message string
}

var TestMessageComponent = donburi.NewComponentType[TestMessage]()
