package router

import (
	"context"
	"fmt"
	"nhooyr.io/websocket"
)

type NetworkClient struct {
	*websocket.Conn
	ctx context.Context
}

func NewNetworkClient(ctx context.Context, underlying *websocket.Conn) *NetworkClient {
	return &NetworkClient{
		Conn: underlying,
		ctx:  ctx,
	}
}

func (c *NetworkClient) SendMessage(msg any) error {
	payload, err := Serialize(msg)
	if err != nil {
		return fmt.Errorf("unable to serialize message: %w", err)
	}

	err = c.Conn.Write(c.ctx, websocket.MessageBinary, payload)
	if err != nil {
		return fmt.Errorf("unable to write message: %w", err)
	}

	return nil
}

func (c *NetworkClient) SendMessageBytes(msgBytes []byte) error {
	err := c.Conn.Write(c.ctx, websocket.MessageBinary, msgBytes)
	if err != nil {
		return err
	}

	return nil
}

func (c *NetworkClient) Id() string {
	return Id(c)
}
