package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gabrielmgaa/coup/internal/engine"
	"github.com/gabrielmgaa/coup/internal/protocol"
)

func recordingModel(sent *[]protocol.FromClient) Model {
	return NewModel(func(message protocol.FromClient) tea.Cmd {
		*sent = append(*sent, message)
		return nil
	}, nil, nil)
}

func deliverAll(model Model, messages ...tea.Msg) Model {
	for _, msg := range messages {
		updated, _ := model.Update(msg)
		model = updated.(Model)
	}
	return model
}

func labelUnderCursor(model Model) string {
	return model.choices()[model.cursor].label
}

func TestAnEnterLeftOverFromYourTurnPassesInTheNextWindow(t *testing.T) {
	sent := []protocol.FromClient{}
	myTurn := protocol.GameState{View: engine.View{You: "tester1", TurnOf: "tester1",
		YourActions: []engine.AvailableAction{{Name: "income"}, {Name: "foreign_aid"}}}}
	aidWindow := protocol.GameState{View: engine.View{You: "tester1", TurnOf: "tester2", Window: &engine.WindowView{
		ID:          9,
		Action:      engine.ActionView{Name: "foreign_aid", By: "tester2"},
		YourOptions: []engine.Option{{Answer: engine.Block, Character: engine.Duke}, {Answer: engine.Pass}},
		WaitingOn:   []string{"tester1"},
	}}}

	model := deliverAll(recordingModel(&sent), updateArrived{state: myTurn}, tea.KeyMsg{Type: tea.KeyEnter}, updateArrived{state: aidWindow})
	if under := labelUnderCursor(model); under != "deixar passar" {
		t.Errorf("the window opens with the cursor on %q, expected %q", under, "deixar passar")
	}
	deliverAll(model, tea.KeyMsg{Type: tea.KeyEnter})
	if len(sent) != 2 || sent[0].Action != "income" || sent[1].Answer != "pass" {
		t.Errorf("two enters sent %+v, expected income and then a pass", sent)
	}
}

func TestTheFirstTurnOpensOnTheFirstActionWhateverTheLobbyCursor(t *testing.T) {
	sent := []protocol.FromClient{}
	lobby := protocol.LobbyView{Room: "K7QM", You: "tester1", Host: "tester1",
		Players: []protocol.SeatView{{Name: "tester1", Ready: true}, {Name: "tester2", Ready: true}}}
	firstTurn := protocol.GameState{View: engine.View{You: "tester1", TurnOf: "tester1",
		YourActions: []engine.AvailableAction{{Name: "income"}, {Name: "foreign_aid"}, {Name: "tax"}}}}

	model := deliverAll(recordingModel(&sent),
		lobbyArrived{state: lobby}, tea.KeyMsg{Type: tea.KeyDown}, tea.KeyMsg{Type: tea.KeyEnter}, updateArrived{state: firstTurn})
	if len(sent) != 1 || sent[0].Type != "start" {
		t.Fatalf("the lobby sent %+v, expected one start", sent)
	}
	if under := labelUnderCursor(model); under != "Renda" {
		t.Errorf("the first turn opens with the cursor on %q, expected %q", under, "Renda")
	}
}
