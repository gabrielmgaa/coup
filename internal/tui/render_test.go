package tui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gabrielmgaa/coup/internal/engine"
	"github.com/gabrielmgaa/coup/internal/protocol"
)

type manualClock struct{ at time.Time }

func (c *manualClock) read() time.Time { return c.at }

func (c *manualClock) advance(by time.Duration) { c.at = c.at.Add(by) }

func modelOnClock(clock *manualClock) Model {
	model := NewModel(nil, nil, nil)
	model.now = clock.read
	return model
}

func TestThePauseCountdownRunsDownOnScreen(t *testing.T) {
	clock := &manualClock{at: time.Date(2026, 9, 26, 12, 0, 0, 0, time.UTC)}
	paused := protocol.GameState{View: engine.View{You: "tester2", TurnOf: "tester1"},
		Paused: &protocol.PausedView{WaitingFor: []string{"tester1"}, ResumesInMs: 30000}}

	model := deliverAll(modelOnClock(clock), updateArrived{state: paused})
	clock.advance(5 * time.Second)
	if screen := model.View(); !strings.Contains(screen, "mesa pausada esperando tester1 voltar (25s)") {
		t.Errorf("5s into the pause the screen does not read 25s:\n%s", screen)
	}
}

func TestTheClockSaysWhoseDecisionItIs(t *testing.T) {
	clock := &manualClock{at: time.Date(2026, 9, 26, 12, 0, 0, 0, time.UTC)}
	for _, scenario := range []struct {
		state    engine.View
		expected string
	}{
		{engine.View{You: "tester1", TurnOf: "tester2"}, "24s até a jogada automática"},
		{engine.View{You: "tester1", TurnOf: "tester1", YourActions: []engine.AvailableAction{{Name: "income"}}}, "você tem 24s para decidir"},
	} {
		model := deliverAll(modelOnClock(clock), updateArrived{state: protocol.GameState{View: scenario.state, ClosesInMs: 24000}})
		if screen := model.View(); !strings.Contains(screen, scenario.expected) {
			t.Errorf("the clock line does not read %q:\n%s", scenario.expected, screen)
		}
	}
}

func TestTheEndingStaysOnScreenUntilTheNextGame(t *testing.T) {
	finished := protocol.GameState{View: engine.View{You: "tester1", Winner: "tester2"}}
	lobby := protocol.LobbyView{Room: "K7QM", You: "tester1", Host: "tester1", LastWinner: "tester2",
		Players: []protocol.SeatView{{Name: "tester1"}, {Name: "tester2"}}}

	model := deliverAll(NewModel(nil, nil, nil),
		updateArrived{state: finished, events: []engine.Event{{N: 30, Text: "tester1 perdeu a última influência."}}},
		lobbyArrived{state: lobby}, lobbyArrived{state: lobby})
	screen := model.View()
	for _, expected := range []string{"tester1 perdeu a última influência.", "tester2 venceu a partida"} {
		if !strings.Contains(screen, expected) {
			t.Errorf("back in the lobby the screen is missing %q:\n%s", expected, screen)
		}
	}

	nextGame := protocol.GameState{View: engine.View{You: "tester1", TurnOf: "tester2"}}
	next := deliverAll(model, updateArrived{state: nextGame, events: []engine.Event{{N: 1, Text: "Agora é a vez de tester2."}}})
	if len(next.log) != 1 {
		t.Errorf("the next game opened with %d log lines, expected only its own 1", len(next.log))
	}
}

func TestTheScreenFitsTheTerminalWithSixSeatsAndNineChoices(t *testing.T) {
	names := []string{"tester1", "tester2", "tester3", "tester4", "tester5", "tester6"}
	players := make([]engine.PlayerView, 0, len(names))
	for _, name := range names {
		players = append(players, engine.PlayerView{Name: name, Coins: 2, Hidden: 2})
	}
	players[0].MyCards = []engine.Character{engine.Duke, engine.Captain}
	state := engine.View{You: "tester1", TurnOf: "tester1", Players: players, YourActions: []engine.AvailableAction{
		{Name: "income"}, {Name: "foreign_aid"}, {Name: "tax"}, {Name: "exchange"}, {Name: "steal", Targets: names[1:]},
	}}
	events := make([]engine.Event, 0, visibleLogLines)
	for position := range visibleLogLines {
		events = append(events, engine.Event{N: position + 1, Text: fmt.Sprintf("tester%d pegou Renda.", position%6+1)})
	}

	model := deliverAll(NewModel(nil, nil, nil),
		tea.WindowSizeMsg{Width: 100, Height: 30}, updateArrived{state: protocol.GameState{View: state}, events: events})
	screen := model.View()
	if lines := len(strings.Split(screen, "\n")); lines > 30 {
		t.Errorf("the screen has %d lines, expected at most 30 — the seats would scroll off the top:\n%s", lines, screen)
	}
	for _, name := range names {
		if !strings.Contains(screen, name) {
			t.Errorf("the screen does not show %s:\n%s", name, screen)
		}
	}
}

func TestAHiddenCardAndARevealedOneSitOneSpaceApart(t *testing.T) {
	drawn := renderCards(engine.PlayerView{Name: "tester2", Hidden: 1, Revealed: []engine.Character{engine.Duke}})
	if !strings.Contains(drawn, "? Duque") || strings.Contains(drawn, "?  ") {
		t.Errorf("tester2's cards read %q, expected %q", drawn, "? Duque")
	}
}

func TestSixSeatsFitAnEightyByTwentyFourTerminal(t *testing.T) {
	names := []string{"tester1", "tester2", "tester3", "tester4", "tester5", "tester6"}
	players := make([]engine.PlayerView, 0, len(names))
	revealed := []engine.Character{engine.Captain, engine.Assassin, engine.Ambassador, engine.Contessa, engine.Ambassador, engine.Duke}
	for position, name := range names {
		players = append(players, engine.PlayerView{Name: name, Coins: position + 1, Hidden: 1, Revealed: revealed[position : position+1]})
	}
	players[0].MyCards = []engine.Character{engine.Duke}
	state := engine.View{You: "tester1", TurnOf: "tester1", Players: players, YourActions: []engine.AvailableAction{
		{Name: "income"}, {Name: "foreign_aid"}, {Name: "tax"}, {Name: "exchange"}, {Name: "steal", Targets: names[1:4]},
	}}
	dropped := protocol.GameState{View: state, Disconnected: []string{"tester4"}}

	model := deliverAll(NewModel(nil, nil, nil), tea.WindowSizeMsg{Width: 80, Height: 24}, updateArrived{state: dropped})
	lines := strings.Split(model.View(), "\n")
	if len(lines) > 24 {
		t.Errorf("the screen has %d lines, expected at most 24", len(lines))
	}
	widest := 0
	for _, line := range lines {
		widest = max(widest, lipgloss.Width(line))
	}
	if widest > 80 {
		t.Errorf("the widest line has %d columns, expected at most 80 — a seat is cut at the edge", widest)
	}
	screen := strings.Join(lines, "\n")
	for position, name := range names {
		if coins := fmt.Sprintf("%d moedas", position+1); !strings.Contains(screen, name) || !strings.Contains(screen, coins) {
			t.Errorf("the screen does not show %s and %s:\n%s", name, coins, screen)
		}
	}
}
