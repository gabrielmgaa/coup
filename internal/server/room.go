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
}

func newRoom(rng *rand.Rand, initialCoins int) *Room {
	return &Room{inbox: make(chan command), rng: rng, initialCoins: initialCoins}
}

func (r *Room) run() {
	for received := range r.inbox {
		switch received.message.Type {
		case "join":
			r.join(received)
		case "leave":
			r.remove(received.from)
		default:
			r.play(received)
		}
	}
}

func (r *Room) join(received command) {
	if r.game != nil {
		r.refuse(received.from, &engine.Refusal{Code: "room_full", Message: "a partida já começou",
			Received: received.message.Name, Expected: "uma sala que ainda não começou"})
		return
	}
	for _, seated := range r.connections {
		if seated.name == received.message.Name {
			r.refuse(received.from, &engine.Refusal{Code: "name_taken",
				Message:  "já tem alguém com esse nome na sala",
				Received: received.message.Name, Expected: "um nome ainda não usado nesta sala"})
			return
		}
	}
	received.from.name = received.message.Name
	r.connections = append(r.connections, received.from)
	if len(r.connections) < playersToStart {
		return
	}
	r.game = engine.NewGame(r.names(), r.rng, r.initialCoins)
	r.broadcast(nil)
}

func (r *Room) play(received command) {
	if r.game == nil {
		r.refuse(received.from, &engine.Refusal{Code: "illegal_action",
			Message:  "a partida ainda não começou",
			Received: received.message.Type, Expected: "join"})
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
		if r.send(c, protocol.NewUpdate(engine.ViewFor(r.game, c.name), events)) {
			connected = append(connected, c)
		}
	}
	r.connections = connected
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
}

func (r *Room) names() []string {
	names := make([]string, 0, len(r.connections))
	for _, c := range r.connections {
		names = append(names, c.name)
	}
	return names
}
