package engine

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"testing"
)

const movesPerRandomGame = 400

type referee struct {
	t    *testing.T
	game *Game
	rng  *rand.Rand
}

func newReferee(t *testing.T, seed uint64, players int) *referee {
	names := []string{"tester1", "tester2", "tester3", "tester4", "tester5", "tester6"}[:players]
	game, err := NewGame(names, rand.New(rand.NewPCG(seed, seed+1)), Setup{Options: Options{IndependentReactions: seed%2 == 0}})
	if err != nil {
		t.Fatalf("seed %d: the game was not dealt: %v", seed, err)
	}
	return &referee{t: t, game: game, rng: rand.New(rand.NewPCG(seed*7, seed*13))}
}

func (r *referee) play(steps int) {
	r.t.Helper()
	for step := 0; step < steps && r.game.phase != Finished; step++ {
		if r.rng.IntN(3) == 0 {
			r.attempt(r.cheat())
			continue
		}
		r.attempt(r.legalMove())
	}
}

func (r *referee) attempt(move Move) {
	r.t.Helper()
	before := r.fingerprint()
	_, err := r.game.Apply(move)
	if err != nil {
		var refusal *Refusal
		if !errors.As(err, &refusal) {
			r.t.Fatalf("move %#v failed with a non-refusal error: %v", move, err)
		}
		if after := r.fingerprint(); after != before {
			r.t.Fatalf("the refused move %#v (%s) changed the table:\nbefore %s\nafter  %s", move, refusal.Code, before, after)
		}
		return
	}
	r.checkInvariants(move)
}

func (r *referee) legalMove() Move {
	awaited := r.game.Awaiting()
	name := awaited[r.rng.IntN(len(awaited))]
	view := ViewFor(r.game, name)
	switch {
	case view.Losing == name:
		cards := view.Players[r.game.indexOf(name)].MyCards
		return LoseInfluence{By: name, Card: cards[r.rng.IntN(len(cards))]}
	case len(view.YourReturns) > 0:
		pair := view.YourReturns[r.rng.IntN(len(view.YourReturns))]
		return ReturnCards{By: name, Cards: [2]Character{pair[0], pair[1]}}
	case view.Window != nil:
		option := view.Window.YourOptions[r.rng.IntN(len(view.Window.YourOptions))]
		return Respond{By: name, Window: view.Window.ID, Answer: option.Answer, Character: option.Character}
	}
	offered := view.YourActions[r.rng.IntN(len(view.YourActions))]
	action, _ := ActionByName(offered.Name)
	target := ""
	if len(offered.Targets) > 0 {
		target = offered.Targets[r.rng.IntN(len(offered.Targets))]
	}
	return Act{By: name, Action: action, Target: target}
}

func (r *referee) cheat() Move {
	names := []string{"tester1", "tester2", "tester3", "tester4", "tester5", "tester6", "stranger", ""}
	by := names[r.rng.IntN(len(names))]
	card := Character(r.rng.IntN(int(Contessa) + 2))
	switch r.rng.IntN(4) {
	case 0:
		return Act{By: by, Action: ActionType(r.rng.IntN(int(Exchange) + 2)), Target: names[r.rng.IntN(len(names))]}
	case 1:
		return Respond{By: by, Window: r.game.decision + r.rng.IntN(3) - 1, Answer: Answer(r.rng.IntN(int(Pass) + 2)), Character: card}
	case 2:
		return LoseInfluence{By: by, Card: card}
	}
	return ReturnCards{By: by, Cards: [2]Character{card, Character(r.rng.IntN(int(Contessa) + 2))}}
}

func (r *referee) fingerprint() string {
	views := make([]View, 0, len(r.game.players))
	for _, seated := range r.game.players {
		views = append(views, ViewFor(r.game, seated.name))
	}
	encoded, _ := json.Marshal(views)
	return fmt.Sprintf("%s deck=%v decision=%d", encoded, r.game.deck, r.game.decision)
}

func (r *referee) checkInvariants(after Move) {
	r.t.Helper()
	counted := map[Character]int{}
	for _, card := range r.game.deck {
		counted[card]++
	}
	for _, seated := range r.game.players {
		if seated.coins < 0 {
			r.t.Fatalf("after %#v %s holds %d coins", after, seated.name, seated.coins)
		}
		if !seated.alive() && seated.coins != 0 {
			r.t.Fatalf("after %#v the eliminated %s still holds %d coins", after, seated.name, seated.coins)
		}
		for _, card := range append(append([]Character{}, seated.hand...), seated.revealed...) {
			counted[card]++
		}
	}
	for _, character := range allCharacters {
		if counted[character] != copiesPerCharacter {
			r.t.Fatalf("after %#v there are %d copies of %s on the table, expected %d", after, counted[character], character, copiesPerCharacter)
		}
	}
	r.checkHiddenHands(after)
}

func (r *referee) checkHiddenHands(after Move) {
	r.t.Helper()
	for _, seated := range r.game.players {
		for _, seen := range ViewFor(r.game, seated.name).Players {
			if seen.Name != seated.name && seen.MyCards != nil {
				r.t.Fatalf("after %#v %s can see %s's hand %v", after, seated.name, seen.Name, seen.MyCards)
			}
		}
	}
}

func TestRandomGamesNeverBreakTheTable(t *testing.T) {
	for seed := uint64(1); seed <= 300; seed++ {
		referee := newReferee(t, seed, 2+int(seed%5))
		referee.play(movesPerRandomGame)
	}
}

func TestEveryRandomGameWithOnlyLegalMovesEnds(t *testing.T) {
	for seed := uint64(1); seed <= 100; seed++ {
		referee := newReferee(t, seed, 2+int(seed%5))
		for step := 0; referee.game.phase != Finished; step++ {
			if step > 5000 {
				t.Fatalf("seed %d: the game did not end within 5000 legal moves", seed)
			}
			referee.attempt(referee.legalMove())
		}
		if referee.game.Winner() == "" {
			t.Fatalf("seed %d: the game finished with no winner", seed)
		}
	}
}

func FuzzAnyMoveSequenceKeepsTheTableWhole(f *testing.F) {
	f.Add(uint64(1), uint8(3), []byte{0, 1, 2, 3, 4, 5, 6, 7})
	f.Add(uint64(42), uint8(6), []byte{9, 9, 9, 0, 0, 0})
	f.Add(uint64(7), uint8(2), []byte{255, 128, 3, 1})
	f.Fuzz(func(t *testing.T, seed uint64, players uint8, choices []byte) {
		referee := newReferee(t, seed, 2+int(players%5))
		for _, choice := range choices {
			if referee.game.phase == Finished {
				return
			}
			referee.rng = rand.New(rand.NewPCG(uint64(choice), seed))
			if choice%3 == 0 {
				referee.attempt(referee.cheat())
				continue
			}
			referee.attempt(referee.legalMove())
		}
	})
}
