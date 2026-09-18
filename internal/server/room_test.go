package server

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/coder/websocket"
	"github.com/gabrielmgaa/coup/internal/engine"
	"github.com/gabrielmgaa/coup/internal/protocol"
)

type received struct {
	Type     string `json:"type"`
	Code     string `json:"code"`
	Received any    `json:"received"`
	Expected any    `json:"expected"`
	State    struct {
		Room    string `json:"room"`
		You     string `json:"you"`
		Host    string `json:"host"`
		TurnOf  string `json:"turn_of"`
		Players []struct {
			Name  string `json:"name"`
			Coins int    `json:"coins"`
			Ready bool   `json:"ready"`
		} `json:"players"`
	} `json:"state"`
}

func (r received) names() []string {
	seated := make([]string, 0, len(r.State.Players))
	for _, player := range r.State.Players {
		seated = append(seated, player.Name)
	}
	return seated
}

func (r received) coinsOf(t *testing.T, name string) int {
	t.Helper()
	for _, player := range r.State.Players {
		if player.Name == name {
			return player.Coins
		}
	}
	t.Fatalf("%q is not in the snapshot, which holds %v", name, r.names())
	return 0
}

func (r received) readyOf(t *testing.T, name string) bool {
	t.Helper()
	for _, player := range r.State.Players {
		if player.Name == name {
			return player.Ready
		}
	}
	t.Fatalf("%q is not in the lobby, which holds %v", name, r.names())
	return false
}

type tab struct {
	t    *testing.T
	conn *websocket.Conn
	room string
	name string
}

func startServer(t *testing.T) string {
	t.Helper()
	running := httptest.NewServer(New(fstest.MapFS{}, rand.New(rand.NewPCG(1, 2)), engine.RulebookCoins))
	t.Cleanup(running.Close)
	return "ws" + strings.TrimPrefix(running.URL, "http") + "/ws"
}

func dial(t *testing.T, url string) *tab {
	t.Helper()
	ctx, stop := context.WithTimeout(context.Background(), 2*time.Second)
	defer stop()
	conn, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		t.Fatalf("could not open the websocket: %v", err)
	}
	t.Cleanup(func() { conn.CloseNow() })
	return &tab{t: t, conn: conn}
}

func createTable(t *testing.T, url, name string) *tab {
	t.Helper()
	opened := dial(t, url)
	opened.name = name
	opened.send(protocol.FromClient{Type: "create_room", Name: name})
	opening := opened.receive()
	if opening.Type != "lobby" {
		t.Fatalf("create_room answered %q, expected lobby", opening.Type)
	}
	opened.room = opening.State.Room
	return opened
}

func enterTable(t *testing.T, url, code, name string) *tab {
	t.Helper()
	entering := dial(t, url)
	entering.room = code
	entering.name = strings.TrimSpace(name)
	entering.send(protocol.FromClient{Type: "join", Room: code, Name: name})
	return entering
}

func tableInPlay(t *testing.T, url string) (*tab, *tab) {
	t.Helper()
	tester1 := createTable(t, url, "tester1")
	tester2 := enterTable(t, url, tester1.room, "tester2")
	tester1.waitForSeats(2)
	tester1.send(protocol.FromClient{Type: "ready", Ready: true})
	tester2.send(protocol.FromClient{Type: "ready", Ready: true})
	tester1.waitForEveryoneReady(2)
	tester1.send(protocol.FromClient{Type: "start"})
	dealt := tester1.waitFor("update")
	tester2.waitFor("update")
	if dealt.State.TurnOf == tester2.name {
		return tester2, tester1
	}
	return tester1, tester2
}

func (a *tab) send(message any) {
	a.t.Helper()
	encoded, err := json.Marshal(message)
	if err != nil {
		a.t.Fatalf("the message does not serialize: %v", err)
	}
	ctx, stop := context.WithTimeout(context.Background(), time.Second)
	defer stop()
	if err := a.conn.Write(ctx, websocket.MessageText, encoded); err != nil {
		a.t.Fatalf("could not write to the socket: %v", err)
	}
}

