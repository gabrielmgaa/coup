# `internal/tui` — a mesa no terminal

O cliente Bubble Tea do `coup join`. Mesmo papel do React em `web/`: **desenha o snapshot que
chega e manda de volta o que foi escolhido**. Não tem regra de Coup aqui dentro.

## Arquivos

| | |
|---|---|
| `tui.go` | `Table` (servidor, nome, código) e `Play`, que disca, manda a primeira mensagem e roda o programa |
| `link.go` | o WebSocket: `receive` vira um `tea.Msg` por mensagem, `outgoing` vira um `tea.Cmd` que escreve; falhar ao conectar ou perder a conexão vira mensagem em pt-BR |
| `incoming.go` | decodifica o envelope do servidor em `lobbyArrived`, `updateArrived`, `refusalArrived` |
| `model.go` | `Model`, `Init`, `Update`: teclas movem o cursor, enter manda a escolha; guarda a altura e a largura do terminal, a partida que acabou e os instantes em que prazo e pausa terminam |
| `choices.go` | a lista de escolhas desenhada a partir de `your_actions` (e do lobby); rótulos em pt-BR, com artigo ("a Condessa", "o Duque") |
| `render.go` | `View`: mesa, status, relógio, log e escolhas, com Lipgloss |
| `session.go` | `session.json` em `coup/` dentro de `os.UserConfigDir` — nome, servidor, e o código e token do último assento |

## O que não pode quebrar

- **As escolhas saem do snapshot, nunca de uma regra.** `choices.go` expande `your_actions`
  (um item por alvo) e as cartas da própria mão; ele não decide o que é legal. "É a minha
  decisão" é "tenho escolhas na tela", não uma leitura de fase.
- **Decisão nova, cursor novo.** Quando a lista de escolhas muda, o cursor volta para "deixar
  passar" se ela existir, senão para a primeira. Um Enter que sobrou da decisão anterior nunca
  cai num bloqueio ou numa contestação.
- **O modelo não escreve na rede dentro do `Update`.** A escrita volta como `tea.Cmd` e roda fora
  do laço do Bubble Tea — o mesmo princípio do actor do servidor.
- **Quando a sala recusa e fecha, o motivo aparece.** Se a conexão cai depois de um `error`, o
  programa sai com a mensagem da recusa; sem recusa, sai com "a conexão com o servidor caiu: "
  seguida da causa técnica, que diz se foi endereço errado, servidor fora do ar ou rede.
- **A tela cabe no terminal.** Com a altura conhecida, o log encolhe até a tela caber; o Bubble
  Tea corta o que sobra por cima, e por cima estão os assentos. Com a largura conhecida, quando a
  fileira de caixas não cabe, os assentos viram uma lista de uma linha por jogador — seis
  assentos com cartas reveladas passam de 90 colunas.
- **O fim fica na tela.** A partida que acabou e o log dela continuam acima do lobby até a
  próxima começar.

## Teste

`tui_test.go` joga uma partida inteira contra o servidor de verdade: um terminal (dirigido por
teclas no `Model`) e um "navegador" (WebSocket cru), até alguém vencer, e confere que os dois
viram o mesmo vencedor. Não precisa de TTY: o laço do Bubble Tea é reproduzido chamando
`Update` com o que o socket entrega. Os outros arquivos de teste cobrem um assunto cada:
`cursor_test.go`, `choices_test.go`, `render_test.go` (com relógio manual no campo `now` do
`Model`) e `link_test.go` (servidor ausente e servidor que cai).

## Desvio do plano

O plano previa o viewport do `bubbles` para o log. O log mostra as últimas 8 linhas, menos
quando o terminal é baixo: rolagem não pagou a dependência.
