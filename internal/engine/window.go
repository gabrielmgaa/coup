package engine

import (
	"encoding/json"
	"fmt"
)

type Answer uint8

const (
	NoAnswer Answer = iota
	Challenge
	Block
	Pass
)

var answerName = map[Answer]string{
	Challenge: "challenge",
	Block:     "block",
	Pass:      "pass",
}

func (a Answer) String() string { return answerName[a] }

func (a Answer) MarshalJSON() ([]byte, error) { return json.Marshal(a.String()) }

func (a *Answer) UnmarshalJSON(encoded []byte) error {
	var written string
	if err := json.Unmarshal(encoded, &written); err != nil {
		return err
	}
	answer, known := AnswerByName(written)
	if !known {
		return &Refusal{Code: "illegal_action", Message: "resposta que não existe",
			Received: written, Expected: []string{"challenge", "block", "pass"}}
	}
	*a = answer
	return nil
}

func AnswerByName(name string) (Answer, bool) {
	for answer, written := range answerName {
		if written == name {
			return answer, true
		}
	}
	return NoAnswer, false
}

type Option struct {
	Answer    Answer    `json:"answer"`
	Character Character `json:"character,omitempty"`
}

type window struct {
	id        int
	block     *pendingBlock
	blockOnly bool
	pending   map[int]bool
	responded map[int]bool
}

type pendingBlock struct {
	by        int
	character Character
}

func (g *Game) openActionWindow() []Event {
	eligible := []int{}
	for i := range g.players {
		if i != g.pending.by && g.players[i].alive() && len(g.reactionsOf(i)) > 0 {
			eligible = append(eligible, i)
		}
	}
	if len(eligible) == 0 {
		return g.resolveAction()
	}
	g.openWindow(eligible, window{})
	return nil
}

func (g *Game) openWindow(eligible []int, shape window) {
	g.decision++
	g.window = &window{id: g.decision, block: shape.block, blockOnly: shape.blockOnly,
		pending: map[int]bool{}, responded: map[int]bool{}}
	for _, i := range eligible {
		g.window.pending[i] = true
	}
	g.phase = AwaitingResponse
}

func (g *Game) reactionsOf(i int) []Option {
	if g.window != nil && g.window.block != nil {
		return []Option{{Answer: Challenge}}
	}
	reactions := []Option{}
	if g.pending.rule.challengeable() && (g.window == nil || !g.window.blockOnly) {
		reactions = append(reactions, Option{Answer: Challenge})
	}
	if g.mayBlock(i) {
		for _, blocker := range g.pending.rule.BlockedBy {
			reactions = append(reactions, Option{Answer: Block, Character: blocker})
		}
	}
	return reactions
}

func (g *Game) mayBlock(i int) bool {
	rule := g.pending.rule
	return len(rule.BlockedBy) > 0 && (!rule.NeedsTarget || i == g.pending.target)
}

func (g *Game) optionsFor(i int) []Option {
	if g.window == nil || !g.window.pending[i] {
		return nil
	}
	return append(g.reactionsOf(i), Option{Answer: Pass})
}

func (g *Game) respond(r Respond) ([]Event, error) {
	responder, err := g.checkResponse(r)
	if err != nil {
		return nil, err
	}
	g.window.responded[responder] = true
	delete(g.window.pending, responder)
	if g.window.block == nil {
		g.pending.reacted[responder] = true
	}
	switch r.Answer {
	case Challenge:
		return g.challenge(responder), nil
	case Block:
		return g.block(responder, r.Character), nil
	}
	return g.pass(responder), nil
}

func (g *Game) checkResponse(r Respond) (int, error) {
	if g.phase != AwaitingResponse {
		return nobody, &Refusal{Code: "illegal_action", Message: "não há janela de reação aberta",
			Received: "respond", Expected: g.phase.String()}
	}
	if r.Window != g.window.id {
		return nobody, &Refusal{Code: "window_closed", Message: fmt.Sprintf("a janela %d já fechou", r.Window),
			Received: r.Window, Expected: g.window.id}
	}
	responder := g.indexOf(r.By)
	if g.window.responded[responder] {
		return nobody, &Refusal{Code: "already_responded", Message: "cada um responde uma vez por janela",
			Received: r.By, Expected: g.waitingOn()}
	}
	offered := g.optionsFor(responder)
	for _, option := range offered {
		if option == (Option{Answer: r.Answer, Character: r.Character}) {
			return responder, nil
		}
	}
	return nobody, &Refusal{Code: "illegal_action", Message: "essa resposta não está entre as suas opções",
		Received: Option{Answer: r.Answer, Character: r.Character}, Expected: offered}
}

func (g *Game) pass(responder int) []Event {
	passed := g.narrate("passed", fmt.Sprintf("%s deixou passar.", g.players[responder].name))
	if len(g.window.pending) > 0 {
		return []Event{passed}
	}
	blocked := g.window.block
	g.window = nil
	if blocked == nil {
		return append([]Event{passed}, g.resolveAction()...)
	}
	held := g.narrate("block_held", fmt.Sprintf("O bloqueio de %s valeu.", g.players[blocked.by].name))
	return append([]Event{passed, held}, g.proceed(endTurn)...)
}

func (g *Game) block(blocker int, character Character) []Event {
	eligible := []int{}
	for i := range g.players {
		if i != blocker && g.players[i].alive() {
			eligible = append(eligible, i)
		}
	}
	g.openWindow(eligible, window{block: &pendingBlock{by: blocker, character: character}})
	return []Event{g.narrate("blocked", fmt.Sprintf("%s alegou %s para bloquear %s.",
		g.players[blocker].name, character.LabelPtBR(), g.players[g.pending.by].name))}
}

func (g *Game) continueAction() []Event {
	eligible := []int{}
	for i := range g.players {
		if g.players[i].alive() && g.mayBlock(i) && (g.options.IndependentReactions || !g.pending.reacted[i]) {
			eligible = append(eligible, i)
		}
	}
	if len(eligible) == 0 {
		return g.resolveAction()
	}
	g.openWindow(eligible, window{blockOnly: true})
	return nil
}

func (g *Game) waitingOn() []string {
	names := []string{}
	for i := range g.players {
		if g.window != nil && g.window.pending[i] {
			names = append(names, g.players[i].name)
		}
	}
	return names
}
