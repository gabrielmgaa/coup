package protocol

import "github.com/gabrielmgaa/coup/internal/engine"

type FromClient struct {
	Type      string         `json:"type"`
	Room      string         `json:"room,omitempty"`
	Name      string         `json:"name,omitempty"`
	Action    string         `json:"action,omitempty"`
	Target    string         `json:"target,omitempty"`
	Card      string         `json:"card,omitempty"`
	Ready     bool           `json:"ready,omitempty"`
	Window    int            `json:"window,omitempty"`
	Answer    string         `json:"answer,omitempty"`
	Character string         `json:"character,omitempty"`
	Cards     []string       `json:"cards,omitempty"`
	Token     string         `json:"token,omitempty"`
	Options   engine.Options `json:"options,omitzero"`
}

type SeatView struct {
	Name  string `json:"name"`
	Ready bool   `json:"ready"`
}

type LobbyView struct {
	Room       string         `json:"room"`
	You        string         `json:"you"`
	Host       string         `json:"host"`
	Players    []SeatView     `json:"players"`
	Options    engine.Options `json:"options"`
	LastWinner string         `json:"last_winner,omitempty"`
}

type Lobby struct {
	Type  string    `json:"type"`
	State LobbyView `json:"state"`
}

func NewLobby(state LobbyView) Lobby {
	return Lobby{Type: "lobby", State: state}
}

type GameState struct {
	engine.View
	Room         string      `json:"room"`
	ClosesInMs   int64       `json:"closes_in_ms,omitempty"`
	Paused       *PausedView `json:"paused"`
	Disconnected []string    `json:"disconnected"`
}

type PausedView struct {
	WaitingFor  []string `json:"waiting_for"`
	ResumesInMs int64    `json:"resumes_in_ms"`
}

type Update struct {
	Type   string         `json:"type"`
	State  GameState      `json:"state"`
	Events []engine.Event `json:"events"`
}

type Welcome struct {
	Type  string `json:"type"`
	Room  string `json:"room"`
	Token string `json:"token"`
}

func NewWelcome(room, token string) Welcome {
	return Welcome{Type: "welcome", Room: room, Token: token}
}

func NewUpdate(state GameState, events []engine.Event) Update {
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
	case "respond":
		return toResponse(message, by)
	case "return_cards":
		return toReturn(message, by)
	}
	return nil, &engine.Refusal{Code: "illegal_action", Message: "mensagem que a sala não entende",
		Received: message.Type,
		Expected: []string{"create_room", "join", "ready", "start", "play", "respond", "lose_influence", "return_cards", "leave"}}
}

func toReturn(message FromClient, by string) (engine.Move, error) {
	returned := engine.ReturnCards{By: by}
	if len(message.Cards) != len(returned.Cards) {
		return nil, &engine.Refusal{Code: "illegal_action", Message: "a troca devolve exatamente 2 cartas",
			Received: message.Cards, Expected: len(returned.Cards)}
	}
	for position, name := range message.Cards {
		card, known := engine.CharacterByName(name)
		if !known {
			return nil, &engine.Refusal{Code: "illegal_action", Message: "personagem que não existe",
				Received: name, Expected: engine.CharacterNames()}
		}
		returned.Cards[position] = card
	}
	return returned, nil
}

func toResponse(message FromClient, by string) (engine.Move, error) {
	answer, known := engine.AnswerByName(message.Answer)
	if !known {
		return nil, &engine.Refusal{Code: "illegal_action", Message: "resposta que não existe",
			Received: message.Answer, Expected: []string{"challenge", "block", "pass"}}
	}
	character := engine.NoCharacter
	if message.Character != "" {
		if character, known = engine.CharacterByName(message.Character); !known {
			return nil, &engine.Refusal{Code: "illegal_action", Message: "personagem que não existe",
				Received: message.Character, Expected: engine.CharacterNames()}
		}
	}
	return engine.Respond{By: by, Window: message.Window, Answer: answer, Character: character}, nil
}
