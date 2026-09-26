package engine

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestStealTakesTwoCoins(t *testing.T) {
	game := threePlayerGame(Captain, Duke, Contessa, Ambassador, Assassin, Captain)
	apply(t, game, Act{By: "tester1", Action: Steal, Target: "tester2"})
	respond(t, game, "tester2", Pass)
	respond(t, game, "tester3", Pass)

	expectStanding(t, game, standing{"tester1", 4, 2}, standing{"tester2", 0, 2})
}

func TestStealFromSomeoneWithOneCoinTakesOne(t *testing.T) {
	game := threePlayerGame(Captain, Duke, Contessa, Ambassador, Assassin, Captain)
	giveCoins(game, "tester2", 1)
	apply(t, game, Act{By: "tester1", Action: Steal, Target: "tester2"})
	respond(t, game, "tester2", Pass)
	respond(t, game, "tester3", Pass)

	expectStanding(t, game, standing{"tester1", 3, 2}, standing{"tester2", 0, 2})
}

func TestSomeoneWithNoCoinsIsNeitherOfferedNorAcceptedAsAStealTarget(t *testing.T) {
	game := threePlayerGame(Captain, Duke, Contessa, Ambassador, Assassin, Captain)
	giveCoins(game, "tester2", 0)
	for _, offered := range ViewFor(game, "tester1").YourActions {
		if offered.Name == "steal" && (len(offered.Targets) != 1 || offered.Targets[0] != "tester3") {
			t.Errorf("steal offers targets %v, expected only tester3", offered.Targets)
		}
	}
	_, err := game.Apply(Act{By: "tester1", Action: Steal, Target: "tester2"})
	refusal := refusalFrom(t, err)
	if refusal.Code != "invalid_target" {
		t.Errorf("code %q, expected invalid_target", refusal.Code)
	}
	if expected, listed := refusal.Expected.([]string); !listed || len(expected) != 1 || expected[0] != "tester3" {
		t.Errorf("the refusal expects %v, wanted [tester3]", refusal.Expected)
	}
}

func TestWithEveryoneBrokeStealDisappearsFromTheActions(t *testing.T) {
	game := threePlayerGame(Captain, Duke, Contessa, Ambassador, Assassin, Captain)
	giveCoins(game, "tester2", 0)
	giveCoins(game, "tester3", 0)
	for _, offered := range ViewFor(game, "tester1").YourActions {
		if offered.Name == "steal" {
			t.Errorf("steal is offered with nobody to steal from: %+v", offered)
		}
	}
}

func TestOnlyTheTargetBlocksAStealWithCaptainOrAmbassador(t *testing.T) {
	game := threePlayerGame(Captain, Duke, Contessa, Ambassador, Assassin, Captain)
	apply(t, game, Act{By: "tester1", Action: Steal, Target: "tester2"})

	target := ViewFor(game, "tester2").Window.YourOptions
	expected := []Option{{Answer: Challenge}, {Answer: Block, Character: Captain}, {Answer: Block, Character: Ambassador}, {Answer: Pass}}
	if len(target) != len(expected) {
		t.Fatalf("tester2 is offered %v, expected %v", target, expected)
	}
	for position := range expected {
		if target[position] != expected[position] {
			t.Errorf("option %d is %v, expected %v", position, target[position], expected[position])
		}
	}
	if bystander := ViewFor(game, "tester3").Window.YourOptions; len(bystander) != 2 {
		t.Errorf("tester3 is offered %v, expected challenge and pass", bystander)
	}
}

func TestTheClaimedBlockerDecidesTheChallenge(t *testing.T) {
	game := threePlayerGame(Captain, Duke, Contessa, Captain, Assassin, Duke)
	apply(t, game, Act{By: "tester1", Action: Steal, Target: "tester2"})
	blockWith(t, game, "tester2", Ambassador)
	respond(t, game, "tester1", Challenge)
	apply(t, game, LoseInfluence{By: "tester2", Card: Contessa})

	expectStanding(t, game, standing{"tester1", 4, 2}, standing{"tester2", 0, 1})
}

func TestExchangeDrawsTwoAndOffersEveryPairToReturn(t *testing.T) {
	game := threePlayerGame(Ambassador, Duke, Contessa, Captain, Assassin, Captain)
	deckBefore := len(game.deck)
	apply(t, game, Act{By: "tester1", Action: Exchange})
	respond(t, game, "tester2", Pass)
	respond(t, game, "tester3", Pass)

	snapshot := ViewFor(game, "tester1")
	if snapshot.Phase != "awaiting_exchange" {
		t.Fatalf("phase is %q, expected awaiting_exchange", snapshot.Phase)
	}
	mine := entryFor(t, snapshot, "tester1")
	if len(mine.MyCards) != 4 || len(game.deck) != deckBefore-2 {
		t.Errorf("tester1 holds %v and the deck %d, expected 4 cards and %d", mine.MyCards, len(game.deck), deckBefore-2)
	}
	if pairs := snapshot.YourReturns; len(pairs) == 0 || len(pairs) > 6 {
		t.Errorf("tester1 is offered %d pairs to return, expected between 1 and 6", len(pairs))
	}
	for _, pair := range snapshot.YourReturns {
		if _, held := handWithout(mine.MyCards, pair); !held {
			t.Errorf("the pair %v is offered but tester1 does not hold it", pair)
		}
	}
}

