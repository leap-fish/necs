package wrapws

import (
	"context"
	"io"
	"nhooyr.io/websocket"
	"time"
)

type WebSocketClient struct {
	handler EventHandler
}

func NewWebSocketClient(handler EventHandler) *WebSocketClient {
	return &WebSocketClient{
		handler: handler,
	}
}

func (ws *WebSocketClient) Dial(addr string, options *websocket.DialOptions, callback func(conn *websocket.Conn)) error {
	ctx := context.Background()

	conn, _, err := websocket.Dial(ctx, addr, options)
	if err != nil {
		return err
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	if callback != nil {
		callback(conn)
	}

	// Connect callback
	ws.handler.OnConnect(ctx, conn)

	_ = ws.readLoop(ctx, conn)

	err = conn.Close(websocket.StatusNormalClosure, "")
	ws.handler.OnDisconnect(ctx, conn, err)

	return nil
}

func (ws *WebSocketClient) readLoop(ctx context.Context, conn *websocket.Conn) error {
	ctx, cancel := context.WithTimeout(ctx, maxConnectionTime)
	defer cancel()

	for {
		payload, err := ws.read(ctx, conn)
		if err != nil {
			ws.handler.OnError(ctx, conn, err)
			return err
		}

		ws.handler.OnMessage(ctx, conn, payload)
	}
}

func (ws *WebSocketClient) read(ctx context.Context, c *websocket.Conn) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, idleTimeout)
	defer cancel()

	_, r, err := c.Reader(ctx)
	if err != nil {
		return nil, err
	}

	time.AfterFunc(maxMessageReadTime, cancel)

	return io.ReadAll(r)
}
