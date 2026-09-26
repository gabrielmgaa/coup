package server

import (
	"time"

	"github.com/gabrielmgaa/coup/internal/engine"
)

type expiryKind uint8

const (
	deadlineExpired expiryKind = iota
	graceExpired
	idleExpired
)

type expiry struct {
	kind expiryKind
	id   int
}

const noDecision = -1

type clock struct {
	deadline     *time.Timer
	deadlineFor  int
	deadlineAt   time.Time
	deadlineLeft time.Duration
	grace        *time.Timer
	graceFor     int
	graceLeft    time.Duration
	pauseID      int
	paused       []string
	resumeAt     time.Time
	idle         *time.Timer
	idleID       int
}

func newClock() clock { return clock{deadlineFor: noDecision, graceFor: noDecision} }

func fire(deliver func(command) bool, fired expiry) func() {
	return func() { deliver(command{expired: &fired}) }
}

func (c *clock) armDeadline(decision int, full time.Duration, deliver func(command) bool) {
	if c.deadline != nil && c.deadlineFor == decision {
		return
	}
	c.stopDeadline()
	if c.deadlineFor != decision {
		c.deadlineFor, c.deadlineLeft = decision, full
	}
	c.deadlineAt = time.Now().Add(c.deadlineLeft)
	c.deadline = time.AfterFunc(c.deadlineLeft, fire(deliver, expiry{kind: deadlineExpired, id: decision}))
}

func (c *clock) stopDeadline() {
	if c.deadline == nil {
		return
	}
	c.deadline.Stop()
	c.deadline = nil
	c.deadlineLeft = max(0, time.Until(c.deadlineAt))
}

func (c *clock) pause(decision int, waiting []string, full time.Duration, deliver func(command) bool) {
	c.stopDeadline()
	if c.paused == nil {
		if c.graceFor != decision {
			c.graceFor, c.graceLeft = decision, full
		}
		c.pauseID++
		c.resumeAt = time.Now().Add(c.graceLeft)
		c.grace = time.AfterFunc(c.graceLeft, fire(deliver, expiry{kind: graceExpired, id: c.pauseID}))
	}
	c.paused = waiting
}

func (c *clock) resume() {
	if c.paused == nil {
		return
	}
	c.grace.Stop()
	c.graceLeft = max(0, time.Until(c.resumeAt))
	c.paused = nil
}

func (c *clock) deadlineFired() {
	c.stopDeadline()
	c.deadlineFor = noDecision
}

func (c *clock) endGame() {
	c.stopDeadline()
	c.resume()
	c.deadlineFor, c.graceFor = noDecision, noDecision
}

func (c *clock) startIdle(ttl time.Duration, deliver func(command) bool) {
	if c.idle != nil {
		return
	}
	c.idleID++
	c.idle = time.AfterFunc(ttl, fire(deliver, expiry{kind: idleExpired, id: c.idleID}))
}

func (c *clock) stopIdle() {
	if c.idle != nil {
		c.idle.Stop()
		c.idle = nil
	}
}

func (c *clock) stopAll() {
	c.stopDeadline()
	c.resume()
	c.stopIdle()
}

func (c *clock) closesInMs() int64 {
	if c.deadline == nil || c.paused != nil {
		return 0
	}
	return max(1, time.Until(c.deadlineAt).Milliseconds())
}

func (c *clock) resumesInMs() int64 {
	return max(1, time.Until(c.resumeAt).Milliseconds())
}

func (r *Room) settleClock() {
	if r.game.Winner() != "" {
		r.clock.endGame()
		return
	}
	if waiting := r.stalled(); len(waiting) > 0 {
		r.clock.pause(r.game.Decision(), waiting, r.config.Grace, r.deliver)
		return
	}
	r.clock.resume()
	r.clock.armDeadline(r.game.Decision(), r.config.Deadline, r.deliver)
}

func (r *Room) stalled() []string {
	waiting := []string{}
	for _, name := range r.game.Awaiting() {
		if awaited := r.seatNamed(name); !awaited.connected() && !awaited.autopilot {
			waiting = append(waiting, name)
		}
	}
	return waiting
}

func (r *Room) expire(fired expiry) {
	switch fired.kind {
	case deadlineExpired:
		r.deadlinePassed(fired.id)
	case graceExpired:
		r.gracePassed(fired.id)
	case idleExpired:
		r.idlePassed(fired.id)
	}
}

func (r *Room) deadlinePassed(decision int) {
	if r.clock.deadline == nil || r.clock.deadlineFor != decision {
		return
	}
	r.clock.deadlineFired()
	var events []engine.Event
	for _, name := range r.game.Awaiting() {
		move, _ := r.game.SafeMove(name)
		applied, err := r.game.Apply(move)
		if err == nil {
			events = append(events, applied...)
		}
	}
	r.afterChange(events)
}

func (r *Room) gracePassed(pauseID int) {
	if r.clock.paused == nil || r.clock.pauseID != pauseID {
		return
	}
	for _, name := range r.clock.paused {
		r.seatNamed(name).autopilot = true
	}
	r.afterChange(nil)
}

func (r *Room) idlePassed(idleID int) {
	if r.clock.idle == nil || r.clock.idleID != idleID || r.anyoneConnected() {
		return
	}
	r.close()
}
