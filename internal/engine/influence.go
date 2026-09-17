package engine

import "fmt"

func (g *Game) loseInfluence(chosen LoseInfluence) ([]Event, error) {
	if g.phase != AwaitingInfluenceLoss {
		return nil, &Refusal{Code: "illegal_action", Message: "ninguém precisa revelar carta agora",
			Received: "lose_influence", Expected: g.phase.String()}
	}
	target := g.players[g.losing]
	if target.name != chosen.By {
		return nil, &Refusal{Code: "not_your_turn", Message: "a escolha é de quem levou o golpe",
			Received: chosen.By, Expected: target.name}
	}
	if !handHas(target.hand, chosen.Card) {
		return nil, &Refusal{Code: "illegal_action", Message: "carta que não está na mão",
			Received: chosen.Card.String(), Expected: namesOf(target.hand)}
	}

	g.phase = AwaitingAction
	g.losing = nobody
	events := g.reveal(g.indexOf(chosen.By), chosen.Card)
	return append(events, g.closeTurn()...), nil
}

func (g *Game) openInfluenceLoss(target int) []Event {
	if len(g.players[target].hand) == 1 {
		return g.reveal(target, g.players[target].hand[0])
	}
	g.phase = AwaitingInfluenceLoss
	g.losing = target
	return nil
}

func (g *Game) reveal(index int, card Character) []Event {
	who := &g.players[index]
	for position, inHand := range who.hand {
		if inHand == card {
			who.hand = append(who.hand[:position], who.hand[position+1:]...)
			break
		}
	}
	who.revealed = append(who.revealed, card)
	events := []Event{g.narrate("influence_lost",
		fmt.Sprintf("%s revelou %s e perdeu uma influência.", who.name, card.LabelPtBR()))}
	if who.alive() {
		return events
	}
	returned := who.coins
	who.coins = 0
	return append(events, g.narrate("player_eliminated",
		fmt.Sprintf("%s está fora do jogo e devolveu %d moedas ao Tesouro.", who.name, returned)))
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
	if survivor == nobody {
		return nil, false
	}
	g.phase = Finished
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
