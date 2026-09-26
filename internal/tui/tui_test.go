package tui

import (
	"context"
	"encoding/json"
	"math/rand/v2"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/coder/websocket"
	"github.com/gabrielmgaa/coup/internal/engine"
	"github.com/gabrielmgaa/coup/internal/protocol"
	"github.com/gabrielmgaa/coup/internal/server"
)

const startingCoinsForThreeCoups = 14

func startServer(t *testing.T) string {
	t.Helper()
	running := httptest.NewServer(server.New(fstest.MapFS{}, rand.New(rand.NewPCG(1, 2)),
		server.Config{InitialCoins: startingCoinsForThreeCoups, Deadline: time.Hour, Grace: time.Hour, IdleTTL: time.Hour, Handshake: time.Hour}))
	t.Cleanup(running.Close)
	return "ws" + strings.TrimPrefix(running.URL, "http") + "/ws"
}

type terminal struct {
	t     *testing.T
	link  *link
	model Model
}

func openTerminal(t *testing.T, address string, first protocol.FromClient) *terminal {
	t.Helper()
	ctx, stop := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(stop)
	opened, err := dialTable(ctx, address)
	if err != nil {
		t.Fatalf("the terminal could not reach the server: %v", err)
	}
	t.Cleanup(opened.close)
	if err := opened.write(first); err != nil {
		t.Fatalf("the first message did not leave: %v", err)
	}
	return &terminal{t: t, link: opened, model: NewModel(opened.outgoing, opened.receive, func(string, string) tea.Cmd { return nil })}
}

func (term *terminal) deliver(msg tea.Msg) tea.Cmd {
	updated, cmd := term.model.Update(msg)
	term.model = updated.(Model)
	return cmd
}

func (term *terminal) until(done func(Model) bool) {
	term.t.Helper()
	for !done(term.model) {
		arrived := term.link.receive()
		if lost, dropped := arrived.(connectionLost); dropped {
			term.t.Fatalf("the terminal lost the connection: %v", lost.err)
		}
		term.deliver(arrived)
	}
}

func (term *terminal) press(key tea.KeyMsg) {
	term.t.Helper()
	cmd := term.deliver(key)
	if cmd == nil {
		return
	}
	if lost, dropped := cmd().(connectionLost); dropped {
		term.t.Fatalf("the key press did not reach the server: %v", lost.err)
	}
}

func (term *terminal) pick(label string) {
	term.t.Helper()
	available := term.model.choices()
	for range available {
		if available[term.model.cursor].label == label {
			term.press(tea.KeyMsg{Type: tea.KeyEnter})
			return
		}
		term.press(tea.KeyMsg{Type: tea.KeyDown})
	}
	term.t.Fatalf("no choice labelled %q; the screen offers %v", label, labelsOf(available))
}

func labelsOf(available []choice) []string {
	labels := make([]string, 0, len(available))
	for _, option := range available {
		labels = append(labels, option.label)
	}
	return labels
}

type browser struct {
	t    *testing.T
	conn *websocket.Conn
}

func openBrowser(t *testing.T, address string, first protocol.FromClient) *browser {
	t.Helper()
	ctx, stop := context.WithTimeout(context.Background(), 2*time.Second)
	defer stop()
	conn, _, err := websocket.Dial(ctx, address, nil)
	if err != nil {
		t.Fatalf("the browser could not reach the server: %v", err)
	}
	t.Cleanup(func() { conn.CloseNow() })
	opened := &browser{t: t, conn: conn}
	opened.send(first)
	return opened
}

func (b *browser) send(message protocol.FromClient) {
	b.t.Helper()
	encoded, _ := json.Marshal(message)
	if err := b.conn.Write(context.Background(), websocket.MessageText, encoded); err != nil {
		b.t.Fatalf("the browser could not write: %v", err)
	}
}

func (b *browser) until(done func(engine.View) bool) engine.View {
	b.t.Helper()
	for {
		ctx, stop := context.WithTimeout(context.Background(), 2*time.Second)
		_, encoded, err := b.conn.Read(ctx)
		stop()
		if err != nil {
			b.t.Fatalf("nothing reached the browser within 2s: %v", err)
		}
		decoded, err := decode(encoded)
		if err != nil {
			b.t.Fatalf("the browser received json it cannot read: %v", err)
		}
		if update, isUpdate := decoded.(updateArrived); isUpdate && done(update.state.View) {
			return update.state.View
		}
	}
}

