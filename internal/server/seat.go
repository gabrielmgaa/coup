package server

import (
	"crypto/rand"
	"crypto/subtle"
	"slices"
	"strings"

	"github.com/gabrielmgaa/coup/internal/engine"
	"github.com/gabrielmgaa/coup/internal/protocol"
)

type seat struct {
	name      string
	token     string
	ready     bool
	conn      *connection
	autopilot bool
}

func (s *seat) connected() bool { return s.conn != nil }

type connection struct {
	outbox chan []byte
	closed bool
	seat   *seat
}

func (c *connection) drop() {
	if c.closed {
		return
	}
	c.closed = true
	close(c.outbox)
}

func (r *Room) join(c *connection, name string) {
	if refusal := r.checkJoin(name); refusal != nil {
		r.turnAway(c, refusal)
		return
	}
	joined := &seat{name: name, token: rand.Text(), conn: c}
	c.seat = joined
	r.seats = append(r.seats, joined)
	r.send(c, protocol.NewWelcome(r.code, joined.token))
	r.clock.stopIdle()
	r.broadcast(nil)
}

func (r *Room) checkJoin(name string) *engine.Refusal {
	if r.game != nil {
		return gameStarted(name)
	}
	if len(r.seats) >= engine.MaxPlayers {
		return &engine.Refusal{Code: "room_full", Message: "a sala está cheia",
			Received: len(r.seats) + 1, Expected: engine.MaxPlayers}
	}
	if r.nameTaken(name) {
		return &engine.Refusal{Code: "name_taken", Message: "já tem alguém com esse nome na sala",
			Received: name, Expected: "um nome ainda não usado nesta sala"}
	}
	return nil
}

func (r *Room) reconnect(c *connection, token string) {
	returning := r.seatWithToken(token)
	if returning == nil {
		r.turnAway(c, &engine.Refusal{Code: "invalid_token", Message: "essa sessão não existe mais nesta sala",
			Received: "token desconhecido", Expected: "o token recebido no welcome desta sala"})
		return
	}
	if returning.connected() {
		displaced := returning.conn
		displaced.seat = nil
		r.turnAway(displaced, &engine.Refusal{Code: "seat_taken", Message: "você abriu esta mesa em outro lugar",
			Received: "um reconnect com o token deste assento", Expected: "uma conexão por assento"})
	}
	returning.conn = c
	returning.autopilot = false
	c.seat = returning
	r.send(c, protocol.NewWelcome(r.code, returning.token))
	r.clock.stopIdle()
	r.afterSeatsChanged()
}

func (r *Room) disconnect(c *connection) {
	c.drop()
	c.seat.conn = nil
	if r.game == nil {
		r.removeSeat(c.seat)
	}
	if !r.anyoneConnected() {
		r.clock.startIdle(r.config.IdleTTL, r.deliver)
	}
	r.afterSeatsChanged()
}

func (r *Room) afterSeatsChanged() {
	if r.game == nil {
		r.broadcast(nil)
		return
	}
	r.afterChange(nil)
}

func (r *Room) removeSeat(target *seat) {
	remaining := make([]*seat, 0, len(r.seats))
	for _, seated := range r.seats {
		if seated != target {
			remaining = append(remaining, seated)
		}
	}
	r.seats = remaining
}

func (r *Room) nameTaken(name string) bool {
	return slices.ContainsFunc(r.seats, func(seated *seat) bool { return strings.EqualFold(seated.name, name) })
}

func (r *Room) seatNamed(name string) *seat {
	for _, seated := range r.seats {
		if seated.name == name {
			return seated
		}
	}
	return nil
}

func (r *Room) seatWithToken(token string) *seat {
	for _, seated := range r.seats {
		if subtle.ConstantTimeCompare([]byte(seated.token), []byte(token)) == 1 {
			return seated
		}
	}
	return nil
}

func (r *Room) anyoneConnected() bool {
	for _, seated := range r.seats {
		if seated.connected() {
			return true
		}
	}
	return false
}

func (r *Room) host() string {
	if len(r.seats) == 0 {
		return ""
	}
	return r.seats[0].name
}

func (r *Room) notReady() []string {
	waiting := []string{}
	for _, seated := range r.seats {
		if !seated.ready {
			waiting = append(waiting, seated.name)
		}
	}
	return waiting
}

func (r *Room) names() []string {
	names := make([]string, 0, len(r.seats))
	for _, seated := range r.seats {
		names = append(names, seated.name)
	}
	return names
}
