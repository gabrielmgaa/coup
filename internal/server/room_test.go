package server

import (
	"context"
	"encoding/json"
	"math/rand/v2"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/coder/websocket"
	"github.com/gabrielmgaa/coup/internal/engine"
)

type received struct {
	Type  string `json:"type"`
	Code  string `json:"code"`
	State struct {
		You     string `json:"you"`
		TurnOf  string `json:"turn_of"`
		Players []struct {
			Name  string `json:"name"`
			Coins int    `json:"coins"`
		} `json:"players"`
	} `json:"state"`
}

func (r received) coinsOf(t *testing.T, name string) int {
	t.Helper()
	for _, player := range r.State.Players {
		if player.Name == name {
			return player.Coins
		}
	}
	t.Fatalf("%q does not appear in the snapshot that arrived", name)
	return 0
}

type tab struct {
	t    *testing.T
	conn *websocket.Conn
}

func startServer(t *testing.T) string {
	t.Helper()
	running := httptest.NewServer(New(fstest.MapFS{}, rand.New(rand.NewPCG(1, 2)), engine.RulebookCoins))
	t.Cleanup(running.Close)
	return "ws" + strings.TrimPrefix(running.URL, "http") + "/ws"
}

func openTab(t *testing.T, url, name string) *tab {
	t.Helper()
	conn, _, err := websocket.Dial(context.Background(), url, nil)
	if err != nil {
		t.Fatalf("%s could not connect: %v", name, err)
	}
	t.Cleanup(func() { conn.CloseNow() })
	opened := &tab{t: t, conn: conn}
	opened.send(map[string]string{"type": "join", "name": name})
	return opened
}

func (a *tab) send(message any) {
	a.t.Helper()
	encoded, err := json.Marshal(message)
	if err != nil {
		a.t.Fatalf("message does not serialize: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := a.conn.Write(ctx, websocket.MessageText, encoded); err != nil {
		a.t.Fatalf("could not send: %v", err)
	}
}

func (a *tab) receive() received {
	a.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, encoded, err := a.conn.Read(ctx)
	if err != nil {
		a.t.Fatalf("nothing arrived: %v", err)
	}
	var message received
	if err := json.Unmarshal(encoded, &message); err != nil {
		a.t.Fatalf("unreadable message: %v", err)
	}
	return message
}

func (a *tab) nothingArrives(within time.Duration) {
	a.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), within)
	defer cancel()
	if _, encoded, err := a.conn.Read(ctx); err == nil {
		a.t.Fatalf("a message arrived that should not have: %s", encoded)
	}
}

func TestWhatOneTabDoesShowsUpInTheOther(t *testing.T) {
	url := startServer(t)
	isabely := openTab(t, url, "isabely")
	marina := openTab(t, url, "marina")

	first := isabely.receive()
	if first.State.TurnOf != "isabely" {
		t.Fatalf("the game started on %q; this test is written with isabely going first", first.State.TurnOf)
	}
	marina.receive()

	isabely.send(map[string]string{"type": "play", "action": "income"})

	inMarinasTab := marina.receive()
	if coins := inMarinasTab.coinsOf(t, "isabely"); coins != 2 {
		t.Errorf("marina's tab shows isabely with %d coins, expected 2", coins)
	}
	if inMarinasTab.State.You != "marina" {
		t.Errorf("the snapshot that reached marina belongs to %q — each player gets their own cut",
			inMarinasTab.State.You)
	}
}

func TestARefusalOnlyReachesThePlayerWhoCausedIt(t *testing.T) {
	url := startServer(t)
	isabely := openTab(t, url, "isabely")
	marina := openTab(t, url, "marina")
	isabely.receive()
	marina.receive()

	marina.send(map[string]string{"type": "play", "action": "income"})

	answer := marina.receive()
	if answer.Type != "error" || answer.Code != "not_your_turn" {
		t.Errorf("marina received %s/%s, expected error/not_your_turn", answer.Type, answer.Code)
	}
	isabely.nothingArrives(300 * time.Millisecond)
}

func TestAClientThatStopsReadingIsDropped(t *testing.T) {
	room := newRoom(rand.New(rand.NewPCG(1, 2)), engine.RulebookCoins)
	stalled := &connection{name: "marina", outbox: make(chan []byte, outboxCapacity)}
	attentive := &connection{name: "isabely", outbox: make(chan []byte, outboxCapacity)}
	room.connections = []*connection{attentive, stalled}
	room.game = engine.NewGame([]string{"isabely", "marina"}, rand.New(rand.NewPCG(1, 2)), engine.RulebookCoins)

	done := make(chan struct{})
	go func() {
		defer close(done)
		for range outboxCapacity {
			room.broadcast(nil)
			<-attentive.outbox
		}
		room.broadcast(nil)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("broadcast blocked waiting for marina's queue to drain — one stalled tab freezes the whole table")
	}

	if len(room.connections) != 1 || room.connections[0] != attentive {
		t.Errorf("the room kept %d connections, expected only isabely's", len(room.connections))
	}
	if !stalled.closed {
		t.Error("marina's connection was not dropped even with a full queue")
	}
}
