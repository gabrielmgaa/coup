# `web` — a mesa no navegador

Vite + React + TypeScript. Desenha a mesa a partir do snapshot que chega pelo WebSocket e manda de
volta o que foi clicado. **Não tem regra de Coup aqui dentro, e não pode ganhar nenhuma.**

## Arquivos

| | |
|---|---|
| `src/coup.ts` | os tipos do fio (`View`, `PlayerView`, `GameEvent`, `FromServer`…) e a tradução pra tela: `actionLabel`, `cardLabel` |
| `src/App.tsx` | o componente único: formulário de entrada, mesa, mão, log, botões |
| `src/index.css` | o estilo |
| `embed.go` | pacote Go de uma função: `Dist()` devolve o `dist/` embutido no binário |

## O cliente não sabe as regras

Os botões saem de `state.your_actions`, que o servidor manda já filtrado — com os alvos válidos
dentro. O React não decide que Extorsão não mira quem tem 0 moedas, nem que com 10 moedas só
sobra o Golpe. Ele desenha a lista que recebeu.

O motivo é economia: cada regra escrita aqui seria escrita **de novo** em Go no motor e mais uma
vez na TUI da 0.3 — e as três divergiriam.

Os nomes cruzam o fio em inglês (`income`, `coup`, `duke`) e viram pt-BR na borda, em
`actionLabel` e `cardLabel`. Texto de evento já chega pronto do servidor, em pt-BR.

## Dois processos em dev, um em release

```sh
pnpm dev      # :5173, hot reload. O proxy de /ws pro :8080 está em vite.config.ts
pnpm build    # roda tsc -b e gera dist/ — precisa vir ANTES do go build
pnpm lint     # oxlint
```

## Os dois `.gitkeep` que parecem lixo

`go:embed` é **erro de compilação** quando o padrão não casa arquivo nenhum, então `dist/` nunca
pode ficar vazia. `dist/.gitkeep` está no git e faz um clone novo compilar antes de qualquer
build; `public/.gitkeep` é copiado pra dentro do `dist` por todo `pnpm build`, repondo o
primeiro, que o Vite apaga ao esvaziar a pasta. Apagar qualquer um dos dois quebra
`go build ./...`.

A diretiva `//go:embed` não aceita `..`, e é só por isso que `embed.go` mora aqui e não em
`internal/server` ou `cmd/coup`.
