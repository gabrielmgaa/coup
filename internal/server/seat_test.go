package server

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/coder/websocket"
)

func TestTheTabThatLosesItsSeatIsToldWhy(t *testing.T) {
	url := startServer(t)
	playing := seatTable(t, url, "tester1", "tester2")
	old := playing.seats[playing.onTurn()]

	rejoin(t, url, old.room, old.token)

	told := old.waitFor("error")
	if told.Code != "seat_taken" || told.Message != "você abriu esta mesa em outro lugar" {
		t.Errorf("the old tab was told %q %q, expected seat_taken \"você abriu esta mesa em outro lugar\"", told.Code, told.Message)
	}
}

func TestANameThatDiffersOnlyInCaseIsTaken(t *testing.T) {
	url := startServer(t)
	host := createTable(t, url, "tester1")

	impostor := enterTable(t, url, host.room, "TESTER1")
	if refused := impostor.receive(); refused.Code != "name_taken" {
		t.Errorf("joining as TESTER1 next to tester1 answered %q %q, expected name_taken", refused.Type, refused.Code)
	}
	enterTable(t, url, host.room, "tester2")
	if seated := host.waitForSeats(2).names(); len(seated) != 2 || seated[1] != "tester2" {
		t.Errorf("the lobby holds %v, expected tester1 and tester2 — TESTER1 must not have sat", seated)
	}
}

func TestARefusedJoinClosesWithTheRefusalAsTheReason(t *testing.T) {
	url := startServer(t)
	host := createTable(t, url, "tester1")

	twin := enterTable(t, url, host.room, "tester1")
	twin.waitFor("error")
	ctx, stop := context.WithTimeout(context.Background(), 2*time.Second)
	defer stop()
	_, _, err := twin.conn.Read(ctx)
	var closed websocket.CloseError
	if !errors.As(err, &closed) || closed.Reason != "name_taken" {
		t.Errorf("the refused socket closed with %v, expected the reason name_taken", err)
	}
}
