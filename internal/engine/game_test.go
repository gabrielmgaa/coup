package engine

import (
	"encoding/json"
	"errors"
	"math/rand/v2"
	"strings"
	"testing"
)

func fixedDeck(first ...Character) []Character {
	remaining := map[Character]int{}
	for _, character := range allCharacters {
		remaining[character] = copiesPerCharacter
	}
	deck := make([]Character, 0, len(allCharacters)*copiesPerCharacter)
	for _, card := range first {
		if remaining[card] == 0 {
			panic("too many copies of " + card.String() + " in the test deck")
		}
		remaining[card]--
		deck = append(deck, card)
	}
	for _, character := range allCharacters {
		for range remaining[character] {
			deck = append(deck, character)
		}
	}
	return deck
}

func twoPlayerGame(first ...Character) *Game {
	return newGameWithDeck([]string{"tester1", "tester2"}, fixedDeck(first...))
}

func threePlayerGame(first ...Character) *Game {
	return newGameWithDeck([]string{"tester1", "tester2", "tester3"}, fixedDeck(first...))
}

func apply(t *testing.T, game *Game, move Move) {
	t.Helper()
	if _, err := game.Apply(move); err != nil {
		t.Fatalf("move %#v was refused: %v", move, err)
	}
}

func refusalFrom(t *testing.T, err error) *Refusal {
	t.Helper()
	var refusal *Refusal
	if !errors.As(err, &refusal) {
		t.Errorf("the engine accepted the move instead of refusing it (err = %v)", err)
		return &Refusal{}
	}
	return refusal
}

func entryFor(t *testing.T, snapshot View, name string) PlayerView {
	t.Helper()
	for _, seen := range snapshot.Players {
		if seen.Name == name {
			return seen
		}
	}
	t.Fatalf("%q does not appear in the snapshot", name)
	return PlayerView{}
}

func leaveOneCard(game *Game, name string, kept, revealed Character) {
	for i := range game.players {
		if game.players[i].name == name {
			game.players[i].hand = []Character{kept}
			game.players[i].revealed = []Character{revealed}
			return
		}
	}
	panic("no such player: " + name)
}

func typesOf(events []Event) []string {
	kinds := make([]string, 0, len(events))
	for _, seen := range events {
		kinds = append(kinds, seen.Type)
	}
	return kinds
}

func giveCoins(game *Game, name string, coins int) {
	for i := range game.players {
		if game.players[i].name == name {
			game.players[i].coins = coins
			return
		}
	}
	panic("no such player: " + name)
}

func TestTwoPlayerGameStartsWithOneCoin(t *testing.T) {
	snapshot := ViewFor(twoPlayerGame(), "tester1")
	for _, name := range []string{"tester1", "tester2"} {
		if coins := entryFor(t, snapshot, name).Coins; coins != 1 {
			t.Errorf("%s started with %d coins, expected 1", name, coins)
		}
	}
}

func TestThreePlayerGameStartsWithTwoCoins(t *testing.T) {
	snapshot := ViewFor(threePlayerGame(), "tester1")
	for _, name := range []string{"tester1", "tester2", "tester3"} {
		if coins := entryFor(t, snapshot, name).Coins; coins != 2 {
			t.Errorf("%s started with %d coins, expected 2", name, coins)
		}
	}
}

func TestDeckHasFifteenCardsAndElevenAreLeft(t *testing.T) {
	base := baseDeck()
	if len(base) != 15 {
		t.Errorf("the base deck has %d cards, expected 15", len(base))
	}
	count := map[Character]int{}
	for _, card := range base {
		count[card]++
	}
	for _, character := range allCharacters {
		if count[character] != 3 {
			t.Errorf("%s appears %d times in the base deck, expected 3", character, count[character])
		}
	}
	game, err := NewGame([]string{"tester1", "tester2"}, rand.New(rand.NewPCG(1, 2)), RulebookCoins)
	if err != nil {
		t.Fatalf("two players was refused: %v", err)
	}
	if left := ViewFor(game, "tester1").DeckRemaining; left != 11 {
		t.Errorf("after dealing to 2 players %d cards were left, expected 11", left)
	}
}

func TestIncomeGivesOneCoinAndPassesTheTurn(t *testing.T) {
	game := twoPlayerGame()
	apply(t, game, Act{By: "tester1", Action: Income})

	snapshot := ViewFor(game, "tester1")
	if coins := entryFor(t, snapshot, "tester1").Coins; coins != 2 {
		t.Errorf("tester1 ended with %d coins, expected 2", coins)
	}
	if snapshot.TurnOf != "tester2" {
		t.Errorf("the turn went to %q, expected tester2", snapshot.TurnOf)
	}
}

