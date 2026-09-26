# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

Open source implementation of the board game **Coup**, playable in the browser and in the
terminal. Authoritative Go server, pure rules engine, typed event protocol.
One binary with the site inside it.

## Language split — the rule that trips everyone

**Code is English. Player-facing strings are pt-BR. Comments do not exist.**

- Identifiers, file names, packages, test names, JSON keys, refusal codes, CLI flags: English
  (`turn_of`, `my_cards`, `not_your_turn`, `income`, `coup`, `duke`, `contessa`).
- Anything a human reads on screen: pt-BR. Event text (`"tester1 pagou 7 e deu um Golpe de
  Estado em tester2."`), refusal messages (`"não é a vez de quem jogou"`), button labels,
  the join form.
- Card and action names cross the wire in English and become pt-BR at the edge:
  `Character.LabelPtBR()` in the engine's narration, `actionLabel()`/`cardLabel()` in
  `web/src/coup.ts` for the UI.
- **No comments anywhere**, including doc comments. If a fact needs explaining, it goes in a
  name, or in prose in `README.md` (see the `.gitkeep` invariant below).

The user's global rules (`~/.claude/CLAUDE.md`) also apply: declarative names everywhere, early
returns, explicit types, no `any`, refusals carry received *and* expected values.

## Commands

```sh
# dev — TWO processes. Vite proxies /ws to the Go server (web/vite.config.ts).
cd web && pnpm install && pnpm dev    # :5173, hot reload
go run ./cmd/coup serve               # :8080

# release — ONE process. pnpm build MUST come first; go build embeds web/dist.
make build && ./coup serve
make release                           # static binaries for linux and macos in release/
./coup join -name tester1              # terminal client; no code opens a new table

# verify, one at a time, never in parallel
make verify                           # go test -race, go vet, gofmt, pnpm build, pnpm lint

# a single test, or one group
go test ./internal/engine -run TestCoupChargesSevenOnDeclaration -v
go test ./internal/engine -run 'Coins|Coin' -v
```

`serve` flags: `-port` (8080) and `-starting-coins`. `-starting-coins 0` follows the rulebook
(2 coins, or 1 in a two-player game) and is the default; **`-starting-coins 14` is how you
actually test a full game by hand** — with rulebook coins and only Income available, reaching
the second Coup takes ~29 turns of clicking.

## Architecture

### The actor: one owner goroutine per room, zero mutexes

`internal/server/room.go` is the whole concurrency design. A `Room` owns its `*engine.Game`;
nothing else can reach it. Every input — a click, a disconnect, a deadline firing — arrives as a
`command` on `Room.inbox`, and `run()` pulls one at a time. The race
"someone answers at the instant the clock expires" does not get resolved, it **cannot exist**.

Two consequences you must preserve:

- **The owner goroutine never writes to the network.** `send()` drops the message into a
  per-connection `outbox` (capacity 16) with `select`/`default`; a full queue means a dead
  client, so it gets dropped. There is no other place to send from, which is the point — a
  mutex design would need "no network write while holding the lock" repeated in every new
  function, and `localhost` never fails that way in tests.
- **A room has 2 goroutines per player plus its own**: a reader (socket → inbox) and a writer
  (outbox → socket). Blocking reads are normal Go; the runtime parks them.

Rooms live in the `registry` (`map[code]*Room`, the only mutex in the server). A connection with
no seat may only send `join` or `reconnect`; once seated, those are refused — that is what stops
a client from renaming itself or trying someone else's token. Timers (`clock.go`) are commands
too, carrying the ID of what armed them, and a stale one is dropped.

### Hidden information is enforced by the compiler

`engine.Game` has only lowercase fields. `encoding/json` serializes exported fields only, so
`json.Marshal(game)` returns `{}` — leaking a hand by accident is impossible, even if the code
is wrong. What leaves the server is `engine.ViewFor(game, name)`: a per-recipient snapshot
where `MyCards` is populated for **one** entry. `internal/protocol` wraps that in an envelope
and never reshapes it.

`View` lives in the engine, not in `protocol`, because hiding cards is a rule of Coup rather
than a transport detail. Putting it in `protocol` produces an import cycle, which is why
`docs/plan/major/03-protocolo.md` now says so explicitly — it used to say the opposite.

### Snapshot, not deltas

