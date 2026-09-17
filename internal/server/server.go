package server

import (
	"context"
	"encoding/json"
	"io/fs"
	"math/rand/v2"
	"net/http"

	"github.com/coder/websocket"
	"github.com/gabrielmgaa/coup/internal/protocol"
)

func New(site fs.FS, rng *rand.Rand, initialCoins int) http.Handler {
	room := newRoom(rng, initialCoins)
	go room.run()

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", room.accept)
	mux.Handle("/", http.FileServerFS(site))
	return mux
}

func (r *Room) accept(w http.ResponseWriter, request *http.Request) {
	conn, err := websocket.Accept(w, request, nil)
	if err != nil {
		return
	}
	ctx, shutDown := context.WithCancel(context.Background())
	defer shutDown()

	c := &connection{outbox: make(chan []byte, outboxCapacity)}
	go write(ctx, conn, c)
	r.read(ctx, conn, c)
}

func (r *Room) read(ctx context.Context, conn *websocket.Conn, c *connection) {
	defer conn.CloseNow()
	for {
		_, encoded, err := conn.Read(ctx)
		if err != nil {
			r.inbox <- command{from: c, message: protocol.FromClient{Type: "leave"}}
			return
		}
		var message protocol.FromClient
		if err := json.Unmarshal(encoded, &message); err != nil {
			message = protocol.FromClient{Type: "unreadable_json"}
		}
		r.inbox <- command{from: c, message: message}
	}
}

func write(ctx context.Context, conn *websocket.Conn, c *connection) {
	for encoded := range c.outbox {
		if err := conn.Write(ctx, websocket.MessageText, encoded); err != nil {
			return
		}
	}
	conn.Close(websocket.StatusPolicyViolation, "client too far behind")
}