func (a *tab) receive() received {
	a.t.Helper()
	ctx, stop := context.WithTimeout(context.Background(), 2*time.Second)
	defer stop()
	_, encoded, err := a.conn.Read(ctx)
	if err != nil {
		a.t.Fatalf("nothing arrived within 2s: %v", err)
	}
	var arrived received
	if err := json.Unmarshal(encoded, &arrived); err != nil {
		a.t.Fatalf("the server sent json that does not decode: %v — %s", err, encoded)
	}
	return arrived
}

func (a *tab) waitFor(kind string) received {
	a.t.Helper()
	for {
		arrived := a.receive()
		if arrived.Type == kind {
			return arrived
		}
	}
}

func (a *tab) waitForSeats(count int) received {
	a.t.Helper()
	for {
		if lobby := a.waitFor("lobby"); len(lobby.State.Players) >= count {
			return lobby
		}
	}
}

func (a *tab) waitForEveryoneReady(count int) received {
	a.t.Helper()
	for {
		lobby := a.waitFor("lobby")
		ready := 0
		for _, seated := range lobby.State.Players {
			if seated.Ready {
				ready++
			}
		}
		if len(lobby.State.Players) == count && ready == count {
			return lobby
		}
	}
}

func (a *tab) noUpdateArrives(within time.Duration) {
	a.t.Helper()
	ctx, stop := context.WithTimeout(context.Background(), within)
	defer stop()
	for {
		_, encoded, err := a.conn.Read(ctx)
		if err != nil {
			return
		}
		var arrived received
		if err := json.Unmarshal(encoded, &arrived); err == nil && arrived.Type == "update" {
			a.t.Fatalf("the game was dealt when it should not have been: %s", encoded)
		}
	}
}

func (a *tab) nothingArrives(within time.Duration) {
	a.t.Helper()
	ctx, stop := context.WithTimeout(context.Background(), within)
	defer stop()
	if _, encoded, err := a.conn.Read(ctx); err == nil {
		a.t.Fatalf("something arrived that should not have: %s", encoded)
	}
}

func decodeLobby(t *testing.T, encoded []byte) received {
	t.Helper()
	var arrived received
	if err := json.Unmarshal(encoded, &arrived); err != nil {
		t.Fatalf("the lobby message does not decode: %v — %s", err, encoded)
	}
	return arrived
}

func TestAnEmptyNameNeverSitsAtTheTable(t *testing.T) {
	url := startServer(t)
	tester1 := createTable(t, url, "tester1")

	intruder := dial(t, url)
	intruder.send(protocol.FromClient{Type: "join", Room: tester1.room, Name: ""})

	refused := intruder.receive()
	if refused.Code != "invalid_name" {
		t.Errorf("code %q, expected invalid_name", refused.Code)
	}
	if refused.Received != "" || refused.Expected != "2 a 16 caracteres" {
		t.Errorf("refusal says received %v expected %v, wanted \"\" and \"2 a 16 caracteres\"",
			refused.Received, refused.Expected)
	}
	tester1.nothingArrives(300 * time.Millisecond)
}

func TestTheNameLengthLimitsHoldOnBothEnds(t *testing.T) {
	url := startServer(t)
	host := createTable(t, url, "tester1")

	sixteen := strings.Repeat("tester2", 3)[:16]
	seventeen := sixteen + "a"
	for _, refusedName := range []string{"a", seventeen} {
		turned := dial(t, url)
		turned.send(protocol.FromClient{Type: "join", Room: host.room, Name: refusedName})
		if code := turned.receive().Code; code != "invalid_name" {
			t.Errorf("%q (%d chars) answered %q, expected invalid_name", refusedName, len(refusedName), code)
		}
	}
	for _, acceptedName := range []string{"ab", sixteen} {
		enterTable(t, url, host.room, acceptedName)
	}

	lobby := host.waitForSeats(3)
	if len(lobby.State.Players) != 3 {
		t.Errorf("the lobby holds %v, expected tester1 plus the two accepted names", lobby.names())
	}
}

