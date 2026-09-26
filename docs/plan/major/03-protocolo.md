# Protocolo

WebSocket em `/ws`. JSON, `encoding/json`, tagged union pelo campo `type`. Uma conexão por
jogador, aberta uma vez e mantida aberta.

**Nome vai em inglês no fio, texto vai em pt-BR.** Chave de JSON, `type`, código de erro e nome
de ação são identificadores; `text` e `message` são o que o humano lê. O que a fase 0.1 já
fala está marcado nas tabelas abaixo; o resto é o desenho das fases seguintes.

## O princípio que rege tudo: o cliente não sabe as regras

O servidor manda **quais ações são legais agora** (`your_actions`) e **quais respostas cabem
nesta janela** (`your_options`). O React e o Bubble Tea são renderizadores puros: desenham
botões a partir de uma lista e mandam de volta o que foi clicado.

Sem isso, "Extorquir não pode mirar quem tem 0 moedas" precisaria ser escrito três vezes — em
Go no motor, em TypeScript no navegador, em Go de novo na TUI — e divergiria. Com isso, é
escrito uma vez.

Mesmo motivo do snapshot em vez de delta: **regra e estado moram no servidor, ponto.**

## Cliente → servidor

| `type` | Campos | Quando | Fase |
|---|---|---|---|
| `join` | `room`, `name` | Entrar numa sala existente | **0.2** |
| `play` | `action`, `target?` | Só no seu turno | **0.1** |
| `lose_influence` | `card` | Quando a fase pede que você escolha | **0.1** |
| `leave` | — | Sai da sala | **0.1** |
| `create_room` | `name`, `options?` | Sem sala ainda. O código volta no primeiro `lobby` | **0.2** (sem `options` até a 0.9) |
| `ready` | `ready: bool` | Só no lobby | **0.2** |
| `start` | — | Só o host, só com todos prontos e ≥2 jogadores | **0.2** |
| `respond` | `window`, `answer`, `character?` | `answer` ∈ `challenge` / `block` / `pass` | **0.4** |
| `return_cards` | `cards: [duas]` | Após Trocar (Embaixador); os pares válidos chegam em `your_returns` | **0.7** |
| `reconnect` | `room`, `token` | Retomar a sessão de antes | **0.8** |

`character` só acompanha `block`, porque Extorsão aceita dois bloqueadores diferentes
(Capitão ou Embaixador) e o motor precisa saber qual foi alegado pra resolver a contestação.

## Servidor → cliente

| `type` | Campos | Fase |
|---|---|---|
| `update` | `state`, `events` | **0.1** |
| `error` | `code`, `message`, `received`, `expected` | **0.1** |
| `lobby` | `state` com `room`, `you`, `host` e `players` de `{name, ready}` | **0.2** |
| `welcome` | `room`, `token` — só para quem acabou de sentar | **0.8** |

**`lobby` e `update` são mensagens distintas.** Enquanto a partida não começou, a sala manda
`lobby`; depois do `start`, manda `update` e nunca mais `lobby`. O cliente troca de tela pela
mudança de tipo, e não precisa adivinhar pela forma do `state`. O `welcome` que este documento
previa para a 0.2 não existe: o código da sala chega no campo `room` do primeiro `lobby`, e o
token só faz sentido quando houver reconexão.

`state` e `events` viajam **na mesma mensagem**, sempre. Nunca há estado sem a narração do
que causou ele, nem narração sem o estado resultante.

## O snapshot

Um snapshot completo, recortado pro destinatário. ~700 bytes numa mesa de 6; uma partida inteira
gasta ~84 KB. Mandar tudo a cada ação é irrelevante.

```json
{
  "type": "update",
  "state": {
    "room": "K7QM",
    "phase": "awaiting_response",
    "you": "tester3",
    "host": "tester2",
    "options": { "independent_reactions": false },
    "turn_of": "tester2",
    "deck_remaining": 7,
    "paused": null,
    "winner": null,
    "players": [
      { "name": "tester2",  "coins": 2, "hidden": 2, "revealed": [],
        "connected": true,  "ready": true, "eliminated": false },
      { "name": "tester3",   "coins": 3, "hidden": 2, "revealed": [],
        "connected": true,  "ready": true, "eliminated": false,
        "my_cards": ["contessa", "captain"] },
      { "name": "tester5",  "coins": 2, "hidden": 1, "revealed": ["assassin"],
        "connected": true,  "ready": true, "eliminated": false },
      { "name": "tester4", "coins": 0, "hidden": 2, "revealed": [],
        "connected": false, "ready": true, "eliminated": false }
    ],
    "window": {
      "id": 42,
      "action": { "name": "assassinate", "by": "tester2", "target": "tester3", "claims": "assassin" },
      "block": null,
      "your_options": [{ "answer": "challenge" }, { "answer": "block", "character": "contessa" }, { "answer": "pass" }],
      "waiting_on": ["tester3", "tester5"],
      "closes_in_ms": 25000
    },
    "your_actions": null
  },
  "events": [
    { "n": 47, "type": "action_declared",
      "text": "tester2 pagou 3 e alegou Assassino contra tester3.",
      "data": { "by": "tester2", "action": "assassinate", "target": "tester3", "cost": 3 } }
  ]
}
```

