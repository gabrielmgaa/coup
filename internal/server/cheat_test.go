package server

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/gabrielmgaa/coup/internal/protocol"
)

func TestCheatsFromTheWireAreRefusedPrivatelyAndChangeNothing(t *testing.T) {
	url := startServer(t)
	playing := seatTable(t, url, "tester1", "tester2", "tester3")
	actor := playing.onTurn()
	cheater := playing.someoneElse(actor)
	witness := ""
	for name := range playing.seats {
		if name != actor && name != cheater {
			witness = name
		}
	}

	attempts := []struct {
		name    string
		message protocol.FromClient
		code    string
	}{
		{"play out of turn", protocol.FromClient{Type: "play", Action: "income"}, "not_your_turn"},
		{"coup with two coins", protocol.FromClient{Type: "play", Action: "coup", Target: actor}, "not_your_turn"},
		{"invent an action", protocol.FromClient{Type: "play", Action: "print_money"}, "illegal_action"},
		{"reveal a card for nobody", protocol.FromClient{Type: "lose_influence", Card: "duke"}, "illegal_action"},
		{"answer a window that is not open", protocol.FromClient{Type: "respond", Window: 1, Answer: "challenge"}, "illegal_action"},
		{"return cards with no exchange", protocol.FromClient{Type: "return_cards", Cards: []string{"duke", "duke"}}, "illegal_action"},
		{"forge a deadline", protocol.FromClient{Type: "deadline"}, "illegal_action"},
		{"forge an expiry", protocol.FromClient{Type: "expired"}, "illegal_action"},
		{"mark ready mid game", protocol.FromClient{Type: "ready", Ready: true}, "game_started"},
		{"restart the game", protocol.FromClient{Type: "start"}, "game_started"},
		{"sit a second time", protocol.FromClient{Type: "join", Name: "tester9"}, "illegal_action"},
	}
	for _, attempt := range attempts {
		refused := playing.refused(cheater, attempt.message)
		if refused.Code != attempt.code {
			t.Errorf("%s: answered %q, expected %q", attempt.name, refused.Code, attempt.code)
		}
	}

	playing.seats[actor].send(protocol.FromClient{Type: "play", Action: "income"})
	if next := playing.seats[witness].receive(); next.Type != "update" {
		t.Errorf("the witness received %q before the next move — a refusal leaked", next.Type)
	}
	for _, name := range []string{actor, cheater} {
		playing.views[name] = playing.seats[name].nextView()
	}
	playing.views[witness] = playing.views[actor]
	for _, name := range []string{actor, cheater, witness} {
		if seen := playing.player(name); seen.Hidden != 2 {
			t.Errorf("%s holds %d cards after the cheats, expected 2", name, seen.Hidden)
		}
	}
	if coins := playing.player(cheater).Coins; coins != 2 {
		t.Errorf("the cheater holds %d coins, expected the 2 dealt", coins)
	}
}

func TestUnreadableJsonIsRefusedAndTheSeatStays(t *testing.T) {
	url := startServer(t)
	playing := seatTable(t, url, "tester1", "tester2")
	actor := playing.onTurn()
	playing.seats[actor].conn.Write(context.Background(), websocket.MessageText, []byte("{not json"))

	if code := playing.seats[actor].waitFor("error").Code; code != "illegal_action" {
		t.Errorf("garbage answered %q, expected illegal_action", code)
	}
	playing.play(actor, protocol.FromClient{Type: "play", Action: "income"})
	if coins := playing.player(actor).Coins; coins != 2 {
		t.Errorf("%s holds %d coins after income, expected 2 — the seat did not survive the garbage", actor, coins)
	}
}

func TestAnEliminatedPlayerCannotAct(t *testing.T) {
	url := startServerWithCoins(t, 14)
	playing := seatTable(t, url, "tester1", "tester2", "tester3")
	killer := playing.onTurn()
	victim := playing.someoneElse(killer)
	playing.play(killer, protocol.FromClient{Type: "play", Action: "coup", Target: victim})
	playing.play(victim, protocol.FromClient{Type: "lose_influence", Card: playing.hand(victim)[0].String()})
	for playing.player(victim).Hidden > 0 {
		turn := playing.onTurn()
		if losing := playing.views[turn].Losing; losing != "" {
			playing.play(losing, protocol.FromClient{Type: "lose_influence", Card: playing.hand(losing)[0].String()})
			continue
		}
		target := victim
		if turn == victim {
			target = playing.coupTarget(turn)
		}
		playing.play(turn, protocol.FromClient{Type: "play", Action: "coup", Target: target})
	}
	if playing.anyView().Winner != "" {
		t.Fatal("the game ended before the eliminated seat could try anything")
	}

	for _, attempt := range []protocol.FromClient{
		{Type: "play", Action: "income"},
		{Type: "lose_influence", Card: "duke"},
	} {
		if refused := playing.refused(victim, attempt); refused.Code == "" {
			t.Errorf("the eliminated %s sent %s and it was not refused", victim, attempt.Type)
		}
	}
}

