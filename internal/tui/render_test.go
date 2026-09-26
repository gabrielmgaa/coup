package tui

import (
	"strings"
	"testing"
	"time"

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