func TestCoupChargesSevenOnDeclaration(t *testing.T) {
	game := twoPlayerGame(Duke, Captain, Duke, Contessa)
	giveCoins(game, "tester1", 7)
	apply(t, game, Act{By: "tester1", Action: Coup, Target: "tester2"})

	snapshot := ViewFor(game, "tester2")
	if snapshot.Phase != "awaiting_influence_loss" {
		t.Fatalf("phase is %q, expected awaiting_influence_loss", snapshot.Phase)
	}
	if coins := entryFor(t, snapshot, "tester1").Coins; coins != 0 {
		t.Errorf("tester1 holds %d coins while tester2 picks a card, expected 0", coins)
	}
}

func TestCoupWithSixCoinsIsRefused(t *testing.T) {
	game := twoPlayerGame()
	giveCoins(game, "tester1", 6)

	_, err := game.Apply(Act{By: "tester1", Action: Coup, Target: "tester2"})
	if code := refusalFrom(t, err).Code; code != "insufficient_coins" {
		t.Errorf("code %q, expected insufficient_coins", code)
	}
	snapshot := ViewFor(game, "tester1")
	if coins := entryFor(t, snapshot, "tester1").Coins; coins != 6 {
		t.Errorf("tester1 ended with %d coins, expected 6", coins)
	}
	if snapshot.TurnOf != "tester1" {
		t.Errorf("the turn went to %q, expected tester1", snapshot.TurnOf)
	}
}

func TestPlayingOutOfTurnIsRefused(t *testing.T) {
	game := twoPlayerGame()

	_, err := game.Apply(Act{By: "tester2", Action: Income})
	if code := refusalFrom(t, err).Code; code != "not_your_turn" {
		t.Errorf("code %q, expected not_your_turn", code)
	}
	snapshot := ViewFor(game, "tester2")
	if coins := entryFor(t, snapshot, "tester2").Coins; coins != 1 {
		t.Errorf("tester2 ended with %d coins, expected 1", coins)
	}
	if snapshot.TurnOf != "tester1" {
		t.Errorf("the turn went to %q, expected tester1", snapshot.TurnOf)
	}
}

func TestWithTenCoinsOnlyCoupIsLeft(t *testing.T) {
	game := twoPlayerGame()
	giveCoins(game, "tester1", 10)

	offered := ViewFor(game, "tester1").YourActions
	if len(offered) != 1 || offered[0].Name != "coup" {
		t.Errorf("with 10 coins the offered actions are %v, expected coup only", offered)
	}
	_, err := game.Apply(Act{By: "tester1", Action: Income})
	if code := refusalFrom(t, err).Code; code != "coup_required" {
		t.Errorf("code %q, expected coup_required", code)
	}
}

func TestCoupAgainstAnEliminatedPlayerIsRefused(t *testing.T) {
	game := threePlayerGame(Duke, Captain, Duke, Contessa, Assassin, Ambassador)
	leaveOneCard(game, "tester2", Contessa, Duke)
	giveCoins(game, "tester1", 14)
	apply(t, game, Act{By: "tester1", Action: Coup, Target: "tester2"})
	apply(t, game, Act{By: "tester3", Action: Income})

	_, err := game.Apply(Act{By: "tester1", Action: Coup, Target: "tester2"})
	if code := refusalFrom(t, err).Code; code != "invalid_target" {
		t.Errorf("code %q, expected invalid_target", code)
	}
	if coins := entryFor(t, ViewFor(game, "tester1"), "tester1").Coins; coins != 7 {
		t.Errorf("tester1 ended with %d coins, expected 7", coins)
	}
}

func TestTargetPicksWhichCardToReveal(t *testing.T) {
	game := twoPlayerGame(Duke, Captain, Duke, Contessa)
	giveCoins(game, "tester1", 7)
	apply(t, game, Act{By: "tester1", Action: Coup, Target: "tester2"})
	apply(t, game, LoseInfluence{By: "tester2", Card: Contessa})

	seen := entryFor(t, ViewFor(game, "tester2"), "tester2")
	if len(seen.Revealed) != 1 || seen.Revealed[0] != Contessa {
		t.Errorf("tester2 revealed %v, expected [contessa]", seen.Revealed)
	}
	if seen.Hidden != 1 {
		t.Errorf("tester2 kept %d hidden cards, expected 1", seen.Hidden)
	}
	if len(seen.MyCards) != 1 || seen.MyCards[0] != Duke {
		t.Errorf("tester2's hand is %v, expected [duke]", seen.MyCards)
	}
}

