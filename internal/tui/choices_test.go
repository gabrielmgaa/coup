package tui

import (
	"testing"

	"github.com/gabrielmgaa/coup/internal/engine"
)

func TestTheContessaTakesTheFeminineArticle(t *testing.T) {
	challengeOrPass := []engine.Option{{Answer: engine.Challenge}, {Answer: engine.Pass}}
	blockedWithContessa := engine.WindowView{ID: 3, YourOptions: challengeOrPass,
		Action: engine.ActionView{Name: "assassinate", By: "tester1", Target: "tester2", Claims: engine.Assassin},
		Block:  &engine.BlockView{By: "tester2", Character: engine.Contessa}}
	taxClaimingDuke := engine.WindowView{ID: 4, YourOptions: challengeOrPass,
		Action: engine.ActionView{Name: "tax", By: "tester2", Claims: engine.Duke}}
	losingContessa := engine.View{You: "tester1", Losing: "tester1",
		Players: []engine.PlayerView{{Name: "tester1", Hidden: 1, MyCards: []engine.Character{engine.Contessa}}}}

	for _, scenario := range []struct {
		offered  []choice
		expected string
	}{
		{responseChoices(blockedWithContessa), "contestar a Condessa de tester2"},
		{responseChoices(taxClaimingDuke), "contestar o Duque de tester2"},
		{gameChoices(losingContessa), "revelar a Condessa"},
	} {
		if first := scenario.offered[0].label; first != scenario.expected {
			t.Errorf("the choice reads %q, expected %q", first, scenario.expected)
		}
	}
}
