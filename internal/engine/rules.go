package engine

import "fmt"

type ActionType uint8

const (
	NoAction ActionType = iota
	Income
	ForeignAid
	Coup
	Tax
	Assassinate
)

type Rule struct {
	Name        string
	Cost        int
	Claims      Character
	NeedsTarget bool
	BlockedBy   []Character
	Declaration string
	Effect      func(g *Game, by, target int) []Event
}

func (r Rule) challengeable() bool { return r.Claims != NoCharacter }

var actionOrderForDeterministicView = []ActionType{Income, ForeignAid, Tax, Assassinate, Coup}

var rules = map[ActionType]Rule{
	Income: {
		Name:        "income",
		Declaration: "%[1]s pegou Renda.",
		Effect:      gainCoins(1),
	},
	ForeignAid: {
		Name:        "foreign_aid",
		BlockedBy:   []Character{Duke},
		Declaration: "%[1]s pediu Ajuda Externa.",
		Effect:      gainCoins(2),
	},
	Tax: {
		Name:        "tax",
		Claims:      Duke,
		Declaration: "%[1]s alegou Duque para cobrar Taxas.",
		Effect:      gainCoins(3),
	},
	Assassinate: {
		Name:        "assassinate",
		Cost:        3,
		Claims:      Assassin,
		NeedsTarget: true,
		BlockedBy:   []Character{Contessa},
		Declaration: "%[1]s pagou 3 e alegou Assassino contra %[2]s.",
		Effect:      targetLosesInfluence,
	},
	Coup: {
		Name:        "coup",
		Cost:        7,
		NeedsTarget: true,
		Declaration: "%[1]s pagou 7 e deu um Golpe de Estado em %[2]s.",
		Effect:      targetLosesInfluence,
	},
}

func gainCoins(amount int) func(g *Game, by, target int) []Event {
	return func(g *Game, by, _ int) []Event {
		g.players[by].coins += amount
		gained := g.narrate("coins_gained", fmt.Sprintf("%s recebeu %s.", g.players[by].name, coinsPtBR(amount)))
		return append([]Event{gained}, g.proceed(endTurn)...)
	}
}

func targetLosesInfluence(g *Game, _, target int) []Event {
	return g.loseInfluenceThen(target, endTurn)
}

func coinsPtBR(amount int) string {
	if amount == 1 {
		return "1 moeda"
	}
	return fmt.Sprintf("%d moedas", amount)
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
