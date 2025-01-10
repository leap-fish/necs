package transports

import (
	"context"
	"fmt"
	"time"

	"github.com/coder/websocket"
	"github.com/leap-fish/necs/router"
	"github.com/leap-fish/necs/wrapws"
)

type WsServerTransport struct {
	Port    uint
	Address string

	server *wrapws.WebSocketServer
}

func NewWsServerTransport(port uint, Address string, options *websocket.AcceptOptions) *WsServerTransport {
	return &WsServerTransport{
		Port:    port,
		Address: Address,
		server:  wrapws.NewWebSocketServer(wsEventHandler{}, options),
	}
}

func (n *WsServerTransport) Start() error {
	err := n.server.Serve(fmt.Sprintf("%s:%d", n.Address, n.Port))
	if err != nil {
		return fmt.Errorf("could not start server transport: %w", err)
	}

	return nil
}

type wsEventHandler struct {
	deadline time.Duration
}

func (w wsEventHandler) OnConnect(ctx context.Context, conn *websocket.Conn) {
	router.CallConnect(conn)
}

func (w wsEventHandler) OnDisconnect(ctx context.Context, conn *websocket.Conn, err error) {
	router.CallDisconnect(conn, err)
}

func (w wsEventHandler) OnError(ctx context.Context, conn *websocket.Conn, err error) {
	router.CallError(conn, err)
}

func (w wsEventHandler) OnMessage(ctx context.Context, conn *websocket.Conn, payload []byte) {
	err := router.CallProcessMessage(conn, payload)
	if err != nil {
		router.CallError(conn, fmt.Errorf("unable to process message: %w", err))
	}
}
