package engine

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestCardsAndAnswersRoundTripThroughJSON(t *testing.T) {
	sent := Option{Answer: Block, Character: Ambassador}
	encoded, err := json.Marshal(sent)
	if err != nil {
		t.Fatalf("the option does not serialize: %v", err)
	}
	if string(encoded) != `{"answer":"block","character":"ambassador"}` {
		t.Errorf("the option serialized as %s", encoded)
	}
	var received Option
	if err := json.Unmarshal(encoded, &received); err != nil || received != sent {
		t.Errorf("the option came back as %+v, %v; expected %+v", received, err, sent)
	}
}

func TestUnknownNamesOnTheWireAreRefusals(t *testing.T) {
	var card Character
	var answer Answer
	for _, attempt := range []error{
		json.Unmarshal([]byte(`"king"`), &card),
		json.Unmarshal([]byte(`"cheat"`), &answer),
	} {
		var refusal *Refusal
		if !errors.As(attempt, &refusal) || refusal.Code != "illegal_action" {
			t.Errorf("an unknown name decoded with %v, expected an illegal_action refusal", attempt)
		}
	}
	for _, attempt := range []error{json.Unmarshal([]byte(`3`), &card), json.Unmarshal([]byte(`3`), &answer)} {
		if attempt == nil {
			t.Error("a number decoded as a name")
		}
	}
}

func TestARefusalReadsWithBothValues(t *testing.T) {
	written := (&Refusal{Code: "window_closed", Message: "a janela 3 já fechou", Received: 3, Expected: 4}).Error()
	if !strings.Contains(written, "received 3") || !strings.Contains(written, "expected 4") {
		t.Errorf("the refusal reads %q, expected both values in it", written)
	}
}

func TestNamesResolveBothWays(t *testing.T) {
	for _, name := range CharacterNames() {
		if card, known := CharacterByName(name); !known || card.String() != name {
			t.Errorf("%q resolved to %v, %v", name, card, known)
		}
	}
	if _, known := CharacterByName("king"); known {
		t.Error("king resolved to a character")
	}
	for _, name := range ActionNames() {
		if action, known := ActionByName(name); !known || rules[action].Name != name {
			t.Errorf("%q resolved to %v, %v", name, action, known)
		}
	}
	if _, known := ActionByName("print_money"); known {
		t.Error("print_money resolved to an action")
	}
	if _, known := AnswerByName("pass"); !known {
		t.Error("pass did not resolve")
	}
}

type foreignMove struct{ Move }

func TestAMoveFromOutsideTheUnionIsRefused(t *testing.T) {
	_, err := threePlayerGame().Apply(foreignMove{})
	if code := refusalFrom(t, err).Code; code != "illegal_action" {
		t.Errorf("code %q, expected illegal_action", code)
	}
}