func TestReturningTwoCardsKeepsTheRestAndPassesTheTurn(t *testing.T) {
	game := threePlayerGame(Ambassador, Duke, Contessa, Captain, Assassin, Captain)
	deckBefore := len(game.deck)
	apply(t, game, Act{By: "tester1", Action: Exchange})
	respond(t, game, "tester2", Pass)
	respond(t, game, "tester3", Pass)
	pair := ViewFor(game, "tester1").YourReturns[0]
	apply(t, game, ReturnCards{By: "tester1", Cards: [2]Character{pair[0], pair[1]}})

	if held := len(game.players[0].hand); held != 2 || len(game.deck) != deckBefore {
		t.Errorf("tester1 holds %d cards and the deck %d, expected 2 and %d", held, len(game.deck), deckBefore)
	}
	if turn := ViewFor(game, "tester1").TurnOf; turn != "tester2" {
		t.Errorf("the turn went to %q, expected tester2", turn)
	}
}

func TestExchangeWithOneInfluenceKeepsOne(t *testing.T) {
	game := threePlayerGame(Ambassador, Duke, Contessa, Captain, Assassin, Captain)
	leaveOneCard(game, "tester1", Ambassador, Duke)
	apply(t, game, Act{By: "tester1", Action: Exchange})
	respond(t, game, "tester2", Pass)
	respond(t, game, "tester3", Pass)
	hand := handOf(game, "tester1")
	if len(hand) != 3 {
		t.Fatalf("tester1 holds %v during the exchange, expected 3 cards", hand)
	}
	apply(t, game, ReturnCards{By: "tester1", Cards: [2]Character{hand[1], hand[2]}})

	if kept := handOf(game, "tester1"); len(kept) != 1 || kept[0] != Ambassador {
		t.Errorf("tester1 kept %v, expected [ambassador]", kept)
	}
}

func TestReturningCardsYouDoNotHoldIsRefused(t *testing.T) {
	game := threePlayerGame(Ambassador, Duke, Contessa, Captain, Assassin, Captain)
	apply(t, game, Act{By: "tester1", Action: Exchange})
	respond(t, game, "tester2", Pass)
	respond(t, game, "tester3", Pass)
	game.players[0].hand = []Character{Ambassador, Duke, Contessa, Captain}

	for _, returned := range [][2]Character{{Assassin, Duke}, {Ambassador, Ambassador}, {NoCharacter, Duke}} {
		_, err := game.Apply(ReturnCards{By: "tester1", Cards: returned})
		if code := refusalFrom(t, err).Code; code != "illegal_action" {
			t.Errorf("returning %v answered %q, expected illegal_action", returned, code)
		}
	}
	if held := len(game.players[0].hand); held != 4 {
		t.Errorf("tester1 holds %d cards after the refusals, expected 4", held)
	}
}

func TestOnlyTheExchangerReturnsCards(t *testing.T) {
	game := threePlayerGame(Ambassador, Duke, Contessa, Captain, Assassin, Captain)
	apply(t, game, Act{By: "tester1", Action: Exchange})
	respond(t, game, "tester2", Pass)
	respond(t, game, "tester3", Pass)

	_, err := game.Apply(ReturnCards{By: "tester2", Cards: [2]Character{Contessa, Captain}})
	if code := refusalFrom(t, err).Code; code != "not_your_turn" {
		t.Errorf("code %q, expected not_your_turn", code)
	}
	_, err = threePlayerGame().Apply(ReturnCards{By: "tester1", Cards: [2]Character{Duke, Duke}})
	if code := refusalFrom(t, err).Code; code != "illegal_action" {
		t.Errorf("returning cards with no exchange answered %q, expected illegal_action", code)
	}
}

func TestTheCardsDrawnInAnExchangeStayHiddenFromEveryoneElse(t *testing.T) {
	game := threePlayerGame(Ambassador, Ambassador, Contessa, Contessa, Contessa, Ambassador)
	apply(t, game, Act{By: "tester1", Action: Exchange})
	respond(t, game, "tester2", Pass)
	events := respond(t, game, "tester3", Pass)
	if drawn := handOf(game, "tester1")[2:]; drawn[0] != Duke || drawn[1] != Duke {
		t.Fatalf("tester1 drew %v, expected the two dukes on top of the deck", drawn)
	}

	encoded, _ := json.Marshal(ViewFor(game, "tester2"))
	if strings.Contains(string(encoded), "duke") {
		t.Errorf("tester2's snapshot mentions the dukes tester1 just drew: %s", encoded)
	}
	for _, seen := range events {
		if strings.Contains(seen.Text, "Duque") {
			t.Errorf("event %q names a drawn card", seen.Text)
		}
	}
	if returns := ViewFor(game, "tester2").YourReturns; returns != nil {
		t.Errorf("tester2 is offered cards to return: %v", returns)
	}
}
