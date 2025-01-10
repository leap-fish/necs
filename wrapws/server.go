package wrapws

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/coder/websocket"
)

const maxMessageReadTime = time.Second * 30
const maxConnectionTime = time.Hour * 12
const idleTimeout = time.Second * 55
const maxPingTime = time.Minute * 3

type WebSocketServer struct {
	mux     *http.ServeMux
	options *websocket.AcceptOptions

	handler EventHandler
}

func NewWebSocketServer(handler EventHandler, options *websocket.AcceptOptions) *WebSocketServer {
	return &WebSocketServer{
		mux:     http.NewServeMux(),
		options: options,
		handler: handler,
	}
}

func (ws *WebSocketServer) Serve(addr string) error {
	ws.mux.HandleFunc("/", ws.acceptFunc)

	err := http.ListenAndServe(addr, ws.mux)
	if err != nil {
		return err
	}

	return nil
}
func (ws *WebSocketServer) acceptFunc(w http.ResponseWriter, req *http.Request) {
	conn, err := websocket.Accept(w, req, ws.options)
	if err != nil {
		return
	}

	if conn == nil {
		return
	}

	defer conn.CloseNow()

	ws.readLoop(req.Context(), conn)

	err = conn.Close(websocket.StatusNormalClosure, "")
	ws.handler.OnDisconnect(req.Context(), conn, err)
}

func (ws *WebSocketServer) readLoop(ctx context.Context, conn *websocket.Conn) error {
	ctx, cancel := context.WithTimeout(ctx, maxConnectionTime)
	defer cancel()

	// Connect callback
	ws.handler.OnConnect(ctx, conn)

	go func() {
		defer cancel()
		ws.pingLoop(ctx, conn)
	}()

	for {
		payload, err := ws.read(ctx, conn)
		if err != nil {
			ws.handler.OnError(ctx, conn, err)
			return err
		}

		ws.handler.OnMessage(ctx, conn, payload)
	}
}

func (ws *WebSocketServer) read(ctx context.Context, conn *websocket.Conn) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, maxConnectionTime)
	defer cancel()

	_, r, err := conn.Reader(ctx)
	if err != nil {
		return nil, err
	}

	time.AfterFunc(maxMessageReadTime, cancel)

	return io.ReadAll(r)
}

func (ws *WebSocketServer) pingLoop(ctx context.Context, conn *websocket.Conn) {
	t := time.NewTicker(maxPingTime)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		err := ws.ping(ctx, conn)
		if err != nil {
			// The connection has disconnected if ping errors
			// and everything will automatically tear down.
			return
		}
	}
}

func (ws *WebSocketServer) ping(ctx context.Context, conn *websocket.Conn) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	err := conn.Ping(ctx)
	return err
}