func TestAClientFloodingTheServerOnlyHurtsItself(t *testing.T) {
	url := startServer(t)
	playing := seatTable(t, url, "tester1", "tester2", "tester3")
	actor := playing.onTurn()
	flooder := playing.seats[playing.someoneElse(actor)]

	for range 200 {
		encoded := []byte(`{"type":"play","action":"income"}`)
		if err := flooder.conn.Write(context.Background(), websocket.MessageText, encoded); err != nil {
			break
		}
	}
	playing.seats[actor].send(protocol.FromClient{Type: "play", Action: "income"})
	moved := playing.seats[actor].stateUntil(func(state protocol.GameState) bool { return state.TurnOf != actor })
	if moved.TurnOf == actor {
		t.Error("the game stopped while a client flooded the room")
	}
}

func TestAnOversizedMessageDropsOnlyItsSender(t *testing.T) {
	url := startServer(t)
	playing := seatTable(t, url, "tester1", "tester2", "tester3")
	actor := playing.onTurn()
	bystander := playing.someoneElse(actor)
	huge := `{"type":"play","action":"` + strings.Repeat("a", 100_000) + `"}`
	playing.seats[bystander].conn.Write(context.Background(), websocket.MessageText, []byte(huge))

	seen := playing.seats[actor].stateUntil(func(state protocol.GameState) bool { return len(state.Disconnected) > 0 })
	if !contains(seen.Disconnected, bystander) {
		t.Errorf("disconnected lists %v, expected %s who sent 100 KB", seen.Disconnected, bystander)
	}
	playing.seats[actor].send(protocol.FromClient{Type: "play", Action: "income"})
	if moved := playing.seats[actor].nextState(); moved.TurnOf == actor {
		t.Error("the game did not go on after the oversized message")
	}
}

func TestTheFirstMessageMustOpenOrEnterARoom(t *testing.T) {
	url := startServer(t)
	for _, first := range []string{`{"type":"play","action":"income"}`, `{"type":"start"}`, `garbage`} {
		intruder := dial(t, url)
		intruder.conn.Write(context.Background(), websocket.MessageText, []byte(first))
		if refused := intruder.receive(); refused.Code != "illegal_action" {
			t.Errorf("first message %s answered %q, expected illegal_action", first, refused.Code)
		}
	}
}

func TestAHandNeverTravelsToAnotherSeat(t *testing.T) {
	url := startServer(t)
	playing := seatTable(t, url, "tester1", "tester2", "tester3")
	for viewer, view := range playing.views {
		for _, seen := range view.Players {
			if seen.Name != viewer && seen.MyCards != nil {
				t.Errorf("%s received %s's hand: %v", viewer, seen.Name, seen.MyCards)
			}
		}
	}
	actor := playing.onTurn()
	playing.play(actor, protocol.FromClient{Type: "play", Action: "exchange"})
	for _, name := range playing.views[actor].Window.WaitingOn {
		playing.respond(name, "pass")
	}
	for viewer, view := range playing.views {
		if viewer != actor && view.YourReturns != nil {
			t.Errorf("%s was offered %s's exchange: %v", viewer, actor, view.YourReturns)
		}
	}
}

func TestASilentSocketIsClosedAfterTheHandshakeTimeout(t *testing.T) {
	config := calmConfig(0)
	config.Handshake = 100 * time.Millisecond
	url := startServerWith(t, config)
	silent := dial(t, url)

	started := time.Now()
	ctx, stop := context.WithTimeout(context.Background(), 2*time.Second)
	defer stop()
	_, _, err := silent.conn.Read(ctx)
	if err == nil || time.Since(started) > time.Second {
		t.Errorf("a socket that never spoke stayed open for %v (err %v), expected it closed after 100ms", time.Since(started), err)
	}
}

func TestNamesThatCouldPaintOverSomeoneElsesTerminalAreRefused(t *testing.T) {
	url := startServer(t)
	host := createTable(t, url, "tester1")
	for _, name := range []string{"\x1b[2Jtester2", "tester\n2", "tester\t2", "tester​2x"} {
		intruder := dial(t, url)
		intruder.send(protocol.FromClient{Type: "join", Room: host.room, Name: name})
		if refused := intruder.receive(); refused.Code != "invalid_name" {
			t.Errorf("the name %q answered %q, expected invalid_name", name, refused.Code)
		}
	}
	for _, name := range []string{"José da Silva", "tester-2!", "ação"} {
		entering := enterTable(t, url, host.room, name)
		if welcome := entering.receive(); welcome.Type != "welcome" {
			t.Errorf("the name %q answered %q, expected to sit", name, welcome.Type)
		}
	}
}
