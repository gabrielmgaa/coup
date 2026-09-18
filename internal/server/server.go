package server

import (
	"context"
	"encoding/json"
	"io/fs"
	"math/rand/v2"
	"net/http"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/coder/websocket"
	"github.com/gabrielmgaa/coup/internal/engine"
	"github.com/gabrielmgaa/coup/internal/protocol"
)

const (
	codeAlphabet  = "23456789ABCDEFGHJKLMNPQRSTUVWXYZ"
	codeLength    = 4
	minNameLength = 2
	maxNameLength = 16
)

type registry struct {
	mu           sync.Mutex
	rooms        map[string]*Room
	rng          *rand.Rand
	initialCoins int
	newCode      func() string
}

func newRegistry(rng *rand.Rand, initialCoins int) *registry {
	desk := &registry{rooms: map[string]*Room{}, rng: rng, initialCoins: initialCoins}
	desk.newCode = desk.drawCode
	return desk
}

func New(site fs.FS, rng *rand.Rand, initialCoins int) http.Handler {
	desk := newRegistry(rng, initialCoins)

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", desk.accept)
	mux.Handle("/", http.FileServerFS(site))
	return mux
}

func (d *registry) roomFor(first protocol.FromClient) (*Room, *engine.Refusal) {
	d.mu.Lock()
	defer d.mu.Unlock()

	switch first.Type {
	case "create_room":
		opened := newRoom(d.seedRoom(), d.initialCoins, d.freeCode())
		d.rooms[opened.code] = opened
		go opened.run()
		return opened, nil
	case "join":
		waiting, known := d.rooms[first.Room]
		if !known {
			return nil, &engine.Refusal{Code: "room_not_found",
				Message:  "não existe sala com esse código",
				Received: first.Room, Expected: "o código de uma sala aberta"}
		}
		return waiting, nil
	}
	return nil, &engine.Refusal{Code: "illegal_action",
		Message:  "a primeira mensagem precisa abrir uma sala ou entrar numa",
		Received: first.Type, Expected: []string{"create_room", "join"}}
}

func (d *registry) freeCode() string {
	for {
		drawn := d.newCode()
		if _, taken := d.rooms[drawn]; !taken {
			return drawn
		}
	}
}

func (d *registry) drawCode() string {
	drawn := make([]byte, codeLength)
	for position := range drawn {
		drawn[position] = codeAlphabet[d.rng.IntN(len(codeAlphabet))]
	}
	return string(drawn)
}

func (d *registry) seedRoom() *rand.Rand {
	return rand.New(rand.NewPCG(d.rng.Uint64(), d.rng.Uint64()))
}

func (d *registry) accept(w http.ResponseWriter, request *http.Request) {
	conn, err := websocket.Accept(w, request, nil)
	if err != nil {
		return
	}
	ctx, shutDown := context.WithCancel(context.Background())
	defer shutDown()

	first, err := readOne(ctx, conn)
	if err != nil {
		refuseAndClose(ctx, conn, &engine.Refusal{Code: "illegal_action",
			Message:  "a primeira mensagem não pôde ser lida",
			Received: err.Error(), Expected: []string{"create_room", "join"}})
		return
	}
	if refusal := checkName(first.Name); refusal != nil {
		refuseAndClose(ctx, conn, refusal)
		return
	}
	opened, refusal := d.roomFor(first)
	if refusal != nil {
		refuseAndClose(ctx, conn, refusal)
		return
	}

	c := &connection{outbox: make(chan []byte, outboxCapacity)}
	go write(ctx, conn, c)
	opened.inbox <- command{from: c,
		message: protocol.FromClient{Type: "join", Name: strings.TrimSpace(first.Name)}}
	opened.read(ctx, conn, c)
}

func checkName(raw string) *engine.Refusal {
	size := utf8.RuneCountInString(strings.TrimSpace(raw))
	if size < minNameLength || size > maxNameLength {
		return &engine.Refusal{Code: "invalid_name", Message: "nome fora do tamanho permitido",
			Received: raw, Expected: "2 a 16 caracteres"}
	}
	return nil
}

func readOne(ctx context.Context, conn *websocket.Conn) (protocol.FromClient, error) {
	_, encoded, err := conn.Read(ctx)
	if err != nil {
		return protocol.FromClient{}, err
	}
	var message protocol.FromClient
	if err := json.Unmarshal(encoded, &message); err != nil {
		return protocol.FromClient{}, err
	}
	return message, nil
}

func refuseAndClose(ctx context.Context, conn *websocket.Conn, refusal *engine.Refusal) {
	if encoded, err := json.Marshal(protocol.NewRefusal(refusal)); err == nil {
		_ = conn.Write(ctx, websocket.MessageText, encoded)
	}
	conn.Close(websocket.StatusPolicyViolation, refusal.Code)
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
