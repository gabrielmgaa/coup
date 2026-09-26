package server

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/gabrielmgaa/coup/internal/engine"
	"github.com/gabrielmgaa/coup/internal/protocol"
)

type table struct {
	t     *testing.T
	seats map[string]*tab
	views map[string]engine.View
}

func seatTable(t *testing.T, url string, names ...string) *table {
	t.Helper()
	host := createTable(t, url, names[0])
	seated := map[string]*tab{names[0]: host}
	for _, name := range names[1:] {
		seated[name] = enterTable(t, url, host.room, name)
	}
	host.waitForSeats(len(names))
	for _, name := range names {
		seated[name].send(protocol.FromClient{Type: "ready", Ready: true})
	}
	host.waitForEveryoneReady(len(names))
	host.send(protocol.FromClient{Type: "start"})
	playing := &table{t: t, seats: seated, views: map[string]engine.View{}}
	for _, name := range names {
		playing.views[name] = seated[name].nextView()
	}
	return playing
}

func (a *tab) nextView() engine.View {
	a.t.Helper()
	for {
		ctx, stop := context.WithTimeout(context.Background(), 2*time.Second)
		_, encoded, err := a.conn.Read(ctx)
		stop()
		if err != nil {
			a.t.Fatalf("%s received nothing within 2s: %v", a.name, err)
		}
		var arrived struct {
			Type  string      `json:"type"`
			State engine.View `json:"state"`
		}
		if err := json.Unmarshal(encoded, &arrived); err != nil {
			a.t.Fatalf("%s received json that does not decode: %v — %s", a.name, err, encoded)
		}
		if arrived.Type == "update" {
			return arrived.State
		}
	}
}

func (p *table) play(name string, message protocol.FromClient) {
	p.t.Helper()
	p.seats[name].send(message)
	for seated, seat := range p.seats {
		p.views[seated] = seat.nextView()
	}
}

func (p *table) refused(name string, message protocol.FromClient) received {
	p.t.Helper()
	p.seats[name].send(message)
	return p.seats[name].waitFor("error")
}

func (p *table) respond(name string, answer string) {
	p.t.Helper()
	p.play(name, protocol.FromClient{Type: "respond", Window: p.views[name].Window.ID, Answer: answer})
}

func (p *table) hand(name string) []engine.Character {
	for _, seen := range p.views[name].Players {
		if seen.Name == name {
			return seen.MyCards
		}
	}
	return nil
}

func (p *table) player(name string) engine.PlayerView {
	for _, seen := range p.views[name].Players {
		if seen.Name == name {
			return seen
		}
	}
	p.t.Fatalf("%q is not at the table", name)
	return engine.PlayerView{}
}

func (p *table) onTurn() string {
	for _, view := range p.views {
		return view.TurnOf
	}
	return ""
}

func (p *table) someoneElse(than string) string {
	for name := range p.seats {
		if name != than {
			return name
		}
	}
	return ""
}

func holds(hand []engine.Character, card engine.Character) bool {
	for _, held := range hand {
		if held == card {
			return true
		}
	}
	return false
}

func TestThreeTabsCatchADukeClaimAndTheLoserPicksACard(t *testing.T) {
	url := startServer(t)
	playing := seatTable(t, url, "tester1", "tester2", "tester3")
	claimant := playing.onTurn()
	challenger := playing.someoneElse(claimant)
	bluffing := !holds(playing.hand(claimant), engine.Duke)

	playing.play(claimant, protocol.FromClient{Type: "play", Action: "tax"})
	playing.respond(challenger, "challenge")

	loser := challenger
	if bluffing {
		loser = claimant
	}
	if losing := playing.views[loser].Losing; losing != loser {
		t.Fatalf("the table waits on %q to reveal, expected %s (bluffing = %v)", losing, loser, bluffing)
	}
	playing.play(loser, protocol.FromClient{Type: "lose_influence", Card: playing.hand(loser)[0].String()})
	if hidden := playing.player(loser).Hidden; hidden != 1 {
		t.Errorf("%s kept %d cards after losing the challenge, expected 1", loser, hidden)
	}
	expectedCoins := 5
	if bluffing {
		expectedCoins = 2
	}
	if coins := playing.player(claimant).Coins; coins != expectedCoins {
		t.Errorf("%s holds %d coins, expected %d (bluffing = %v)", claimant, coins, expectedCoins, bluffing)
	}
}

