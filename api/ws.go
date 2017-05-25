package api

import (
	"github.com/cpacia/ens-lite"
	"github.com/gorilla/websocket"
	"net/http"
	"time"
	"fmt"
)

type connection struct {
	// The websocket connection
	ws *websocket.Conn

	// Buffered channel of outbound messages
	send chan []byte

	// The hub
	h *hub
}

func (c *connection) reader() {
	for {
		_, message, err := c.ws.ReadMessage()
		if err != nil {
			break
		}

		// Just echo for now until we set up the API
		c.h.Broadcast <- message
	}
	c.ws.Close()
}

func (c *connection) writer() {
	for message := range c.send {
		err := c.ws.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			break
		}
	}
	c.ws.Close()
}

var upgrader = &websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

var handler wsHandler

type wsHandler struct {
	h      *hub
	client *ens.ENSLiteClient
}

func newWSAPIHandler(ensClient *ens.ENSLiteClient) *wsHandler {
	hub := newHub()
	go hub.run()
	handler = wsHandler{
		h:      hub,
		client: ensClient,
	}
	go handler.serveSyncProgress()
	return &handler
}

func (wsh wsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	c := &connection{send: make(chan []byte, 256), ws: ws, h: wsh.h}
	c.h.register <- c
	defer func() { c.h.unregister <- c }()
	go c.writer()
	c.reader()
}

func (wsh wsHandler) serveSyncProgress() {
	time.Sleep(time.Second*5)
	t := time.NewTicker(time.Millisecond * 100)
	for range t.C {
		sp, err := wsh.client.SyncProgress()
		if err != nil && err == ens.ErrorNodeInitializing {
			wsh.h.Broadcast <- []byte("Node initializing...")
			continue
		} else if sp == nil {
			wsh.h.Broadcast <- []byte("Fully synced")
			return
		}
		start := sp.StartingBlock
		highest := sp.HighestBlock
		current := sp.CurrentBlock

		total := highest - start
		downloaded := current - start

		progress := float64(downloaded) / float64(total)
		wsh.h.Broadcast <- []byte(fmt.Sprintf(`%.2f`, progress))
	}
}