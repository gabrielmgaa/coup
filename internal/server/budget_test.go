package server

import (
	"testing"
	"time"

	"github.com/gabrielmgaa/coup/internal/engine"
	"github.com/gabrielmgaa/coup/internal/protocol"
)

func TestReconnectingWithinTheGraceResumesWithTheTimeThatWasLeft(t *testing.T) {
	config := calmConfig(engine.RulebookCoins)
	config.Deadline = 2 * time.Second
	url := startServerWith(t, config)
	playing := seatTable(t, url, "tester1", "tester2")
	actor := playing.onTurn()
	watcher := playing.seats[playing.someoneElse(actor)]

	time.Sleep(time.Second)
	playing.seats[actor].conn.CloseNow()
	watcher.stateUntil(func(state protocol.GameState) bool { return state.Paused != nil })
	back := rejoin(t, url, playing.seats[actor].room, playing.seats[actor].token)
	if welcome := back.receive(); welcome.Type != "welcome" {
		t.Fatalf("reconnect answered %q, expected welcome", welcome.Type)
	}

	resumed := back.nextState()
	if resumed.Paused != nil {
		t.Errorf("the table is still paused after %s came back", actor)
	}
	if resumed.ClosesInMs < 700 || resumed.ClosesInMs > 1100 {
		t.Errorf("closes_in_ms is %d after the return, expected between 700 and 1100 — the second already spent stays spent",
			resumed.ClosesInMs)
	}
}

func TestDroppingAndReconnectingOverAndOverCannotHoldTheTurn(t *testing.T) {
	config := calmConfig(engine.RulebookCoins)
	config.Deadline, config.Grace = 300*time.Millisecond, 400*time.Millisecond
	url := startServerWith(t, config)
	playing := seatTable(t, url, "tester1", "tester2")
	actor := playing.onTurn()
	saboteur := playing.seats[actor]
	room, token := saboteur.room, saboteur.token
	started := time.Now()

	for time.Since(started) < 1500*time.Millisecond {
		time.Sleep(100 * time.Millisecond)
		saboteur.conn.CloseNow()
		saboteur = rejoin(t, url, room, token)
		saboteur.receive()
		if seen := saboteur.nextState(); seen.TurnOf != actor {
			if spent := time.Since(started); spent > time.Second {
				t.Errorf("the turn passed after %v, expected within 1s — 300ms of deadline plus 400ms of grace", spent)
			}
			return
		}
	}
	t.Errorf("after 1.5s of dropping and reconnecting the turn is still %s's", actor)
}

func TestASecondDropInTheSameDecisionGetsOnlyTheGraceThatWasLeft(t *testing.T) {
	config := calmConfig(engine.RulebookCoins)
	config.Grace = 600 * time.Millisecond
	url := startServerWith(t, config)
	playing := seatTable(t, url, "tester1", "tester2")
	actor := playing.onTurn()
	watcher := playing.seats[playing.someoneElse(actor)]

	playing.seats[actor].conn.CloseNow()
	watcher.stateUntil(func(state protocol.GameState) bool { return state.Paused != nil })
	time.Sleep(400 * time.Millisecond)
	back := rejoin(t, url, playing.seats[actor].room, playing.seats[actor].token)
	back.receive()
	back.nextState()
	watcher.stateUntil(func(state protocol.GameState) bool { return state.Paused == nil })
	back.conn.CloseNow()

	paused := watcher.stateUntil(func(state protocol.GameState) bool { return state.Paused != nil })
	if left := paused.Paused.ResumesInMs; left > 250 {
		t.Errorf("resumes_in_ms is %d on the second drop, expected at most 250 — 400ms of the 600ms grace were already used", left)
	}
}

func TestADeadlineThatFiredGivesTheSameDecisionAFullDeadlineWhenArmedAgain(t *testing.T) {
	ticking := newClock()
	deliverNowhere := func(command) bool { return true }
	defer ticking.stopAll()

	ticking.armDeadline(3, time.Second, deliverNowhere)
	time.Sleep(1050 * time.Millisecond)
	ticking.deadlineFired()
	ticking.armDeadline(3, time.Second, deliverNowhere)

	if left := ticking.closesInMs(); left < 900 || left > 1000 {
		t.Errorf("closes_in_ms is %d after the deadline fired, expected between 900 and 1000 — an empty budget re-fires at once, forever", left)
	}
}
