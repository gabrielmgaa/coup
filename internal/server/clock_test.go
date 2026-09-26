package server

import (
	"context"
	"math/rand/v2"
	"strings"
	"testing"
	"time"

	"github.com/gabrielmgaa/coup/internal/engine"
	"github.com/gabrielmgaa/coup/internal/protocol"
)

const (
	shortDeadline = 200 * time.Millisecond
	shortGrace    = 300 * time.Millisecond
	shortIdle     = 200 * time.Millisecond
)

func hurriedConfig() Config {
	return Config{InitialCoins: engine.RulebookCoins, Deadline: shortDeadline, Grace: shortGrace, IdleTTL: shortIdle}
}

func patientTimersExceptDeadline() Config {
	config := hurriedConfig()
	config.Grace, config.IdleTTL = time.Hour, time.Hour
	return config
}

func rejoin(t *testing.T, url, room, token string) *tab {
	t.Helper()
	returning := dial(t, url)
	returning.room = room
	returning.send(protocol.FromClient{Type: "reconnect", Room: room, Token: token})
	return returning
}

func (a *tab) stateUntil(done func(protocol.GameState) bool) protocol.GameState {
	a.t.Helper()
	for {
		if state := a.nextState(); done(state) {
			return state
		}
	}
}

func contains(names []string, wanted string) bool {
	for _, name := range names {
		if name == wanted {
			return true
		}
	}
	return false
}

func TestTheSnapshotCarriesTheTimeLeftToDecide(t *testing.T) {
	url := startServerWith(t, patientTimersExceptDeadline())
	playing := seatTable(t, url, "tester1", "tester2")
	actor := playing.onTurn()
	playing.seats[actor].send(protocol.FromClient{Type: "play", Action: "foreign_aid"})
	state := playing.seats[actor].nextState()
	if state.ClosesInMs <= 0 || state.ClosesInMs > shortDeadline.Milliseconds() {
		t.Errorf("closes_in_ms is %d, expected between 1 and %d", state.ClosesInMs, shortDeadline.Milliseconds())
	}
}

func TestAnUnansweredTurnPlaysIncomeWhenTheDeadlineRuns(t *testing.T) {
	url := startServerWith(t, patientTimersExceptDeadline())
	playing := seatTable(t, url, "tester1", "tester2")
	actor := playing.onTurn()
	watcher := playing.seats[playing.someoneElse(actor)]

	started := time.Now()
	moved := watcher.stateUntil(func(state protocol.GameState) bool { return state.TurnOf != actor })
	if waited := time.Since(started); waited < shortDeadline/2 {
		t.Errorf("the turn moved after %v, before the %v deadline", waited, shortDeadline)
	}
	for _, seen := range moved.Players {
		if seen.Name == actor && seen.Coins != 2 {
			t.Errorf("%s holds %d coins after the timeout, expected 2 — income is the safe default", actor, seen.Coins)
		}
	}
}

func TestAnUnansweredWindowPassesForEveryoneWhenTheDeadlineRuns(t *testing.T) {
	url := startServerWith(t, patientTimersExceptDeadline())
	playing := seatTable(t, url, "tester1", "tester2", "tester3")
	actor := playing.onTurn()
	playing.seats[actor].send(protocol.FromClient{Type: "play", Action: "tax"})

	resolved := playing.seats[actor].stateUntil(func(state protocol.GameState) bool { return state.TurnOf != actor })
	for _, seen := range resolved.Players {
		if seen.Name == actor && seen.Coins != 5 {
			t.Errorf("%s holds %d coins after nobody answered, expected 5", actor, seen.Coins)
		}
	}
}

func TestAStaleDeadlineChangesNothing(t *testing.T) {
	room := newRoom(rand.New(rand.NewPCG(1, 2)), hurriedConfig(), "K7QM", func(string) {})
	dealt, _ := engine.NewGame([]string{"tester1", "tester2"}, rand.New(rand.NewPCG(1, 2)), engine.Setup{})
	room.game = dealt
	room.seats = []*seat{{name: "tester1"}, {name: "tester2"}}
	room.clock.armDeadline(dealt.Decision(), time.Hour, room.deliver)
	defer room.clock.stopAll()
	decision, turn := dealt.Decision(), engine.ViewFor(dealt, "tester1").TurnOf

	room.handle(command{expired: &expiry{kind: deadlineExpired, id: decision - 1}})

	if dealt.Decision() != decision || engine.ViewFor(dealt, "tester1").TurnOf != turn {
		t.Errorf("a stale deadline moved the game from decision %d on %s to %d on %s",
			decision, turn, dealt.Decision(), engine.ViewFor(dealt, "tester1").TurnOf)
	}
}

