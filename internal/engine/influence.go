package engine

import "fmt"

func (g *Game) loseInfluence(chosen LoseInfluence) ([]Event, error) {
	if g.phase != AwaitingInfluenceLoss {
		return nil, &Refusal{Code: "illegal_action", Message: "ninguém precisa revelar carta agora",
			Received: "lose_influence", Expected: g.phase.String()}
	}
	target := g.players[g.losing]
	if target.name != chosen.By {
		return nil, &Refusal{Code: "not_your_turn", Message: "a escolha é de quem perdeu a influência",
			Received: chosen.By, Expected: target.name}
	}
	if !handHas(target.hand, chosen.Card) {
		return nil, &Refusal{Code: "illegal_action", Message: "carta que não está na mão",
			Received: chosen.Card.String(), Expected: namesOf(target.hand)}
	}
	events := g.reveal(g.losing, chosen.Card)
	g.losing = nobody
	return append(events, g.proceed(g.afterLoss)...), nil
}

func (g *Game) loseInfluenceThen(target int, then followUp) []Event {
	hand := g.players[target].hand
	switch len(hand) {
	case 0:
		return g.proceed(then)
	case 1:
		return append(g.reveal(target, hand[0]), g.proceed(then)...)
	}
	g.phase = AwaitingInfluenceLoss
	g.losing = target
	g.afterLoss = then
	g.decision++
	return nil
}

func (g *Game) reveal(index int, card Character) []Event {
	who := &g.players[index]
	who.hand = withoutOne(who.hand, card)
	who.revealed = append(who.revealed, card)
	events := []Event{g.narrate("influence_lost",
		fmt.Sprintf("%s revelou %s e perdeu uma influência.", who.name, card.LabelPtBR()))}
	if who.alive() {
		return events
	}
	returned := who.coins
	who.coins = 0
	return append(events, g.narrate("player_eliminated",
		fmt.Sprintf("%s está fora do jogo e devolveu %s ao Tesouro.", who.name, coinsPtBR(returned))))
}

func (g *Game) checkGameOver() ([]Event, bool) {
	survivor := nobody
	for i := range g.players {
		if !g.players[i].alive() {
			continue
		}
		if survivor != nobody {
			return nil, false
		}
		survivor = i
	}
	g.phase = Finished
	g.pending = nil
	g.window = nil
	g.winner = g.players[survivor].name
	return []Event{g.narrate("game_over", fmt.Sprintf("%s venceu a partida.", g.winner))}, true
}

func handHas(hand []Character, card Character) bool {
	for _, inHand := range hand {
		if inHand == card {
			return true
		}
	}
	return false
}
