package engine

import "testing"

func blockWith(t *testing.T, game *Game, by string, character Character) {
	t.Helper()
	_, err := game.Apply(Respond{By: by, Window: openWindowID(t, game), Answer: Block, Character: character})
	if err != nil {
		t.Fatalf("%s blocking with %s was refused: %v", by, character, err)
	}
}

func TestForeignAidUnansweredPaysTwo(t *testing.T) {
	game := threePlayerGame(Captain, Contessa, Duke, Ambassador, Assassin, Captain)
	apply(t, game, Act{By: "tester1", Action: ForeignAid})
	respond(t, game, "tester2", Pass)
	respond(t, game, "tester3", Pass)

	if coins := game.players[0].coins; coins != 4 {
		t.Errorf("tester1 ended with %d coins, expected 4", coins)
	}
}

func TestForeignAidCannotBeChallengedButAnyoneMayBlockWithTheDuke(t *testing.T) {
	game := threePlayerGame(Captain, Contessa, Duke, Ambassador, Assassin, Captain)
	apply(t, game, Act{By: "tester1", Action: ForeignAid})

	for _, name := range []string{"tester2", "tester3"} {
		offered := ViewFor(game, name).Window.YourOptions
		if len(offered) != 2 || offered[0] != (Option{Answer: Block, Character: Duke}) || offered[1].Answer != Pass {
			t.Errorf("%s is offered %v, expected block with duke and pass", name, offered)
		}
	}
	_, err := game.Apply(Respond{By: "tester2", Window: openWindowID(t, game), Answer: Challenge})
	if code := refusalFrom(t, err).Code; code != "illegal_action" {
		t.Errorf("challenging foreign aid answered %q, expected illegal_action — nobody claimed a character", code)
	}
}

func TestForeignAidBlockedAndUnchallengedPaysNothing(t *testing.T) {
	game := threePlayerGame(Captain, Contessa, Ambassador, Assassin, Duke, Captain)
	apply(t, game, Act{By: "tester1", Action: ForeignAid})
	blockWith(t, game, "tester3", Duke)

	window := ViewFor(game, "tester1").Window
	if window.Block == nil || window.Block.By != "tester3" || window.Block.Character != Duke {
		t.Fatalf("the window shows block %v, expected tester3 claiming duke", window.Block)
	}
	if waiting := window.WaitingOn; len(waiting) != 2 || waiting[0] != "tester1" || waiting[1] != "tester2" {
		t.Errorf("the block window waits on %v, expected tester1 and tester2", waiting)
	}
	respond(t, game, "tester1", Pass)
	respond(t, game, "tester2", Pass)

	snapshot := ViewFor(game, "tester1")
	if coins := entryFor(t, snapshot, "tester1").Coins; coins != 2 {
		t.Errorf("tester1 ended with %d coins, expected 2 — the block held", coins)
	}
	if snapshot.TurnOf != "tester2" {
		t.Errorf("the turn went to %q, expected tester2", snapshot.TurnOf)
	}
}

func TestABlockProvenCostsTheChallengerAndTheActionStillFails(t *testing.T) {
	game := threePlayerGame(Captain, Contessa, Ambassador, Assassin, Duke, Captain)
	apply(t, game, Act{By: "tester1", Action: ForeignAid})
	blockWith(t, game, "tester3", Duke)
	events := respond(t, game, "tester1", Challenge)

	if !containsType(events, "card_swapped") {
		t.Errorf("tester3's proven duke was not swapped; events %v", typesOf(events))
	}
	apply(t, game, LoseInfluence{By: "tester1", Card: Contessa})
	snapshot := ViewFor(game, "tester1")
	actor, blocker := entryFor(t, snapshot, "tester1"), entryFor(t, snapshot, "tester3")
	if actor.Coins != 2 || actor.Hidden != 1 {
		t.Errorf("tester1 has %d coins and %d cards, expected 2 and 1", actor.Coins, actor.Hidden)
	}
	if blocker.Hidden != 2 {
		t.Errorf("tester3 has %d cards, expected 2", blocker.Hidden)
	}
	if snapshot.TurnOf != "tester2" {
		t.Errorf("the turn went to %q, expected tester2", snapshot.TurnOf)
	}
}

func TestABluffedBlockCostsTheBlockerAndTheActionGoesThrough(t *testing.T) {
	game := threePlayerGame(Captain, Contessa, Ambassador, Assassin, Contessa, Captain)
	apply(t, game, Act{By: "tester1", Action: ForeignAid})
	blockWith(t, game, "tester3", Duke)
	respond(t, game, "tester2", Challenge)
	apply(t, game, LoseInfluence{By: "tester3", Card: Captain})

	snapshot := ViewFor(game, "tester1")
	if coins := entryFor(t, snapshot, "tester1").Coins; coins != 4 {
		t.Errorf("tester1 ended with %d coins, expected 4 — the block was a bluff", coins)
	}
	if hidden := entryFor(t, snapshot, "tester3").Hidden; hidden != 1 {
		t.Errorf("tester3 kept %d cards, expected 1", hidden)
	}
	if hidden := entryFor(t, snapshot, "tester2").Hidden; hidden != 2 {
		t.Errorf("tester2 kept %d cards, expected 2 — she was right", hidden)
	}
}

func TestTheBlockerCannotChallengeHerOwnBlock(t *testing.T) {
	game := threePlayerGame(Captain, Contessa, Ambassador, Assassin, Duke, Captain)
	apply(t, game, Act{By: "tester1", Action: ForeignAid})
	blockWith(t, game, "tester3", Duke)

	_, err := game.Apply(Respond{By: "tester3", Window: openWindowID(t, game), Answer: Challenge})
	if code := refusalFrom(t, err).Code; code != "illegal_action" {
		t.Errorf("code %q, expected illegal_action", code)
	}
}

func TestBlockingWithTheWrongCharacterIsRefused(t *testing.T) {
	game := threePlayerGame(Captain, Contessa, Ambassador, Assassin, Duke, Captain)
	apply(t, game, Act{By: "tester1", Action: ForeignAid})

	_, err := game.Apply(Respond{By: "tester2", Window: openWindowID(t, game), Answer: Block, Character: Contessa})
	refusal := refusalFrom(t, err)
	if refusal.Code != "illegal_action" {
		t.Errorf("code %q, expected illegal_action", refusal.Code)
	}
	if offered, listed := refusal.Expected.([]Option); !listed || len(offered) != 2 {
		t.Errorf("the refusal lists %v as expected, wanted the two options tester2 does have", refusal.Expected)
	}
}

func TestTaxCannotBeBlocked(t *testing.T) {
	game := threePlayerGame(Duke, Contessa, Ambassador, Assassin, Duke, Captain)
	apply(t, game, Act{By: "tester1", Action: Tax})

	_, err := game.Apply(Respond{By: "tester2", Window: openWindowID(t, game), Answer: Block, Character: Duke})
	if code := refusalFrom(t, err).Code; code != "illegal_action" {
		t.Errorf("code %q, expected illegal_action", code)
	}
}
