# `internal/tui` — a mesa no terminal

O cliente Bubble Tea do `coup join`. Mesmo papel do React em `web/`: **desenha o snapshot que
chega e manda de volta o que foi escolhido**. Não tem regra de Coup aqui dentro.

## Arquivos

| | |
|---|---|
| `tui.go` | `Table` (servidor, nome, código) e `Play`, que disca, manda a primeira mensagem e roda o programa |
| `link.go` | o WebSocket: `receive` vira um `tea.Msg` por mensagem, `outgoing` vira um `tea.Cmd` que escreve |
| `incoming.go` | decodifica o envelope do servidor em `lobbyArrived`, `updateArrived`, `refusalArrived` |
| `model.go` | `Model`, `Init`, `Update`: teclas movem o cursor, enter manda a escolha |
| `choices.go` | a lista de escolhas desenhada a partir de `your_actions` (e do lobby); rótulos em pt-BR |
| `render.go` | `View`: mesa, status, log e escolhas, com Lipgloss |
| `session.go` | `~/.config/coup/session.json` — nome e servidor, para não digitar de novo |

## O que não pode quebrar

- **As escolhas saem do snapshot, nunca de uma regra.** `choices.go` expande `your_actions`
  (um item por alvo) e as cartas da própria mão; ele não decide o que é legal.
- **O modelo não escreve na rede dentro do `Update`.** A escrita volta como `tea.Cmd` e roda fora
  do laço do Bubble Tea — o mesmo princípio do actor do servidor.
- **Quando a sala recusa e fecha, o motivo aparece.** Se a conexão cai depois de um `error`, o
  programa sai com a mensagem da recusa, não com o erro de socket.

## Teste

`tui_test.go` joga uma partida inteira contra o servidor de verdade: um terminal (dirigido por
teclas no `Model`) e um "navegador" (WebSocket cru), até alguém vencer, e confere que os dois
viram o mesmo vencedor. Não precisa de TTY: o laço do Bubble Tea é reproduzido chamando
`Update` com o que o socket entrega.

## Desvio do plano

O plano previa o viewport do `bubbles` para o log. O log mostra as últimas 8 linhas e mais
nada: rolagem não pagou a dependência.