func TestANameIsTrimmedBeforeItIsMeasuredAndBeforeItSits(t *testing.T) {
	url := startServer(t)
	host := createTable(t, url, "tester1")

	enterTable(t, url, host.room, "  tester2  ")
	lobby := host.waitFor("lobby")
	if seated := lobby.names(); len(seated) != 2 || seated[1] != "tester2" {
		t.Errorf("the lobby holds %v, expected [tester1 tester2] with no padding", seated)
	}

	tester3 := enterTable(t, url, host.room, "tester2")
	if code := tester3.receive().Code; code != "name_taken" {
		t.Errorf("a second tester2 answered %q, expected name_taken", code)
	}

	blank := dial(t, url)
	blank.send(protocol.FromClient{Type: "join", Room: host.room, Name: "   "})
	refused := blank.receive()
	if refused.Code != "invalid_name" {
		t.Errorf("three spaces answered %q, expected invalid_name", refused.Code)
	}
	if refused.Received != "   " {
		t.Errorf("the refusal reports received %q, expected the raw \"   \" the player sent", refused.Received)
	}
}

func TestARoomCodeIsFourCharactersWithNoLookalikes(t *testing.T) {
	url := startServer(t)
	tester1 := createTable(t, url, "tester1")

	if len(tester1.room) != 4 {
		t.Errorf("the room code %q has %d characters, expected 4", tester1.room, len(tester1.room))
	}
	desk := newRegistry(rand.New(rand.NewPCG(1, 2)), engine.RulebookCoins)
	for range 200 {
		code := desk.newCode()
		if len(code) != 4 {
			t.Fatalf("generated %q, expected 4 characters", code)
		}
		for _, letter := range code {
			if strings.ContainsRune("O0I1", letter) {
				t.Fatalf("generated %q, which contains %q — O, 0, I and 1 look alike when read aloud", code, letter)
			}
			if !strings.ContainsRune("23456789", letter) && !strings.ContainsRune("ABCDEFGHJKLMNPQRSTUVWXYZ", letter) {
				t.Fatalf("generated %q, which contains %q — expected an uppercase letter or a digit from 2 to 9", code, letter)
			}
		}
	}
}

func TestARepeatedCodeDoesNotStealTheRoomThatAlreadyHasIt(t *testing.T) {
	desk := newRegistry(rand.New(rand.NewPCG(1, 2)), engine.RulebookCoins)
	drawn := []string{"K7QM", "K7QM", "K7QN"}
	desk.newCode = func() string {
		next := drawn[0]
		drawn = drawn[1:]
		return next
	}

	first, refusal := desk.roomFor(protocol.FromClient{Type: "create_room"})
	if refusal != nil {
		t.Fatalf("create_room was refused: %v", refusal)
	}
	tester1 := &connection{outbox: make(chan []byte, outboxCapacity)}
	first.inbox <- command{from: tester1, message: protocol.FromClient{Type: "join", Name: "tester1"}}
	opening := decodeLobby(t, <-tester1.outbox)
	if opening.State.Room != "K7QM" {
		t.Fatalf("the first room got code %q, expected K7QM", opening.State.Room)
	}

	second, refusal := desk.roomFor(protocol.FromClient{Type: "create_room"})
	if refusal != nil {
		t.Fatalf("the second create_room was refused: %v", refusal)
	}
	if second.code == "K7QM" {
		t.Error("the second room took the code K7QM, which was already in use")
	}
	if desk.rooms["K7QM"] != first {
		t.Error("K7QM no longer points at the room tester1 is sitting in")
	}
	tester2 := &connection{outbox: make(chan []byte, outboxCapacity)}
	first.inbox <- command{from: tester2, message: protocol.FromClient{Type: "join", Name: "tester2"}}
	<-tester1.outbox
	stillThere := decodeLobby(t, <-tester2.outbox)
	if seated := stillThere.names(); len(seated) != 2 || seated[0] != "tester1" {
		t.Errorf("K7QM holds %v, expected tester1 still in her seat", seated)
	}
}

