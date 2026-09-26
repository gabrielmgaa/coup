package protocol

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/gabrielmgaa/coup/internal/engine"
)

func refusalCode(t *testing.T, err error) string {
	t.Helper()
	var refusal *engine.Refusal
	if !errors.As(err, &refusal) {
		t.Fatalf("expected a refusal, got %v", err)
	}
	return refusal.Code
}

func TestEveryMoveIsSignedByTheConnectionNotByTheMessage(t *testing.T) {
	for _, message := range []FromClient{
		{Type: "play", Action: "income", Name: "tester2"},
		{Type: "lose_influence", Card: "duke", Name: "tester2"},
		{Type: "respond", Window: 3, Answer: "pass", Name: "tester2"},
		{Type: "return_cards", Cards: []string{"duke", "captain"}, Name: "tester2"},
	} {
		move, err := ToMove(message, "tester1")
		if err != nil {
			t.Fatalf("%s was refused: %v", message.Type, err)
		}
		encoded, _ := json.Marshal(move)
		if !strings.Contains(string(encoded), `"By":"tester1"`) {
			t.Errorf("%s produced %s, expected it signed by tester1", message.Type, encoded)
		}
	}
}

func TestARespondCarriesWindowAnswerAndCharacter(t *testing.T) {
	move, err := ToMove(FromClient{Type: "respond", Window: 9, Answer: "block", Character: "contessa"}, "tester2")
	if err != nil {
		t.Fatalf("respond was refused: %v", err)
	}
	expected := engine.Respond{By: "tester2", Window: 9, Answer: engine.Block, Character: engine.Contessa}
	if move != expected {
		t.Errorf("respond became %+v, expected %+v", move, expected)
	}
}

func TestMalformedMovesAreRefusedBeforeReachingTheEngine(t *testing.T) {
	for _, message := range []FromClient{
		{Type: "play", Action: "print_money"},
		{Type: "lose_influence", Card: "king"},
		{Type: "respond", Answer: "cheat"},
		{Type: "respond", Answer: "block", Character: "king"},
		{Type: "return_cards", Cards: []string{"duke"}},
		{Type: "return_cards", Cards: []string{"duke", "duke", "duke"}},
		{Type: "return_cards", Cards: []string{"duke", "king"}},
		{Type: "deadline"},
		{Type: ""},
	} {
		_, err := ToMove(message, "tester1")
		if code := refusalCode(t, err); code != "illegal_action" {
			t.Errorf("%+v answered %q, expected illegal_action", message, code)
		}
	}
}

func TestAnUpdateNeverCarriesNullEvents(t *testing.T) {
	encoded, _ := json.Marshal(NewUpdate(GameState{}, nil))
	if !strings.Contains(string(encoded), `"events":[]`) {
		t.Errorf("the update serialized as %s, expected an empty events list", encoded)
	}
}

func TestLobbyAndRefusalCarryTheirType(t *testing.T) {
	lobby, _ := json.Marshal(NewLobby(LobbyView{Room: "K7QM"}))
	refusal, _ := json.Marshal(NewRefusal(&engine.Refusal{Code: "room_full"}))
	if !strings.HasPrefix(string(lobby), `{"type":"lobby"`) || !strings.HasPrefix(string(refusal), `{"type":"error"`) {
		t.Errorf("lobby %s and refusal %s do not lead with their type", lobby, refusal)
	}
}

func TestAWelcomeCarriesRoomAndToken(t *testing.T) {
	encoded, _ := json.Marshal(NewWelcome("K7QM", "secret"))
	if string(encoded) != `{"type":"welcome","room":"K7QM","token":"secret"}` {
		t.Errorf("the welcome serialized as %s", encoded)
	}
}