func TestTerminalAndBrowserPlayTheSameGameToTheEnd(t *testing.T) {
	address := startServer(t)
	cli := openTerminal(t, address, protocol.FromClient{Type: "create_room", Name: "tester1"})
	cli.until(func(m Model) bool { return m.lobby != nil })
	web := openBrowser(t, address, protocol.FromClient{Type: "join", Room: cli.model.lobby.Room, Name: "tester2"})
	cli.until(func(m Model) bool { return len(m.lobby.Players) == 2 })
	if screen := cli.model.View(); !strings.Contains(screen, "tester1") || !strings.Contains(screen, "host · você") {
		t.Errorf("the lobby screen does not show tester1 as the host:\n%s", screen)
	}

	web.send(protocol.FromClient{Type: "ready", Ready: true})
	cli.pick("estou pronto")
	cli.until(func(m Model) bool {
		return len(m.lobby.Players) == 2 && m.lobby.Players[0].Ready && m.lobby.Players[1].Ready
	})
	cli.pick("começar a partida")
	cli.until(func(m Model) bool { return m.game != nil })

	for cli.model.game.Winner == "" {
		played := len(cli.model.log)
		playOneStep(cli, web)
		cli.until(func(m Model) bool { return len(m.log) > played })
	}

	cli.until(func(m Model) bool { return m.game.Winner != "" })
	seenByBrowser := web.until(func(state engine.View) bool { return state.Winner != "" })
	if seenByBrowser.Winner != cli.model.game.Winner {
		t.Errorf("the terminal saw %q win and the browser saw %q", cli.model.game.Winner, seenByBrowser.Winner)
	}
	if screen := cli.model.View(); !strings.Contains(screen, seenByBrowser.Winner+" venceu a partida") {
		t.Errorf("the terminal screen does not announce the winner:\n%s", screen)
	}
}

func playOneStep(cli *terminal, web *browser) {
	state := cli.model.game.View
	switch {
	case state.Losing == "tester1":
		cli.pick("revelar " + entryOf(state, "tester1").MyCards[0].LabelPtBR())
	case state.Losing == "tester2":
		seen := web.until(func(state engine.View) bool { return state.Losing == "tester2" })
		web.send(protocol.FromClient{Type: "lose_influence", Card: entryOf(seen, "tester2").MyCards[0].String()})
	case state.TurnOf == "tester1":
		cli.pick("Golpe de Estado (7) em tester2")
	default:
		web.send(protocol.FromClient{Type: "play", Action: "coup", Target: "tester1"})
	}
}

func entryOf(state engine.View, name string) engine.PlayerView {
	for _, seen := range state.Players {
		if seen.Name == name {
			return seen
		}
	}
	return engine.PlayerView{}
}

func TestATerminalNeverDrawsSomeoneElsesCards(t *testing.T) {
	state := engine.View{You: "tester1", TurnOf: "tester1", Players: []engine.PlayerView{
		{Name: "tester1", Hidden: 2, MyCards: []engine.Character{engine.Duke, engine.Captain}},
		{Name: "tester2", Hidden: 1, Revealed: []engine.Character{engine.Contessa}},
	}}
	screen := Model{game: &protocol.GameState{View: state}}.View()
	for _, own := range []string{"Duque", "Capitão", "Condessa"} {
		if !strings.Contains(screen, own) {
			t.Errorf("the screen is missing %q, which tester1 is allowed to see:\n%s", own, screen)
		}
	}
	if !strings.Contains(screen, "? ") {
		t.Errorf("tester2's hidden card is not drawn face down:\n%s", screen)
	}
}

func TestOnlyTheHostIsOfferedTheStartButton(t *testing.T) {
	lobby := protocol.LobbyView{Room: "K7QM", You: "tester2", Host: "tester1",
		Players: []protocol.SeatView{{Name: "tester1"}, {Name: "tester2", Ready: true}}}
	guest := labelsOf(lobbyChoices(lobby))
	if len(guest) != 1 || guest[0] != "ainda não estou pronto" {
		t.Errorf("tester2 is offered %v, expected only to take back the ready mark", guest)
	}
	lobby.You = "tester1"
	host := labelsOf(lobbyChoices(lobby))
	if len(host) != 2 || host[0] != "estou pronto" || host[1] != "começar a partida" {
		t.Errorf("the host is offered %v, expected ready and start", host)
	}
}