func TestAWindowAnswerSentByAnotherSeatCannotSpeakForSomeoneElse(t *testing.T) {
	url := startServer(t)
	playing := seatTable(t, url, "tester1", "tester2", "tester3")
	claimant := playing.onTurn()
	playing.play(claimant, protocol.FromClient{Type: "play", Action: "tax"})

	forged, _ := json.Marshal(map[string]any{"type": "respond", "window": playing.views[claimant].Window.ID,
		"answer": "pass", "by": playing.someoneElse(claimant), "name": playing.someoneElse(claimant)})
	playing.seats[claimant].conn.Write(context.Background(), websocket.MessageText, forged)

	if code := playing.seats[claimant].waitFor("error").Code; code != "illegal_action" {
		t.Errorf("code %q, expected illegal_action — the claimant tried to pass on behalf of another seat", code)
	}
}

func TestAThirdSeatBlocksForeignAidOverTheWire(t *testing.T) {
	url := startServer(t)
	playing := seatTable(t, url, "tester1", "tester2", "tester3")
	actor := playing.onTurn()
	blocker := playing.someoneElse(actor)
	coinsBefore := playing.player(actor).Coins

	playing.play(actor, protocol.FromClient{Type: "play", Action: "foreign_aid"})
	playing.play(blocker, protocol.FromClient{Type: "respond", Window: playing.views[blocker].Window.ID,
		Answer: "block", Character: "duke"})
	for _, name := range playing.views[actor].Window.WaitingOn {
		playing.respond(name, "pass")
	}

	if coins := playing.player(actor).Coins; coins != coinsBefore {
		t.Errorf("%s holds %d coins, expected %d — the duke block held", actor, coins, coinsBefore)
	}
	if turn := playing.onTurn(); turn == actor {
		t.Errorf("the turn stayed with %s after the blocked foreign aid", actor)
	}
}

func TestTheTargetBlocksAnAssassinationOverTheWireAndTheCoinsStaySpent(t *testing.T) {
	url := startServerWithCoins(t, 5)
	playing := seatTable(t, url, "tester1", "tester2", "tester3")
	actor := playing.onTurn()
	target := playing.someoneElse(actor)

	playing.play(actor, protocol.FromClient{Type: "play", Action: "assassinate", Target: target})
	if coins := playing.player(actor).Coins; coins != 2 {
		t.Fatalf("%s holds %d coins during the window, expected 2", actor, coins)
	}
	bystander := ""
	for name := range playing.seats {
		if name != actor && name != target {
			bystander = name
		}
	}
	refused := playing.refused(bystander, protocol.FromClient{Type: "respond",
		Window: playing.views[bystander].Window.ID, Answer: "block", Character: "contessa"})
	if refused.Code != "illegal_action" {
		t.Errorf("a bystander blocking answered %q, expected illegal_action", refused.Code)
	}
	playing.play(target, protocol.FromClient{Type: "respond", Window: playing.views[target].Window.ID,
		Answer: "block", Character: "contessa"})
	for _, name := range playing.views[actor].Window.WaitingOn {
		playing.respond(name, "pass")
	}

	if coins, hidden := playing.player(actor).Coins, playing.player(target).Hidden; coins != 2 || hidden != 2 {
		t.Errorf("%s has %d coins and %s %d cards, expected 2 and 2", actor, coins, target, hidden)
	}
}
