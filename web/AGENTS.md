<!-- BEGIN:nextjs-agent-rules -->

# This is NOT the Next.js you know

This version has breaking changes — APIs, conventions, and file structure may all differ from your training data. Read the relevant guide in `node_modules/next/dist/docs/` (resolved from this file's directory; in monorepos the `next` package may not be visible from the repo root) before writing any code. Heed deprecation notices.

This block is written and re-added by `next dev` — verify at `node_modules/next/dist/server/lib/generate-agent-files.js`. Removing it from a diff only re-creates the uncommitted change; committing it with your work keeps the tree clean.

<!-- END:nextjs-agent-rules -->

# The web client of Coup

The repository root `CLAUDE.md` holds the project rules; everything there applies here. This
file adds what is specific to `web/`.

## What this folder is

Next.js (App Router) + Tailwind v4 + TypeScript, built as a **static export**
(`output: 'export'`, `distDir: 'dist'`). The Go server embeds `dist/` into the single `coup`
binary and serves it; the page talks to that server over one WebSocket at `/ws`.

- **No server-side Next.** No SSR, no route handlers, no server actions, no middleware, no
  `next/image` optimization, no rewrites in production. Anything that needs a Node server breaks
  the single binary. The only rewrite (`/ws` → `:8080`) exists in `next dev`.
- **One page, client only.** `app/page.tsx` loads `components/Coup` with `next/dynamic` and
  `ssr: false`, because the page reads `localStorage` and `location` on first render.
- **Fonts come from `@fontsource`**, bundled in the build, so a LAN table works with no internet.
  Never `next/font/google` and never a Google Fonts `<link>`.
- **`dist/.gitkeep` and `public/.gitkeep` must survive** — `go:embed` fails to compile on an empty
  `dist/`. See the root `CLAUDE.md`.

## The client knows no rules

Buttons come from `your_actions`, `window.your_options` and `your_returns`, exactly as the server
sends them. Never filter, derive or validate a move here: if a rule is missing on screen, the fix
belongs in `internal/engine`, which feeds both this page and the terminal client. The card
captions in `components/cards.tsx` are printed card text, not rules.

## Language and style

- Identifiers, file names and JSON keys in English; every string a player reads in pt-BR.
- No comments, including JSX comments and doc comments.
- Explicit types, no `any`, early returns, declarative names.
- Card and action names cross the wire in English and become pt-BR only through `actionLabel`,
  `cardLabel` and `optionLabel` in `lib/coup.ts`.

## The Esmalte design

The visual direction comes from the design canvas "Coup — Direção Visual" (direction
"Esmalte"). Its tokens are the `@theme` block in `app/globals.css`; use them, never raw hex in a
component.

- Paper `bg-paper` for the page, `bg-card` for anything lifted, `bg-table` for bands and the
  arena; 3 px `border-ink` outlines; hard shadows `shadow-lift`, `shadow-lift-lg`, `shadow-gold`,
  `shadow-blood` — never a blurred shadow, never a gradient wash.
- One hue per card, always with its glyph and its name — color is never the only channel.
  Tailwind only generates class names it can read in full, so a card's color comes from the maps
  in `lib/palette.ts` (`tint`, `inkOf`), never from a template like `` `bg-${card}` ``.
- Bricolage Grotesque for everything; JetBrains Mono (`numeric`) only for coins, codes and
  clocks. Uppercase only in `label` and `tag`.
- Reuse the utilities `btn`, `btn-primary`, `tag`, `label`, `panel`, `numeric` before adding new
  ones. Touch targets stay at least 44 px; phones (below `md`) stack everything in one column.

## Commands

```sh
pnpm dev      # :3000; run `go run ./cmd/coup serve` beside it for /ws
pnpm build    # tsc + next build → dist/, must run before `go build`
pnpm lint     # oxlint
```

`make verify` at the root runs these together with the Go suite.
