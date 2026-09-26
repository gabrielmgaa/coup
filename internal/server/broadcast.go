package server

import (
	"encoding/json"
	"errors"
	"log"

	"github.com/gabrielmgaa/coup/internal/engine"
	"github.com/gabrielmgaa/coup/internal/protocol"
)

func (r *Room) broadcast(events []engine.Event) {
	for _, seated := range r.seats {
		if seated.connected() {
			r.send(seated.conn, r.messageFor(seated, events))
		}
	}
}

func (r *Room) messageFor(seated *seat, events []engine.Event) any {
	if r.game == nil {
		return protocol.NewLobby(r.lobbyFor(seated))
	}
	return protocol.NewUpdate(r.gameStateFor(seated), events)
}

func (r *Room) gameStateFor(seated *seat) protocol.GameState {
	state := protocol.GameState{
		View:         engine.ViewFor(r.game, seated.name),
		Room:         r.code,
		ClosesInMs:   r.clock.closesInMs(),
		Disconnected: []string{},
	}
	for _, other := range r.seats {
		if !other.connected() {
			state.Disconnected = append(state.Disconnected, other.name)
		}
	}
	if state.ClosesInMs > 0 {
		state.DecisionMs = r.config.Deadline.Milliseconds()
	}
	if r.clock.paused != nil {
		state.Paused = &protocol.PausedView{WaitingFor: r.clock.paused, ResumesInMs: r.clock.resumesInMs()}
	}
	return state
}

func (r *Room) lobbyFor(seated *seat) protocol.LobbyView {
	seats := make([]protocol.SeatView, 0, len(r.seats))
	for _, other := range r.seats {
		seats = append(seats, protocol.SeatView{Name: other.name, Ready: other.ready})
	}
	return protocol.LobbyView{Room: r.code, You: seated.name, Host: r.host(), Players: seats,
		Options: r.options, LastWinner: r.winner}
}

func (r *Room) turnAway(c *connection, reason *engine.Refusal) {
	r.send(c, protocol.NewRefusal(reason))
	c.drop(reason.Code)
}

func (r *Room) refuse(c *connection, reason error) {
	var refusal *engine.Refusal
	if !errors.As(reason, &refusal) {
		refusal = &engine.Refusal{Code: "illegal_action", Message: "a sala não conseguiu aplicar a jogada"}
	}
	r.send(c, protocol.NewRefusal(refusal))
}

func (r *Room) send(c *connection, message any) {
	if c.closed {
		return
	}
	encoded, err := json.Marshal(message)
	if err != nil {
		log.Printf("message for room %s does not serialize: %v", r.code, err)
		return
	}
	select {
	case c.outbox <- encoded:
	default:
		c.drop("client too far behind")
	}
}
