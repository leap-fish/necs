package transports

import (
	"context"
	"github.com/leap-fish/necs/router"
	"github.com/leap-fish/necs/wrapws"
	"golang.org/x/sync/errgroup"
	"nhooyr.io/websocket"
	"time"
)

type WsClientTransport struct {
	dialAddress string

	client *wrapws.WebSocketClient
}

func NewWsClientTransport(dialAddress string) *WsClientTransport {
	return &WsClientTransport{
		dialAddress: dialAddress,

		client: wrapws.NewWebSocketClient(wsClientEventHandler{}),
	}
}

func (n *WsClientTransport) Start(callback func(conn *websocket.Conn)) error {
	errs, _ := errgroup.WithContext(context.Background())
	errs.Go(func() error {
		err := n.client.Dial(n.dialAddress, nil, callback)
		return err
	})

	return errs.Wait()
}

type wsClientEventHandler struct {
	deadline time.Duration
}

func (w wsClientEventHandler) OnConnect(ctx context.Context, conn *websocket.Conn) {
	router.CallConnect(conn)
}

func (w wsClientEventHandler) OnDisconnect(ctx context.Context, conn *websocket.Conn, err error) {
	router.CallDisconnect(conn, err)
}

func (w wsClientEventHandler) OnError(ctx context.Context, conn *websocket.Conn, err error) {
	router.CallError(conn, err)
}

func (w wsClientEventHandler) OnMessage(ctx context.Context, conn *websocket.Conn, payload []byte) {
	err := router.CallProcessMessage(conn, payload)
	if err != nil {
		router.CallError(conn, err)
	}
}

// https://github.com/nhooyr/websocket/issues/86