func TestTheCursorStaysInsideTheList(t *testing.T) {
	sent := []protocol.FromClient{}
	model := NewModel(func(message protocol.FromClient) tea.Cmd {
		sent = append(sent, message)
		return nil
	}, nil, nil)
	state := engine.View{You: "tester1", TurnOf: "tester1", YourActions: []engine.AvailableAction{{Name: "income"}}}
	updated, _ := model.Update(updateArrived{state: protocol.GameState{View: state}})
	for _, key := range []tea.KeyMsg{
		{Type: tea.KeyUp}, {Type: tea.KeyDown}, {Type: tea.KeyDown}, {Type: tea.KeyEnter},
	} {
		updated, _ = updated.Update(key)
	}
	if len(sent) != 1 || sent[0].Action != "income" {
		t.Errorf("the terminal sent %v, expected one income", sent)
	}
}

func TestWithNothingToChooseEnterSendsNothing(t *testing.T) {
	model := NewModel(func(protocol.FromClient) tea.Cmd {
		t.Error("something was sent while there was nothing to choose")
		return nil
	}, nil, nil)
	state := engine.View{You: "tester2", TurnOf: "tester1"}
	updated, _ := model.Update(updateArrived{state: protocol.GameState{View: state}})
	if _, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyEnter}); cmd != nil {
		t.Error("enter produced a command with nothing on screen to choose")
	}
}

func TestARefusalShowsUpAndExplainsTheDisconnect(t *testing.T) {
	model := NewModel(nil, nil, nil)
	updated, _ := model.Update(refusalArrived{message: "não existe sala com esse código"})
	if screen := updated.View(); !strings.Contains(screen, "não existe sala com esse código") {
		t.Errorf("the refusal is not on screen:\n%s", screen)
	}
	updated, cmd := updated.Update(connectionLost{err: context.Canceled})
	if cmd == nil {
		t.Error("losing the connection did not quit the program")
	}
	if lost := updated.(Model).Lost(); lost == nil || lost.Error() != "não existe sala com esse código" {
		t.Errorf("the exit reason is %v, expected the refusal the server sent", lost)
	}
}

func TestQuittingLeavesWithoutAnError(t *testing.T) {
	for _, key := range []tea.KeyMsg{{Type: tea.KeyCtrlC}, {Type: tea.KeyRunes, Runes: []rune("q")}} {
		updated, cmd := NewModel(nil, nil, nil).Update(key)
		if cmd == nil {
			t.Errorf("%q did not quit", key.String())
		}
		if lost := updated.(Model).Lost(); lost != nil {
			t.Errorf("%q quit with error %v", key.String(), lost)
		}
	}
}

func TestPlayReturnsWhenTheServerIsNotThere(t *testing.T) {
	err := Play(context.Background(), Table{Server: "ws://127.0.0.1:1/ws", Name: "tester1"}, nil)
	if err == nil {
		t.Error("Play returned no error for a server that does not exist")
	}
}

func TestPlayRunsUntilTheRoomRefusesTheCode(t *testing.T) {
	address := startServer(t)
	err := Play(context.Background(), Table{Server: address, Name: "tester1", Room: "ZZZZ"}, nil,
		tea.WithInput(strings.NewReader("")), tea.WithOutput(&strings.Builder{}), tea.WithoutRenderer())
	if err == nil || err.Error() != "não existe sala com esse código" {
		t.Errorf("Play ended with %v, expected the room_not_found message", err)
	}
}

func TestSessionSurvivesBetweenRuns(t *testing.T) {
	path := filepath.Join(t.TempDir(), "coup", "session.json")
	empty, err := LoadSession(path)
	if err != nil || empty != (Session{}) {
		t.Fatalf("a missing file loaded as %+v, %v; expected an empty session", empty, err)
	}
	saved := Session{Name: "tester1", Server: "ws://example:8080/ws"}
	if err := saved.Save(path); err != nil {
		t.Fatalf("the session could not be saved: %v", err)
	}
	loaded, err := LoadSession(path)
	if err != nil || loaded != saved {
		t.Errorf("loaded %+v, %v; expected %+v", loaded, err, saved)
	}
	if info, _ := os.Stat(path); info.Mode().Perm() != 0o600 {
		t.Errorf("the session file has mode %v, expected 0600 — it will hold a token", info.Mode().Perm())
	}
}

