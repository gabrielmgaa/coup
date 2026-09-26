package server

import (
	"encoding/json"
	"math/rand/v2"
	"testing"

	"github.com/gabrielmgaa/coup/internal/engine"
	"github.com/gabrielmgaa/coup/internal/protocol"
)

func (p *table) coupUntilSomeoneWins() string {
	p.t.Helper()
	for {
		turn := p.onTurn()
		if turn == "" {
			return p.anyView().Winner
		}
		if losing := p.views[turn].Losing; losing != "" {
			p.play(losing, protocol.FromClient{Type: "lose_influence", Card: p.hand(losing)[0].String()})
			continue
		}
		p.play(turn, protocol.FromClient{Type: "play", Action: "coup", Target: p.coupTarget(turn)})
	}
}

func (p *table) coupTarget(turn string) string {
	for _, offered := range p.views[turn].YourActions {
		if offered.Name == "coup" {
			return offered.Targets[0]
		}
	}
	p.t.Fatalf("%s cannot coup; the offered actions are %v", turn, p.views[turn].YourActions)
	return ""
}

func (p *table) anyView() engine.View {
	for _, view := range p.views {
		return view
	}
	return engine.View{}
}

func TestAfterTheGameTheTableReturnsToTheLobbyAndTheWinnerStarts(t *testing.T) {
	url := startServerWithCoins(t, 14)
	playing := seatTable(t, url, "tester1", "tester2")
	host := playing.seats["tester1"]

	winner := playing.coupUntilSomeoneWins()

	lobby := host.waitFor("lobby")
	if lobby.State.Room != host.room || len(lobby.State.Players) != 2 {
		t.Fatalf("the lobby is room %q with %v, expected %s with both players", lobby.State.Room, lobby.names(), host.room)
	}
	if lobby.readyOf(t, "tester1") || lobby.readyOf(t, "tester2") {
		t.Error("someone is still marked ready after the game ended")
	}
	for _, name := range []string{"tester1", "tester2"} {
		playing.seats[name].send(protocol.FromClient{Type: "ready", Ready: true})
	}
	host.waitForEveryoneReady(2)
	host.send(protocol.FromClient{Type: "start"})
	if turn := host.nextView().TurnOf; turn != winner {
		t.Errorf("the second game opened on %s, expected the winner %s", turn, winner)
	}
}

func TestTheLobbyAfterTheGameNamesTheWinner(t *testing.T) {
	url := startServerWithCoins(t, 14)
	playing := seatTable(t, url, "tester1", "tester2")
	winner := playing.coupUntilSomeoneWins()

	var lobby struct {
		State protocol.LobbyView `json:"state"`
	}
	playing.seats["tester2"].decodeNext("lobby", &lobby)
	if lobby.State.LastWinner != winner {
		t.Errorf("the lobby names %q as the last winner, expected %s", lobby.State.LastWinner, winner)
	}
}

func TestWhoeverDroppedDuringTheGameIsNotInTheNextLobby(t *testing.T) {
	room := newRoom(rand.New(rand.NewPCG(1, 2)), calmConfig(14), "K7QM", func(string) {})
	connections := map[string]*connection{}
	for _, name := range []string{"tester1", "tester2", "tester3"} {
		connections[name] = &connection{outbox: make(chan []byte, 4096)}
		room.handle(command{from: connections[name], message: protocol.FromClient{Type: "join", Name: name}})
		room.handle(command{from: connections[name], message: protocol.FromClient{Type: "ready", Ready: true}})
	}
	room.handle(command{from: connections["tester1"], message: protocol.FromClient{Type: "start"}})
	room.handle(command{from: connections["tester3"], message: protocol.FromClient{Type: "leave"}})

	for steps := 0; room.game != nil; steps++ {
		if steps > 500 {
			t.Fatal("the game did not end within 500 steps")
		}
		playOneSafeStep(t, room)
	}

	if seated := room.names(); len(seated) != 2 || contains(seated, "tester3") {
		t.Errorf("the lobby after the game holds %v, expected tester1 and tester2 without tester3", seated)
	}
	if room.winner == "" {
		t.Error("the room forgot who won")
	}
}

