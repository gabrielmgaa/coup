# `internal/server` — HTTP, WebSocket e o dono da partida

Serve o site que está dentro do binário, aceita conexões WebSocket e roda a sala. É aqui que
mora o desenho de concorrência inteiro do projeto.

## Arquivos

| | |
|---|---|
| `server.go` | `Config` (prazos), o `registry` de salas, o sorteio de código, a validação de nome, e `accept`/`read`/`write` — a vida de uma conexão |
| `room.go` | `Room`, o actor: `run`, `handle`, `admit`, `markReady`, `start`, `play`, `afterChange` e o piloto automático |
| `seat.go` | `seat` (nome, token, pronto, conexão) e `connection`; `join`, `reconnect`, `disconnect` |
| `clock.go` | os três timers — prazo da decisão, carência da pausa, TTL da sala vazia — e o que cada um faz ao disparar |
| `broadcast.go` | `broadcast`, o snapshot de cada assento (`GameState`), `refuse`, `turnAway`, `send` |

## Onde a conexão entra

O endereço é sempre `/ws`, sem query param. **A primeira mensagem decide a sala**: `create_room`
abre uma nova, `join` carrega o código de quem já tem um, `reconnect` carrega código e token.
Quem senta recebe `welcome` com o código e o token do assento — só ele recebe.
`accept` lê essa primeira mensagem, valida o nome (2–16 caracteres, aparado; repetido na sala é
recusado sem distinguir maiúsculas, para `TESTER1` não se passar por `tester1`) e só então entrega
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

Uma mesa tem **3 goroutines por jogador mais a da sala**: uma leitora (socket → inbox), uma
escritora (outbox → socket) e a `keepAlive`, que manda ping. Leitura bloqueante é Go normal; o
runtime estaciona a goroutine.

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
- **Sair do lobby difunde.** Cair no lobby remove o assento e todo mundo recebe a lista nova.
- **Uma conexão só entra uma vez.** Enquanto não tem assento, ela só pode mandar `join` ou
  `reconnect` (`admit`); depois de sentada, esses dois tipos viram `illegal_action`. Sem isso, um
  cliente trocaria de nome no meio da sala ou tentaria o token de outra pessoa pela conexão que
  já tem.
- **A jogada é assinada pelo assento, nunca pela mensagem.** `ToMove` recebe o nome do assento
  da conexão; um campo `name` ou `by` mandado pelo cliente é ignorado.
- **Conexão surda cai sozinha.** `keepAlive` pinga a cada `PingEvery` (15 s) e fecha a conexão
  que não responde em `PongWait` (10 s); toda escrita tem prazo de `WriteWait` (10 s). A leitora
  então falha e o `leave` segue o caminho normal: pausa se a partida espera por aquele assento.
  O navegador responde ping mesmo com o JS travado, então aba congelada continua parecendo viva —
  quem segura a mesa nesse caso é o prazo da decisão. Medido no Chromium 151 e no Brave 154: sem
  ping, o buffer do sistema absorve de 787 a 2643 mensagens antes de a escrita travar, então o
  prazo de escrita sozinho quase nunca dispararia — é o ping que detecta, em até 25 s.
- **O motivo do fechamento é o da recusa.** `drop` guarda o código (`name_taken`,
  `seat_taken`…) e a escritora fecha com ele; fila cheia fecha com `client too far behind`.
- **Timer é só mais um `command`.** `time.AfterFunc` entrega um `expiry` no inbox, com o ID de
  quem o armou. Chegando velho — decisão que já mudou, pausa que já acabou, sala que voltou a
  ter gente — é jogado fora. `expiry` não tem representação JSON: cliente nenhum consegue forjar.

## Tempo e queda (0.8)

- **Prazo:** toda decisão tem `Deadline` (25 s). Estourou, o servidor aplica `engine.SafeMove`
  por cada um que a partida espera: Renda no turno (Golpe se tiver 10+), passar na janela, a
  primeira carta na perda de influência, devolver as duas compradas na troca.
- **Pausa só quando a partida depende de quem caiu.** Queda de quem não tem decisão pendente só
  aparece em `disconnected`. Queda de quem tem para o prazo e abre `Grace` (30 s).
- **Prazo e carência são saldo da decisão, não recarregam.** Quem volta dentro da carência
  (`reconnect` com o token do `welcome`) recebe o snapshot atual e o prazo continua de onde
  parou; cair de novo na mesma decisão pausa só com o que sobrou da carência. Decisão nova — turno,
  janela, perda de influência, troca — começa com os dois cheios. Assim cair e voltar em loop não
  segura a mesa: o pior caso é `Deadline` + `Grace` por decisão.
- **Não voltou:** o assento vira piloto automático — a partida joga o `SafeMove` dele na hora,
  sem pausar de novo. Voltando depois, retoma o assento e o piloto desliga.
- **Reconectar com a aba antiga aberta** toma o assento dela: a conexão antiga recebe
  `seat_taken` e é fechada. O site para de reconectar ao ver esse código, senão as duas abas
  brigariam pelo assento para sempre.
- **TTL:** sala sem nenhuma conexão viva por `IdleTTL` (30 min) fecha, sai do `registry`, e quem
  tentar entrar recebe `room_not_found`. `deliver` usa `done` para nunca travar mandando para
  uma sala que já fechou.

## Estado de hoje (0.8)

Salas com código de 4 caracteres sem `O`, `0`, `I` e `1`; lobby com pronto e host; teto de 6;
prazo por decisão, pausa, reconexão por token e TTL. O fim da partida volta para o lobby com os
mesmos assentos conectados e o mesmo código; quem venceu começa a próxima (`Setup.Starter`), e a
opção `independent_reactions` escolhida no `create_room` vale para todas as partidas da sala.