func TestStaleGraceAndIdleTimersChangeNothing(t *testing.T) {
	room := newRoom(rand.New(rand.NewPCG(1, 2)), hurriedConfig(), "K7QM", func(string) {
		t.Error("a stale idle timer closed the room")
	})
	dealt, _ := engine.NewGame([]string{"tester1", "tester2"}, rand.New(rand.NewPCG(1, 2)), engine.Setup{})
	room.game = dealt
	room.seats = []*seat{{name: "tester1"}, {name: "tester2"}}

	room.handle(command{expired: &expiry{kind: graceExpired, id: 7}})
	room.handle(command{expired: &expiry{kind: idleExpired, id: 7}})

	for _, seated := range room.seats {
		if seated.autopilot {
			t.Errorf("%s was put on autopilot by a grace timer that belongs to no pause", seated.name)
		}
	}
	if room.closed {
		t.Error("the room closed on an idle timer it never armed")
	}
}

func TestDroppingWhileTheGameWaitsOnYouPausesTheTable(t *testing.T) {
	url := startServerWith(t, calmConfig(engine.RulebookCoins))
	playing := seatTable(t, url, "tester1", "tester2", "tester3")
	actor := playing.onTurn()
	watcher := playing.seats[playing.someoneElse(actor)]

	playing.seats[actor].conn.CloseNow()

	paused := watcher.stateUntil(func(state protocol.GameState) bool { return state.Paused != nil })
	if waiting := paused.Paused.WaitingFor; len(waiting) != 1 || waiting[0] != actor {
		t.Errorf("the table waits for %v, expected [%s]", waiting, actor)
	}
	if paused.ClosesInMs != 0 {
		t.Errorf("closes_in_ms is %d while paused, expected 0 — the deadline does not run", paused.ClosesInMs)
	}
	if !contains(paused.Disconnected, actor) {
		t.Errorf("disconnected lists %v, expected %s in it", paused.Disconnected, actor)
	}
}

func TestADropTheGameDoesNotWaitOnDoesNotPause(t *testing.T) {
	url := startServerWith(t, calmConfig(engine.RulebookCoins))
	playing := seatTable(t, url, "tester1", "tester2", "tester3")
	actor := playing.onTurn()
	bystander := playing.someoneElse(actor)

	playing.seats[bystander].conn.CloseNow()

	seen := playing.seats[actor].stateUntil(func(state protocol.GameState) bool { return len(state.Disconnected) > 0 })
	if seen.Paused != nil {
		t.Errorf("the table paused for %v, who had nothing to decide", seen.Paused.WaitingFor)
	}
	playing.seats[actor].send(protocol.FromClient{Type: "play", Action: "income"})
	if moved := playing.seats[actor].nextState(); moved.TurnOf == actor {
		t.Error("the game did not go on after a bystander dropped")
	}
}

func TestReconnectingWithinTheGraceResumesWithAFullDeadline(t *testing.T) {
	config := calmConfig(engine.RulebookCoins)
	config.Deadline = 2 * time.Second
	url := startServerWith(t, config)
	playing := seatTable(t, url, "tester1", "tester2")
	actor := playing.onTurn()
	watcher := playing.seats[playing.someoneElse(actor)]
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
	if resumed.ClosesInMs < config.Deadline.Milliseconds()-300 {
		t.Errorf("closes_in_ms is %d after the return, expected close to the full %d", resumed.ClosesInMs, config.Deadline.Milliseconds())
	}
	if resumed.You != actor || len(resumed.YourActions) == 0 {
		t.Errorf("the returning snapshot is for %q with actions %v, expected %s on turn", resumed.You, resumed.YourActions, actor)
	}
}

func TestAfterTheGraceTheTablePlaysOnWithoutPausingAgain(t *testing.T) {
	config := calmConfig(engine.RulebookCoins)
	config.Grace = shortGrace
	url := startServerWith(t, config)
	playing := seatTable(t, url, "tester1", "tester2")
	actor := playing.onTurn()
	other := playing.someoneElse(actor)
	watcher := playing.seats[other]
	playing.seats[actor].conn.CloseNow()

	moved := watcher.stateUntil(func(state protocol.GameState) bool { return state.TurnOf == other })
	if moved.Paused != nil {
		t.Errorf("the table is still paused after the grace ran out: %+v", moved.Paused)
	}
	watcher.send(protocol.FromClient{Type: "play", Action: "income"})
	again := watcher.nextState()
	if again.TurnOf != other || again.Paused != nil {
		t.Errorf("turn %q paused %+v, expected %s's turn played at once and the turn back with %s",
			again.TurnOf, again.Paused, actor, other)
	}
}