func TestATypoInTheRoomCodeIsRefusedInsteadOfOpeningANewRoom(t *testing.T) {
	url := startServer(t)
	createTable(t, url, "tester1")

	tester2 := dial(t, url)
	tester2.send(protocol.FromClient{Type: "join", Room: "K7QN", Name: "tester2"})

	refused := tester2.receive()
	if refused.Code != "room_not_found" {
		t.Errorf("code %q, expected room_not_found", refused.Code)
	}
	if refused.Received != "K7QN" {
		t.Errorf("the refusal reports received %v, expected K7QN", refused.Received)
	}
}

func TestTwoRoomsDoNotSeeEachOther(t *testing.T) {
	url := startServer(t)
	tester1 := createTable(t, url, "tester1")
	tester2 := enterTable(t, url, tester1.room, "tester2")
	tester3 := createTable(t, url, "tester3")
	enterTable(t, url, tester3.room, "tester4")
	tester3.waitForSeats(2)
	tester2.waitForSeats(2)

	tester1.send(protocol.FromClient{Type: "ready", Ready: true})

	seen := tester2.waitFor("lobby")
	if !seen.readyOf(t, "tester1") {
		t.Error("tester2 did not see tester1 mark ready in their own room")
	}
	tester3.nothingArrives(300 * time.Millisecond)
}

func TestTheSeventhPlayerIsTurnedAway(t *testing.T) {
	url := startServer(t)
	tester1 := createTable(t, url, "tester1")
	for _, name := range []string{"tester2", "tester3", "tester4", "tester5", "tester6"} {
		enterTable(t, url, tester1.room, name)
	}

	tester5 := dial(t, url)
	tester5.send(protocol.FromClient{Type: "join", Room: tester1.room, Name: "tester7"})

	refused := tester5.receive()
	if refused.Code != "room_full" {
		t.Errorf("code %q, expected room_full", refused.Code)
	}
	if refused.Received != float64(7) || refused.Expected != float64(engine.MaxPlayers) {
		t.Errorf("refusal says received %v expected %v, wanted 7 and 6", refused.Received, refused.Expected)
	}
	lobby := tester1.waitForSeats(6)
	if len(lobby.State.Players) != 6 {
		t.Errorf("the lobby holds %d people, expected 6", len(lobby.State.Players))
	}
}

func TestSomeoneArrivingAfterTheStartIsToldTheGameBegan(t *testing.T) {
	url := startServer(t)
	onTurn, waiting := tableInPlay(t, url)

	tester3 := dial(t, url)
	tester3.send(protocol.FromClient{Type: "join", Room: onTurn.room, Name: "tester3"})

	if code := tester3.receive().Code; code != "game_started" {
		t.Errorf("code %q, expected game_started — room_full says the wrong thing here", code)
	}
	onTurn.send(protocol.FromClient{Type: "play", Action: "income"})
	if turn := onTurn.waitFor("update").State.TurnOf; turn != waiting.name {
		t.Errorf("the turn is %q, expected %s — the late arrival disturbed the game", turn, waiting.name)
	}
}

func TestWhoeverOpensTheRoomIsTheHost(t *testing.T) {
	url := startServer(t)
	tester1 := createTable(t, url, "tester1")
	tester2 := enterTable(t, url, tester1.room, "tester2")

	if host := tester2.waitFor("lobby").State.Host; host != "tester1" {
		t.Errorf("tester2 sees host %q, expected tester1", host)
	}
	if host := tester1.waitFor("lobby").State.Host; host != "tester1" {
		t.Errorf("tester1 sees host %q, expected herself", host)
	}
}

func TestWhenTheHostLeavesTheOldestSeatTakesOver(t *testing.T) {
	url := startServer(t)
	tester1 := createTable(t, url, "tester1")
	tester2 := enterTable(t, url, tester1.room, "tester2")
	enterTable(t, url, tester1.room, "tester3")
	tester2.waitForSeats(3)

	tester1.conn.CloseNow()

	lobby := tester2.waitFor("lobby")
	for len(lobby.State.Players) > 2 {
		lobby = tester2.waitFor("lobby")
	}
	if lobby.State.Host != "tester2" {
		t.Errorf("the host is %q, expected tester2 — the oldest seat still connected", lobby.State.Host)
	}
	if len(lobby.State.Players) != 2 {
		t.Errorf("the lobby holds %v, expected tester2 and tester3", lobby.names())
	}
}

