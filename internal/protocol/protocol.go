package protocol

import "github.com/gabrielmgaa/coup/internal/engine"

type FromClient struct {
	Type   string `json:"type"`
	Room   string `json:"room,omitempty"`
	Name   string `json:"name,omitempty"`
	Action string `json:"action,omitempty"`
	Target string `json:"target,omitempty"`
	Card   string `json:"card,omitempty"`
	Ready  bool   `json:"ready,omitempty"`
}

type SeatView struct {
	Name  string `json:"name"`
	Ready bool   `json:"ready"`
}

type LobbyView struct {
	Room    string     `json:"room"`
	You     string     `json:"you"`
	Host    string     `json:"host"`
	Players []SeatView `json:"players"`
}

type Lobby struct {
	Type  string    `json:"type"`
	State LobbyView `json:"state"`
}

func NewLobby(state LobbyView) Lobby {
	return Lobby{Type: "lobby", State: state}
}

type Update struct {
	Type   string         `json:"type"`
	State  engine.View    `json:"state"`
	Events []engine.Event `json:"events"`
}

func NewUpdate(state engine.View, events []engine.Event) Update {
	if events == nil {
		events = []engine.Event{}
	}
	return Update{Type: "update", State: state, Events: events}
}

type RefusalMessage struct {
	Type string `json:"type"`
	*engine.Refusal
}

func NewRefusal(reason *engine.Refusal) RefusalMessage {
	return RefusalMessage{Type: "error", Refusal: reason}
}

func ToMove(message FromClient, by string) (engine.Move, error) {
	switch message.Type {
	case "play":
		action, known := engine.ActionByName(message.Action)
		if !known {
			return nil, &engine.Refusal{Code: "illegal_action", Message: "ação que não existe",
				Received: message.Action, Expected: engine.ActionNames()}
		}
		return engine.Act{By: by, Action: action, Target: message.Target}, nil
	case "lose_influence":
		card, known := engine.CharacterByName(message.Card)
		if !known {
			return nil, &engine.Refusal{Code: "illegal_action", Message: "personagem que não existe",
				Received: message.Card, Expected: engine.CharacterNames()}
		}
		return engine.LoseInfluence{By: by, Card: card}, nil
	}
	return nil, &engine.Refusal{Code: "illegal_action", Message: "mensagem que a sala não entende",
		Received: message.Type,
		Expected: []string{"create_room", "join", "ready", "start", "play", "lose_influence", "leave"}}
}