func TestAReturnAfterTheGraceTakesTheSeatBack(t *testing.T) {
	config := calmConfig(engine.RulebookCoins)
	config.Grace = shortGrace
	url := startServerWith(t, config)
	playing := seatTable(t, url, "tester1", "tester2")
	actor := playing.onTurn()
	watcher := playing.seats[playing.someoneElse(actor)]
	playing.seats[actor].conn.CloseNow()
	watcher.stateUntil(func(state protocol.GameState) bool { return state.TurnOf != actor })

	back := rejoin(t, url, playing.seats[actor].room, playing.seats[actor].token)
	back.receive()
	back.nextState()
	watcher.send(protocol.FromClient{Type: "play", Action: "income"})

	mine := back.stateUntil(func(state protocol.GameState) bool { return state.TurnOf == actor })
	if len(mine.YourActions) == 0 || len(mine.Disconnected) != 0 {
		t.Errorf("%s came back and got actions %v with %v disconnected, expected to play again",
			actor, mine.YourActions, mine.Disconnected)
	}
}

func TestAnUnknownTokenIsRefused(t *testing.T) {
	url := startServer(t)
	tester1 := createTable(t, url, "tester1")
	other := createTable(t, url, "tester2")

	for _, attempt := range []struct{ room, token string }{
		{tester1.room, "made-up"},
		{tester1.room, ""},
		{other.room, tester1.token},
	} {
		refused := rejoin(t, url, attempt.room, attempt.token).receive()
		if refused.Code != "invalid_token" {
			t.Errorf("reconnect to %s with %q answered %q, expected invalid_token", attempt.room, attempt.token, refused.Code)
		}
		if echoed, _ := refused.Received.(string); echoed == tester1.token {
			t.Error("the refusal echoed a real token back")
		}
	}
}

func TestALobbySeatDoesNotSurviveADisconnect(t *testing.T) {
	url := startServer(t)
	tester1 := createTable(t, url, "tester1")
	tester2 := enterTable(t, url, tester1.room, "tester2")
	welcome := tester2.receive()
	tester1.waitForSeats(2)
	tester2.conn.CloseNow()
	for len(tester1.waitFor("lobby").State.Players) != 1 {
	}

	if refused := rejoin(t, url, tester1.room, welcome.Token).receive(); refused.Code != "invalid_token" {
		t.Errorf("reconnecting to a lobby seat that was removed answered %q, expected invalid_token", refused.Code)
	}
}

func TestReconnectingTakesTheSeatFromAStillOpenTab(t *testing.T) {
	url := startServer(t)
	playing := seatTable(t, url, "tester1", "tester2")
	actor := playing.onTurn()
	old := playing.seats[actor]

	fresh := rejoin(t, url, old.room, old.token)
	fresh.receive()
	fresh.nextState()
	ctx, stop := context.WithTimeout(context.Background(), 2*time.Second)
	defer stop()
	for {
		if _, _, err := old.conn.Read(ctx); err != nil {
			break
		}
	}
	fresh.send(protocol.FromClient{Type: "play", Action: "income"})
	moved := fresh.nextState()
	if moved.TurnOf == actor || len(moved.Disconnected) != 0 {
		t.Errorf("after the takeover the turn is %q and %v are disconnected, expected the move to count and nobody out",
			moved.TurnOf, moved.Disconnected)
	}
}

func TestAnEmptyRoomExpires(t *testing.T) {
	config := calmConfig(engine.RulebookCoins)
	config.IdleTTL = shortIdle
	url := startServerWith(t, config)
	tester1 := createTable(t, url, "tester1")
	tester1.conn.CloseNow()
	time.Sleep(shortIdle * 3)

	late := dial(t, url)
	late.send(protocol.FromClient{Type: "join", Room: tester1.room, Name: "tester2"})
	if refused := late.receive(); refused.Code != "room_not_found" {
		t.Errorf("joining an expired room answered %q, expected room_not_found", refused.Code)
	}
}

func TestSomeoneArrivingInTimeKeepsTheRoomAlive(t *testing.T) {
	config := calmConfig(engine.RulebookCoins)
	config.IdleTTL = 400 * time.Millisecond
	url := startServerWith(t, config)
	tester1 := createTable(t, url, "tester1")
	tester1.conn.CloseNow()
	time.Sleep(100 * time.Millisecond)
	tester2 := enterTable(t, url, tester1.room, "tester2")
	if welcome := tester2.receive(); welcome.Type != "welcome" {
		t.Fatalf("joining the empty room answered %q, expected welcome", welcome.Type)
	}
	time.Sleep(600 * time.Millisecond)

	tester3 := enterTable(t, url, tester1.room, "tester3")
	if welcome := tester3.receive(); welcome.Type != "welcome" {
		t.Errorf("the room expired with tester2 sitting in it; tester3 was answered %q", welcome.Type)
	}
}

