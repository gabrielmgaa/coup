package engine

import (
	"fmt"
	"slices"
)

func (g *Game) returnCards(chosen ReturnCards) ([]Event, error) {
	if g.phase != AwaitingExchange {
		return nil, &Refusal{Code: "illegal_action", Message: "ninguém está trocando cartas agora",
			Received: "return_cards", Expected: g.phase.String()}
	}
	actor := &g.players[g.pending.by]
	if actor.name != chosen.By {
		return nil, &Refusal{Code: "not_your_turn", Message: "quem devolve é quem trocou",
			Received: chosen.By, Expected: actor.name}
	}
	kept, held := handWithout(actor.hand, chosen.Cards[:])
	if !held {
		return nil, &Refusal{Code: "illegal_action", Message: "só dá pra devolver cartas que estão na mão",
			Received: namesOf(chosen.Cards[:]), Expected: namesOf(actor.hand)}
	}
	actor.hand = kept
	g.deck = append(g.deck, chosen.Cards[:]...)
	shuffle(g.deck, g.rng)
	exchanged := g.narrate("cards_returned", fmt.Sprintf("%s devolveu 2 cartas ao baralho.", actor.name))
	return append([]Event{exchanged}, g.proceed(endTurn)...), nil
}

func handWithout(hand, returned []Character) ([]Character, bool) {
	kept := slices.Clone(hand)
	for _, card := range returned {
		position := slices.Index(kept, card)
		if position < 0 {
			return nil, false
		}
		kept = slices.Delete(kept, position, position+1)
	}
	return kept, true
}

func returnablePairs(hand []Character) [][]Character {
	pairs := [][]Character{}
	for first := range hand {
		for second := first + 1; second < len(hand); second++ {
			pair := []Character{min(hand[first], hand[second]), max(hand[first], hand[second])}
			if !slices.ContainsFunc(pairs, func(seen []Character) bool { return slices.Equal(seen, pair) }) {
				pairs = append(pairs, pair)
			}
		}
	}
	return pairs
}
