package main

import (
	"errors"
	"net"
	"strconv"
	"testing"

	"github.com/gabrielmgaa/coup/internal/tui"
)

func TestWithoutACommandTheUsageIsShown(t *testing.T) {
	for _, args := range [][]string{{}, {"dance"}} {
		if err := run(args); !errors.Is(err, errUsage) {
			t.Errorf("run(%v) returned %v, expected the usage", args, err)
		}
	}
}

func TestABadServeFlagIsReportedInsteadOfListening(t *testing.T) {
	if err := run([]string{"serve", "-port", "not-a-number"}); err == nil {
		t.Error("serve accepted a port that is not a number")
	}
}

func TestJoinReadsTheCodeAndFallsBackToTheSavedSession(t *testing.T) {
	saved := tui.Session{Name: "tester1", Server: "ws://example:9/ws"}
	table, err := tableFrom([]string{"k7qm"}, saved)
	if err != nil {
		t.Fatalf("join was refused: %v", err)
	}
	expected := tui.Table{Server: "ws://example:9/ws", Name: "tester1", Room: "K7QM"}
	if table != expected {
		t.Errorf("join built %+v, expected %+v", table, expected)
	}
	fresh, err := tableFrom([]string{"-name", "tester2"}, tui.Session{})
	if err != nil || fresh.Server != defaultServer || fresh.Room != "" {
		t.Errorf("join with no session built %+v, %v; expected the default server and a new table", fresh, err)
	}
}

func TestTheHouseRuleFlagTravelsInTheTable(t *testing.T) {
	table, err := tableFrom([]string{"-name", "tester1", "-independent-reactions"}, tui.Session{})
	if err != nil || !table.Options.IndependentReactions {
		t.Errorf("join built %+v, %v; expected independent reactions on", table, err)
	}
}

func TestJoinWithoutAnyNameIsRefused(t *testing.T) {
	if _, err := tableFrom([]string{"K7QM"}, tui.Session{}); err == nil {
		t.Error("join without a name was accepted")
	}
	if _, err := tableFrom([]string{"-bogus"}, tui.Session{}); err == nil {
		t.Error("join accepted an unknown flag")
	}
}

func TestReconnectUsesTheSavedSeat(t *testing.T) {
	saved := tui.Session{Name: "tester1", Server: "ws://example:9/ws", Room: "K7QM", Token: "secret"}
	table, err := tableFrom([]string{"-reconnect"}, saved)
	if err != nil {
		t.Fatalf("reconnect was refused: %v", err)
	}
	if table.Token != "secret" || table.Room != "K7QM" {
		t.Errorf("reconnect built %+v, expected the saved room and token", table)
	}
	if _, err := tableFrom([]string{"-reconnect"}, tui.Session{Name: "tester1"}); err == nil {
		t.Error("reconnect without a saved seat was accepted")
	}
}

func TestJoinFailsCleanlyWhenTheServerIsNotThere(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	err := run([]string{"join", "-server", "ws://127.0.0.1:1/ws", "-name", "tester1"})
	if err == nil {
		t.Error("join returned no error for a server that does not exist")
	}
}

func TestTheSeedComesFromTheSystem(t *testing.T) {
	first, err := seededRand()
	if err != nil {
		t.Fatalf("no entropy: %v", err)
	}
	second, _ := seededRand()
	if first.Uint64() == second.Uint64() {
		t.Error("two seeds drew the same first number")
	}
}

func TestServeReportsAPortThatIsAlreadyTaken(t *testing.T) {
	taken, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("could not hold a port: %v", err)
	}
	defer taken.Close()
	port := taken.Addr().(*net.TCPAddr).Port
	if err := run([]string{"serve", "-port", strconv.Itoa(port)}); err == nil {
		t.Errorf("serve returned no error on port %d, which is already taken", port)
	}
}
