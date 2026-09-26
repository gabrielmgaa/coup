package tui

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/gabrielmgaa/coup/internal/engine"
	"github.com/gabrielmgaa/coup/internal/protocol"
)

const visibleLogLines = 8

var (
	titleStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#d9a441"))
	boxStyle     = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)
	turnStyle    = boxStyle.BorderForeground(lipgloss.Color("#d9a441"))
	deadStyle    = boxStyle.Foreground(lipgloss.Color("#666666"))
	refusalStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#b5443a"))
	faintStyle   = lipgloss.NewStyle().Faint(true)
	cursorStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#d9a441"))
)

func (m Model) View() string {
	sections := []string{titleStyle.Render("COUP")}
	switch {
	case m.game != nil:
		sections = append(sections, renderTable(*m.game), renderStatus(m.game.View), m.renderClock(), renderLog(m.log))
	case m.lobby != nil:
		sections = append(sections, renderLobby(*m.lobby))
	default:
		sections = append(sections, faintStyle.Render("conectando…"))
	}
	sections = append(sections, m.renderChoices())
	if m.refusal != "" {
		sections = append(sections, refusalStyle.Render(m.refusal))
	}
	sections = append(sections, faintStyle.Render("↑/↓ escolhe · enter confirma · q sai"))
	return strings.Join(sections, "\n\n") + "\n"
}

func renderLobby(lobby protocol.LobbyView) string {
	lines := []string{fmt.Sprintf("mesa %s", titleStyle.Render(lobby.Room))}
	if lobby.LastWinner != "" {
		lines = append(lines, titleStyle.Render(lobby.LastWinner+" venceu a última partida e começa a próxima"))
	}
	if lobby.Options.IndependentReactions {
		lines = append(lines, faintStyle.Render("regra da casa: reações independentes"))
	}
	for _, seat := range lobby.Players {
		status := "esperando"
		if seat.Ready {
			status = "pronto"
		}
		lines = append(lines, fmt.Sprintf("  %-16s %s%s", seat.Name, status, seatTags(lobby, seat.Name)))
	}
	return strings.Join(lines, "\n")
}

func seatTags(lobby protocol.LobbyView, name string) string {
	tags := ""
	if name == lobby.Host {
		tags += " · host"
	}
	if name == lobby.You {
		tags += " · você"
	}
	return tags
}

func renderTable(state protocol.GameState) string {
	seats := make([]string, 0, len(state.Players))
	for _, seen := range state.Players {
		seats = append(seats, seatStyle(state.View, seen).Render(renderSeat(state, seen)))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, seats...)
}

func seatStyle(state engine.View, seen engine.PlayerView) lipgloss.Style {
	if seen.Eliminated {
		return deadStyle
	}
	if seen.Name == state.TurnOf {
		return turnStyle
	}
	return boxStyle
}

func renderSeat(state protocol.GameState, seen engine.PlayerView) string {
	name := seen.Name
	if seen.Name == state.You {
		name += " (você)"
	}
	if slices.Contains(state.Disconnected, seen.Name) {
		name += " (caiu)"
	}
	return strings.Join([]string{name, fmt.Sprintf("%d moedas", seen.Coins), renderCards(seen)}, "\n")
}

func renderCards(seen engine.PlayerView) string {
	cards := []string{}
	for _, card := range seen.MyCards {
		cards = append(cards, card.LabelPtBR())
	}
	if seen.MyCards == nil {
		cards = append(cards, strings.Repeat("? ", seen.Hidden))
	}
	for _, card := range seen.Revealed {
		cards = append(cards, faintStyle.Strikethrough(true).Render(card.LabelPtBR()))
	}
	return strings.Join(cards, " ")
}

func renderStatus(state engine.View) string {
	switch {
	case state.Winner != "":
		return titleStyle.Render(state.Winner + " venceu a partida")
	case state.Losing == state.You:
		return "você perdeu uma influência — qual carta revela?"
	case state.Losing != "":
		return state.Losing + " está escolhendo qual carta perder…"
	case state.Window != nil:
		return renderWindow(*state.Window)
	case len(state.YourReturns) > 0:
		return "escolha as 2 cartas que voltam para o baralho"
	case state.Phase == "awaiting_exchange":
		return state.TurnOf + " está escolhendo cartas…"
	case state.TurnOf == state.You:
		return "sua vez"
	}
	return "é a vez de " + state.TurnOf + "…"
}

func renderWindow(window engine.WindowView) string {
	declared := fmt.Sprintf("%s declarou %s", window.Action.By, actionLabel(window.Action.Name))
	if window.Action.Target != "" {
		declared += " em " + window.Action.Target
	}
	if window.Block != nil {
		declared += fmt.Sprintf("; %s bloqueou com %s", window.Block.By, window.Block.Character.LabelPtBR())
	}
	return declared + " — esperando " + strings.Join(window.WaitingOn, ", ")
}

func (m Model) renderClock() string {
	if m.game.Paused != nil {
		return refusalStyle.Render(fmt.Sprintf("mesa pausada esperando %s voltar (%ds)",
			strings.Join(m.game.Paused.WaitingFor, ", "), m.game.Paused.ResumesInMs/1000))
	}
	if m.game.ClosesInMs == 0 {
		return ""
	}
	return faintStyle.Render(fmt.Sprintf("%ds para decidir", max(0, int(time.Until(m.closesAt).Seconds()))))
}

func renderLog(events []engine.Event) string {
	start := max(0, len(events)-visibleLogLines)
	lines := make([]string, 0, visibleLogLines)
	for _, event := range events[start:] {
		lines = append(lines, faintStyle.Render("· ")+event.Text)
	}
	return strings.Join(lines, "\n")
}

func (m Model) renderChoices() string {
	available := m.choices()
	lines := make([]string, 0, len(available))
	for position, option := range available {
		if position == m.cursor {
			lines = append(lines, cursorStyle.Render("> "+option.label))
			continue
		}
		lines = append(lines, "  "+option.label)
	}
	return strings.Join(lines, "\n")
}
