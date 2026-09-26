package tui

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/coder/websocket"
	"github.com/gabrielmgaa/coup/internal/protocol"
)

var errServerDropped = errors.New("a conexão com o servidor caiu")

type link struct {
	ctx  context.Context
	conn *websocket.Conn
}

func dialTable(ctx context.Context, address string) (*link, error) {
	conn, _, err := websocket.Dial(ctx, address, nil)
	if err != nil {
		return nil, fmt.Errorf("não consegui conectar ao servidor em %s: %w", address, err)
	}
	return &link{ctx: ctx, conn: conn}, nil
}

func (l *link) receive() tea.Msg {
	_, encoded, err := l.conn.Read(l.ctx)
	if err != nil {
		return connectionLost{err: err}
	}
	decoded, err := decode(encoded)
	if err != nil {
		return connectionLost{err: err}
	}
	return decoded
}

func (l *link) outgoing(message protocol.FromClient) tea.Cmd {
	return func() tea.Msg {
		if err := l.write(message); err != nil {
			return connectionLost{err: err}
		}
		return nil
	}
}

func (l *link) write(message protocol.FromClient) error {
	encoded, err := json.Marshal(message)
	if err != nil {
		return err
	}
	return l.conn.Write(l.ctx, websocket.MessageText, encoded)
}

func (l *link) close() { l.conn.CloseNow() }
