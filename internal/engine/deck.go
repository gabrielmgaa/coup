package engine

import (
	"encoding/json"
	"math/rand/v2"
)

type Character uint8

const (
	NoCharacter Character = iota
	Duke
	Assassin
	Captain
	Ambassador
	Contessa
)

const copiesPerCharacter = 3

var allCharacters = []Character{Duke, Assassin, Captain, Ambassador, Contessa}

var characterName = map[Character]string{
	Duke:       "duke",
	Assassin:   "assassin",
	Captain:    "captain",
	Ambassador: "ambassador",
	Contessa:   "contessa",
}

var characterLabelPtBR = map[Character]string{
	Duke:       "Duque",
	Assassin:   "Assassino",
	Captain:    "Capitão",
	Ambassador: "Embaixador",
	Contessa:   "Condessa",
}

func (c Character) String() string { return characterName[c] }

func (c Character) LabelPtBR() string { return characterLabelPtBR[c] }

func (c Character) MarshalJSON() ([]byte, error) { return json.Marshal(c.String()) }

func (c *Character) UnmarshalJSON(encoded []byte) error {
	var written string
	if err := json.Unmarshal(encoded, &written); err != nil {
		return err
	}
	character, known := CharacterByName(written)
	if !known {
		return &Refusal{Code: "illegal_action", Message: "personagem que não existe",
			Received: written, Expected: CharacterNames()}
	}
	*c = character
	return nil
}

func CharacterByName(name string) (Character, bool) {
	for character, written := range characterName {
		if written == name {
			return character, true
		}
	}
	return NoCharacter, false
}

func CharacterNames() []string { return namesOf(allCharacters) }

func namesOf(cards []Character) []string {
	names := make([]string, 0, len(cards))
	for _, card := range cards {
		names = append(names, card.String())
	}
	return names
}

func baseDeck() []Character {
	deck := make([]Character, 0, len(allCharacters)*copiesPerCharacter)
	for _, character := range allCharacters {
		for range copiesPerCharacter {
			deck = append(deck, character)
		}
	}
	return deck
}

func shuffle(deck []Character, rng *rand.Rand) {
	rng.Shuffle(len(deck), func(i, k int) { deck[i], deck[k] = deck[k], deck[i] })
}
