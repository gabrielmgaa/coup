package engine

import (
	"slices"
	"strings"
	"testing"
)

func openWindowID(t *testing.T, game *Game) int {
	t.Helper()
	window := ViewFor(game, "tester1").Window
	if window == nil {
		t.Fatalf("no window is open; phase is %s", game.phase)
	}
	return window.ID
}

func respond(t *testing.T, game *Game, by string, answer Answer) []Event {
	t.Helper()
	events, err := game.Apply(Respond{By: by, Window: openWindowID(t, game), Answer: answer})
	if err != nil {
		t.Fatalf("%s answering %s was refused: %v", by, answer, err)
	}
	return events
}

func handOf(game *Game, name string) []Character {
	return append([]Character(nil), game.players[game.indexOf(name)].hand...)
}

func TestTaxOpensAWindowForEveryoneButTheClaimant(t *testing.T) {
	game := threePlayerGame(Duke, Captain, Contessa, Ambassador, Assassin, Captain)
	apply(t, game, Act{By: "tester1", Action: Tax})

	snapshot := ViewFor(game, "tester2")
	if snapshot.Phase != "awaiting_response" {
		t.Fatalf("phase is %q, expected awaiting_response", snapshot.Phase)
	}
	offered := snapshot.Window.YourOptions
	if len(offered) != 2 || offered[0].Answer != Challenge || offered[1].Answer != Pass {
		t.Errorf("tester2 is offered %v, expected challenge and pass", offered)
	}
	if mine := ViewFor(game, "tester1").Window.YourOptions; len(mine) != 0 {
		t.Errorf("tester1 is offered %v on her own claim, expected nothing", mine)
	}
	if coins := entryFor(t, snapshot, "tester1").Coins; coins != 2 {
		t.Errorf("tester1 holds %d coins before anyone answered, expected 2", coins)
	}
	if claims := snapshot.Window.Action.Claims; claims != Duke {
		t.Errorf("the window shows the claim %v, expected duke", claims)
	}
}

func TestTaxNobodyChallengesPaysThree(t *testing.T) {
	game := threePlayerGame(Duke, Captain, Contessa, Ambassador, Assassin, Captain)
	apply(t, game, Act{By: "tester1", Action: Tax})
	respond(t, game, "tester2", Pass)
	if coins := game.players[0].coins; coins != 2 {
		t.Errorf("tester1 was paid after one pass of two, holding %d, expected 2", coins)
	}
	respond(t, game, "tester3", Pass)

	snapshot := ViewFor(game, "tester1")
	if coins := entryFor(t, snapshot, "tester1").Coins; coins != 5 {
		t.Errorf("tester1 ended with %d coins, expected 5", coins)
	}
	if snapshot.TurnOf != "tester2" || snapshot.Window != nil {
		t.Errorf("turn %q with window %v, expected tester2 and no window", snapshot.TurnOf, snapshot.Window)
	}
}

func TestTaxBluffCaughtCostsAnInfluenceAndPaysNothing(t *testing.T) {
	game := threePlayerGame(Captain, Contessa, Duke, Ambassador, Assassin, Captain)
	apply(t, game, Act{By: "tester1", Action: Tax})
	respond(t, game, "tester2", Challenge)

	if losing := ViewFor(game, "tester1").Losing; losing != "tester1" {
		t.Fatalf("the game waits on %q to reveal, expected tester1 who bluffed", losing)
	}
	apply(t, game, LoseInfluence{By: "tester1", Card: Captain})

	snapshot := ViewFor(game, "tester1")
	bluffer, challenger := entryFor(t, snapshot, "tester1"), entryFor(t, snapshot, "tester2")
	if bluffer.Coins != 2 || bluffer.Hidden != 1 {
		t.Errorf("tester1 has %d coins and %d cards, expected 2 and 1", bluffer.Coins, bluffer.Hidden)
	}
	if challenger.Coins != 2 || challenger.Hidden != 2 {
		t.Errorf("tester2 has %d coins and %d cards, expected 2 and 2", challenger.Coins, challenger.Hidden)
	}
	if snapshot.TurnOf != "tester2" {
		t.Errorf("the turn went to %q, expected tester2", snapshot.TurnOf)
	}
}