func TestOnlyTheHostCanStartTheGame(t *testing.T) {
	url := startServer(t)
	tester1 := createTable(t, url, "tester1")
	tester2 := enterTable(t, url, tester1.room, "tester2")
	tester1.waitForSeats(2)
	tester1.send(protocol.FromClient{Type: "ready", Ready: true})
	tester2.send(protocol.FromClient{Type: "ready", Ready: true})

	tester2.send(protocol.FromClient{Type: "start"})

	refused := tester2.waitFor("error")
	if refused.Code != "not_host" {
		t.Errorf("code %q, expected not_host", refused.Code)
	}
	if refused.Received != "tester2" || refused.Expected != "tester1" {
		t.Errorf("refusal says received %v expected %v, wanted tester2 and tester1",
			refused.Received, refused.Expected)
	}
	tester1.send(protocol.FromClient{Type: "ready", Ready: false})
	if kind := tester1.waitFor("lobby").Type; kind != "lobby" {
		t.Errorf("tester1 is receiving %q, expected to still be in the lobby", kind)
	}
}

func TestTheGameDoesNotStartWhileSomeoneIsNotReady(t *testing.T) {
	url := startServer(t)
	tester1 := createTable(t, url, "tester1")
	tester2 := enterTable(t, url, tester1.room, "tester2")
	tester1.waitForSeats(2)
	tester1.send(protocol.FromClient{Type: "ready", Ready: true})

	tester1.send(protocol.FromClient{Type: "start"})

	refused := tester1.waitFor("error")
	if refused.Code != "not_all_ready" {
		t.Errorf("code %q, expected not_all_ready", refused.Code)
	}
	if fmt.Sprint(refused.Received) != "[tester2]" {
		t.Errorf("the refusal names %v as missing, expected [tester2]", refused.Received)
	}
	tester2.noUpdateArrives(300 * time.Millisecond)
}

func TestAPlayerAloneCannotStart(t *testing.T) {
	url := startServer(t)
	tester1 := createTable(t, url, "tester1")
	tester1.send(protocol.FromClient{Type: "ready", Ready: true})

	tester1.send(protocol.FromClient{Type: "start"})

	refused := tester1.waitFor("error")
	if refused.Code != "not_enough_players" {
		t.Errorf("code %q, expected not_enough_players", refused.Code)
	}
	if refused.Received != float64(1) || refused.Expected != "2 a 6" {
		t.Errorf("refusal says received %v expected %v, wanted 1 and \"2 a 6\"",
			refused.Received, refused.Expected)
	}
	tester1.nothingArrives(300 * time.Millisecond)
}

func TestStartDealsTheGameAndClosesTheDoor(t *testing.T) {
	url := startServer(t)
	tester1 := createTable(t, url, "tester1")
	tester2 := enterTable(t, url, tester1.room, "tester2")
	tester3 := enterTable(t, url, tester1.room, "tester3")
	tester1.waitForSeats(3)
	for _, playing := range []*tab{tester1, tester2, tester3} {
		playing.send(protocol.FromClient{Type: "ready", Ready: true})
	}
	tester1.waitForEveryoneReady(3)

	tester1.send(protocol.FromClient{Type: "start"})

	dealt := tester3.waitFor("update")
	if len(dealt.State.Players) != 3 {
		t.Errorf("the game was dealt to %v, expected three players", dealt.names())
	}
	for _, name := range []string{"tester1", "tester2", "tester3"} {
		if coins := dealt.coinsOf(t, name); coins != 2 {
			t.Errorf("%s starts with %d coins, expected 2 at a three-player table", name, coins)
		}
	}
	tester4 := dial(t, url)
	tester4.send(protocol.FromClient{Type: "join", Room: tester1.room, Name: "tester4"})
	if code := tester4.receive().Code; code != "game_started" {
		t.Errorf("tester4 was answered %q, expected game_started", code)
	}
}

