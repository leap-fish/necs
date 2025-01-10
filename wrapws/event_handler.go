package wrapws

import (
	"context"

	"github.com/coder/websocket"
)

type EventHandler interface {
	OnConnect(ctx context.Context, conn *websocket.Conn)
	OnDisconnect(ctx context.Context, conn *websocket.Conn, err error)
	OnError(ctx context.Context, conn *websocket.Conn, err error)
	OnMessage(ctx context.Context, conn *websocket.Conn, payload []byte)
}