func TestAWinnerWhoLeftIsNotAnnouncedInTheNextLobby(t *testing.T) {
	room := newRoom(rand.New(rand.NewPCG(1, 2)), calmConfig(14), "K7QM", func(string) {})
	connections := map[string]*connection{}
	for _, name := range []string{"tester1", "tester2", "tester3"} {
		connections[name] = &connection{outbox: make(chan []byte, 4096)}
		room.handle(command{from: connections[name], message: protocol.FromClient{Type: "join", Name: name}})
		room.handle(command{from: connections[name], message: protocol.FromClient{Type: "ready", Ready: true}})
	}
	room.handle(command{from: connections["tester1"], message: protocol.FromClient{Type: "start"}})
	room.handle(command{from: connections["tester2"], message: protocol.FromClient{Type: "leave"}})
	for steps := 0; room.game != nil; steps++ {
		if steps > 500 {
			t.Fatal("the game did not end within 500 steps")
		}
		playOneSafeStep(t, room)
	}
	if seated := room.names(); contains(seated, "tester2") {
		t.Fatalf("the lobby holds %v, expected tester2 gone — the scenario needs the winner to have left", seated)
	}

	announced := lastLobbyIn(t, connections["tester1"]).LastWinner
	if announced != "" {
		t.Errorf("the lobby announces %q as the last winner, expected nobody — that seat is gone", announced)
	}
}

func lastLobbyIn(t *testing.T, c *connection) protocol.LobbyView {
	t.Helper()
	var latest protocol.LobbyView
	for len(c.outbox) > 0 {
		var arrived protocol.Lobby
		if json.Unmarshal(<-c.outbox, &arrived) == nil && arrived.Type == "lobby" {
			latest = arrived.State
		}
	}
	return latest
}

func playOneSafeStep(t *testing.T, room *Room) {
	t.Helper()
	if room.clock.paused != nil {
		room.handle(command{expired: &expiry{kind: graceExpired, id: room.clock.pauseID}})
		return
	}
	move, _ := room.game.SafeMove(room.game.Awaiting()[0])
	events, err := room.game.Apply(move)
	if err != nil {
		t.Fatalf("the safe move %+v was refused: %v", move, err)
	}
	room.afterChange(events)
}

func TestRoomOptionsTravelFromCreateRoomToTheGame(t *testing.T) {
	url := startServer(t)
	opener := dial(t, url)
	opener.name = "tester1"
	opener.send(protocol.FromClient{Type: "create_room", Name: "tester1", Options: engine.Options{IndependentReactions: true}})
	opener.waitFor("welcome")
	var lobby struct {
		State protocol.LobbyView `json:"state"`
	}
	opener.decodeNext("lobby", &lobby)
	if !lobby.State.Options.IndependentReactions {
		t.Fatalf("the lobby shows options %+v, expected independent_reactions on", lobby.State.Options)
	}
	tester2 := enterTable(t, url, lobby.State.Room, "tester2")
	opener.waitForSeats(2)
	opener.send(protocol.FromClient{Type: "ready", Ready: true})
	tester2.send(protocol.FromClient{Type: "ready", Ready: true})
	opener.waitForEveryoneReady(2)
	opener.send(protocol.FromClient{Type: "start"})
	if !opener.nextView().Options.IndependentReactions {
		t.Error("the game was dealt without independent_reactions")
	}
}

func TestTheLastWinnerAlwaysOpensTheNextGame(t *testing.T) {
	for _, winner := range []string{"tester1", "tester2", "tester3", "tester2", "tester3"} {
		room := newRoom(rand.New(rand.NewPCG(1, 2)), calmConfig(engine.RulebookCoins), "K7QM", func(string) {})
		for _, name := range []string{"tester1", "tester2", "tester3"} {
			seated := &connection{outbox: make(chan []byte, 64)}
			room.handle(command{from: seated, message: protocol.FromClient{Type: "join", Name: name}})
			room.handle(command{from: seated, message: protocol.FromClient{Type: "ready", Ready: true}})
		}
		room.winner = winner
		room.handle(command{from: room.seats[0].conn, message: protocol.FromClient{Type: "start"}})
		room.clock.stopAll()
		if turn := engine.ViewFor(room.game, winner).TurnOf; turn != winner {
			t.Errorf("after %s won, the next game opened on %s", winner, turn)
		}
	}
}
