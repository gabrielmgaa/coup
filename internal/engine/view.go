package engine

type View struct {
	Phase         string            `json:"phase"`
	You           string            `json:"you"`
	TurnOf        string            `json:"turn_of"`
	Losing        string            `json:"losing,omitempty"`
	Winner        string            `json:"winner,omitempty"`
	DeckRemaining int               `json:"deck_remaining"`
	Players       []PlayerView      `json:"players"`
	YourActions   []AvailableAction `json:"your_actions"`
}

type PlayerView struct {
	Name       string      `json:"name"`
	Coins      int         `json:"coins"`
	Hidden     int         `json:"hidden"`
	Revealed   []Character `json:"revealed"`
	Eliminated bool        `json:"eliminated"`
	MyCards    []Character `json:"my_cards,omitempty"`
}

type AvailableAction struct {
	Name    string   `json:"name"`
	Cost    int      `json:"cost,omitempty"`
	Targets []string `json:"targets,omitempty"`
}

func ViewFor(g *Game, name string) View {
	snapshot := View{
		Phase:         g.phase.String(),
		You:           name,
		Winner:        g.winner,
		DeckRemaining: len(g.deck),
		YourActions:   []AvailableAction{},
	}
	if g.phase != Finished {
		snapshot.TurnOf = g.players[g.turn].name
	}
	if g.phase == AwaitingInfluenceLoss {
		snapshot.Losing = g.players[g.losing].name
	}
	snapshot.Players = g.playersAsSeenBy(name)
	if g.phase == AwaitingAction && g.players[g.turn].name == name {
		snapshot.YourActions = g.actionsFor(g.turn)
	}
	return snapshot
}

func (g *Game) playersAsSeenBy(name string) []PlayerView {
	seen := make([]PlayerView, 0, len(g.players))
	for i := range g.players {
		who := &g.players[i]
		visible := PlayerView{
			Name:       who.name,
			Coins:      who.coins,
			Hidden:     len(who.hand),
			Revealed:   append([]Character{}, who.revealed...),
			Eliminated: !who.alive(),
		}
		if who.name == name {
			visible.MyCards = append([]Character{}, who.hand...)
		}
		seen = append(seen, visible)
	}
	return seen
}

func (g *Game) actionsFor(by int) []AvailableAction {
	mustCoup := g.players[by].coins >= coinsForcingCoup
	available := []AvailableAction{}
	for _, action := range actionOrderForDeterministicView {
		rule := rules[action]
		if mustCoup && action != Coup {
			continue
		}
		if g.players[by].coins < rule.Cost {
			continue
		}
		offered := AvailableAction{Name: rule.Name, Cost: rule.Cost}
		if rule.NeedsTarget {
			if offered.Targets = g.validTargetNames(by); len(offered.Targets) == 0 {
				continue
			}
		}
		available = append(available, offered)
	}
	return available
}
