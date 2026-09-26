package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gabrielmgaa/coup/internal/engine"
	"github.com/gabrielmgaa/coup/internal/protocol"
)

type Table struct {
	Server  string
	Name    string
	Room    string
	Token   string
	Options engine.Options
}

func (t Table) firstMessage() protocol.FromClient {
	switch {
	case t.Token != "":
		return protocol.FromClient{Type: "reconnect", Room: t.Room, Token: t.Token}
	case t.Room == "":
		return protocol.FromClient{Type: "create_room", Name: t.Name, Options: t.Options}
	}
	return protocol.FromClient{Type: "join", Room: t.Room, Name: t.Name}
}

func Play(ctx context.Context, table Table, remember func(room, token string) error, options ...tea.ProgramOption) error {
	opened, err := dialTable(ctx, table.Server)
	if err != nil {
		return err
	}
	defer opened.close()
	if err := opened.write(table.firstMessage()); err != nil {
		return err
	}
	final, err := tea.NewProgram(NewModel(opened.outgoing, opened.receive, rememberCmd(remember)), options...).Run()
	if err != nil {
		return err
	}
	return final.(Model).Lost()
}

func rememberCmd(remember func(room, token string) error) func(room, token string) tea.Cmd {
	return func(room, token string) tea.Cmd {
		return func() tea.Msg {
			if err := remember(room, token); err != nil {
				return sessionNotSaved{err: err}
			}
			return nil
		}
	}
}
