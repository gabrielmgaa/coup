# `internal/engine` — as regras

O jogo de Coup em si: cartas, moedas, turnos, quem morreu, quem venceu. **Sem rede, sem
clock, sem I/O, sem goroutine.** Determinístico dado um `*rand.Rand`, o que faz a suíte
inteira rodar em milissegundos, sem `Sleep` e sem flake.

É o mesmo pacote que o servidor, a CLI (0.3) e, um dia, os bots vão usar. Ele não conhece
nenhum deles.

## Arquivos

| | |
|---|---|
| `game.go` | `Game`, `NewGame`, `Phase`, `Apply` — o type switch e o pipeline de uma ação |
| `move.go` | a união selada de entradas: `Act`, `LoseInfluence` |
| `rules.go` | `Rule` e a tabela `rules`: uma linha por ação, com `Cost`, `NeedsTarget`, `Effect` |
| `influence.go` | perder carta, escolher qual, eliminação, condição de vitória |
| `deck.go` | `Character`, as 15 cartas, embaralhar |
| `event.go` | `Event` e `narrate`: `n` sequencial + o texto pt-BR que o jogador lê |
| `refusal.go` | `Refusal`: `code`, `message`, e sempre o `received` **e** o `expected` |
| `view.go` | `View` e `ViewFor` — o snapshot recortado por destinatário |

## O que não pode quebrar

- **`Game` só tem campos minúsculos.** `encoding/json` só serializa campo exportado, então
  `json.Marshal(game)` devolve `{}`. Vazar a mão de alguém por acidente é impossível, mesmo
  escrevendo o código errado. Exportar um campo aqui derruba essa garantia inteira.
- **`ViewFor` mora aqui, não em `protocol`.** Esconder carta é regra do Coup, não detalhe de
  transporte — e `protocol` importando `engine` e vice-versa é ciclo de import, que nem compila.
- **Evento não se perde no caminho.** Todo efeito que muda o jogo devolve seus eventos pra quem
  chamou; engolir um abre buraco no `n` e o jogador perde a explicação na tela.
- **Aleatoriedade é injetada, não mockada.** Teste passa `rand.New(rand.NewPCG(1, 2))`, produção
  semeia do `crypto/rand`. Pra mão legível no teste existe `newGameWithDeck`, interno ao pacote.
- **A assimetria do dinheiro mora só na tabela:** contestação bem-sucedida devolve o custo,
  bloqueio bem-sucedido não.

## O que não entra aqui

Deadline, `time`, WebSocket, JSON de envelope, código de sala, host, reconexão. Nada disso é regra
de Coup. O motor não sabe que horas são — quem carimba deadline é `internal/server`.
