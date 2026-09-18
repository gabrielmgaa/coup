# `internal/server` — HTTP, WebSocket e o dono da partida

Serve o site que está dentro do binário, aceita conexões WebSocket e roda a sala. É aqui que
mora o desenho de concorrência inteiro do projeto.

## Arquivos

| | |
|---|---|
| `server.go` | o `registry` de salas, o sorteio de código, a validação de nome, e `accept`/`read`/`write` — a vida de uma conexão |
| `room.go` | `Room`, o actor: `run`, `join`, `markReady`, `start`, `play`, `broadcast`, `refuse`, `send`, `remove` |

## Onde a conexão entra

O endereço é sempre `/ws`, sem query param. **A primeira mensagem decide a sala**: `create_room`
abre uma nova e devolve o código no primeiro `lobby`; `join` carrega o código de quem já tem um.
`accept` lê essa primeira mensagem, valida o nome (2–16 caracteres, aparado) e só então entrega
a conexão ao goroutine da sala. Nome inválido nunca cria sala órfã, porque a checagem vem antes
do registro.

O `registry` é o único lugar do pacote com mutex, e a seção crítica é uma busca em map mais o
sorteio do código. Cada sala nasce com seu próprio `*rand.Rand`, semeado ali dentro — duas salas
sorteando do mesmo `*rand.Rand` seria data race, porque `rand.Rand` não é seguro para uso
concorrente.

## O actor: uma goroutine dona, zero mutex

A `Room` é dona do `*engine.Game`. Ninguém mais alcança o jogo. Todo evento de entrada — um
clique, uma queda de conexão, e da 0.8 um deadline estourando — chega como `command` no
`Room.inbox`, e `run()` tira **um por vez**. A corrida "alguém responde no instante em que o
clock estoura" não é resolvida: ela não pode existir.

Uma mesa tem **2 goroutines por jogador mais a da sala**: uma leitora (socket → inbox) e uma
escritora (outbox → socket). Leitura bloqueante é Go normal; o runtime estaciona a goroutine.

## O que não pode quebrar

- **A goroutine dona nunca escreve na rede.** `send` larga a mensagem no `outbox` da conexão
  (capacidade 16) com `select`/`default`; queue cheia é cliente morto, então ele é derrubado.
  Não existe outro lugar de onde mandar mensagem — é isso que torna impossível esquecer. Um
  desenho com mutex exigiria repetir "nenhuma escrita de rede segurando o lock" em cada
  função nova, e `localhost` nunca falha nos testes pra acusar.
- **O refusal é privado.** `refuse` responde a **uma** conexão; só `broadcast` fala com a mesa.
- **`drop` é idempotente.** Fechar um channel duas vezes é panic, e uma conexão pode ser
  derrubada por caminhos diferentes no mesmo instante.
- **O host é `connections[0]`, não um campo.** A lista está em ordem de chegada, então o host
  novo aparece sozinho quando o antigo sai — desde que a escolha venha *depois* da remoção.
- **Sair do lobby difunde.** `remove` chama `broadcast` enquanto não há partida; sem isso a lista
  das outras telas nunca encolhe e o host fica esperando o pronto de quem já foi embora.

## Estado de hoje (0.2)

Salas de verdade, com código de 4 caracteres sem `O`, `0`, `I` e `1`; lobby com pronto e host;
teto de 6. Ainda **não** existe: prazo por janela, detecção de queda no meio da partida, pausa,
reconexão e TTL de sala — uma sala vazia continua viva até o processo morrer, e isso fecha na
0.8. `remove` no meio de uma partida tira a conexão e não difunde, porque decidir o que a mesa
vê quando alguém cai é justamente o assunto da 0.8.