func TestATokenOnlyReachesItsOwner(t *testing.T) {
	url := startServer(t)
	tester1 := createTable(t, url, "tester1")
	tester2 := enterTable(t, url, tester1.room, "tester2")
	own := tester2.receive()

	ctx, stop := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer stop()
	for {
		_, encoded, err := tester1.conn.Read(ctx)
		if err != nil {
			break
		}
		if strings.Contains(string(encoded), own.Token) {
			t.Errorf("tester2's token reached tester1: %s", encoded)
		}
	}
}

func TestASeatedTabCannotJoinOrReconnectAgain(t *testing.T) {
	url := startServer(t)
	tester1 := createTable(t, url, "tester1")
	tester2 := enterTable(t, url, tester1.room, "tester2")
	tester2.receive()
	tester1.waitForSeats(2)

	tester2.send(protocol.FromClient{Type: "join", Room: tester1.room, Name: "tester9"})
	if code := tester2.waitFor("error").Code; code != "illegal_action" {
		t.Errorf("a second join answered %q, expected illegal_action", code)
	}
	tester2.send(protocol.FromClient{Type: "reconnect", Room: tester1.room, Token: tester1.token})
	if code := tester2.waitFor("error").Code; code != "illegal_action" {
		t.Errorf("reconnecting into tester1's seat from tester2's tab answered %q, expected illegal_action", code)
	}
	tester2.send(protocol.FromClient{Type: "ready", Ready: true})
	lobby := tester1.waitFor("lobby")
	if seated := lobby.names(); len(seated) != 2 || seated[1] != "tester2" {
		t.Errorf("the lobby holds %v, expected tester1 and tester2 unchanged", seated)
	}
}

func TestADeadlineQueuedBeforeAPauseChangesNothing(t *testing.T) {
	room := newRoom(rand.New(rand.NewPCG(1, 2)), hurriedConfig(), "K7QM", func(string) {})
	dealt, _ := engine.NewGame([]string{"tester1", "tester2"}, rand.New(rand.NewPCG(1, 2)), engine.Setup{})
	room.game = dealt
	room.seats = []*seat{{name: "tester1"}, {name: "tester2"}}
	room.clock.armDeadline(dealt.Decision(), time.Hour, room.deliver)
	room.clock.pause([]string{"tester1"}, time.Hour, room.deliver)
	defer room.clock.stopAll()
	decision := dealt.Decision()

	room.handle(command{expired: &expiry{kind: deadlineExpired, id: decision}})

	if dealt.Decision() != decision {
		t.Errorf("a deadline that fired as the table paused moved the game from decision %d to %d", decision, dealt.Decision())
	}
}

func TestAPauseStopsTheDeadline(t *testing.T) {
	config := calmConfig(engine.RulebookCoins)
	config.Deadline = shortDeadline
	url := startServerWith(t, config)
	playing := seatTable(t, url, "tester1", "tester2", "tester3")
	actor := playing.onTurn()
	watcher := playing.seats[playing.someoneElse(actor)]
	playing.seats[actor].conn.CloseNow()
	watcher.stateUntil(func(state protocol.GameState) bool { return state.Paused != nil })

	ctx, stop := context.WithTimeout(context.Background(), shortDeadline*3)
	defer stop()
	if _, encoded, err := watcher.conn.Read(ctx); err == nil {
		t.Errorf("the paused table moved when the deadline would have run: %s", encoded)
	}
}

func TestComingBackAndDroppingAgainEarnsAFreshPause(t *testing.T) {
	config := calmConfig(engine.RulebookCoins)
	config.Grace = shortGrace
	url := startServerWith(t, config)
	playing := seatTable(t, url, "tester1", "tester2")
	actor := playing.onTurn()
	other := playing.someoneElse(actor)
	watcher := playing.seats[other]
	playing.seats[actor].conn.CloseNow()
	watcher.stateUntil(func(state protocol.GameState) bool { return state.TurnOf == other })

	back := rejoin(t, url, playing.seats[actor].room, playing.seats[actor].token)
	back.receive()
	back.nextState()
	watcher.send(protocol.FromClient{Type: "play", Action: "income"})
	back.stateUntil(func(state protocol.GameState) bool { return state.TurnOf == actor })
	back.conn.CloseNow()

	paused := watcher.stateUntil(func(state protocol.GameState) bool { return len(state.Disconnected) > 0 })
	if paused.Paused == nil || paused.TurnOf != actor {
		t.Errorf("turn %q paused %+v, expected a new pause waiting on %s, who had come back", paused.TurnOf, paused.Paused, actor)
	}
}

func TestTheDefaultConfigFollowsThePlan(t *testing.T) {
	config := DefaultConfig(engine.RulebookCoins)
	if config.Deadline != 25*time.Second || config.Grace != 30*time.Second || config.IdleTTL != 30*time.Minute {
		t.Errorf("the default config is %+v, expected 25s, 30s and 30min", config)
	}
}
