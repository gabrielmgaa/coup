package tui

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/coder/websocket"
)

func quietProgram() []tea.ProgramOption {
	return []tea.ProgramOption{tea.WithInput(strings.NewReader("")), tea.WithOutput(&strings.Builder{}), tea.WithoutRenderer()}
}

func TestPlayExplainsInPortugueseWhenTheServerIsNotThere(t *testing.T) {
	err := Play(context.Background(), Table{Server: "ws://127.0.0.1:1/ws", Name: "tester1"}, nil)
	opening := "não consegui conectar ao servidor em ws://127.0.0.1:1/ws: "
	if err == nil || !strings.HasPrefix(err.Error(), opening) || !strings.Contains(err.Error(), "connection refused") {
		t.Errorf("Play ended with %v, expected %q followed by the cause, connection refused", err, opening)
	}
}

func TestPlayExplainsInPortugueseWhenTheServerDrops(t *testing.T) {
	dropping := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		conn, err := websocket.Accept(w, request, nil)
		if err != nil {
			return
		}
		conn.Read(request.Context())
		conn.CloseNow()
	}))
	t.Cleanup(dropping.Close)

	address := "ws" + strings.TrimPrefix(dropping.URL, "http") + "/ws"
	err := Play(context.Background(), Table{Server: address, Name: "tester1"}, nil, quietProgram()...)
	opening := "a conexão com o servidor caiu: "
	if err == nil || !strings.HasPrefix(err.Error(), opening) || len(err.Error()) == len(opening) {
		t.Errorf("Play ended with %v, expected %q followed by the cause", err, opening)
	}
}
