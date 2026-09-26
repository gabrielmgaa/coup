package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gabrielmgaa/coup/internal/protocol"
)

type Table struct {
	Server string
	Name   string
	Room   string
}

func (t Table) firstMessage() protocol.FromClient {
	if t.Room == "" {
		return protocol.FromClient{Type: "create_room", Name: t.Name}
	}
	return protocol.FromClient{Type: "join", Room: t.Room, Name: t.Name}
}

func Play(ctx context.Context, table Table, options ...tea.ProgramOption) error {
	opened, err := dialTable(ctx, table.Server)
	if err != nil {
		return err
	}
	defer opened.close()
	if err := opened.write(table.firstMessage()); err != nil {
		return err
	}
	final, err := tea.NewProgram(NewModel(opened.outgoing, opened.receive), options...).Run()
	if err != nil {
		return err
	}
	return final.(Model).Lost()
}
