package engine

type ActionType uint8

const (
	NoAction ActionType = iota
	Income
	Coup
)

type Rule struct {
	Name        string
	Cost        int
	NeedsTarget bool
	Effect      func(g *Game, by, target int) []Event
}

var actionOrderForDeterministicView = []ActionType{Income, Coup}

var rules = map[ActionType]Rule{
	Income: {
		Name: "income",
		Effect: func(g *Game, by, _ int) []Event {
			g.players[by].coins++
			return nil
		},
	},
	Coup: {
		Name:        "coup",
		Cost:        7,
		NeedsTarget: true,
		Effect:      func(g *Game, _, target int) []Event { return g.openInfluenceLoss(target) },
	},
}

func ActionByName(name string) (ActionType, bool) {
	for action, rule := range rules {
		if rule.Name == name {
			return action, true
		}
	}
	return NoAction, false
}

func ActionNames() []string {
	names := make([]string, 0, len(actionOrderForDeterministicView))
	for _, action := range actionOrderForDeterministicView {
		names = append(names, rules[action].Name)
	}
	return names
}
