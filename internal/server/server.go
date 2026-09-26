package server

import (
	"context"
	"encoding/json"
	"io/fs"
	"math/rand/v2"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
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

type Config struct {
	InitialCoins int
	Deadline     time.Duration
	Grace        time.Duration
	IdleTTL      time.Duration
	Handshake    time.Duration
	PingEvery    time.Duration
	PongWait     time.Duration
	WriteWait    time.Duration
}

func DefaultConfig(initialCoins int) Config {
	return Config{
		InitialCoins: initialCoins,
		Deadline:     25 * time.Second,
		Grace:        30 * time.Second,
		IdleTTL:      30 * time.Minute,
		Handshake:    10 * time.Second,
		PingEvery:    15 * time.Second,
		PongWait:     10 * time.Second,
		WriteWait:    10 * time.Second,
	}
}

type registry struct {
	mu      sync.Mutex
	rooms   map[string]*Room
	rng     *rand.Rand
	config  Config
	newCode func() string
}

func newRegistry(rng *rand.Rand, config Config) *registry {
	desk := &registry{rooms: map[string]*Room{}, rng: rng, config: config}
	desk.newCode = desk.drawCode
	return desk
}

func New(site fs.FS, rng *rand.Rand, config Config) http.Handler {
	desk := newRegistry(rng, config)

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
		if refusal := checkName(first.Name); refusal != nil {
			return nil, refusal
		}
		opened := newRoom(d.seedRoom(), d.config, d.freeCode(), d.forget)
		opened.options = first.Options
		d.rooms[opened.code] = opened
		go opened.run()
		return opened, nil
	case "join":
		if refusal := checkName(first.Name); refusal != nil {
			return nil, refusal
		}
		return d.existing(first.Room)
	case "reconnect":
		return d.existing(first.Room)
	}
	return nil, &engine.Refusal{Code: "illegal_action",
		Message:  "a primeira mensagem precisa abrir uma sala ou entrar numa",
		Received: first.Type, Expected: []string{"create_room", "join", "reconnect"}}
}

func (d *registry) existing(code string) (*Room, *engine.Refusal) {
	waiting, known := d.rooms[code]
	if !known {
		return nil, roomNotFound(code)
	}
	return waiting, nil
}

func roomNotFound(code string) *engine.Refusal {
	return &engine.Refusal{Code: "room_not_found", Message: "não existe sala com esse código",
		Received: code, Expected: "o código de uma sala aberta"}
}

func (d *registry) forget(code string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.rooms, code)
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

	first, err := readFirst(ctx, conn, d.config.Handshake)
	if err != nil {
		refuseAndClose(ctx, conn, &engine.Refusal{Code: "illegal_action",
			Message:  "a primeira mensagem não pôde ser lida",
			Received: err.Error(), Expected: []string{"create_room", "join", "reconnect"}})
		return
	}
	opened, refusal := d.roomFor(first)
	if refusal != nil {
		refuseAndClose(ctx, conn, refusal)
		return
	}

	c := &connection{outbox: make(chan []byte, outboxCapacity)}
	if !opened.deliver(command{from: c, message: admission(first)}) {
		refuseAndClose(ctx, conn, roomNotFound(first.Room))
		return
	}
	go write(ctx, conn, c, d.config.WriteWait)
	go keepAlive(ctx, conn, d.config.PingEvery, d.config.PongWait)
	opened.read(ctx, conn, c)
}

func admission(first protocol.FromClient) protocol.FromClient {
	if first.Type == "reconnect" {
		return protocol.FromClient{Type: "reconnect", Token: first.Token}
	}
	return protocol.FromClient{Type: "join", Name: strings.TrimSpace(first.Name)}
}

func checkName(raw string) *engine.Refusal {
	size := utf8.RuneCountInString(strings.TrimSpace(raw))
	if size < minNameLength || size > maxNameLength {
		return &engine.Refusal{Code: "invalid_name", Message: "nome fora do tamanho permitido",
			Received: raw, Expected: "2 a 16 caracteres"}
	}
	if strings.ContainsFunc(raw, func(letter rune) bool { return !unicode.IsPrint(letter) && letter != ' ' }) {
		return &engine.Refusal{Code: "invalid_name", Message: "nome com caractere que não se imprime",
			Received: strconv.QuoteToASCII(raw), Expected: "letras, números, espaços e sinais"}
	}
	return nil
}

func readFirst(ctx context.Context, conn *websocket.Conn, patience time.Duration) (protocol.FromClient, error) {
	ctx, stop := context.WithTimeout(ctx, patience)
	defer stop()
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
			r.deliver(command{from: c, message: protocol.FromClient{Type: "leave"}})
			return
		}
		var message protocol.FromClient
		if err := json.Unmarshal(encoded, &message); err != nil {
			message = protocol.FromClient{Type: "unreadable_json"}
		}
		if !r.deliver(command{from: c, message: message}) {
			return
		}
	}
}

func write(ctx context.Context, conn *websocket.Conn, c *connection, patience time.Duration) {
	for encoded := range c.outbox {
		if err := writeWithin(ctx, conn, encoded, patience); err != nil {
			conn.CloseNow()
			return
		}
	}
	conn.Close(websocket.StatusPolicyViolation, "client too far behind")
}

func writeWithin(ctx context.Context, conn *websocket.Conn, encoded []byte, patience time.Duration) error {
	ctx, stop := context.WithTimeout(ctx, patience)
	defer stop()
	return conn.Write(ctx, websocket.MessageText, encoded)
}

func keepAlive(ctx context.Context, conn *websocket.Conn, every, patience time.Duration) {
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !answersPing(ctx, conn, patience) {
				conn.CloseNow()
				return
			}
		}
	}
}

func answersPing(ctx context.Context, conn *websocket.Conn, patience time.Duration) bool {
	ctx, stop := context.WithTimeout(ctx, patience)
	defer stop()
	return conn.Ping(ctx) == nil
}
