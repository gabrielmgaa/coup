package tui

import (
	"fmt"

	"github.com/gabrielmgaa/coup/internal/engine"
	"github.com/gabrielmgaa/coup/internal/protocol"
)

type choice struct {
	label   string
	message protocol.FromClient
}

func lobbyChoices(lobby protocol.LobbyView) []choice {
	choices := []choice{readyChoice(lobby)}
	if lobby.You == lobby.Host {
		choices = append(choices, choice{label: "começar a partida", message: protocol.FromClient{Type: "start"}})
	}
	return choices
}

func readyChoice(lobby protocol.LobbyView) choice {
	for _, seat := range lobby.Players {
		if seat.Name == lobby.You && seat.Ready {
			return choice{label: "ainda não estou pronto", message: protocol.FromClient{Type: "ready", Ready: false}}
		}
	}
	return choice{label: "estou pronto", message: protocol.FromClient{Type: "ready", Ready: true}}
}

func gameChoices(state engine.View) []choice {
	if state.Losing == state.You {
		return revealChoices(state)
	}
	if state.Window != nil {
		return responseChoices(*state.Window)
	}
	choices := []choice{}
	for _, action := range state.YourActions {
		choices = append(choices, actionChoices(action)...)
	}
	return choices
}

func revealChoices(state engine.View) []choice {
	choices := []choice{}
	for _, card := range myCards(state) {
		choices = append(choices, choice{
			label:   "revelar " + card.LabelPtBR(),
			message: protocol.FromClient{Type: "lose_influence", Card: card.String()},
		})
	}
	return choices
}

func responseChoices(window engine.WindowView) []choice {
	choices := make([]choice, 0, len(window.YourOptions))
	for _, option := range window.YourOptions {
		choices = append(choices, choice{
			label: optionLabel(window, option),
			message: protocol.FromClient{Type: "respond", Window: window.ID,
				Answer: option.Answer.String(), Character: option.Character.String()},
		})
	}
	return choices
}

func optionLabel(window engine.WindowView, option engine.Option) string {
	switch option.Answer {
	case engine.Challenge:
		return fmt.Sprintf("contestar o %s de %s", window.Action.Claims.LabelPtBR(), window.Action.By)
	case engine.Block:
		return "bloquear com " + option.Character.LabelPtBR()
	}
	return "deixar passar"
}

func actionChoices(action engine.AvailableAction) []choice {
	label := actionLabel(action.Name) + priceTag(action.Cost)
	if len(action.Targets) == 0 {
		return []choice{{label: label, message: protocol.FromClient{Type: "play", Action: action.Name}}}
	}
	choices := make([]choice, 0, len(action.Targets))
	for _, target := range action.Targets {
		choices = append(choices, choice{
			label:   fmt.Sprintf("%s em %s", label, target),
			message: protocol.FromClient{Type: "play", Action: action.Name, Target: target},
		})
	}
	return choices
}

func priceTag(cost int) string {
	if cost == 0 {
		return ""
	}
	return fmt.Sprintf(" (%d)", cost)
}

func myCards(state engine.View) []engine.Character {
	for _, seen := range state.Players {
		if seen.Name == state.You {
			return seen.MyCards
		}
	}
	return nil
}

var actionLabels = map[string]string{
	"income": "Renda",
	"tax":    "Taxas",
	"coup":   "Golpe de Estado",
}

func actionLabel(name string) string {
	if label, known := actionLabels[name]; known {
		return label
	}
	return name
}
