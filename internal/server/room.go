package server

import (
	"encoding/json"
	"errors"
	"log"
	"math/rand/v2"

	"github.com/gabrielmgaa/coup/internal/engine"
	"github.com/gabrielmgaa/coup/internal/protocol"
)

const (
	outboxCapacity = 16
	playersToStart = 2
)

type connection struct {
	name   string
	ready  bool
	outbox chan []byte
	closed bool
}

func (c *connection) drop() {
	if c.closed {
		return
	}
	c.closed = true
	close(c.outbox)
}

type command struct {
	from    *connection
	message protocol.FromClient
}

type Room struct {
	inbox        chan command
	connections  []*connection
	game         *engine.Game
	rng          *rand.Rand
	initialCoins int
	code         string
}

func newRoom(rng *rand.Rand, initialCoins int, code string) *Room {
	return &Room{inbox: make(chan command), rng: rng, initialCoins: initialCoins, code: code}
}

func (r *Room) run() {
	for received := range r.inbox {
		switch received.message.Type {
		case "join":
			r.join(received)
		case "ready":
			r.markReady(received)
		case "start":
			r.start(received)
		case "leave":
			r.remove(received.from)
		default:
			r.play(received)
		}
	}
}

func (r *Room) join(received command) {
	if r.game != nil {
		r.turnAway(received.from, &engine.Refusal{Code: "game_started",
			Message:  "a partida já começou",
			Received: received.message.Name, Expected: "uma sala que ainda não começou"})
		return
	}
	if len(r.connections) >= engine.MaxPlayers {
		r.turnAway(received.from, &engine.Refusal{Code: "room_full", Message: "a sala está cheia",
			Received: len(r.connections) + 1, Expected: engine.MaxPlayers})
		return
	}
	for _, seated := range r.connections {
		if seated.name == received.message.Name {
			r.turnAway(received.from, &engine.Refusal{Code: "name_taken",
				Message:  "já tem alguém com esse nome na sala",
				Received: received.message.Name, Expected: "um nome ainda não usado nesta sala"})
			return
		}
	}
	received.from.name = received.message.Name
	r.connections = append(r.connections, received.from)
	r.broadcast(nil)
}

func (r *Room) markReady(received command) {
	if r.game != nil {
		r.refuse(received.from, &engine.Refusal{Code: "game_started",
			Message:  "a partida já começou",
			Received: "ready", Expected: "uma sala que ainda não começou"})
		return
	}
	received.from.ready = received.message.Ready
	r.broadcast(nil)
}

func (r *Room) start(received command) {
	if r.game != nil {
		r.refuse(received.from, &engine.Refusal{Code: "game_started",
			Message:  "a partida já começou",
			Received: "start", Expected: "uma sala que ainda não começou"})
		return
	}
	if host := r.host(); received.from.name != host {
		r.refuse(received.from, &engine.Refusal{Code: "not_host",
			Message:  "só quem abriu a sala começa a partida",
			Received: received.from.name, Expected: host})
		return
	}
	if waiting := r.notReady(); len(waiting) > 0 {
		r.refuse(received.from, &engine.Refusal{Code: "not_all_ready",
			Message:  "ainda tem gente sem marcar pronto",
			Received: waiting, Expected: "todos prontos"})
		return
	}
	if len(r.connections) < playersToStart {
		r.refuse(received.from, &engine.Refusal{Code: "not_enough_players",
			Message:  "uma pessoa sozinha não joga Coup",
			Received: len(r.connections), Expected: "2 a 6"})
		return
	}
	dealt, err := engine.NewGame(r.names(), r.rng, r.initialCoins)
	if err != nil {
		r.refuse(received.from, err)
		return
	}
	r.game = dealt
	r.broadcast(nil)
}

func (r *Room) play(received command) {
	if r.game == nil {
		r.refuse(received.from, &engine.Refusal{Code: "illegal_action",
			Message:  "a partida ainda não começou",
			Received: received.message.Type, Expected: []string{"ready", "start"}})
		return
	}
	move, err := protocol.ToMove(received.message, received.from.name)
	if err != nil {
		r.refuse(received.from, err)
		return
	}
	events, err := r.game.Apply(move)
	if err != nil {
		r.refuse(received.from, err)
		return
	}
	r.broadcast(events)
}

func (r *Room) broadcast(events []engine.Event) {
	connected := make([]*connection, 0, len(r.connections))
	for _, c := range r.connections {
		if r.send(c, r.messageFor(c, events)) {
			connected = append(connected, c)
		}
	}
	r.connections = connected
}

func (r *Room) messageFor(c *connection, events []engine.Event) any {
	if r.game == nil {
		return protocol.NewLobby(r.lobbyFor(c))
	}
	return protocol.NewUpdate(engine.ViewFor(r.game, c.name), events)
}

func (r *Room) lobbyFor(c *connection) protocol.LobbyView {
	seats := make([]protocol.SeatView, 0, len(r.connections))
	for _, seated := range r.connections {
		seats = append(seats, protocol.SeatView{Name: seated.name, Ready: seated.ready})
	}
	return protocol.LobbyView{Room: r.code, You: c.name, Host: r.host(), Players: seats}
}

func (r *Room) host() string {
	if len(r.connections) == 0 {
		return ""
	}
	return r.connections[0].name
}

func (r *Room) notReady() []string {
	waiting := []string{}
	for _, seated := range r.connections {
		if !seated.ready {
			waiting = append(waiting, seated.name)
		}
	}
	return waiting
}

func (r *Room) turnAway(c *connection, reason *engine.Refusal) {
	r.send(c, protocol.NewRefusal(reason))
	c.drop()
}

func (r *Room) refuse(c *connection, reason error) {
	var refusal *engine.Refusal
	if !errors.As(reason, &refusal) {
		refusal = &engine.Refusal{Code: "illegal_action", Message: "a sala não conseguiu aplicar a jogada"}
	}
	if !r.send(c, protocol.NewRefusal(refusal)) {
		r.remove(c)
	}
}

func (r *Room) send(c *connection, message any) bool {
	if c.closed {
		return false
	}
	encoded, err := json.Marshal(message)
	if err != nil {
		log.Printf("message for %q does not serialize: %v", c.name, err)
		return true
	}
	select {
	case c.outbox <- encoded:
		return true
	default:
		c.drop()
		return false
	}
}

func (r *Room) remove(target *connection) {
	target.drop()
	remaining := make([]*connection, 0, len(r.connections))
	for _, c := range r.connections {
		if c != target {
			remaining = append(remaining, c)
		}
	}
	r.connections = remaining
	if r.game == nil {
		r.broadcast(nil)
	}
}

func (r *Room) names() []string {
	names := make([]string, 0, len(r.connections))
	for _, c := range r.connections {
		names = append(names, c.name)
	}
	return names
}
