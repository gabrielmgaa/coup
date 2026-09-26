package engine

import "testing"

func applySafeMove(t *testing.T, game *Game, name string) {
	t.Helper()
	move, awaited := game.SafeMove(name)
	if !awaited {
		t.Fatalf("the game is not waiting on %s; it waits on %v", name, game.Awaiting())
	}
	apply(t, game, move)
}

func TestTheSafeActionIsIncome(t *testing.T) {
	game := threePlayerGame()
	applySafeMove(t, game, "tester1")
	expectStanding(t, game, standing{"tester1", 3, 2})
}

func TestWithTenCoinsTheSafeActionIsTheForcedCoup(t *testing.T) {
	game := threePlayerGame(Duke, Captain, Contessa, Ambassador, Assassin, Captain)
	giveCoins(game, "tester1", 10)
	applySafeMove(t, game, "tester1")
	expectStanding(t, game, standing{"tester1", 3, 2})
	if losing := ViewFor(game, "tester1").Losing; losing != "tester2" {
		t.Errorf("the forced coup hit %q, expected the first living opponent tester2", losing)
	}
}

func TestTheSafeAnswerIsPass(t *testing.T) {
	game := threePlayerGame(Captain, Contessa, Duke, Ambassador, Assassin, Captain)
	apply(t, game, Act{By: "tester1", Action: Tax})
	applySafeMove(t, game, "tester2")
	applySafeMove(t, game, "tester3")
	expectStanding(t, game, standing{"tester1", 5, 2}, standing{"tester2", 2, 2}, standing{"tester3", 2, 2})
}

func TestTheSafeLossIsTheFirstCard(t *testing.T) {
	game := twoPlayerGame(Duke, Captain, Contessa, Ambassador)
	giveCoins(game, "tester1", 7)
	apply(t, game, Act{By: "tester1", Action: Coup, Target: "tester2"})
	applySafeMove(t, game, "tester2")
	if kept := handOf(game, "tester2"); len(kept) != 1 || kept[0] != Ambassador {
		t.Errorf("tester2 kept %v, expected [ambassador] after revealing the first card", kept)
	}
}

func TestTheSafeExchangeGivesBackWhatWasDrawn(t *testing.T) {
	game := threePlayerGame(Ambassador, Contessa, Duke, Captain, Assassin, Captain)
	apply(t, game, Act{By: "tester1", Action: Exchange})
	respond(t, game, "tester2", Pass)
	respond(t, game, "tester3", Pass)
	applySafeMove(t, game, "tester1")
	if kept := handOf(game, "tester1"); len(kept) != 2 || kept[0] != Ambassador || kept[1] != Contessa {
		t.Errorf("tester1 kept %v, expected the original [ambassador contessa]", kept)
	}
}

func TestNoSafeMoveForSomeoneTheGameIsNotWaitingOn(t *testing.T) {
	game := threePlayerGame()
	for _, name := range []string{"tester2", "stranger"} {
		if move, awaited := game.SafeMove(name); awaited {
			t.Errorf("SafeMove(%q) returned %v, expected nothing", name, move)
		}
	}
	giveCoins(game, "tester1", 7)
	leaveOneCard(game, "tester2", Duke, Captain)
	leaveOneCard(game, "tester3", Duke, Captain)
	apply(t, game, Act{By: "tester1", Action: Coup, Target: "tester2"})
	apply(t, game, Act{By: "tester3", Action: Income})
	giveCoins(game, "tester1", 7)
	apply(t, game, Act{By: "tester1", Action: Coup, Target: "tester3"})
	if awaited := game.Awaiting(); awaited != nil {
		t.Errorf("a finished game waits on %v, expected nobody", awaited)
	}
	if game.Winner() != "tester1" {
		t.Errorf("Winner() is %q, expected tester1", game.Winner())
	}
}

func TestTheDecisionChangesWithEachThingTheGameWaitsFor(t *testing.T) {
	game := threePlayerGame(Captain, Contessa, Duke, Ambassador, Assassin, Captain)
	start := game.Decision()
	apply(t, game, Act{By: "tester1", Action: Tax})
	window := game.Decision()
	respond(t, game, "tester2", Pass)
	if game.Decision() != window {
		t.Errorf("one pass of two moved the decision from %d to %d; the window is the same", window, game.Decision())
	}
	respond(t, game, "tester3", Pass)
	if start == window || window == game.Decision() {
		t.Errorf("decisions went %d → %d → %d, expected three different values", start, window, game.Decision())
	}
}
