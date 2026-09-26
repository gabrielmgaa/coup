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
	Steal
	Exchange
)

type Rule struct {
	Name        string
	Cost        int
	Claims      Character
	NeedsTarget bool
	BlockedBy   []Character
	ValidTarget func(target *player) bool
	Declaration string
	Effect      func(g *Game, by, target int) []Event
}

func (r Rule) challengeable() bool { return r.Claims != NoCharacter }

var actionOrderForDeterministicView = []ActionType{Income, ForeignAid, Tax, Exchange, Steal, Assassinate, Coup}

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
	Steal: {
		Name:        "steal",
		Claims:      Captain,
		NeedsTarget: true,
		BlockedBy:   []Character{Captain, Ambassador},
		ValidTarget: hasCoins,
		Declaration: "%[1]s alegou Capitão para extorquir %[2]s.",
		Effect:      steal,
	},
	Exchange: {
		Name:        "exchange",
		Claims:      Ambassador,
		Declaration: "%[1]s alegou Embaixador para trocar cartas.",
		Effect:      drawForExchange,
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

func hasCoins(target *player) bool { return target.coins > 0 }

func steal(g *Game, by, target int) []Event {
	taken := min(maxStolenCoins, g.players[target].coins)
	g.players[target].coins -= taken
	g.players[by].coins += taken
	stolen := g.narrate("coins_stolen", fmt.Sprintf("%s tirou %s de %s.",
		g.players[by].name, coinsPtBR(taken), g.players[target].name))
	return append([]Event{stolen}, g.proceed(endTurn)...)
}

func drawForExchange(g *Game, by, _ int) []Event {
	g.players[by].hand = append(g.players[by].hand, g.deck[:exchangeDraw]...)
	g.deck = g.deck[exchangeDraw:]
	g.phase = AwaitingExchange
	g.decision++
	return []Event{g.narrate("cards_drawn", fmt.Sprintf("%s comprou 2 cartas do baralho.", g.players[by].name))}
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
