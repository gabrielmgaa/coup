package engine

import (
	"fmt"
	"math/rand/v2"
)

type Phase uint8

const (
	AwaitingAction Phase = iota
	AwaitingResponse
	AwaitingInfluenceLoss
	AwaitingExchange
	Finished
)

var phaseName = map[Phase]string{
	AwaitingAction:        "awaiting_action",
	AwaitingResponse:      "awaiting_response",
	AwaitingInfluenceLoss: "awaiting_influence_loss",
	AwaitingExchange:      "awaiting_exchange",
	Finished:              "finished",
}

func (p Phase) String() string { return phaseName[p] }

type followUp uint8

const (
	endTurn followUp = iota
	continueAction
	resolveAction
)

const (
	startingInfluences = 2
	startingCoins      = 2
	startingCoinsDuel  = 1
	coinsForcingCoup   = 10
	maxStolenCoins     = 2
	exchangeDraw       = 2
	nobody             = -1

	RulebookCoins = 0
	MaxPlayers    = 6
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
	rng           *rand.Rand
	phase         Phase
	turn          int
	pending       *pendingAction
	window        *window
	losing        int
	afterLoss     followUp
	decision      int
	winner        string
	eventsEmitted int
}

func NewGame(names []string, rng *rand.Rand, initialCoins int) (*Game, error) {
	if len(names) > MaxPlayers {
		return nil, &Refusal{Code: "too_many_players", Message: "gente demais para um baralho só",
			Received: len(names), Expected: MaxPlayers}
	}
	deck := baseDeck()
	shuffle(deck, rng)
	game := newGameWithDeck(names, deck)
	game.rng = rng
	if initialCoins != RulebookCoins {
		for i := range game.players {
			game.players[i].coins = initialCoins
		}
	}
	game.turn = rng.IntN(len(game.players))
	return game, nil
}

func newGameWithDeck(names []string, deck []Character) *Game {
	coins := startingCoins
	if len(names) == 2 {
		coins = startingCoinsDuel
	}
	game := &Game{deck: deck, rng: rand.New(rand.NewPCG(1, 2)), losing: nobody}
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
	case Respond:
		return g.respond(chosen)
	case LoseInfluence:
		return g.loseInfluence(chosen)
	case ReturnCards:
		return g.returnCards(chosen)
	}
	return nil, &Refusal{Code: "illegal_action", Message: "jogada desconhecida"}
}

type pendingAction struct {
	rule    Rule
	by      int
	target  int
	reacted map[int]bool
}

func (g *Game) act(a Act) ([]Event, error) {
	declared, err := g.checkAction(a)
	if err != nil {
		return nil, err
	}
	g.players[declared.by].coins -= declared.rule.Cost
	g.pending = &declared
	events := []Event{g.narrate("action_declared", g.declarationPtBR(declared))}
	return append(events, g.openActionWindow()...), nil
}

func (g *Game) checkAction(a Act) (pendingAction, error) {
	if g.phase != AwaitingAction {
		return pendingAction{}, &Refusal{Code: "illegal_action", Message: "não é hora de agir",
			Received: "act", Expected: g.phase.String()}
	}
	by, err := g.indexOnTurn(a.By)
	if err != nil {
		return pendingAction{}, err
	}
	rule, known := rules[a.Action]
	if !known {
		return pendingAction{}, &Refusal{Code: "illegal_action", Message: "ação que não existe",
			Received: int(a.Action), Expected: ActionNames()}
	}
	if g.players[by].coins >= coinsForcingCoup && a.Action != Coup {
		return pendingAction{}, &Refusal{Code: "coup_required",
			Message:  "com 10 moedas ou mais o turno inteiro é um Golpe",
			Received: rule.Name, Expected: "coup"}
	}
	target := nobody
	if rule.NeedsTarget {
		if target, err = g.indexOfValidTarget(by, rule, a.Target); err != nil {
			return pendingAction{}, err
		}
	}
	if g.players[by].coins < rule.Cost {
		return pendingAction{}, &Refusal{Code: "insufficient_coins",
			Message:  "saldo menor que o custo da ação",
			Received: g.players[by].coins, Expected: rule.Cost}
	}
	return pendingAction{rule: rule, by: by, target: target, reacted: map[int]bool{}}, nil
}

func (g *Game) resolveAction() []Event {
	return g.pending.rule.Effect(g, g.pending.by, g.pending.target)
}

func (g *Game) proceed(then followUp) []Event {
	if events, over := g.checkGameOver(); over {
		return events
	}
	switch then {
	case continueAction:
		return g.continueAction()
	case resolveAction:
		return g.resolveAction()
	}
	return g.closeTurn()
}

func (g *Game) closeTurn() []Event {
	g.pending = nil
	g.window = nil
	g.phase = AwaitingAction
	g.passTurn()
	g.decision++
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

func (g *Game) indexOfValidTarget(by int, rule Rule, name string) (int, error) {
	target := g.indexOf(name)
	if target == nobody || !g.validTarget(by, rule, target) {
		return nobody, &Refusal{Code: "invalid_target", Message: "alvo que esta ação não pode mirar",
			Received: name, Expected: g.validTargetNames(by, rule)}
	}
	return target, nil
}

func (g *Game) validTarget(by int, rule Rule, target int) bool {
	who := &g.players[target]
	return target != by && who.alive() && (rule.ValidTarget == nil || rule.ValidTarget(who))
}

func (g *Game) validTargetNames(by int, rule Rule) []string {
	names := []string{}
	for i := range g.players {
		if g.validTarget(by, rule, i) {
			names = append(names, g.players[i].name)
		}
	}
	return names
}

func (g *Game) declarationPtBR(declared pendingAction) string {
	target := ""
	if declared.target != nobody {
		target = g.players[declared.target].name
	}
	return fmt.Sprintf(declared.rule.Declaration, g.players[declared.by].name, target)
}
