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
}

type lobbyArrived struct{ state protocol.LobbyView }

type updateArrived struct {
	state  engine.View
	events []engine.Event
}

type refusalArrived struct{ message string }

type connectionLost struct{ err error }

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
		var state engine.View
		err := json.Unmarshal(arrived.State, &state)
		return updateArrived{state: state, events: arrived.Events}, err
	}
	return refusalArrived{message: arrived.Message}, nil
}