func TestTaxChallengedWithTheDukeCostsTheChallengerAndSwapsTheCard(t *testing.T) {
	game := threePlayerGame(Duke, Captain, Contessa, Ambassador, Assassin, Captain)
	deckBefore := len(game.deck)
	apply(t, game, Act{By: "tester1", Action: Tax})
	events := respond(t, game, "tester2", Challenge)

	if !containsType(events, "card_swapped") {
		t.Errorf("the proven duke was not swapped; events were %v", typesOf(events))
	}
	if len(game.players[0].hand) != 2 || len(game.deck) != deckBefore {
		t.Errorf("tester1 holds %d cards and the deck %d, expected 2 and %d",
			len(game.players[0].hand), len(game.deck), deckBefore)
	}
	if losing := ViewFor(game, "tester1").Losing; losing != "tester2" {
		t.Fatalf("the game waits on %q, expected tester2 who challenged a real duke", losing)
	}
	apply(t, game, LoseInfluence{By: "tester2", Card: Contessa})

	snapshot := ViewFor(game, "tester1")
	if coins := entryFor(t, snapshot, "tester1").Coins; coins != 5 {
		t.Errorf("tester1 ended with %d coins, expected 5 — the tax survived the challenge", coins)
	}
	if hidden := entryFor(t, snapshot, "tester2").Hidden; hidden != 1 {
		t.Errorf("tester2 kept %d cards, expected 1", hidden)
	}
	if snapshot.TurnOf != "tester2" {
		t.Errorf("the turn went to %q, expected tester2", snapshot.TurnOf)
	}
}

func TestTheSwapNeverNamesTheCardThatWasDrawn(t *testing.T) {
	game := threePlayerGame(Duke, Captain, Contessa, Ambassador, Assassin, Captain)
	apply(t, game, Act{By: "tester1", Action: Tax})
	events := respond(t, game, "tester2", Challenge)

	drawn := handOf(game, "tester1")[1]
	if drawn == Duke {
		t.Skip("the seed drew another duke; the leak cannot be told apart from the shown card")
	}
	for _, seen := range events {
		if strings.Contains(seen.Text, drawn.LabelPtBR()) {
			t.Errorf("event %q names the drawn %s, which only tester1 may know", seen.Text, drawn.LabelPtBR())
		}
	}
}

func TestTheFirstChallengeClosesTheWindowForEveryoneElse(t *testing.T) {
	game := threePlayerGame(Duke, Captain, Contessa, Ambassador, Assassin, Captain)
	apply(t, game, Act{By: "tester1", Action: Tax})
	window := openWindowID(t, game)
	respond(t, game, "tester2", Challenge)

	_, err := game.Apply(Respond{By: "tester3", Window: window, Answer: Challenge})
	if code := refusalFrom(t, err).Code; code != "illegal_action" {
		t.Errorf("code %q, expected illegal_action — the window already closed", code)
	}
	if hidden := len(game.players[2].hand); hidden != 2 {
		t.Errorf("tester3 holds %d cards, expected 2 — a late challenge must cost nothing", hidden)
	}
}

func TestAnswerToAnOldWindowIsRefusedWithBothIDs(t *testing.T) {
	game := threePlayerGame(Duke, Captain, Duke, Ambassador, Assassin, Captain)
	apply(t, game, Act{By: "tester1", Action: Tax})
	old := openWindowID(t, game)
	respond(t, game, "tester2", Pass)
	respond(t, game, "tester3", Pass)
	apply(t, game, Act{By: "tester2", Action: Tax})
	current := openWindowID(t, game)

	_, err := game.Apply(Respond{By: "tester3", Window: old, Answer: Challenge})
	refusal := refusalFrom(t, err)
	if refusal.Code != "window_closed" {
		t.Errorf("code %q, expected window_closed", refusal.Code)
	}
	if refusal.Received != old || refusal.Expected != current {
		t.Errorf("refusal says received %v expected %v, wanted %d and %d", refusal.Received, refusal.Expected, old, current)
	}
}

func TestNobodyAnswersTheSameWindowTwice(t *testing.T) {
	game := threePlayerGame(Captain, Contessa, Duke, Ambassador, Assassin, Captain)
	apply(t, game, Act{By: "tester1", Action: Tax})
	respond(t, game, "tester2", Pass)

	_, err := game.Apply(Respond{By: "tester2", Window: openWindowID(t, game), Answer: Challenge})
	refusal := refusalFrom(t, err)
	if refusal.Code != "already_responded" {
		t.Errorf("code %q, expected already_responded", refusal.Code)
	}
	if hidden := len(game.players[0].hand); hidden != 2 {
		t.Errorf("tester1 lost a card to a second answer from tester2; she holds %d", hidden)
	}
}

