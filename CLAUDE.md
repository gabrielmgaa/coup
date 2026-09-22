# CLAUDE.md

Open source implementation of the board game **Coup**, playable in the browser and in the
terminal. Authoritative Go server, pure rules engine, typed event protocol, one binary with the
site inside it.

**Where to read before writing:** `docs/plan/major/README.md` for scope and decisions, then
`05-fases.md`. Each package explains itself in its own `README.md` — `cmd/coup`,
`internal/engine`, `internal/protocol`, `internal/server`, `internal/tui`, `web`. Open the one you
are about to touch. A fact about a folder belongs there, not here.

## Language split — the rule that trips everyone

**Code is English. Player-facing strings are pt-BR. Comments do not exist.**

- English: identifiers, file names, packages, test names, JSON keys, refusal codes, CLI flags —
  `turn_of`, `my_cards`, `not_your_turn`, `income`, `coup`, `duke`, `contessa`.
- pt-BR: anything a human reads on screen — event text, refusal messages, button labels, forms.
- Card and action names cross the wire in English and become pt-BR at the edge:
  `Character.LabelPtBR()` in the engine, `actionLabel()`/`cardLabel()` in `web/lib/coup.ts`,
  `actionLabel()` in `internal/tui/choices.go`.
- No comments anywhere, including doc comments. A fact that needs explaining goes in a name, or
  in prose in that package's `README.md`.
- Test actors are `tester1`, `tester2`, … — never a person's name.

Also applies, from `~/.claude/CLAUDE.md`: declarative names, early returns, explicit types, no
`any`, and refusals carrying received *and* expected values.

## Commands

```sh
# dev — TWO processes. next dev rewrites /ws to the Go server (web/next.config.ts, dev only).
cd web && pnpm install && pnpm dev    # :3000, hot reload
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

`serve` flags: `-port` (8080) and `-starting-coins`. The default `0` follows the rulebook (2
coins, 1 in a two-player game); **`-starting-coins 14` is how you test a full game by hand** —
with rulebook coins and only Income available, the second Coup is ~29 turns of clicking away.

## Invariants — breaking one of these is a bug, not a style choice

- **`engine.Game` has only lowercase fields**, so `json.Marshal(game)` returns `{}`. Exporting
  one makes leaking a hand possible.
- **`engine.ViewFor` produces everything that leaves the server.** `internal/protocol` wraps it
  in an envelope and never reshapes it.
- **`View` lives in `internal/engine`, not in `protocol`.** `ViewFor` reads private fields of
  `Game`, so the other way around is an import cycle.
- **A `Room` owns its game, and every input is a command on `Room.inbox`** — a click, a
  disconnect, a deadline firing — pulled one at a time. The owner goroutine never writes to the
  network: it queues into a per-connection `outbox` with `select`/`default`; a full queue means a
  dead client, so the client is dropped. The `registry` is the only mutex in the server.
- **The client knows no rules.** The server sends `your_actions` (targets already filtered),
  `your_options` for the open window and `your_returns` during an exchange. Nothing under `web/`
  or `internal/tui/` has a game rule in it, and nothing there may gain one.
- **The engine has no clock.** `Apply` is pure given a `*rand.Rand`. Deadlines belong to the
  server: a timer publishes an `expiry` into the inbox carrying `game.Decision()`; a stale one is
  dropped by ID, a live one plays `game.SafeMove(name)` for everyone awaited.
- **Two facts in `internal/engine/rules.go` are derived, never stored**: challengeable = "claims
  a character"; who may block = "has a target → only the target; no target → anyone". The money
  asymmetry (a won challenge refunds the cost, a won block does not) stays in that one place.
- **`engine.MaxPlayers` is the only source of the six-player ceiling**, read by the engine and
  by the room. The deck holds 15 cards and each player takes 2.
- **`web/` is a Next.js static export** (`output: 'export'`, `distDir: 'dist'`): no SSR, no
  server routes, no rewrites in production. This Next may differ from what you remember — read
  `web/AGENTS.md` and `node_modules/next/dist/docs/` before touching Next APIs.
- **`web/dist/.gitkeep` and `web/public/.gitkeep` are load-bearing.** Deleting either breaks
  `go build ./...`; `web/README.md` explains the mechanism.
- **RNG is injected, never mocked.** Tests pass `rand.New(rand.NewPCG(1, 2))`; production seeds
  from `crypto/rand`. For a readable hand, tests use the package-internal `newGameWithDeck`.

## Working method

- Phases 0.1 through 0.9 are done and the core is closed; anything new is outside it.
- The decisions in `docs/plan/major/README.md` were each stress-tested before being written
  down. **Do not reopen one without a new fact.**
- `01-regras.md` is the rulebook distilled — 11 traps, the 8-branch decision tree, 3 deliberate
  divergences. Do not re-read the PDF or re-derive the rules.
- Before writing any test, list the test points in the six-field format the user's global rules
  require (Situação / Afirma / Antes→depois / Vermelho / Onde roda / Cobre) and **wait for
  confirmation**.
- After writing them, run `/prove`: revert one production fix at a time and see which points
  fall. A fix that drops no test is a finding, not a pass.
- Changed a package's shape? Its `README.md` changes in the same commit. Closed a phase? Fix
  `05-fases.md` in the same breath, so the plan never drifts again.
- Conventional Commits in English, one-line subject, no body, no co-author. Commit only when
  asked, never `--no-verify`.
- Plan docs are pt-BR prose. When a doc and the code disagree on a name, the code wins; trust
  the docs for intent.

Skills that fit this repo: `block` (runs a phase end to end), `tdd`, `prove`, `clean-code-guard`
after non-trivial production code, `grilling` only for a genuinely new decision. `ponytail` is
active by hook — the core must not inflate.

## Legal

Game rules are not copyrightable, so a clean implementation is legitimate. **Box art never
enters this repository.** The root `README.md` credits Rikki Tahta, La Mame Games, Indie Boards
& Cards and Mandala Jogos. Code is MIT.
