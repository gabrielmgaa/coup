package engine

func (g *Game) Decision() int { return g.decision }

func (g *Game) Winner() string { return g.winner }

func (g *Game) Awaiting() []string {
	switch g.phase {
	case AwaitingAction:
		return []string{g.players[g.turn].name}
	case AwaitingResponse:
		return g.waitingOn()
	case AwaitingInfluenceLoss:
		return []string{g.players[g.losing].name}
	case AwaitingExchange:
		return []string{g.players[g.pending.by].name}
	}
	return nil
}

func (g *Game) SafeMove(name string) (Move, bool) {
	index := g.indexOf(name)
	if index == nobody || !g.awaits(index) {
		return nil, false
	}
	switch g.phase {
	case AwaitingAction:
		return g.safeAction(index), true
	case AwaitingResponse:
		return Respond{By: name, Window: g.window.id, Answer: Pass}, true
	case AwaitingInfluenceLoss:
		return LoseInfluence{By: name, Card: g.players[index].hand[0]}, true
	}
	hand := g.players[index].hand
	return ReturnCards{By: name, Cards: [2]Character{hand[len(hand)-2], hand[len(hand)-1]}}, true
}

func (g *Game) awaits(index int) bool {
	for _, name := range g.Awaiting() {
		if name == g.players[index].name {
			return true
		}
	}
	return false
}

func (g *Game) safeAction(index int) Move {
	if g.players[index].coins < coinsForcingCoup {
		return Act{By: g.players[index].name, Action: Income}
	}
	return Act{By: g.players[index].name, Action: Coup, Target: g.validTargetNames(index, rules[Coup])[0]}
}
