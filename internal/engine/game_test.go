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
	return newGameWithDeck([]string{"isabely", "marina"}, fixedDeck(first...))
}

func threePlayerGame(first ...Character) *Game {
	return newGameWithDeck([]string{"isabely", "marina", "pedro"}, fixedDeck(first...))
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
	snapshot := ViewFor(twoPlayerGame(), "isabely")
	for _, name := range []string{"isabely", "marina"} {
		if coins := entryFor(t, snapshot, name).Coins; coins != 1 {
			t.Errorf("%s started with %d coins, expected 1", name, coins)
		}
	}
}

func TestThreePlayerGameStartsWithTwoCoins(t *testing.T) {
	snapshot := ViewFor(threePlayerGame(), "isabely")
	for _, name := range []string{"isabely", "marina", "pedro"} {
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
	game := NewGame([]string{"isabely", "marina"}, rand.New(rand.NewPCG(1, 2)), RulebookCoins)
	if left := ViewFor(game, "isabely").DeckRemaining; left != 11 {
		t.Errorf("after dealing to 2 players %d cards were left, expected 11", left)
	}
}

func TestIncomeGivesOneCoinAndPassesTheTurn(t *testing.T) {
	game := twoPlayerGame()
	apply(t, game, Act{By: "isabely", Action: Income})

	snapshot := ViewFor(game, "isabely")
	if coins := entryFor(t, snapshot, "isabely").Coins; coins != 2 {
		t.Errorf("isabely ended with %d coins, expected 2", coins)
	}
	if snapshot.TurnOf != "marina" {
		t.Errorf("the turn went to %q, expected marina", snapshot.TurnOf)
	}
}

func TestCoupChargesSevenOnDeclaration(t *testing.T) {
	game := twoPlayerGame(Duke, Captain, Duke, Contessa)
	giveCoins(game, "isabely", 7)
	apply(t, game, Act{By: "isabely", Action: Coup, Target: "marina"})

	snapshot := ViewFor(game, "marina")
	if snapshot.Phase != "awaiting_influence_loss" {
		t.Fatalf("phase is %q, expected awaiting_influence_loss", snapshot.Phase)
	}
	if coins := entryFor(t, snapshot, "isabely").Coins; coins != 0 {
		t.Errorf("isabely holds %d coins while marina picks a card, expected 0", coins)
	}
}

func TestCoupWithSixCoinsIsRefused(t *testing.T) {
	game := twoPlayerGame()
	giveCoins(game, "isabely", 6)

	_, err := game.Apply(Act{By: "isabely", Action: Coup, Target: "marina"})
	if code := refusalFrom(t, err).Code; code != "insufficient_coins" {
		t.Errorf("code %q, expected insufficient_coins", code)
	}
	snapshot := ViewFor(game, "isabely")
	if coins := entryFor(t, snapshot, "isabely").Coins; coins != 6 {
		t.Errorf("isabely ended with %d coins, expected 6", coins)
	}
	if snapshot.TurnOf != "isabely" {
		t.Errorf("the turn went to %q, expected isabely", snapshot.TurnOf)
	}
}

func TestPlayingOutOfTurnIsRefused(t *testing.T) {
	game := twoPlayerGame()

	_, err := game.Apply(Act{By: "marina", Action: Income})
	if code := refusalFrom(t, err).Code; code != "not_your_turn" {
		t.Errorf("code %q, expected not_your_turn", code)
	}
	snapshot := ViewFor(game, "marina")
	if coins := entryFor(t, snapshot, "marina").Coins; coins != 1 {
		t.Errorf("marina ended with %d coins, expected 1", coins)
	}
	if snapshot.TurnOf != "isabely" {
		t.Errorf("the turn went to %q, expected isabely", snapshot.TurnOf)
	}
}

func TestWithTenCoinsOnlyCoupIsLeft(t *testing.T) {
	game := twoPlayerGame()
	giveCoins(game, "isabely", 10)

	offered := ViewFor(game, "isabely").YourActions
	if len(offered) != 1 || offered[0].Name != "coup" {
		t.Errorf("with 10 coins the offered actions are %v, expected coup only", offered)
	}
	_, err := game.Apply(Act{By: "isabely", Action: Income})
	if code := refusalFrom(t, err).Code; code != "coup_required" {
		t.Errorf("code %q, expected coup_required", code)
	}
}

func TestCoupAgainstAnEliminatedPlayerIsRefused(t *testing.T) {
	game := threePlayerGame(Duke, Captain, Duke, Contessa, Assassin, Ambassador)
	leaveOneCard(game, "marina", Contessa, Duke)
	giveCoins(game, "isabely", 14)
	apply(t, game, Act{By: "isabely", Action: Coup, Target: "marina"})
	apply(t, game, Act{By: "pedro", Action: Income})

	_, err := game.Apply(Act{By: "isabely", Action: Coup, Target: "marina"})
	if code := refusalFrom(t, err).Code; code != "invalid_target" {
		t.Errorf("code %q, expected invalid_target", code)
	}
	if coins := entryFor(t, ViewFor(game, "isabely"), "isabely").Coins; coins != 7 {
		t.Errorf("isabely ended with %d coins, expected 7", coins)
	}
}

func TestTargetPicksWhichCardToReveal(t *testing.T) {
	game := twoPlayerGame(Duke, Captain, Duke, Contessa)
	giveCoins(game, "isabely", 7)
	apply(t, game, Act{By: "isabely", Action: Coup, Target: "marina"})
	apply(t, game, LoseInfluence{By: "marina", Card: Contessa})

	seen := entryFor(t, ViewFor(game, "marina"), "marina")
	if len(seen.Revealed) != 1 || seen.Revealed[0] != Contessa {
		t.Errorf("marina revealed %v, expected [contessa]", seen.Revealed)
	}
	if seen.Hidden != 1 {
		t.Errorf("marina kept %d hidden cards, expected 1", seen.Hidden)
	}
	if len(seen.MyCards) != 1 || seen.MyCards[0] != Duke {
		t.Errorf("marina's hand is %v, expected [duke]", seen.MyCards)
	}
}

func TestWithOneCardThereIsNothingToPick(t *testing.T) {
	game := threePlayerGame(Duke, Captain, Duke, Contessa, Assassin, Ambassador)
	leaveOneCard(game, "marina", Contessa, Duke)
	giveCoins(game, "isabely", 7)
	apply(t, game, Act{By: "isabely", Action: Coup, Target: "marina"})

	snapshot := ViewFor(game, "isabely")
	if snapshot.Phase != "awaiting_action" {
		t.Errorf("phase is %q, expected awaiting_action — nobody had a choice to make", snapshot.Phase)
	}
	if snapshot.TurnOf != "pedro" {
		t.Errorf("the turn went to %q, expected pedro — marina is out of the game", snapshot.TurnOf)
	}
}

func TestRevealingACardNotInHandIsRefused(t *testing.T) {
	game := twoPlayerGame(Duke, Captain, Duke, Contessa)
	giveCoins(game, "isabely", 7)
	apply(t, game, Act{By: "isabely", Action: Coup, Target: "marina"})

	_, err := game.Apply(LoseInfluence{By: "marina", Card: Captain})
	if code := refusalFrom(t, err).Code; code != "illegal_action" {
		t.Errorf("code %q, expected illegal_action", code)
	}
	seen := entryFor(t, ViewFor(game, "marina"), "marina")
	if seen.Hidden != 2 {
		t.Errorf("marina kept %d hidden cards after the refusal, expected 2", seen.Hidden)
	}
	if len(seen.Revealed) != 0 {
		t.Errorf("marina now shows %v revealed — a card she never held", seen.Revealed)
	}
}

func TestOnlyTheTargetPicksTheCard(t *testing.T) {
	game := twoPlayerGame(Duke, Captain, Duke, Contessa)
	giveCoins(game, "isabely", 7)
	apply(t, game, Act{By: "isabely", Action: Coup, Target: "marina"})

	_, err := game.Apply(LoseInfluence{By: "isabely", Card: Duke})
	if code := refusalFrom(t, err).Code; code != "not_your_turn" {
		t.Errorf("code %q, expected not_your_turn", code)
	}
	snapshot := ViewFor(game, "isabely")
	if revealed := entryFor(t, snapshot, "isabely").Revealed; len(revealed) != 0 {
		t.Errorf("isabely revealed %v, expected none", revealed)
	}
	if snapshot.Losing != "marina" {
		t.Errorf("the game is waiting on %q, expected marina", snapshot.Losing)
	}
}

func TestLosingTheSecondCardEliminatesAndZeroesCoins(t *testing.T) {
	game := threePlayerGame(Duke, Captain, Duke, Contessa, Assassin, Ambassador)
	leaveOneCard(game, "marina", Contessa, Duke)
	giveCoins(game, "marina", 3)
	giveCoins(game, "isabely", 7)
	apply(t, game, Act{By: "isabely", Action: Coup, Target: "marina"})

	seen := entryFor(t, ViewFor(game, "isabely"), "marina")
	if !seen.Eliminated {
		t.Error("marina lost her second card and was not marked as eliminated")
	}
	if seen.Hidden != 0 {
		t.Errorf("marina kept %d hidden cards, expected 0", seen.Hidden)
	}
	if seen.Coins != 0 {
		t.Errorf("marina left the game with %d coins, expected 0 — they go back to the Treasury", seen.Coins)
	}
}

func TestWithOnePlayerLeftThereIsAWinner(t *testing.T) {
	game := twoPlayerGame(Duke, Captain, Duke, Contessa)
	leaveOneCard(game, "marina", Contessa, Duke)
	giveCoins(game, "isabely", 7)
	apply(t, game, Act{By: "isabely", Action: Coup, Target: "marina"})

	snapshot := ViewFor(game, "isabely")
	if snapshot.Winner != "isabely" {
		t.Errorf("winner is %q, expected isabely", snapshot.Winner)
	}
	if snapshot.TurnOf != "" {
		t.Errorf("the turn went to %q after the game ended, expected empty", snapshot.TurnOf)
	}
}

func TestAfterTheGameEndsMovesAreRefused(t *testing.T) {
	game := twoPlayerGame(Duke, Captain, Duke, Contessa)
	leaveOneCard(game, "marina", Contessa, Duke)
	giveCoins(game, "isabely", 7)
	apply(t, game, Act{By: "isabely", Action: Coup, Target: "marina"})

	_, err := game.Apply(Act{By: "isabely", Action: Income})
	if code := refusalFrom(t, err).Code; code != "illegal_action" {
		t.Errorf("code %q, expected illegal_action", code)
	}
	snapshot := ViewFor(game, "isabely")
	if coins := entryFor(t, snapshot, "isabely").Coins; coins != 0 {
		t.Errorf("isabely holds %d coins after the game ended, expected 0", coins)
	}
	if snapshot.Winner != "isabely" {
		t.Errorf("winner became %q, expected isabely", snapshot.Winner)
	}
}

func TestAHandOnlyAppearsInItsOwnersSnapshot(t *testing.T) {
	game := twoPlayerGame(Duke, Captain, Contessa, Ambassador)

	mine := entryFor(t, ViewFor(game, "isabely"), "isabely")
	if len(mine.MyCards) != 2 || mine.MyCards[0] != Duke || mine.MyCards[1] != Captain {
		t.Errorf("isabely sees %v in her own hand, expected [duke captain]", mine.MyCards)
	}
	marinasSnapshot := ViewFor(game, "marina")
	byMarina := entryFor(t, marinasSnapshot, "isabely")
	if byMarina.MyCards != nil {
		t.Errorf("marina could see isabely's hand: %v", byMarina.MyCards)
	}
	if byMarina.Hidden != 2 {
		t.Errorf("marina sees %d hidden cards of isabely, expected 2", byMarina.Hidden)
	}
	encoded, err := json.Marshal(marinasSnapshot)
	if err != nil {
		t.Fatalf("the snapshot does not serialize: %v", err)
	}
	if strings.Contains(string(encoded), "captain") {
		t.Errorf("isabely's captain leaked into the json sent to marina: %s", encoded)
	}
}

func TestYourActionsOnlyReachThePlayerOnTurn(t *testing.T) {
	game := twoPlayerGame()

	if offered := ViewFor(game, "isabely").YourActions; len(offered) == 0 {
		t.Error("isabely is on turn and received no actions at all")
	}
	if offered := ViewFor(game, "marina").YourActions; len(offered) != 0 {
		t.Errorf("marina received %v outside her turn, expected none", offered)
	}
}
