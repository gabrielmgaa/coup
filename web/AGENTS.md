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

## shadcn/ui

Every interactive or framed element comes from `components/ui/` — `Button`, `Badge`, `Card`,
`Input`, `Label`, `Checkbox`, `Alert`, `Progress`, `Separator`. They are shadcn/ui components
(style `new-york`, Radix primitives from `radix-ui`, icons from `lucide-react`, class merging
from the `cn` package through `lib/utils.ts`), configured by `components.json` and restyled to
Esmalte: this project owns their code, so the look lives in their `cva` variants, not in
overrides at the call site.

- **Reach for a variant before a className.** `Button` has `default` (ink), `outline` (paper
  card), `challenge`, one variant per card (`duke`, `assassin`, `captain`, `ambassador`,
  `contessa`) and `bare` for a card that is itself the button; sizes `default`, `md`, `lg`. `Badge`
  has `default`, `solid`, `ready` and `revealed`. If a new look repeats, it becomes a variant.
- **The theme is shadcn's semantic tokens mapped onto Esmalte** in `app/globals.css`:
  `primary` is ink, `background` is paper, `secondary`/`muted` are the table band,
  `muted-foreground` is the secondary text, `accent` is the gold wash, `destructive` is blood,
  `border`/`input` are the ink outline, `ring` is gold. Use the semantic name when one fits;
  the Esmalte names (`paper`, `table`, `ink`, `gold`, `blood`, the card hues) cover the rest.
- **Adding a component:** `pnpm dlx shadcn@latest add <name>` when the network reaches
  `ui.shadcn.com`; otherwise copy it from `github.com/shadcn-ui/ui`, path
  `apps/v4/registry/new-york-v4/ui/<name>.tsx`, point its `cn` import at `@/lib/utils`, restyle it
  to Esmalte, and keep it free of comments. The playing card is `PlayingCard` in
  `components/cards.tsx` — `Card` is the shadcn panel.

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
- Two utilities complete the components: `eyebrow` (the small uppercase label) and `numeric`
  (mono, tabular). Touch targets stay at least 44 px; phones (below `md`) stack everything in one
  column.

## Commands

```sh
pnpm dev      # :3000; run `go run ./cmd/coup serve` beside it for /ws
pnpm build    # tsc + next build → dist/, must run before `go build`
pnpm lint     # oxlint
```

`make verify` at the root runs these together with the Go suite.