func TestMarkingReadyShowsUpOnEveryoneElsesScreen(t *testing.T) {
	url := startServer(t)
	tester1 := createTable(t, url, "tester1")
	tester2 := enterTable(t, url, tester1.room, "tester2")
	tester1.waitForSeats(2)

	tester2.send(protocol.FromClient{Type: "ready", Ready: true})

	seen := tester1.waitFor("lobby")
	if !seen.readyOf(t, "tester2") {
		t.Error("tester1 does not see tester2 as ready")
	}
	if seen.readyOf(t, "tester1") {
		t.Error("tester1 was marked ready by tester2's click")
	}
}

func TestClosingATabInTheLobbyRemovesThatNameForEveryone(t *testing.T) {
	url := startServer(t)
	tester1 := createTable(t, url, "tester1")
	tester2 := enterTable(t, url, tester1.room, "tester2")
	tester3 := enterTable(t, url, tester1.room, "tester3")
	tester1.waitForSeats(3)
	tester2.waitForSeats(3)

	tester3.conn.CloseNow()

	lobby := tester1.waitFor("lobby")
	for len(lobby.State.Players) > 2 {
		lobby = tester1.waitFor("lobby")
	}
	if seated := lobby.names(); len(seated) != 2 {
		t.Errorf("the lobby holds %v, expected tester1 and tester2 only", seated)
	}
	for _, seated := range lobby.names() {
		if seated == "tester3" {
			t.Error("tester3 closed his tab and is still listed")
		}
	}
}

func TestWhatOneTabDoesShowsUpInTheOther(t *testing.T) {
	url := startServer(t)
	onTurn, waiting := tableInPlay(t, url)

	onTurn.send(protocol.FromClient{Type: "play", Action: "income"})

	seen := waiting.waitFor("update")
	if coins := seen.coinsOf(t, onTurn.name); coins != 2 {
		t.Errorf("%s sees %s with %d coins after income, expected 2", waiting.name, onTurn.name, coins)
	}
	if seen.State.You != waiting.name {
		t.Errorf("the snapshot says you = %q, expected %s", seen.State.You, waiting.name)
	}
}

func TestARefusalOnlyReachesThePlayerWhoCausedIt(t *testing.T) {
	url := startServer(t)
	onTurn, waiting := tableInPlay(t, url)

	waiting.send(protocol.FromClient{Type: "play", Action: "income"})

	refused := waiting.waitFor("error")
	if refused.Code != "not_your_turn" {
		t.Errorf("code %q, expected not_your_turn", refused.Code)
	}
	onTurn.nothingArrives(300 * time.Millisecond)
}

func TestAClientThatStopsReadingIsDropped(t *testing.T) {
	room := newRoom(rand.New(rand.NewPCG(1, 2)), engine.RulebookCoins, "K7QM")
	stalled := &connection{name: "tester2", outbox: make(chan []byte, outboxCapacity)}
	attentive := &connection{name: "tester1", outbox: make(chan []byte, outboxCapacity)}
	room.connections = []*connection{attentive, stalled}
	dealt, err := engine.NewGame([]string{"tester1", "tester2"}, rand.New(rand.NewPCG(1, 2)), engine.RulebookCoins)
	if err != nil {
		t.Fatalf("the two player game was refused: %v", err)
	}
	room.game = dealt

	done := make(chan struct{})
	go func() {
		for range outboxCapacity {
			room.broadcast(nil)
			<-attentive.outbox
		}
		room.broadcast(nil)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("broadcast blocked on the client that stopped reading")
	}

	if len(room.connections) != 1 || room.connections[0] != attentive {
		t.Errorf("the room kept %d connections, expected only the attentive one", len(room.connections))
	}
	if !stalled.closed {
		t.Error("the stalled connection was not closed")
	}
}
