package engine

import "fmt"

func (g *Game) challenge(challenger int) []Event {
	blocked := g.window.block
	g.window = nil
	if blocked != nil {
		return g.challengeBlock(challenger, *blocked)
	}
	claimant, claimed := g.pending.by, g.pending.rule.Claims
	events := g.narrateChallenge(challenger, claimant, claimed)
	if handHas(g.players[claimant].hand, claimed) {
		events = append(events, g.swapProvenCard(claimant, claimed)...)
		return append(events, g.loseInfluenceThen(challenger, continueAction)...)
	}
	events = append(events, g.narrateBluff(claimant, claimed))
	events = append(events, g.refundCost()...)
	return append(events, g.loseInfluenceThen(claimant, endTurn)...)
}

func (g *Game) challengeBlock(challenger int, blocked pendingBlock) []Event {
	events := g.narrateChallenge(challenger, blocked.by, blocked.character)
	if handHas(g.players[blocked.by].hand, blocked.character) {
		events = append(events, g.swapProvenCard(blocked.by, blocked.character)...)
		return append(events, g.loseInfluenceThen(challenger, endTurn)...)
	}
	events = append(events, g.narrateBluff(blocked.by, blocked.character))
	return append(events, g.loseInfluenceThen(blocked.by, resolveAction)...)
}

func (g *Game) narrateChallenge(challenger, claimant int, claimed Character) []Event {
	return []Event{g.narrate("challenged", fmt.Sprintf("%s contestou o %s de %s.",
		g.players[challenger].name, claimed.LabelPtBR(), g.players[claimant].name))}
}

func (g *Game) narrateBluff(claimant int, claimed Character) Event {
	return g.narrate("challenge_won", fmt.Sprintf("%s não tinha %s.", g.players[claimant].name, claimed.LabelPtBR()))
}

func (g *Game) swapProvenCard(claimant int, card Character) []Event {
	who := &g.players[claimant]
	who.hand = withoutOne(who.hand, card)
	g.deck = append(g.deck, card)
	shuffle(g.deck, g.rng)
	who.hand = append(who.hand, g.deck[0])
	g.deck = g.deck[1:]
	return []Event{g.narrate("card_swapped", fmt.Sprintf(
		"%s mostrou %s, devolveu a carta ao baralho e puxou outra.", who.name, card.LabelPtBR()))}
}

func (g *Game) refundCost() []Event {
	cost := g.pending.rule.Cost
	if cost == 0 {
		return nil
	}
	g.players[g.pending.by].coins += cost
	return []Event{g.narrate("coins_refunded", fmt.Sprintf("%s recebeu de volta %s.",
		g.players[g.pending.by].name, coinsPtBR(cost)))}
}

func withoutOne(hand []Character, card Character) []Character {
	kept := make([]Character, 0, len(hand))
	removed := false
	for _, inHand := range hand {
		if inHand == card && !removed {
			removed = true
			continue
		}
		kept = append(kept, inHand)
	}
	return kept
}
