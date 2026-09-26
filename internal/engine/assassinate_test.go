package engine

import "testing"

func assassinationTable(t *testing.T, deck ...Character) *Game {
	t.Helper()
	game := threePlayerGame(deck...)
	giveCoins(game, "tester1", 5)
	apply(t, game, Act{By: "tester1", Action: Assassinate, Target: "tester2"})
	return game
}

type standing struct {
	name   string
	coins  int
	hidden int
}

func expectStanding(t *testing.T, game *Game, expected ...standing) {
	t.Helper()
	snapshot := ViewFor(game, "tester1")
	for _, want := range expected {
		seen := entryFor(t, snapshot, want.name)
		if seen.Coins != want.coins || seen.Hidden != want.hidden {
			t.Errorf("%s has %d coins and %d cards, expected %d and %d",
				want.name, seen.Coins, seen.Hidden, want.coins, want.hidden)
		}
	}
}

func TestAssassinateChargesThreeOnDeclaration(t *testing.T) {
	game := assassinationTable(t, Assassin, Duke, Contessa, Captain, Ambassador, Captain)
	expectStanding(t, game, standing{"tester1", 2, 2})
	if phase := ViewFor(game, "tester1").Phase; phase != "awaiting_response" {
		t.Errorf("phase is %q, expected awaiting_response", phase)
	}
}

func TestOnlyTheTargetMayBlockAnAssassination(t *testing.T) {
	game := assassinationTable(t, Assassin, Duke, Contessa, Captain, Ambassador, Captain)
	target := ViewFor(game, "tester2").Window.YourOptions
	if len(target) != 3 || target[1] != (Option{Answer: Block, Character: Contessa}) {
		t.Errorf("tester2 is offered %v, expected challenge, block with contessa, pass", target)
	}
	bystander := ViewFor(game, "tester3").Window.YourOptions
	if len(bystander) != 2 || bystander[0].Answer != Challenge || bystander[1].Answer != Pass {
		t.Errorf("tester3 is offered %v, expected challenge and pass only", bystander)
	}
	_, err := game.Apply(Respond{By: "tester3", Window: openWindowID(t, game), Answer: Block, Character: Contessa})
	if code := refusalFrom(t, err).Code; code != "illegal_action" {
		t.Errorf("tester3 blocking answered %q, expected illegal_action", code)
	}
}

func TestBranchANobodyReactsAndTheTargetLosesOne(t *testing.T) {
	game := assassinationTable(t, Assassin, Duke, Contessa, Captain, Ambassador, Captain)
	respond(t, game, "tester2", Pass)
	respond(t, game, "tester3", Pass)
	apply(t, game, LoseInfluence{By: "tester2", Card: Captain})

	expectStanding(t, game, standing{"tester1", 2, 2}, standing{"tester2", 2, 1})
}

func TestBranchB1ContessaUnchallengedKeepsTheThreeCoinsSpent(t *testing.T) {
	game := assassinationTable(t, Assassin, Duke, Contessa, Captain, Ambassador, Captain)
	blockWith(t, game, "tester2", Contessa)
	respond(t, game, "tester1", Pass)
	respond(t, game, "tester3", Pass)

	expectStanding(t, game, standing{"tester1", 2, 2}, standing{"tester2", 2, 2})
	if turn := ViewFor(game, "tester1").TurnOf; turn != "tester2" {
		t.Errorf("the turn went to %q, expected tester2", turn)
	}
}

func TestBranchB2ContessaProvenCostsTheAssassinAndTheCoinsStaySpent(t *testing.T) {
	game := assassinationTable(t, Assassin, Duke, Contessa, Captain, Ambassador, Captain)
	blockWith(t, game, "tester2", Contessa)
	respond(t, game, "tester1", Challenge)
	apply(t, game, LoseInfluence{By: "tester1", Card: Duke})

	expectStanding(t, game, standing{"tester1", 2, 1}, standing{"tester2", 2, 2})
}

func TestBranchB3ContessaBluffLosesTwiceAndLeavesTheGame(t *testing.T) {
	game := assassinationTable(t, Assassin, Duke, Captain, Ambassador, Contessa, Captain)
	blockWith(t, game, "tester2", Contessa)
	respond(t, game, "tester1", Challenge)
	apply(t, game, LoseInfluence{By: "tester2", Card: Captain})

	expectStanding(t, game, standing{"tester1", 2, 2}, standing{"tester2", 0, 0})
	if !entryFor(t, ViewFor(game, "tester1"), "tester2").Eliminated {
		t.Error("tester2 bluffed contessa, lost the challenge and the assassination, and is still in the game")
	}
}

func TestBranchC1TheActionSurvivesAndTheTargetStillMayBlock(t *testing.T) {
	game := assassinationTable(t, Assassin, Duke, Contessa, Captain, Ambassador, Captain)
	respond(t, game, "tester3", Challenge)
	apply(t, game, LoseInfluence{By: "tester3", Card: Ambassador})

	window := ViewFor(game, "tester2").Window
	if window == nil {
		t.Fatal("no block-only window reopened for tester2, who never reacted")
	}
	if len(window.YourOptions) != 2 || window.YourOptions[0] != (Option{Answer: Block, Character: Contessa}) {
		t.Errorf("tester2 is offered %v, expected block with contessa and pass", window.YourOptions)
	}
	if waiting := window.WaitingOn; len(waiting) != 1 || waiting[0] != "tester2" {
		t.Errorf("the reopened window waits on %v, expected only tester2", waiting)
	}
	respond(t, game, "tester2", Pass)
	apply(t, game, LoseInfluence{By: "tester2", Card: Contessa})

	expectStanding(t, game, standing{"tester1", 2, 2}, standing{"tester2", 2, 1}, standing{"tester3", 2, 1})
}