**O snapshot de hoje é menor.** A 0.1 tem `phase`, `you`, `turn_of`, `losing`, `winner`,
`deck_remaining`, `players` (com `name`, `coins`, `hidden`, `revealed`, `eliminated`,
`my_cards`) e `your_actions` — e `event` sem `data`, só `n`, `type` e `text`. `room`, `host`,
`options`, `paused`, `window`, `connected` e `ready` chegam com as fases que os pedem.

`losing` é o nome de quem tem de escolher qual carta revelar; vem preenchido só na fase
`awaiting_influence_loss`.

### O mesmo snapshot, mandado pro tester5

Muda em três lugares, e é aí que a informação oculta acontece:

```json
  "you": "tester5",
    { "name": "tester3",  "coins": 3, "hidden": 2, "revealed": [] },
    { "name": "tester5", "coins": 2, "hidden": 1, "revealed": ["assassin"],
      "my_cards": ["duke"] },
  "window": { "your_options": ["challenge", "pass"] }
```

O tester5 vê que o tester3 tem 2 cartas ocultas, **nunca quais são** — `my_cards` só existe
na entrada dele mesmo. E ele não recebe `block_with_contessa` porque bloquear Assassinato é
só do alvo.

### Quando é a sua vez

`window` vem `null` e `your_actions` vem preenchido, já filtrado pelo que é legal:

```json
"phase": "awaiting_action",
"turn_of": "tester3",
"your_actions": [
  { "name": "income" },
  { "name": "foreign_aid" },
  { "name": "tax" },
  { "name": "exchange" },
  { "name": "steal",       "targets": ["tester2", "tester5"] },
  { "name": "assassinate", "targets": ["tester2", "tester5", "tester4"], "cost": 3 }
]
```

tester4 não aparece nos alvos de `steal` porque está com 0 moedas. `coup` não aparece
porque tester3 tem menos de 7. Se tester3 tivesse 10+, a lista teria **só** `coup` — e isso já é
assim desde a 0.1.

### Sala pausada

```json
"paused": { "waiting_for": "tester3", "resumes_in_ms": 30000 }
```

Presente só quando a partida trava por queda de quem tem decisão pendente. Enquanto está
presente, `window.closes_in_ms` não corre.

## Eventos

```json
{ "n": 47, "type": "action_declared", "text": "...", "data": { ... } }
```

- `n` é sequencial por sala e nunca reseta durante a partida. É o log que o projeto quer desde
  o começo, mesmo hoje descartado no fim. **Um evento suprimido abre buraco na numeração**, e
  foi assim que um bug apareceu na 0.1: quem perdia a última carta não gerava `influence_lost`
  nem `player_eliminated`, e o log pulava de 1 pra 4.
- `text` é o que a CLI imprime no log e a web mostra no feed. Vem pronto do servidor, em
  pt-BR, porque montar a frase no cliente significaria montá-la **duas vezes**, em Go e em TS.
- `data` é o que a interface usa pra animar (qual carta virou, quem perdeu, quanto de dinheiro
  andou). **Ainda não existe:** o `Event` de hoje tem `n`, `type` e `text`, e nasce com a
  primeira animação que precisar dele.

Quando i18n entrar (fase futura), o cliente monta a string a partir de `type` + `data` e
`text` vira fallback. Até lá, `text` é a fonte.

**Tipos de evento.** Existem hoje: `action_declared`, `influence_lost`, `player_eliminated`,
`turn_passed`, `game_over`. Previstos: `game_started`, `window_opened`, `passed`, `challenged`,
`challenge_won`, `challenge_lost`, `blocked`, `block_held`, `block_failed`, `card_swapped`,
`coins_refunded`, `action_resolved`, `player_dropped`, `player_returned`, `room_paused`,
`room_resumed`, `auto_resolved`.