Every server message carries the full state plus the events that caused it, together. ~700
bytes for a six-player table; a whole game is ~84 KB. Reconnection (0.8) is "send the snapshot
again" — no replay, no client-side reassembly that would have to be written twice, in Go and
in TypeScript.

### The client knows no rules

The server sends `your_actions` (what is legal right now, targets already filtered),
`your_options` for the open window, and `your_returns` during an exchange. React and the future Bubble Tea client are pure
renderers: draw buttons from a list, send back what was clicked. `web/src/App.tsx` has no game
rule in it, and must not gain one.

### The engine has no clock

`Apply` is pure given a `*rand.Rand`. Deadlines belong to the server: a `time.AfterFunc`
publishes an `expiry` into the same inbox carrying `game.Decision()`; a stale one is recognized
by ID and thrown away, and a live one plays `game.SafeMove(name)` for everyone awaited. This is what lets the engine suite run
with no fake clock, no `Sleep`, and no flakes.

RNG is injected, not mocked: tests use `rand.New(rand.NewPCG(1, 2))`, production seeds from
`crypto/rand`. For readable hands, tests use the package-internal `newGameWithDeck`, so
"tester2 holds Duke and Contessa" is visible in the test rather than hidden behind a seed.

### Rules live in a table

`internal/engine/rules.go` holds one row per action (`Cost`, `Claims`, `NeedsTarget`,
`BlockedBy`, `ValidTarget`, `Declaration`, `Effect`). Two facts are **derived, never stored**: challengeable = "claims a
character", and who may block = "has a target → only the target; no target → anyone". The
money asymmetry of Coup (a successful challenge refunds the cost, a successful block does not)
must stay in that one place.

### The load-bearing empty files

`go:embed` is a **compile error** when its pattern matches nothing, so `web/dist/` can never be
empty. Two empty files guarantee that: `web/dist/.gitkeep` is committed so a fresh clone builds
before anyone runs `pnpm build`, and `web/public/.gitkeep` is copied into `dist` by every
`pnpm build`, replacing the first one, which Vite deletes when it empties the folder. Deleting
either breaks `go build ./...`. Also: the `//go:embed` directive cannot use `..`, which is why
it lives in `web/embed.go` rather than in `server` or `cmd`.

## Working method in this repo

`docs/plan/major/` is the source of truth for scope and sequencing — read `README.md` there
first, then `05-fases.md`. Phases 0.1 through 0.9 are all done, each with its "pronto quando"
and the decisions taken while building it; the core is closed and anything new is outside it.
The decisions in that README were each stress-tested before being written down: **do not
reopen one without a new fact.** Two of them deliberately contradict the brief and are marked
as such.

**Every package and top-level folder carries its own `README.md`** — `cmd/coup`,
`internal/engine`, `internal/protocol`, `internal/server`, `internal/tui`, `web`. That is where a fact about a
folder goes, since comments do not exist here: what lives inside, which invariant must not
break, what does not belong. Changed a package's shape? Its README changes in the same commit.

`01-regras.md` is the rulebook distilled: 11 traps a from-memory engine gets silently wrong,
the full 8-branch decision tree of one action, 3 deliberate divergences, and one invariant the
rulebook never states ("one reaction per player per action" — without it, assassination never
kills anyone). Do not re-read the PDF or re-derive the rules.

**The plan documents are pt-BR prose, and their identifiers were reconciled with the code after
0.1 shipped.** Names that exist today are the code's; names for phases 0.2–0.9 are projections
of the same language rule and will move when the code lands. Each doc marks which phase a name
belongs to. Trust the code for names, the docs for intent — and when a phase closes, fix the
doc in the same breath rather than letting the gap reopen.

Before writing any test: list the test points in the six-field format the user's global rules
require (Situação / Afirma / Antes→depois / Vermelho / Onde roda / Cobre) **and wait for
confirmation**. After writing them, run `/prove` — it reverts one production fix at a time and
shows which points fall, so a decorative test cannot reach a commit. A fix that drops no test
is a finding, not a pass.

Skills that fit this repo: `block` (runs a phase end to end), `tdd`, `prove`, `clean-code-guard`
after non-trivial production code, `grilling` only for a genuinely new decision. `ponytail` is
active by hook — the core must not inflate.

## Legal

Game rules are not copyrightable, so a clean implementation is legitimate. **Box art never
enters this repository.** The README credits Rikki Tahta, La Mame Games, Indie Boards & Cards,
and Mandala Jogos. Code is MIT.
