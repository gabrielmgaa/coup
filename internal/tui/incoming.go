package tui

import (
	"encoding/json"

	"github.com/gabrielmgaa/coup/internal/engine"
	"github.com/gabrielmgaa/coup/internal/protocol"
)

type envelope struct {
	Type    string          `json:"type"`
	State   json.RawMessage `json:"state"`
	Events  []engine.Event  `json:"events"`
	Message string          `json:"message"`
	Room    string          `json:"room"`
	Token   string          `json:"token"`
}

type welcomeArrived struct{ room, token string }

type lobbyArrived struct{ state protocol.LobbyView }

type updateArrived struct {
	state  protocol.GameState
	events []engine.Event
}

type refusalArrived struct{ message string }

type connectionLost struct{ err error }

type sessionNotSaved struct{ err error }

func decode(encoded []byte) (any, error) {
	var arrived envelope
	if err := json.Unmarshal(encoded, &arrived); err != nil {
		return nil, err
	}
	switch arrived.Type {
	case "lobby":
		var state protocol.LobbyView
		err := json.Unmarshal(arrived.State, &state)
		return lobbyArrived{state: state}, err
	case "update":
		var state protocol.GameState
		err := json.Unmarshal(arrived.State, &state)
		return updateArrived{state: state, events: arrived.Events}, err
	case "welcome":
		return welcomeArrived{room: arrived.Room, token: arrived.Token}, nil
	}
	return refusalArrived{message: arrived.Message}, nil
}