func TestTheClaimantAnsweringTheirOwnClaimIsToldWhoIsAwaited(t *testing.T) {
	game := threePlayerGame(Captain, Contessa, Duke, Ambassador, Assassin, Captain)
	apply(t, game, Act{By: "tester1", Action: Tax})

	_, err := game.Apply(Respond{By: "tester1", Window: openWindowID(t, game), Answer: Pass})
	refusal := refusalFrom(t, err)
	if refusal.Code != "not_your_turn" {
		t.Errorf("code %q, expected not_your_turn", refusal.Code)
	}
	if awaited, listed := refusal.Expected.([]string); !listed || !slices.Equal(awaited, []string{"tester2", "tester3"}) {
		t.Errorf("the refusal expects %v, expected [tester2 tester3]", refusal.Expected)
	}
}

func TestSomeoneOutsideTheGameCannotAnswer(t *testing.T) {
	game := threePlayerGame(Captain, Contessa, Duke, Ambassador, Assassin, Captain)
	apply(t, game, Act{By: "tester1", Action: Tax})

	_, err := game.Apply(Respond{By: "stranger", Window: openWindowID(t, game), Answer: Challenge})
	if code := refusalFrom(t, err).Code; code != "illegal_action" {
		t.Errorf("code %q, expected illegal_action", code)
	}
}

func TestAnEliminatedPlayerIsNotAskedToAnswer(t *testing.T) {
	game := threePlayerGame(Captain, Contessa, Duke, Ambassador, Assassin, Captain)
	game.players[2].revealed = append(game.players[2].revealed, game.players[2].hand...)
	game.players[2].hand = nil
	apply(t, game, Act{By: "tester1", Action: Tax})

	if waiting := ViewFor(game, "tester1").Window.WaitingOn; len(waiting) != 1 || waiting[0] != "tester2" {
		t.Errorf("the window waits on %v, expected only tester2", waiting)
	}
	_, err := game.Apply(Respond{By: "tester3", Window: openWindowID(t, game), Answer: Challenge})
	if code := refusalFrom(t, err).Code; code != "not_your_turn" {
		t.Errorf("code %q, expected not_your_turn — the window does not wait on an eliminated player", code)
	}
}

func TestAnsweringWithNoWindowOpenIsRefused(t *testing.T) {
	game := threePlayerGame()
	_, err := game.Apply(Respond{By: "tester2", Window: 0, Answer: Challenge})
	if code := refusalFrom(t, err).Code; code != "illegal_action" {
		t.Errorf("code %q, expected illegal_action", code)
	}
}

func TestIncomeAndCoupOpenNoWindow(t *testing.T) {
	game := threePlayerGame(Duke, Captain, Contessa, Ambassador, Assassin, Captain)
	apply(t, game, Act{By: "tester1", Action: Income})
	if window := ViewFor(game, "tester1").Window; window != nil {
		t.Errorf("income opened window %v", window)
	}
	giveCoins(game, "tester2", 7)
	apply(t, game, Act{By: "tester2", Action: Coup, Target: "tester3"})
	if phase := ViewFor(game, "tester1").Phase; phase != "awaiting_influence_loss" {
		t.Errorf("phase after the coup is %q, expected awaiting_influence_loss — nobody may answer a coup", phase)
	}
}

func TestLosingTheLastCardToAChallengeEndsTheGameBeforeTheTaxIsPaid(t *testing.T) {
	game := twoPlayerGame(Duke, Captain, Contessa, Ambassador)
	leaveOneCard(game, "tester2", Contessa, Ambassador)
	apply(t, game, Act{By: "tester1", Action: Tax})
	events := respond(t, game, "tester2", Challenge)

	snapshot := ViewFor(game, "tester1")
	if snapshot.Winner != "tester1" {
		t.Errorf("winner is %q, expected tester1", snapshot.Winner)
	}
	if last := events[len(events)-1].Type; last != "game_over" {
		t.Errorf("the last event is %q, expected game_over; events %v", last, typesOf(events))
	}
	if coins := entryFor(t, snapshot, "tester1").Coins; coins != 1 {
		t.Errorf("tester1 holds %d coins, expected 1 — the game ended before the tax resolved", coins)
	}
}

func containsType(events []Event, kind string) bool {
	for _, seen := range events {
		if seen.Type == kind {
			return true
		}
	}
	return false
}
