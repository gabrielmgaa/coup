# `internal/engine` — as regras

O jogo de Coup em si: cartas, moedas, turnos, quem morreu, quem venceu. **Sem rede, sem
clock, sem I/O, sem goroutine.** Determinístico dado um `*rand.Rand`, o que faz a suíte
inteira rodar em milissegundos, sem `Sleep` e sem flake.

É o mesmo pacote que o servidor, a CLI (0.3) e, um dia, os bots vão usar. Ele não conhece
nenhum deles.

## Arquivos

| | |
|---|---|
| `game.go` | `Game`, `NewGame`, `Phase`, `Apply`, `MaxPlayers` — o type switch e o pipeline de uma ação |
| `move.go` | a união selada de entradas: `Act`, `Respond`, `LoseInfluence`, `ReturnCards` |
| `rules.go` | `Rule` e a tabela `rules`: as 7 ações, com `Cost`, `Claims`, `NeedsTarget`, `BlockedBy`, `ValidTarget`, `Declaration`, `Effect` |
| `window.go` | a janela de reação: `Answer`, `Option`, quem é elegível, `respond`, first-responder, bloqueio, fechamento quando todos passam |
| `challenge.go` | contestação da ação e do bloqueio, troca da carta provada, devolução do custo |
| `exchange.go` | devolver 2 cartas depois de Trocar, e os pares que o snapshot oferece |
| `influence.go` | perder carta (escolha ou automática), o que vem depois (`followUp`), eliminação, vitória |
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
- **`MaxPlayers` é daqui, e `NewGame` recusa mais que isso.** O baralho tem 15 cartas e cada
  jogador leva 2: no oitavo, distribuir estoura o slice. `NewGame` devolve `too_many_players`
  em vez de panicar, porque a sala não é a única que chama — a CLI da 0.3 e os bots chamam
  direto. `internal/server` lê a mesma constante, então o teto muda num lugar só.

## O pipeline de uma ação

`act` cobra o custo e declara; `openActionWindow` abre a janela para quem pode reagir, ou resolve
direto se ninguém pode (Renda, Golpe). Cada caminho termina em `proceed(followUp)`, que antes de
tudo confere se a partida acabou:

- `endTurn` — passa a vez;
- `continueAction` — a ação sobreviveu a uma contestação e segue;
- `resolveAction` — aplica o `Effect` da linha da tabela.

Perder influência pode pedir escolha do jogador; por isso `loseInfluenceThen` guarda o
`followUp` em `afterLoss`, e `loseInfluence` retoma dali. **Cada `Effect` termina o próprio
turno** — ou abrindo uma perda de influência, ou chamando `proceed(endTurn)`.

`decision` conta cada coisa nova que o jogo passa a esperar (turno, janela, perda de carta). O
`ID` da janela é o `decision` do momento em que ela abriu, e é assim que uma resposta atrasada é
reconhecida (`window_closed`).

## O que não entra aqui

Deadline, `time`, WebSocket, JSON de envelope, código de sala, host, reconexão. Nada disso é regra
de Coup. O motor não sabe que horas são — quem carimba deadline é `internal/server`.
