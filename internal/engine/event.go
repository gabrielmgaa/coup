package engine

type Event struct {
	N    int    `json:"n"`
	Type string `json:"type"`
	Text string `json:"text"`
}

func (g *Game) narrate(kind, textPtBR string) Event {
	g.eventsEmitted++
	return Event{N: g.eventsEmitted, Type: kind, Text: textPtBR}
}