func TestWithOneCardThereIsNothingToPick(t *testing.T) {
	game := threePlayerGame(Duke, Captain, Duke, Contessa, Assassin, Ambassador)
	leaveOneCard(game, "tester2", Contessa, Duke)
	giveCoins(game, "tester1", 7)
	apply(t, game, Act{By: "tester1", Action: Coup, Target: "tester2"})

	snapshot := ViewFor(game, "tester1")
	if snapshot.Phase != "awaiting_action" {
		t.Errorf("phase is %q, expected awaiting_action — nobody had a choice to make", snapshot.Phase)
	}
	if snapshot.TurnOf != "tester3" {
		t.Errorf("the turn went to %q, expected tester3 — tester2 is out of the game", snapshot.TurnOf)
	}
}

func TestRevealingACardNotInHandIsRefused(t *testing.T) {
	game := twoPlayerGame(Duke, Captain, Duke, Contessa)
	giveCoins(game, "tester1", 7)
	apply(t, game, Act{By: "tester1", Action: Coup, Target: "tester2"})

	_, err := game.Apply(LoseInfluence{By: "tester2", Card: Captain})
	if code := refusalFrom(t, err).Code; code != "illegal_action" {
		t.Errorf("code %q, expected illegal_action", code)
	}
	seen := entryFor(t, ViewFor(game, "tester2"), "tester2")
	if seen.Hidden != 2 {
		t.Errorf("tester2 kept %d hidden cards after the refusal, expected 2", seen.Hidden)
	}
	if len(seen.Revealed) != 0 {
		t.Errorf("tester2 now shows %v revealed — a card she never held", seen.Revealed)
	}
}

func TestOnlyTheTargetPicksTheCard(t *testing.T) {
	game := twoPlayerGame(Duke, Captain, Duke, Contessa)
	giveCoins(game, "tester1", 7)
	apply(t, game, Act{By: "tester1", Action: Coup, Target: "tester2"})

	_, err := game.Apply(LoseInfluence{By: "tester1", Card: Duke})
	if code := refusalFrom(t, err).Code; code != "not_your_turn" {
		t.Errorf("code %q, expected not_your_turn", code)
	}
	snapshot := ViewFor(game, "tester1")
	if revealed := entryFor(t, snapshot, "tester1").Revealed; len(revealed) != 0 {
		t.Errorf("tester1 revealed %v, expected none", revealed)
	}
	if snapshot.Losing != "tester2" {
		t.Errorf("the game is waiting on %q, expected tester2", snapshot.Losing)
	}
}

func TestLosingTheSecondCardEliminatesAndZeroesCoins(t *testing.T) {
	game := threePlayerGame(Duke, Captain, Duke, Contessa, Assassin, Ambassador)
	leaveOneCard(game, "tester2", Contessa, Duke)
	giveCoins(game, "tester2", 3)
	giveCoins(game, "tester1", 7)
	apply(t, game, Act{By: "tester1", Action: Coup, Target: "tester2"})

	seen := entryFor(t, ViewFor(game, "tester1"), "tester2")
	if !seen.Eliminated {
		t.Error("tester2 lost her second card and was not marked as eliminated")
	}
	if seen.Hidden != 0 {
		t.Errorf("tester2 kept %d hidden cards, expected 0", seen.Hidden)
	}
	if seen.Coins != 0 {
		t.Errorf("tester2 left the game with %d coins, expected 0 — they go back to the Treasury", seen.Coins)
	}
}

func TestWithOnePlayerLeftThereIsAWinner(t *testing.T) {
	game := twoPlayerGame(Duke, Captain, Duke, Contessa)
	leaveOneCard(game, "tester2", Contessa, Duke)
	giveCoins(game, "tester1", 7)
	apply(t, game, Act{By: "tester1", Action: Coup, Target: "tester2"})

	snapshot := ViewFor(game, "tester1")
	if snapshot.Winner != "tester1" {
		t.Errorf("winner is %q, expected tester1", snapshot.Winner)
	}
	if snapshot.TurnOf != "" {
		t.Errorf("the turn went to %q after the game ended, expected empty", snapshot.TurnOf)
	}
}

func TestAfterTheGameEndsMovesAreRefused(t *testing.T) {
	game := twoPlayerGame(Duke, Captain, Duke, Contessa)
	leaveOneCard(game, "tester2", Contessa, Duke)
	giveCoins(game, "tester1", 7)
	apply(t, game, Act{By: "tester1", Action: Coup, Target: "tester2"})

	_, err := game.Apply(Act{By: "tester1", Action: Income})
	if code := refusalFrom(t, err).Code; code != "illegal_action" {
		t.Errorf("code %q, expected illegal_action", code)
	}
	snapshot := ViewFor(game, "tester1")
	if coins := entryFor(t, snapshot, "tester1").Coins; coins != 0 {
		t.Errorf("tester1 holds %d coins after the game ended, expected 0", coins)
	}
	if snapshot.Winner != "tester1" {
		t.Errorf("winner became %q, expected tester1", snapshot.Winner)
	}
}