func TestACorruptSessionIsReported(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.json")
	os.WriteFile(path, []byte("{"), 0o600)
	if _, err := LoadSession(path); err == nil {
		t.Error("a corrupt session file loaded without error")
	}
}

func TestUnreadableJsonFromTheServerIsAnError(t *testing.T) {
	if _, err := decode([]byte("not json")); err == nil {
		t.Error("decode accepted bytes that are not json")
	}
}

func TestAnOpenWindowOffersExactlyTheServerOptions(t *testing.T) {
	state := engine.View{You: "tester2", TurnOf: "tester1", Window: &engine.WindowView{
		ID:          4,
		Action:      engine.ActionView{Name: "tax", By: "tester1", Claims: engine.Duke},
		YourOptions: []engine.Option{{Answer: engine.Challenge}, {Answer: engine.Pass}},
		WaitingOn:   []string{"tester2", "tester3"},
	}}
	offered := gameChoices(state)
	if labels := labelsOf(offered); len(labels) != 2 || labels[0] != "contestar o Duque de tester1" || labels[1] != "deixar passar" {
		t.Errorf("the terminal offers %v, expected challenge and pass", labels)
	}
	challenge := offered[0].message
	if challenge.Type != "respond" || challenge.Window != 4 || challenge.Answer != "challenge" || challenge.Character != "" {
		t.Errorf("challenging sends %+v, expected respond to window 4 with no character", challenge)
	}
	if screen := (Model{game: &protocol.GameState{View: state}}).View(); !strings.Contains(screen, "esperando tester2, tester3") {
		t.Errorf("the screen does not say who the window waits on:\n%s", screen)
	}
}

func TestABlockWindowLabelsTheChallengeAgainstTheBlocker(t *testing.T) {
	state := engine.View{You: "tester1", TurnOf: "tester1", Window: &engine.WindowView{
		ID:          7,
		Action:      engine.ActionView{Name: "foreign_aid", By: "tester1"},
		Block:       &engine.BlockView{By: "tester3", Character: engine.Duke},
		YourOptions: []engine.Option{{Answer: engine.Challenge}, {Answer: engine.Pass}},
		WaitingOn:   []string{"tester1"},
	}}
	if labels := labelsOf(gameChoices(state)); labels[0] != "contestar o Duque de tester3" {
		t.Errorf("the terminal offers %v, expected to challenge tester3's duke", labels)
	}
	if screen := (Model{game: &protocol.GameState{View: state}}).View(); !strings.Contains(screen, "tester3 bloqueou com Duque") {
		t.Errorf("the screen does not show the block:\n%s", screen)
	}
	blocking := responseChoices(engine.WindowView{ID: 8, YourOptions: []engine.Option{{Answer: engine.Block, Character: engine.Duke}}})
	if blocking[0].label != "bloquear com Duque" || blocking[0].message.Character != "duke" {
		t.Errorf("blocking reads %q and sends %+v, expected duke", blocking[0].label, blocking[0].message)
	}
}

func TestAnExchangeOffersEachPairAsOneChoice(t *testing.T) {
	state := engine.View{You: "tester1", TurnOf: "tester1", Phase: "awaiting_exchange",
		YourReturns: [][]engine.Character{{engine.Duke, engine.Captain}, {engine.Duke, engine.Duke}}}
	offered := gameChoices(state)
	if labels := labelsOf(offered); len(labels) != 2 || labels[0] != "devolver Duque e Capitão" {
		t.Errorf("the terminal offers %v, expected one choice per pair", labels)
	}
	if sent := offered[1].message; sent.Type != "return_cards" || len(sent.Cards) != 2 || sent.Cards[0] != "duke" {
		t.Errorf("the second choice sends %+v, expected return_cards [duke duke]", sent)
	}
	if screen := (Model{game: &protocol.GameState{View: state}}).View(); !strings.Contains(screen, "escolha as 2 cartas") {
		t.Errorf("the screen does not ask which cards go back:\n%s", screen)
	}
	state.You, state.YourReturns = "tester2", nil
	if screen := (Model{game: &protocol.GameState{View: state}}).View(); !strings.Contains(screen, "tester1 está escolhendo cartas") {
		t.Errorf("the other seats are not told the exchange is under way:\n%s", screen)
	}
}

