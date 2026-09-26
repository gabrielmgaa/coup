package engine

type View struct {
	Phase         string            `json:"phase"`
	You           string            `json:"you"`
	TurnOf        string            `json:"turn_of"`
	Losing        string            `json:"losing,omitempty"`
	Winner        string            `json:"winner,omitempty"`
	DeckRemaining int               `json:"deck_remaining"`
	Players       []PlayerView      `json:"players"`
	Window        *WindowView       `json:"window"`
	YourActions   []AvailableAction `json:"your_actions"`
	YourReturns   [][]Character     `json:"your_returns,omitempty"`
}

type WindowView struct {
	ID          int        `json:"id"`
	Action      ActionView `json:"action"`
	Block       *BlockView `json:"block"`
	YourOptions []Option   `json:"your_options"`
	WaitingOn   []string   `json:"waiting_on"`
}

type BlockView struct {
	By        string    `json:"by"`
	Character Character `json:"character"`
}

type ActionView struct {
	Name   string    `json:"name"`
	By     string    `json:"by"`
	Target string    `json:"target,omitempty"`
	Claims Character `json:"claims,omitempty"`
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
	snapshot.Window = g.windowAsSeenBy(name)
	if g.phase == AwaitingAction && g.players[g.turn].name == name {
		snapshot.YourActions = g.actionsFor(g.turn)
	}
	if g.phase == AwaitingExchange && g.players[g.pending.by].name == name {
		snapshot.YourReturns = returnablePairs(g.players[g.pending.by].hand)
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

func (g *Game) windowAsSeenBy(name string) *WindowView {
	if g.window == nil {
		return nil
	}
	declared := ActionView{Name: g.pending.rule.Name, By: g.players[g.pending.by].name, Claims: g.pending.rule.Claims}
	if g.pending.target != nobody {
		declared.Target = g.players[g.pending.target].name
	}
	return &WindowView{
		ID:          g.window.id,
		Action:      declared,
		Block:       g.blockAsSeen(),
		YourOptions: append([]Option{}, g.optionsFor(g.indexOf(name))...),
		WaitingOn:   g.waitingOn(),
	}
}

func (g *Game) blockAsSeen() *BlockView {
	if g.window.block == nil {
		return nil
	}
	return &BlockView{By: g.players[g.window.block.by].name, Character: g.window.block.character}
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
			if offered.Targets = g.validTargetNames(by, rule); len(offered.Targets) == 0 {
				continue
			}
		}
		available = append(available, offered)
	}
	return available
}