func TestAHandOnlyAppearsInItsOwnersSnapshot(t *testing.T) {
	game := twoPlayerGame(Duke, Captain, Contessa, Ambassador)

	mine := entryFor(t, ViewFor(game, "tester1"), "tester1")
	if len(mine.MyCards) != 2 || mine.MyCards[0] != Duke || mine.MyCards[1] != Captain {
		t.Errorf("tester1 sees %v in her own hand, expected [duke captain]", mine.MyCards)
	}
	snapshotForTester2 := ViewFor(game, "tester2")
	tester1AsSeenByTester2 := entryFor(t, snapshotForTester2, "tester1")
	if tester1AsSeenByTester2.MyCards != nil {
		t.Errorf("tester2 could see tester1's hand: %v", tester1AsSeenByTester2.MyCards)
	}
	if tester1AsSeenByTester2.Hidden != 2 {
		t.Errorf("tester2 sees %d hidden cards of tester1, expected 2", tester1AsSeenByTester2.Hidden)
	}
	encoded, err := json.Marshal(snapshotForTester2)
	if err != nil {
		t.Fatalf("the snapshot does not serialize: %v", err)
	}
	if strings.Contains(string(encoded), "captain") {
		t.Errorf("tester1's captain leaked into the json sent to tester2: %s", encoded)
	}
}

func TestYourActionsOnlyReachThePlayerOnTurn(t *testing.T) {
	game := twoPlayerGame()

	if offered := ViewFor(game, "tester1").YourActions; len(offered) == 0 {
		t.Error("tester1 is on turn and received no actions at all")
	}
	if offered := ViewFor(game, "tester2").YourActions; len(offered) != 0 {
		t.Errorf("tester2 received %v outside her turn, expected none", offered)
	}
}

func TestCoupOnTheLastCardNarratesFourEventsInOrder(t *testing.T) {
	game := twoPlayerGame(Duke, Captain, Duke, Contessa)
	leaveOneCard(game, "tester2", Contessa, Duke)
	giveCoins(game, "tester1", 7)

	events, err := game.Apply(Act{By: "tester1", Action: Coup, Target: "tester2"})
	if err != nil {
		t.Fatalf("the coup was refused: %v", err)
	}
	expected := []string{"action_declared", "influence_lost", "player_eliminated", "game_over"}
	if len(events) != len(expected) {
		t.Fatalf("the move returned %d events %v, expected %d %v",
			len(events), typesOf(events), len(expected), expected)
	}
	for position, want := range expected {
		if events[position].Type != want {
			t.Errorf("event %d is %q, expected %q", position+1, events[position].Type, want)
		}
		if events[position].N != position+1 {
			t.Errorf("event %q carries n = %d, expected %d — the log would show a gap",
				events[position].Type, events[position].N, position+1)
		}
	}
}

func TestSevenPlayersIsRefusedInsteadOfDealingFromAnEmptyDeck(t *testing.T) {
	names := []string{"tester1", "tester2", "tester3", "tester4", "tester5", "tester6", "tester7"}

	game, err := NewGame(names, rand.New(rand.NewPCG(1, 2)), RulebookCoins)
	if game != nil {
		t.Error("a game was dealt to 7 players; the base deck only holds 15 cards")
	}
	refusal := refusalFrom(t, err)
	if refusal.Code != "too_many_players" {
		t.Errorf("code %q, expected too_many_players", refusal.Code)
	}
	if refusal.Received != len(names) || refusal.Expected != MaxPlayers {
		t.Errorf("refusal says received %v expected %v, wanted %d and %d",
			refusal.Received, refusal.Expected, len(names), MaxPlayers)
	}
}

func TestSixPlayersLeavesThreeCardsInTheDeck(t *testing.T) {
	names := []string{"tester1", "tester2", "tester3", "tester4", "tester5", "tester6"}

	game, err := NewGame(names, rand.New(rand.NewPCG(1, 2)), RulebookCoins)
	if err != nil {
		t.Fatalf("six players was refused: %v", err)
	}
	if left := ViewFor(game, "tester1").DeckRemaining; left != 3 {
		t.Errorf("after dealing to 6 players %d cards were left, expected 3", left)
	}
}
