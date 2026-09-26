package main

import (
	"context"
	crand "crypto/rand"
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"log"
	"math/rand/v2"
	"net/http"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gabrielmgaa/coup/internal/engine"
	"github.com/gabrielmgaa/coup/internal/server"
	"github.com/gabrielmgaa/coup/internal/tui"
	"github.com/gabrielmgaa/coup/web"
)

const defaultServer = "ws://localhost:8080/ws"

var errUsage = errors.New("uso: coup serve [-port 8080] | coup join [-server URL] [-name NOME] [-reconnect] [CÓDIGO]")

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return errUsage
	}
	switch args[0] {
	case "serve":
		return serve(args[1:])
	case "join":
		return join(args[1:])
	}
	return errUsage
}

func serve(args []string) error {
	flags := flag.NewFlagSet("serve", flag.ContinueOnError)
	port := flags.Int("port", 8080, "HTTP port")
	coins := flags.Int("starting-coins", engine.RulebookCoins,
		"coins each player starts with; 0 follows the rulebook (2, or 1 in a two-player game)")
	if err := flags.Parse(args); err != nil {
		return err
	}
	rng, err := seededRand()
	if err != nil {
		return err
	}
	address := fmt.Sprintf(":%d", *port)
	log.Printf("coup listening on http://localhost%s", address)
	return http.ListenAndServe(address, server.New(web.Dist(), rng, server.DefaultConfig(*coins)))
}

func join(args []string) error {
	path, err := tui.SessionPath()
	if err != nil {
		return err
	}
	saved, err := tui.LoadSession(path)
	if err != nil {
		return err
	}
	table, err := tableFrom(args, saved)
	if err != nil {
		return err
	}
	if err := (tui.Session{Name: table.Name, Server: table.Server, Room: table.Room, Token: table.Token}).Save(path); err != nil {
		return err
	}
	remember := func(room, token string) error {
		return tui.Session{Name: table.Name, Server: table.Server, Room: room, Token: token}.Save(path)
	}
	return tui.Play(context.Background(), table, remember, tea.WithAltScreen())
}

func tableFrom(args []string, saved tui.Session) (tui.Table, error) {
	flags := flag.NewFlagSet("join", flag.ContinueOnError)
	address := flags.String("server", firstFilled(saved.Server, defaultServer), "websocket address of the server")
	name := flags.String("name", saved.Name, "your name at the table")
	reconnect := flags.Bool("reconnect", false, "take back the seat saved in the session")
	if err := flags.Parse(args); err != nil {
		return tui.Table{}, err
	}
	if *reconnect {
		return rejoinedTable(*address, saved)
	}
	if strings.TrimSpace(*name) == "" {
		return tui.Table{}, errors.New("informe seu nome com -name na primeira vez")
	}
	return tui.Table{Server: *address, Name: *name, Room: strings.ToUpper(flags.Arg(0))}, nil
}

func rejoinedTable(address string, saved tui.Session) (tui.Table, error) {
	if saved.Token == "" {
		return tui.Table{}, errors.New("não há mesa salva para voltar; entre com um código")
	}
	return tui.Table{Server: address, Name: saved.Name, Room: saved.Room, Token: saved.Token}, nil
}

func firstFilled(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func seededRand() (*rand.Rand, error) {
	var bytes [16]byte
	if _, err := crand.Read(bytes[:]); err != nil {
		return nil, fmt.Errorf("sem entropia para embaralhar o baralho: %w", err)
	}
	return rand.New(rand.NewPCG(
		binary.NativeEndian.Uint64(bytes[:8]),
		binary.NativeEndian.Uint64(bytes[8:]),
	)), nil
}