func TestTheTerminalRemembersItsSeatAndComesBackToIt(t *testing.T) {
	address := startServer(t)
	cli := openTerminal(t, address, protocol.FromClient{Type: "create_room", Name: "tester1"})
	var room, token string
	cli.model.remember = func(gotRoom, gotToken string) tea.Cmd {
		room, token = gotRoom, gotToken
		return nil
	}
	cli.until(func(m Model) bool { return m.lobby != nil })
	if room == "" || token == "" || room != cli.model.lobby.Room {
		t.Fatalf("the terminal remembered room %q and token %q, expected the lobby's room and a token", room, token)
	}
	web := openBrowser(t, address, protocol.FromClient{Type: "join", Room: room, Name: "tester2"})
	cli.until(func(m Model) bool { return len(m.lobby.Players) == 2 })
	web.send(protocol.FromClient{Type: "ready", Ready: true})
	cli.pick("estou pronto")
	cli.until(func(m Model) bool { return m.lobby.Players[0].Ready && m.lobby.Players[1].Ready })
	cli.pick("começar a partida")
	cli.until(func(m Model) bool { return m.game != nil })
	cli.link.close()

	table := Table{Server: address, Room: room, Token: token}
	back := openTerminal(t, address, table.firstMessage())
	back.until(func(m Model) bool { return m.game != nil })
	if back.model.game.You != "tester1" || len(back.model.game.Disconnected) != 0 {
		t.Errorf("the terminal came back as %q with %v disconnected, expected tester1 and nobody out",
			back.model.game.You, back.model.game.Disconnected)
	}
}

func TestAPausedTableShowsWhoItWaitsFor(t *testing.T) {
	state := protocol.GameState{
		View:         engine.View{You: "tester2", TurnOf: "tester1", Players: []engine.PlayerView{{Name: "tester1"}, {Name: "tester2"}}},
		Paused:       &protocol.PausedView{WaitingFor: []string{"tester1"}, ResumesInMs: 30000},
		Disconnected: []string{"tester1"},
	}
	screen := Model{game: &state}.View()
	for _, expected := range []string{"mesa pausada esperando tester1 voltar (30s)", "tester1 (caiu)"} {
		if !strings.Contains(screen, expected) {
			t.Errorf("the screen is missing %q:\n%s", expected, screen)
		}
	}
}

func TestTheCountdownIsDrawnFromTheSnapshot(t *testing.T) {
	model := NewModel(nil, nil, nil)
	state := protocol.GameState{View: engine.View{You: "tester1", TurnOf: "tester1"}, ClosesInMs: 25000}
	updated, _ := model.Update(updateArrived{state: state})
	if screen := updated.View(); !strings.Contains(screen, "s para decidir") {
		t.Errorf("the screen has no countdown:\n%s", screen)
	}
	if _, cmd := updated.Update(tickArrived{}); cmd == nil {
		t.Error("a tick did not schedule the next one; the countdown would freeze")
	}
}

func TestAFailedSessionSaveIsShownWithoutStoppingTheGame(t *testing.T) {
	failing := rememberCmd(func(string, string) error { return os.ErrPermission })
	msg := failing("K7QM", "token")()
	updated, cmd := NewModel(nil, nil, nil).Update(msg)
	if cmd != nil {
		t.Error("a failed save produced a command; it must not start a second reader")
	}
	if screen := updated.View(); !strings.Contains(screen, "não deu para salvar a sessão") {
		t.Errorf("the failed save is not on screen:\n%s", screen)
	}
	if saved := rememberCmd(func(string, string) error { return nil })("K7QM", "token")(); saved != nil {
		t.Errorf("a successful save produced message %v, expected none", saved)
	}
}