func TestBranchC1TheReopenedBlockCanStillBeChallenged(t *testing.T) {
	game := assassinationTable(t, Assassin, Duke, Contessa, Captain, Ambassador, Captain)
	respond(t, game, "tester3", Challenge)
	apply(t, game, LoseInfluence{By: "tester3", Card: Ambassador})
	blockWith(t, game, "tester2", Contessa)
	respond(t, game, "tester1", Pass)
	respond(t, game, "tester3", Pass)

	expectStanding(t, game, standing{"tester1", 2, 2}, standing{"tester2", 2, 2}, standing{"tester3", 2, 1})
}

func TestBranchC2ABystanderCatchesTheBluffAndTheThreeCoinsComeBack(t *testing.T) {
	game := assassinationTable(t, Duke, Captain, Contessa, Captain, Ambassador, Assassin)
	respond(t, game, "tester3", Challenge)
	apply(t, game, LoseInfluence{By: "tester1", Card: Duke})

	expectStanding(t, game, standing{"tester1", 5, 1}, standing{"tester2", 2, 2}, standing{"tester3", 2, 2})
}

func TestBranchD1TheTargetChallengesLosesAndIsAssassinatedToo(t *testing.T) {
	game := assassinationTable(t, Assassin, Duke, Contessa, Captain, Ambassador, Captain)
	respond(t, game, "tester2", Challenge)
	if window := ViewFor(game, "tester2").Window; window != nil {
		t.Fatalf("a window is open during the challenge loss: %+v", window)
	}
	apply(t, game, LoseInfluence{By: "tester2", Card: Captain})

	expectStanding(t, game, standing{"tester1", 2, 2}, standing{"tester2", 0, 0})
	if window := ViewFor(game, "tester2").Window; window != nil {
		t.Errorf("a block window reopened for tester2, who already spent her reaction: %+v", window)
	}
}

func TestBranchD2TheTargetCatchesTheBluffAndTheThreeCoinsComeBack(t *testing.T) {
	game := assassinationTable(t, Duke, Captain, Contessa, Captain, Ambassador, Assassin)
	respond(t, game, "tester2", Challenge)
	apply(t, game, LoseInfluence{By: "tester1", Card: Captain})

	expectStanding(t, game, standing{"tester1", 5, 1}, standing{"tester2", 2, 2})
}

func TestAPassOnTheFirstWindowSpendsTheReaction(t *testing.T) {
	game := assassinationTable(t, Assassin, Duke, Contessa, Captain, Ambassador, Captain)
	respond(t, game, "tester2", Pass)
	respond(t, game, "tester3", Challenge)
	apply(t, game, LoseInfluence{By: "tester3", Card: Ambassador})

	if window := ViewFor(game, "tester2").Window; window != nil {
		t.Errorf("tester2 passed and was asked again: %+v", window)
	}
	if losing := ViewFor(game, "tester2").Losing; losing != "tester2" {
		t.Errorf("the game waits on %q, expected tester2 to pay for the assassination", losing)
	}
}

func TestAssassinatingWithTwoCoinsIsRefusedAndNotOffered(t *testing.T) {
	game := threePlayerGame()
	for _, offered := range ViewFor(game, "tester1").YourActions {
		if offered.Name == "assassinate" {
			t.Errorf("assassinate was offered with 2 coins: %+v", offered)
		}
	}
	_, err := game.Apply(Act{By: "tester1", Action: Assassinate, Target: "tester2"})
	refusal := refusalFrom(t, err)
	if refusal.Code != "insufficient_coins" || refusal.Received != 2 || refusal.Expected != 3 {
		t.Errorf("refusal %+v, expected insufficient_coins with 2 and 3", refusal)
	}
}

func TestAssassinationOfATargetWhoDiedInTheChallengeDoesNothingMore(t *testing.T) {
	game := threePlayerGame(Assassin, Duke, Contessa, Captain, Ambassador, Captain)
	leaveOneCard(game, "tester2", Contessa, Captain)
	giveCoins(game, "tester1", 5)
	apply(t, game, Act{By: "tester1", Action: Assassinate, Target: "tester2"})
	events := respond(t, game, "tester2", Challenge)

	if count := countType(events, "influence_lost"); count != 1 {
		t.Errorf("%d influences were lost, expected 1 — tester2 had one card; events %v", count, typesOf(events))
	}
	if turn := ViewFor(game, "tester1").TurnOf; turn != "tester3" {
		t.Errorf("the turn went to %q, expected tester3", turn)
	}
}

func countType(events []Event, kind string) int {
	count := 0
	for _, seen := range events {
		if seen.Type == kind {
			count++
		}
	}
	return count
}