## Erros

Formato fixo, sempre com o valor recebido e o esperado:

```json
{ "type": "error", "code": "window_closed",
  "message": "a janela 42 já fechou",
  "received": 42, "expected": 43 }
```

`message` é pt-BR porque é o que aparece na tela; `code`, `received` e `expected` são para quem
está depurando.

| `code` | Significa | Fase |
|---|---|---|
| `illegal_action` | ação não está em `your_actions` | **0.1** |
| `invalid_target` | alvo não está na lista daquela ação | **0.1** |
| `not_your_turn` | jogou fora do turno, ou respondeu a uma janela que não espera por ele; `expected` traz quem ela espera | **0.1** |
| `insufficient_coins` | custo maior que o saldo | **0.1** |
| `coup_required` | tem 10+ moedas e tentou outra coisa | **0.1** |
| `name_taken` | duplicado nesta sala | **0.1** |
| `room_full` | já tem 6 no lobby | **0.2** |
| `invalid_name` | fora de 2–16 caracteres, já aparado | **0.2** |
| `not_host` | tentou `start` sem ser host | **0.2** |
| `room_not_found` | código de sala não existe ou expirou | **0.2** |
| `game_started` | chegou depois do `start` | **0.2** |
| `not_all_ready` | `start` com gente sem marcar pronto; `received` traz quem falta | **0.2** |
| `not_enough_players` | `start` com menos de 2 | **0.2** |
| `too_many_players` | o motor recusou mais de 6 nomes | **0.2** |
| `window_closed` | respondeu a uma janela que não está aberta | **0.4** |
| `already_responded` | segunda resposta na mesma janela | **0.4** |
| `invalid_token` | reconexão com token desconhecido | **0.8** |
| `seat_taken` | outra conexão entrou com o token deste assento; esta é fechada em seguida | **pós-0.9** |

Um erro vai sempre para **uma** conexão e nunca é difundido. Quase sempre responde a uma mensagem
daquele cliente; `seat_taken` é a exceção, porque quem o provoca é a conexão nova.

## Informação oculta, garantida pelo compilador

São **dois tipos Go diferentes**, e não é convenção — é o compilador que impede o vazamento.

```go
// internal/engine/game.go — a VERDADE. Nunca sai do servidor.
type Game struct {
	players []player    // minúsculo = não exportado
	deck    []Character // minúsculo = não exportado
	phase   Phase
	winner  string
}

// internal/engine/view.go — o que VAI pro cliente.
type View struct {
	You     string       `json:"you"`
	TurnOf  string       `json:"turn_of"`
	Players []PlayerView `json:"players"`
	Window  *WindowView  `json:"window,omitempty"`
}
```

**`View` mora em `internal/engine`, não em `internal/protocol`** — e a versão anterior deste
documento dizia o contrário, errado. `ViewFor` precisa ler os campos privados de `Game`, então
morando em `protocol` ele importaria `engine`, e `engine` importaria `protocol` de volta:
ciclo de import, que em Go nem compila. O motivo de fundo é melhor que o mecânico — esconder
carta é **regra do Coup**, não detalhe de transporte. `internal/protocol` só embrulha
`engine.View` no envelope e não remodela nada.

Maiúscula/minúscula **é** o sistema de visibilidade do Go: campo com inicial minúscula é
privado ao pacote. E `encoding/json` **só serializa campos exportados**. Então
`json.Marshal(game)` devolve literalmente `{}` — é *impossível* vazar a mão de alguém por
acidente, mesmo escrevendo o código errado.

Aquela `` `json:"turn_of"` `` é uma **struct tag**: metadado colado no campo dizendo ao
`encoding/json` como nomear ele no JSON.

A projeção é uma função pura:

```go
func ViewFor(g *Game, name string) View
```

## Reconexão

1. No primeiro `join` ou `create_room`, o servidor responde `welcome` com um token opaco de
   128 bits. Web guarda em `localStorage`, CLI em `~/.config/coup/session.json`.
2. Caiu a conexão → o cliente reabre o WebSocket e manda `reconnect {token}`.
3. O servidor associa o token ao assento, marca `connected: true` e responde `update` com
   o snapshot atual. Acabou.

Não existe replay, não existe "me manda do evento 47 em diante", não existe histórico guardado
por sala pra isso. O snapshot **é** o estado.

O token vale enquanto a sala viver (TTL de 30 min sem nenhuma conexão). Reconexão depois disso
devolve `room_not_found`.