func TestTheLobbyNamesTheLastWinnerAndTheHouseRule(t *testing.T) {
	lobby := protocol.LobbyView{Room: "K7QM", You: "tester1", Host: "tester1", LastWinner: "tester2",
		Options: engine.Options{IndependentReactions: true}, Players: []protocol.SeatView{{Name: "tester1"}}}
	screen := Model{lobby: &lobby}.View()
	for _, expected := range []string{"tester2 venceu a última partida", "reações independentes"} {
		if !strings.Contains(screen, expected) {
			t.Errorf("the lobby screen is missing %q:\n%s", expected, screen)
		}
	}
	opening := Table{Name: "tester1", Options: engine.Options{IndependentReactions: true}}.firstMessage()
	if !opening.Options.IndependentReactions {
		t.Errorf("create_room left without the house rule: %+v", opening)
	}
}

func TestBackInTheLobbyTheOldLogIsCleared(t *testing.T) {
	model := NewModel(nil, nil, nil)
	updated, _ := model.Update(updateArrived{state: protocol.GameState{View: engine.View{You: "tester1"}},
		events: []engine.Event{{N: 1, Text: "tester1 pegou Renda."}}})
	updated, _ = updated.Update(lobbyArrived{state: protocol.LobbyView{Room: "K7QM", You: "tester1"}})
	if final := updated.(Model); final.game != nil || len(final.log) != 0 {
		t.Errorf("back in the lobby the terminal kept game %v and %d log lines", final.game, len(final.log))
	}
}

func TestTheFirstMessageDependsOnWhatTheTableKnows(t *testing.T) {
	for _, scenario := range []struct {
		table    Table
		expected string
	}{
		{Table{Name: "tester1"}, "create_room"},
		{Table{Name: "tester1", Room: "K7QM"}, "join"},
		{Table{Room: "K7QM", Token: "secret"}, "reconnect"},
	} {
		if first := scenario.table.firstMessage(); first.Type != scenario.expected {
			t.Errorf("%+v opens with %q, expected %q", scenario.table, first.Type, scenario.expected)
		}
	}
}

func TestTheSessionLivesUnderTheUserConfigFolder(t *testing.T) {
	folder := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", folder)
	t.Setenv("HOME", folder)
	configFolder, _ := os.UserConfigDir()
	path, err := SessionPath()
	if err != nil || path != filepath.Join(configFolder, "coup", "session.json") || !strings.HasPrefix(path, folder) {
		t.Errorf("the session path is %q, %v; expected coup/session.json in the config folder under %s", path, err, folder)
	}
}

func TestASessionThatCannotBeWrittenIsAnError(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "file")
	os.WriteFile(blocker, nil, 0o600)
	if err := (Session{Name: "tester1"}).Save(filepath.Join(blocker, "coup", "session.json")); err == nil {
		t.Error("saving under a regular file succeeded")
	}
}

func TestEveryStatusLineMatchesWhatTheGameWaitsFor(t *testing.T) {
	for _, scenario := range []struct {
		state    engine.View
		expected string
	}{
		{engine.View{You: "tester1", Winner: "tester2"}, "tester2 venceu a partida"},
		{engine.View{You: "tester1", Losing: "tester1"}, "qual carta revela"},
		{engine.View{You: "tester1", Losing: "tester2"}, "tester2 está escolhendo qual carta perder"},
		{engine.View{You: "tester1", TurnOf: "tester1"}, "sua vez"},
		{engine.View{You: "tester1", TurnOf: "tester2"}, "é a vez de tester2"},
	} {
		if status := renderStatus(scenario.state); !strings.Contains(status, scenario.expected) {
			t.Errorf("the status reads %q, expected %q", status, scenario.expected)
		}
	}
	window := renderWindow(engine.WindowView{Action: engine.ActionView{Name: "steal", By: "tester1", Target: "tester2"}, WaitingOn: []string{"tester2"}})
	if !strings.Contains(window, "tester1 declarou Extorquir em tester2") {
		t.Errorf("the window line reads %q", window)
	}
	if label := actionLabel("mystery"); label != "mystery" {
		t.Errorf("an unknown action is labelled %q, expected its own name", label)
	}
}

func TestLosingTheConnectionWithoutARefusalKeepsTheSocketError(t *testing.T) {
	updated, _ := NewModel(nil, nil, nil).Update(connectionLost{err: os.ErrClosed})
	if lost := updated.(Model).Lost(); lost != os.ErrClosed {
		t.Errorf("the exit reason is %v, expected the socket error", lost)
	}
}
