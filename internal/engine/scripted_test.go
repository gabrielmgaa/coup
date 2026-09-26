package engine

import "testing"

type script struct {
	t    *testing.T
	game *Game
	seen map[string]bool
}

func (s *script) act(by string, action ActionType, target string) {
	s.t.Helper()
	events, err := s.game.Apply(Act{By: by, Action: action, Target: target})
	if err != nil {
		s.t.Fatalf("%s playing %s on %q was refused: %v", by, rules[action].Name, target, err)
	}
	s.seen[rules[action].Name] = true
	s.record(events)
}

func (s *script) everyonePasses() {
	s.t.Helper()
	for s.game.window != nil {
		waiting := s.game.waitingOn()
		s.record(respond(s.t, s.game, waiting[0], Pass))
	}
}

func (s *script) block(by string, character Character) {
	s.t.Helper()
	blockWith(s.t, s.game, by, character)
	s.seen["block_with_"+character.String()] = true
}

func (s *script) record(events []Event) {
	for _, seen := range events {
		s.seen[seen.Type] = true
	}
}

func (s *script) expectCoins(name string, coins int) {
	s.t.Helper()
	if held := s.game.players[s.game.indexOf(name)].coins; held != coins {
		s.t.Fatalf("%s holds %d coins, expected %d", name, held, coins)
	}
}

func (s *script) playToTheEnd() {
	s.t.Helper()
	for moves := 0; s.game.phase != Finished; moves++ {
		if moves > 200 {
			s.t.Fatal("the game did not end within 200 moves")
		}
		s.playForcedStep()
	}
}

func (s *script) playForcedStep() {
	s.t.Helper()
	if s.game.phase == AwaitingInfluenceLoss {
		loser := s.game.players[s.game.losing]
		apply(s.t, s.game, LoseInfluence{By: loser.name, Card: loser.hand[0]})
		return
	}
	actor := s.game.players[s.game.turn]
	if actor.coins < rules[Coup].Cost {
		s.act(actor.name, Income, "")
		return
	}
	s.act(actor.name, Coup, s.game.validTargetNames(s.game.turn, rules[Coup])[0])
}

func TestAFourPlayerGameUsesEveryActionAndEveryBlockAndEnds(t *testing.T) {
	game := newGameWithDeck([]string{"tester1", "tester2", "tester3", "tester4"},
		fixedDeck(Duke, Assassin, Captain, Ambassador, Contessa, Duke, Assassin, Captain))
	play := &script{t: t, game: game, seen: map[string]bool{}}

	play.act("tester1", Income, "")
	play.act("tester2", ForeignAid, "")
	play.block("tester3", Duke)
	play.everyonePasses()
	play.expectCoins("tester2", 2)

	play.act("tester3", Tax, "")
	play.everyonePasses()
	play.expectCoins("tester3", 5)

	play.act("tester4", Steal, "tester1")
	play.block("tester1", Ambassador)
	respond(t, game, "tester4", Challenge)
	apply(t, game, LoseInfluence{By: "tester1", Card: Duke})
	play.expectCoins("tester1", 1)
	play.expectCoins("tester4", 4)

	play.act("tester1", Income, "")
	play.act("tester2", Exchange, "")
	play.everyonePasses()
	kept := handOf(game, "tester2")
	apply(t, game, ReturnCards{By: "tester2", Cards: [2]Character{kept[2], kept[3]}})

	play.act("tester3", Income, "")
	play.act("tester4", Assassinate, "tester3")
	play.block("tester3", Contessa)
	play.everyonePasses()
	play.expectCoins("tester4", 1)
	play.expectCoins("tester3", 6)

	play.playToTheEnd()

	for _, expected := range []string{"income", "foreign_aid", "coup", "tax", "assassinate", "steal", "exchange",
		"block_with_duke", "block_with_contessa", "block_with_ambassador", "game_over"} {
		if !play.seen[expected] {
			t.Errorf("the scripted game never saw %s", expected)
		}
	}
	if winner := ViewFor(game, "tester1").Winner; winner == "" {
		t.Error("the game ended with no winner")
	}
}
