package server

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/gabrielmgaa/coup/internal/engine"
	"github.com/gabrielmgaa/coup/internal/protocol"
)

func pingingConfig() Config {
	config := calmConfig(engine.RulebookCoins)
	config.PingEvery, config.PongWait = 100*time.Millisecond, 100*time.Millisecond
	return config
}

func TestAClientThatStopsReadingIsMarkedDisconnected(t *testing.T) {
	url := startServerWith(t, pingingConfig())
	playing := seatTable(t, url, "tester1", "tester2")
	listener := playing.onTurn()
	deaf := playing.someoneElse(listener)

	ctx, stop := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer stop()
	for {
		_, encoded, err := playing.seats[listener].conn.Read(ctx)
		if err != nil {
			t.Fatalf("after 500ms %s, who stopped reading, is still not listed as disconnected", deaf)
		}
		var arrived struct {
			State protocol.GameState `json:"state"`
		}
		if json.Unmarshal(encoded, &arrived) == nil && contains(arrived.State.Disconnected, deaf) {
			return
		}
	}
}

func TestAClientThatKeepsReadingIsNotDropped(t *testing.T) {
	url := startServerWith(t, pingingConfig())
	playing := seatTable(t, url, "tester1", "tester2")
	actor := playing.onTurn()
	seen := make(chan protocol.GameState, 64)
	for _, seated := range playing.seats {
		go readEveryStateInto(seated, seen)
	}

	time.Sleep(time.Second)
	playing.seats[actor].send(protocol.FromClient{Type: "play", Action: "income"})

	waited := time.After(time.Second)
	for {
		select {
		case state := <-seen:
			if len(state.Disconnected) > 0 {
				t.Fatalf("%v were marked disconnected while reading every message", state.Disconnected)
			}
			if state.TurnOf != actor {
				return
			}
		case <-waited:
			t.Fatalf("the income from %s never arrived after a second of pings", actor)
		}
	}
}

func readEveryStateInto(seated *tab, seen chan<- protocol.GameState) {
	for {
		_, encoded, err := seated.conn.Read(context.Background())
		if err != nil {
			return
		}
		var arrived struct {
			Type  string             `json:"type"`
			State protocol.GameState `json:"state"`
		}
		if json.Unmarshal(encoded, &arrived) == nil && arrived.Type == "update" {
			seen <- arrived.State
		}
	}
}

func TestAClientThatStopsReadingIsDroppedByTheWriteDeadline(t *testing.T) {
	config := calmConfig(engine.RulebookCoins)
	config.WriteWait = 200 * time.Millisecond
	url := startServerWith(t, config)
	host := createTable(t, url, "tester1")
	enterTable(t, url, host.room, "tester2")
	host.waitForSeats(2)

	for toggle := 0; toggle < 10000; toggle++ {
		host.send(protocol.FromClient{Type: "ready", Ready: toggle%2 == 0})
		if seated := host.waitFor("lobby").names(); len(seated) == 1 {
			return
		}
	}
	ctx, stop := context.WithTimeout(context.Background(), 3*time.Second)
	defer stop()
	for {
		_, encoded, err := host.conn.Read(ctx)
		if err != nil {
			t.Fatal("3s after the last toggle the lobby still seats tester1 and tester2, expected tester1 alone — tester2 stopped reading")
		}
		var arrived received
		if json.Unmarshal(encoded, &arrived) == nil && arrived.Type == "lobby" && len(arrived.State.Players) == 1 {
			return
		}
	}
}
