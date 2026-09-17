# `internal/server` — HTTP, WebSocket e o dono da partida

Serve o site que está dentro do binário, aceita conexões WebSocket e roda a sala. É aqui que
mora o desenho de concorrência inteiro do projeto.

## Arquivos

| | |
|---|---|
| `server.go` | `New` monta o `http.ServeMux`; `accept`, `read` e `write` são a vida de uma conexão |
| `room.go` | `Room`, o actor: `run`, `join`, `play`, `broadcast`, `refuse`, `send`, `remove` |

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

## Estado de hoje (0.1)

**Uma sala fixa, sem código de sala**: `newRoom` é chamado uma vez em `New`, e a partida começa
no segundo jogador que entrar. O `map[code]*Room`, o `create_room` e a validação de nome nascem
na 0.2 — e é lá que este andaime sai. Hoje `join` aceita qualquer nome, inclusive vazio.
