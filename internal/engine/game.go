package engine

import (
	"fmt"
	"math/rand/v2"
)

type Phase uint8

const (
	AwaitingAction Phase = iota
	AwaitingInfluenceLoss
	Finished
)

var phaseName = map[Phase]string{
	AwaitingAction:        "awaiting_action",
	AwaitingInfluenceLoss: "awaiting_influence_loss",
	Finished:              "finished",
}

func (p Phase) String() string { return phaseName[p] }

const (
	startingInfluences = 2
	startingCoins      = 2
	startingCoinsDuel  = 1
	coinsForcingCoup   = 10
	nobody             = -1

	RulebookCoins = 0
)

type player struct {
	name     string
	coins    int
	hand     []Character
	revealed []Character
}

func (p *player) alive() bool { return len(p.hand) > 0 }

type Game struct {
	players       []player
	deck          []Character
	phase         Phase
	turn          int
	losing        int
	winner        string
	eventsEmitted int
}

func NewGame(names []string, rng *rand.Rand, initialCoins int) *Game {
	deck := baseDeck()
	shuffle(deck, rng)
	game := newGameWithDeck(names, deck)
	if initialCoins != RulebookCoins {
		for i := range game.players {
			game.players[i].coins = initialCoins
		}
	}
	game.turn = rng.IntN(len(game.players))
	return game
}

func newGameWithDeck(names []string, deck []Character) *Game {
	coins := startingCoins
	if len(names) == 2 {
		coins = startingCoinsDuel
	}
	game := &Game{deck: deck, losing: nobody}
	for _, name := range names {
		hand := append([]Character(nil), game.deck[:startingInfluences]...)
		game.deck = game.deck[startingInfluences:]
		game.players = append(game.players, player{name: name, coins: coins, hand: hand})
	}
	return game
}

func (g *Game) Apply(move Move) ([]Event, error) {
	switch chosen := move.(type) {
	case Act:
		return g.act(chosen)
	case LoseInfluence:
		return g.loseInfluence(chosen)
	}
	return nil, &Refusal{Code: "illegal_action", Message: "jogada desconhecida"}
}

type declaredAction struct {
	by     int
	target int
	rule   Rule
}

func (g *Game) act(a Act) ([]Event, error) {
	declared, err := g.checkAction(a)
	if err != nil {
		return nil, err
	}
	g.players[declared.by].coins -= declared.rule.Cost
	events := []Event{g.narrate("action_declared",
		g.actionTextPtBR(declared.rule, declared.by, declared.target))}
	events = append(events, declared.rule.Effect(g, declared.by, declared.target)...)
	return append(events, g.closeTurn()...), nil
}

func (g *Game) checkAction(a Act) (declaredAction, error) {
	if g.phase != AwaitingAction {
		return declaredAction{}, &Refusal{Code: "illegal_action", Message: "não é hora de agir",
			Received: "act", Expected: g.phase.String()}
	}
	by, err := g.indexOnTurn(a.By)
	if err != nil {
		return declaredAction{}, err
	}
	rule, known := rules[a.Action]
	if !known {
		return declaredAction{}, &Refusal{Code: "illegal_action", Message: "ação que não existe",
			Received: int(a.Action), Expected: ActionNames()}
	}
	if g.players[by].coins >= coinsForcingCoup && a.Action != Coup {
		return declaredAction{}, &Refusal{Code: "coup_required",
			Message:  "com 10 moedas ou mais o turno inteiro é um Golpe",
			Received: rule.Name, Expected: "coup"}
	}
	target := nobody
	if rule.NeedsTarget {
		if target, err = g.indexOfLivingTarget(by, a.Target); err != nil {
			return declaredAction{}, err
		}
	}
	if g.players[by].coins < rule.Cost {
		return declaredAction{}, &Refusal{Code: "insufficient_coins",
			Message:  "saldo menor que o custo da ação",
			Received: g.players[by].coins, Expected: rule.Cost}
	}
	return declaredAction{by: by, target: target, rule: rule}, nil
}

func (g *Game) closeTurn() []Event {
	if g.phase == AwaitingInfluenceLoss {
		return nil
	}
	if events, over := g.checkGameOver(); over {
		return events
	}
	g.passTurn()
	return []Event{g.narrate("turn_passed", fmt.Sprintf("Agora é a vez de %s.", g.players[g.turn].name))}
}

func (g *Game) passTurn() {
	for range g.players {
		g.turn = (g.turn + 1) % len(g.players)
		if g.players[g.turn].alive() {
			return
		}
	}
}

func (g *Game) indexOf(name string) int {
	for i := range g.players {
		if g.players[i].name == name {
			return i
		}
	}
	return nobody
}

func (g *Game) indexOnTurn(name string) (int, error) {
	index := g.indexOf(name)
	if index == nobody || index != g.turn {
		return nobody, &Refusal{Code: "not_your_turn", Message: "não é a vez de quem jogou",
			Received: name, Expected: g.players[g.turn].name}
	}
	return index, nil
}

func (g *Game) indexOfLivingTarget(by int, name string) (int, error) {
	target := g.indexOf(name)
	if target == nobody || target == by || !g.players[target].alive() {
		return nobody, &Refusal{Code: "invalid_target", Message: "alvo fora do jogo ou inexistente",
			Received: name, Expected: g.validTargetNames(by)}
	}
	return target, nil
}

func (g *Game) validTargetNames(by int) []string {
	var names []string
	for i := range g.players {
		if i != by && g.players[i].alive() {
			names = append(names, g.players[i].name)
		}
	}
	return names
}

func (g *Game) actionTextPtBR(rule Rule, by, target int) string {
	if rule.NeedsTarget {
		return fmt.Sprintf("%s pagou %d e deu um Golpe de Estado em %s.",
			g.players[by].name, rule.Cost, g.players[target].name)
	}
	return fmt.Sprintf("%s pegou Renda.", g.players[by].name)
}
