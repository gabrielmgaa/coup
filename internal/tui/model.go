package tui

import (
	"errors"
	"fmt"
	"slices"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gabrielmgaa/coup/internal/engine"
	"github.com/gabrielmgaa/coup/internal/protocol"
)

type tickArrived struct{}

type Model struct {
	outgoing  func(protocol.FromClient) tea.Cmd
	receive   tea.Cmd
	remember  func(room, token string) tea.Cmd
	lobby     *protocol.LobbyView
	game      *protocol.GameState
	log       []engine.Event
	refusal   string
	cursor    int
	offered   []choice
	closesAt  time.Time
	resumesAt time.Time
	lost      error
	now       func() time.Time
}

func NewModel(outgoing func(protocol.FromClient) tea.Cmd, receive tea.Cmd, remember func(room, token string) tea.Cmd) Model {
	return Model{outgoing: outgoing, receive: receive, remember: remember, now: time.Now}
}

func (m Model) Init() tea.Cmd { return tea.Batch(m.receive, tick()) }

func tick() tea.Cmd {
	return tea.Tick(time.Second, func(time.Time) tea.Msg { return tickArrived{} })
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch arrived := msg.(type) {
	case tea.KeyMsg:
		return m.press(arrived.String())
	case tickArrived:
		return m, tick()
	case welcomeArrived:
		return m, tea.Batch(m.remember(arrived.room, arrived.token), m.receive)
	case lobbyArrived:
		m.lobby, m.game, m.log, m.refusal = &arrived.state, nil, nil, ""
		return m.settle(), m.receive
	case updateArrived:
		return m.applyUpdate(arrived), m.receive
	case refusalArrived:
		m.refusal = arrived.message
		return m, m.receive
	case sessionNotSaved:
		m.refusal = "não deu para salvar a sessão: " + arrived.err.Error()
		return m, nil
	case connectionLost:
		m.lost = m.explain(arrived.err)
		return m, tea.Quit
	}
	return m, nil
}

func (m Model) applyUpdate(arrived updateArrived) Model {
	m.game, m.refusal = &arrived.state, ""
	m.closesAt = m.now().Add(time.Duration(arrived.state.ClosesInMs) * time.Millisecond)
	if arrived.state.Paused != nil {
		m.resumesAt = m.now().Add(time.Duration(arrived.state.Paused.ResumesInMs) * time.Millisecond)
	}
	m.log = append(m.log, arrived.events...)
	return m.settle()
}

func (m Model) press(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "up", "k":
		m.cursor--
		return m.settle(), nil
	case "down", "j":
		m.cursor++
		return m.settle(), nil
	case "enter", " ":
		return m, m.choose()
	}
	return m, nil
}

func (m Model) choose() tea.Cmd {
	available := m.choices()
	if len(available) == 0 {
		return nil
	}
	return m.outgoing(available[m.cursor].message)
}

func (m Model) settle() Model {
	available := m.choices()
	if !sameChoices(m.offered, available) {
		m.offered = available
		m.cursor = safestChoice(available)
		return m
	}
	m.cursor = max(0, min(m.cursor, len(available)-1))
	return m
}

func sameChoices(before, after []choice) bool {
	return slices.EqualFunc(before, after, func(earlier, later choice) bool {
		return earlier.label == later.label && earlier.message.Window == later.message.Window
	})
}

func safestChoice(available []choice) int {
	for position, option := range available {
		if option.message.Answer == engine.Pass.String() {
			return position
		}
	}
	return 0
}

func (m Model) choices() []choice {
	if m.game != nil {
		return gameChoices(m.game.View)
	}
	if m.lobby != nil {
		return lobbyChoices(*m.lobby)
	}
	return nil
}

func (m Model) explain(cause error) error {
	if m.refusal != "" {
		return errors.New(m.refusal)
	}
	return fmt.Errorf("%w: %w", errServerDropped, cause)
}

func (m Model) Lost() error { return m.lost }
