package engine

import (
	"math/rand/v2"
	"testing"
)

func independentAssassinationTable(t *testing.T, deck ...Character) *Game {
	t.Helper()
	game := threePlayerGame(deck...)
	game.options = Options{IndependentReactions: true}
	giveCoins(game, "tester1", 5)
	apply(t, game, Act{By: "tester1", Action: Assassinate, Target: "tester2"})
	return game
}

func TestBranchC1ReopensTheBlockInBothModes(t *testing.T) {
	for _, independent := range []bool{false, true} {
		game := threePlayerGame(Assassin, Duke, Contessa, Captain, Ambassador, Captain)
		game.options = Options{IndependentReactions: independent}
		giveCoins(game, "tester1", 5)
		apply(t, game, Act{By: "tester1", Action: Assassinate, Target: "tester2"})
		respond(t, game, "tester3", Challenge)
		apply(t, game, LoseInfluence{By: "tester3", Card: Ambassador})

		window := ViewFor(game, "tester2").Window
		if window == nil || len(window.YourOptions) != 2 || window.YourOptions[0].Answer != Block {
			t.Errorf("independent_reactions=%v: tester2 got window %+v, expected block and pass", independent, window)
		}
	}
}

func TestIndependentReactionsLetTheTargetChallengeLoseAndStillBlock(t *testing.T) {
	game := independentAssassinationTable(t, Assassin, Duke, Contessa, Captain, Ambassador, Captain)
	respond(t, game, "tester2", Challenge)
	apply(t, game, LoseInfluence{By: "tester2", Card: Captain})

	window := ViewFor(game, "tester2").Window
	if window == nil {
		t.Fatal("with independent reactions tester2 was not offered the block after losing the challenge")
	}
	blockWith(t, game, "tester2", Contessa)
	respond(t, game, "tester1", Pass)
	respond(t, game, "tester3", Pass)

	expectStanding(t, game, standing{"tester1", 2, 2}, standing{"tester2", 2, 1})
}

func TestIndependentReactionsGiveAPassedTargetASecondChance(t *testing.T) {
	game := independentAssassinationTable(t, Assassin, Duke, Contessa, Captain, Ambassador, Captain)
	respond(t, game, "tester2", Pass)
	respond(t, game, "tester3", Challenge)
	apply(t, game, LoseInfluence{By: "tester3", Card: Ambassador})

	if window := ViewFor(game, "tester2").Window; window == nil {
		t.Error("with independent reactions the target who passed was not asked again after the challenge")
	}
}

func TestTheSnapshotCarriesTheOptions(t *testing.T) {
	game, _ := NewGame([]string{"tester1", "tester2"}, rand.New(rand.NewPCG(1, 2)),
		Setup{Options: Options{IndependentReactions: true}})
	if !ViewFor(game, "tester1").Options.IndependentReactions {
		t.Error("the snapshot does not say independent_reactions is on")
	}
}

func TestTheStarterOpensTheGame(t *testing.T) {
	for _, starter := range []string{"tester1", "tester2", "tester3"} {
		game, _ := NewGame([]string{"tester1", "tester2", "tester3"}, rand.New(rand.NewPCG(1, 2)), Setup{Starter: starter})
		if turn := ViewFor(game, starter).TurnOf; turn != starter {
			t.Errorf("Starter %s: the game opened on %s", starter, turn)
		}
	}
	game, _ := NewGame([]string{"tester1", "tester2"}, rand.New(rand.NewPCG(1, 2)), Setup{Starter: "gone"})
	if turn := ViewFor(game, "tester1").TurnOf; turn != "tester1" && turn != "tester2" {
		t.Errorf("an unknown starter opened the game on %q", turn)
	}
}
