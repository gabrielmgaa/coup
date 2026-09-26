package server

import (
	"math/rand/v2"

	"github.com/gabrielmgaa/coup/internal/engine"
	"github.com/gabrielmgaa/coup/internal/protocol"
)

const (
	outboxCapacity = 16
	playersToStart = 2
)

type command struct {
	from    *connection
	message protocol.FromClient
	expired *expiry
}

type Room struct {
	inbox  chan command
	done   chan struct{}
	closed bool
	seats  []*seat
	game   *engine.Game
	rng    *rand.Rand
	config Config
	code   string
	forget func(code string)
	clock  clock
}

func newRoom(rng *rand.Rand, config Config, code string, forget func(code string)) *Room {
	return &Room{inbox: make(chan command), done: make(chan struct{}), rng: rng, config: config,
		code: code, forget: forget, clock: newClock()}
}

func (r *Room) deliver(received command) bool {
	select {
	case r.inbox <- received:
		return true
	case <-r.done:
		return false
	}
}

func (r *Room) run() {
	for !r.closed {
		r.handle(<-r.inbox)
	}
}

func (r *Room) handle(received command) {
	if received.expired != nil {
		r.expire(*received.expired)
		return
	}
	if received.from.seat == nil {
		r.admit(received)
		return
	}
	switch received.message.Type {
	case "ready":
		r.markReady(received)
	case "start":
		r.start(received)
	case "leave":
		r.disconnect(received.from)
	default:
		r.play(received)
	}
}

func (r *Room) admit(received command) {
	switch received.message.Type {
	case "join":
		r.join(received.from, received.message.Name)
	case "reconnect":
		r.reconnect(received.from, received.message.Token)
	default:
		received.from.drop()
	}
}

func (r *Room) markReady(received command) {
	if r.game != nil {
		r.refuse(received.from, gameStarted("ready"))
		return
	}
	received.from.seat.ready = received.message.Ready
	r.broadcast(nil)
}

func (r *Room) start(received command) {
	if r.game != nil {
		r.refuse(received.from, gameStarted("start"))
		return
	}
	if refusal := r.checkStart(received.from.seat.name); refusal != nil {
		r.refuse(received.from, refusal)
		return
	}
	dealt, err := engine.NewGame(r.names(), r.rng, r.config.InitialCoins)
	if err != nil {
		r.refuse(received.from, err)
		return
	}
	r.game = dealt
	r.afterChange(nil)
}

func (r *Room) checkStart(requester string) *engine.Refusal {
	if host := r.host(); requester != host {
		return &engine.Refusal{Code: "not_host", Message: "só quem abriu a sala começa a partida",
			Received: requester, Expected: host}
	}
	if waiting := r.notReady(); len(waiting) > 0 {
		return &engine.Refusal{Code: "not_all_ready", Message: "ainda tem gente sem marcar pronto",
			Received: waiting, Expected: "todos prontos"}
	}
	if len(r.seats) < playersToStart {
		return &engine.Refusal{Code: "not_enough_players", Message: "uma pessoa sozinha não joga Coup",
			Received: len(r.seats), Expected: "2 a 6"}
	}
	return nil
}

func gameStarted(received string) *engine.Refusal {
	return &engine.Refusal{Code: "game_started", Message: "a partida já começou",
		Received: received, Expected: "uma sala que ainda não começou"}
}

func (r *Room) play(received command) {
	if r.game == nil {
		r.refuse(received.from, &engine.Refusal{Code: "illegal_action",
			Message:  "a partida ainda não começou",
			Received: received.message.Type, Expected: []string{"ready", "start"}})
		return
	}
	move, err := protocol.ToMove(received.message, received.from.seat.name)
	if err != nil {
		r.refuse(received.from, err)
		return
	}
	events, err := r.game.Apply(move)
	if err != nil {
		r.refuse(received.from, err)
		return
	}
	r.afterChange(events)
}

func (r *Room) afterChange(events []engine.Event) {
	events = append(events, r.runAutopilot()...)
	r.settleClock()
	r.broadcast(events)
}

func (r *Room) runAutopilot() []engine.Event {
	var events []engine.Event
	for {
		move, found := r.autopilotMove()
		if !found {
			return events
		}
		applied, err := r.game.Apply(move)
		if err != nil {
			return events
		}
		events = append(events, applied...)
	}
}

func (r *Room) autopilotMove() (engine.Move, bool) {
	for _, name := range r.game.Awaiting() {
		if waiting := r.seatNamed(name); !waiting.connected() && waiting.autopilot {
			return r.game.SafeMove(name)
		}
	}
	return nil, false
}

func (r *Room) close() {
	r.clock.stopAll()
	r.forget(r.code)
	r.closed = true
	close(r.done)
}
