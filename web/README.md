# `web` — a mesa no navegador

Next.js (App Router) + Tailwind v4 + shadcn/ui + TypeScript, como **export estático**. Desenha a mesa a
partir do snapshot que chega pelo WebSocket e manda de volta o que foi clicado. **Não tem regra
de Coup aqui dentro, e não pode ganhar nenhuma.**

## Arquivos

| | |
|---|---|
| `app/layout.tsx` | o HTML de fora: `lang`, título, ícone, cor do tema |
| `app/page.tsx` | a única página; carrega `components/Coup` só no navegador (`ssr: false`) |
| `app/globals.css` | Tailwind, a base do shadcn, e o tema: os nomes semânticos do shadcn (`primary`, `muted-foreground`, `border`, `ring`…) apontando para as cores Esmalte, as sombras sólidas e os utilitários `eyebrow` e `numeric` |
| `components.json` | a configuração do shadcn/ui: estilo `new-york`, aliases `@/components/ui` e `@/lib/utils`, ícones `lucide` |
| `components/ui/` | os componentes shadcn — `button`, `badge`, `card`, `input`, `label`, `checkbox`, `alert`, `progress`, `separator` — com as variantes reescritas na direção Esmalte |
| `lib/utils.ts` | `cn`, do pacote `cn`, para juntar classes |
| `lib/coup.ts` | os tipos do fio (`GameState`, `WindowView`, `FromServer`…), os rótulos pt-BR (`actionLabel`, `cardLabel`, `cardLabelWithArticle`, `optionLabel`) e a sessão no `localStorage` |
| `lib/connection.ts` | `useTable`: WebSocket, `welcome` salvo, reconexão automática (que para em `seat_taken`), volta à entrada quando a sessão se perde, a última partida para a tela de fim; `useSecondsLeft` para os relógios |
| `lib/palette.ts` | a cor de cada carta, como classe Tailwind escrita por extenso |
| `components/cards.tsx` | `PlayingCard` (glifo, legenda, nome), o verso, o chip de carta revelada, as moedas e a narração que pinta o nome da carta na cor dela |
| `components/Coup.tsx` | escolhe a tela: entrada, lobby, fim de partida, mesa, ou aviso de mesa aberta em outra aba |
| `components/JoinForm.tsx`, `Lobby.tsx`, `Ending.tsx` | as três telas fora da partida |
| `components/Displaced.tsx` | o aviso para a aba que perdeu o assento para outra (`seat_taken`): ela para de reconectar e só volta pelo botão |
| `components/Table.tsx` | a mesa: barra do topo, e as quatro áreas abaixo |
| `components/Seats.tsx` | os outros assentos; quem está na vez vira esmalte preto, quem caiu fica dourado, quem saiu fica riscado |
| `components/Arena.tsx` | o centro: a janela de reação (com a barra de tempo), a pausa, ou a última jogada |
| `components/Hand.tsx` | a sua mão e a sua decisão: ações (com alvo em dois toques), revelar carta, escolher o que devolver na troca |
| `components/Log.tsx` | o registro |
| `embed.go` | pacote Go de uma função: `Dist()` devolve o `dist/` embutido no binário |
| `CLAUDE.md`, `AGENTS.md` | as instruções para agentes: o `CLAUDE.md` só importa o `AGENTS.md` com `@AGENTS.md` |

## O cliente não sabe as regras

Os botões saem de `your_actions`, `window.your_options` e `your_returns`, que o servidor manda
já filtrados. Na troca, a tela deixa marcar duas cartas e só libera o envio se o par marcado
estiver em `your_returns` — a lista decide, não a tela. A legenda impressa em cada carta
("taxas · +3", "bloqueia extorsão") é texto de carta, como o nome dela.

## Por que Next só como export estático

O binário único é o requisito: o Go embute `dist/` e serve tudo. `output: 'export'` gera HTML e JS
estáticos; não há SSR, rota de servidor nem rewrite em produção. O rewrite de `/ws` pro `:8080`
existe só no `next dev`. As fontes vêm do `@fontsource`, dentro do build, então a mesa funciona
numa LAN sem internet.

## `AGENTS.md` e `CLAUDE.md`

O Next 16 escreve, no `next dev`, um bloco entre os marcadores `nextjs-agent-rules` avisando que
esta versão pode diferir do que um agente lembra, e mandando ler `node_modules/next/dist/docs/`.
Os dois arquivos estão versionados: o `next dev` encontra o bloco e não mexe em nada. Abaixo do
bloco ficam as regras deste diretório (export estático, cliente sem regra, direção Esmalte,
idioma). O `CLAUDE.md` tem uma linha só, `@AGENTS.md`, e o Claude Code lê o outro por ela.

## Dois processos em dev, um em release

```sh
pnpm dev      # :3000, hot reload; /ws vai pro Go em :8080
pnpm build    # tsc + next build, gera dist/ — precisa vir ANTES do go build
pnpm lint     # oxlint
```

## Os dois `.gitkeep` que parecem lixo

`go:embed` é **erro de compilação** quando o padrão não casa arquivo nenhum, então `dist/` nunca
pode ficar vazia. `dist/.gitkeep` está no git e faz um clone novo compilar antes de qualquer
build; `public/.gitkeep` é copiado pra dentro do `dist` por todo `pnpm build`, repondo o
primeiro, que o `next build` apaga ao esvaziar a pasta. Apagar qualquer um dos dois quebra
`go build ./...`.

A diretiva `//go:embed` não aceita `..`, e é só por isso que `embed.go` mora aqui e não em
`internal/server` ou `cmd/coup`.
