# Arquitetura

Escrito assumindo que você não conhece Go. Cada conceito é explicado onde aparece.

## O desenho, em uma frase

Um processo Go escutando numa porta. Ele serve o site (que está **dentro** do binário) e
aceita conexões WebSocket. Cada sala de jogo é uma goroutine que é dona absoluta daquela
partida; ninguém mexe no jogo direto, todo mundo manda pedido pra ela por uma queue. A cada
pedido aplicado, ela produz uma versão recortada do jogo pra cada jogador e larga no `outbox`
de cada um.

```
processo `coup serve`
│
├── goroutine principal ─ net/http escutando :8080
│     ├── GET /              → index.html (de dentro do binário)
│     ├── GET /assets/*      → JS e CSS (de dentro do binário)
│     └── GET /ws               → vira WebSocket; a 1ª mensagem diz a sala
│
├── registry ─ map[code]*Room, com mutex (só pra criar/achar sala)
│
└── Room K7QM
      ├── 1 goroutine DONA ("actor")
      │     inbox: chan command      ← todo mundo publica aqui
      │     game:  *engine.Game      ← inalcançável de fora
      │
      └── por jogador conectado (4 jogadores = 8 goroutines):
            ├── goroutine leitora    ─ lê do socket → publica no inbox
            ├── goroutine escritora  ─ lê do outbox → escreve no socket
            └── outbox: chan []byte (com espaço pra 16)
```

Mesa de 4 = **9 goroutines**. Mesa de 6 = 13. Isso é nada: cada goroutine começa com 2 KB.

O `registry` é o único lugar do servidor com mutex, e a seção crítica dele é uma busca em map
mais o sorteio do código. Ele nunca toca em estado de partida.

**O código da sala viaja na mensagem, não no endereço.** `create_room` abre uma sala e recebe o
código de volta; `join` manda o código. Assim o endereço do WebSocket é sempre `/ws` e o mesmo
caminho de entrada serve para os dois casos — o desenho alternativo, `/ws?room=K7QM`, obrigaria
um segundo caminho só para pedir um código antes de ter sala.

## O que é uma goroutine

Uma goroutine **não é uma thread do sistema operacional**. O runtime do Go multiplexa milhares
delas em poucas threads reais. Criar uma custa nanossegundos e 2 KB.

Isso muda como se escreve I/O. Vindo de JavaScript:

```javascript
// Node: ler de um socket não bloqueia. Registra callback e sai.
socket.on('message', (data) => { ... })
```

```go
// Go: ler de um socket BLOQUEIA. A linha para ali até chegar byte.
for {
	_, encoded, err := conn.Read(ctx)   // fica parado aqui
	if err != nil {
		return
	}
	room.inbox <- decode(encoded)
}
```

Parece ruim e não é: isso roda dentro de `go room.read(ctx, conn)`, e o runtime **estaciona**
a goroutine enquanto não tem byte, rodando as outras. Você escreve código reto, de cima pra
baixo, sem `async`, sem `await`, sem callback — e ganha concorrência. É por isso que Go faz
"uma goroutine por conexão": é o modelo inteiro.

## O que é um channel

Um `chan T` é uma **queue tipada** entre goroutines.

```go
inbox := make(chan command)      // sem espaço: quem põe espera alguém tirar
outbox := make(chan []byte, 16)  // com espaço pra 16: só espera se encher

inbox <- command{...}   // põe  (seta apontando PRA dentro do canal)
cmd := <-inbox          // tira (seta saindo do canal)
```

E o laço que a goroutine dona roda pra sempre:

```go
for cmd := range room.inbox {   // tira um, processa, tira o próximo, pra sempre
	...
}
```

`range` num channel é diferente de `range` num slice: não percorre uma coleção, **fica
esperando** o próximo valor chegar, e só termina quando o channel é fechado.

## Por que isso resolve a corrida sozinho

O cenário que mata implementações ingênuas: a janela fecha em 25 s; no segundo **24,998** o
tester5 clica em contestar; no segundo **25,000** o clock estoura. Duas coisas querem mexer
no jogo ao mesmo tempo.

Go garante que um channel entrega **um valor por vez, na ordem em que chegaram**. Então a
corrida não é "resolvida" — ela **não existe**. Só há uma mão, a goroutine dona, tirando um
item da queue de cada vez.

```go
func (r *Room) run() {
	for received := range r.inbox {
		switch received.message.Type {
		case "join":
			r.join(received)
		case "leave":
			r.remove(received.from)
		default:
			r.play(received)
		}
	}
}
```

Esse `for` de 11 linhas **é a sala inteira** — `play` converte a mensagem, chama
`game.Apply` e difunde; `refuse` responde só a quem errou. Lê de cima a baixo e você sabe tudo que pode
acontecer com uma partida.

## Backpressure: por que tem `select`/`default`

A goroutine dona **nunca escreve na rede** — escrever na rede pode demorar. Ela larga a
mensagem no `outbox` de cada jogador e segue:

```go
func (r *Room) broadcast(events []engine.Event) {
	for _, c := range r.connections {
		message := protocol.NewUpdate(engine.ViewFor(r.game, c.name), events)
		select {
		case c.outbox <- encode(message):
			// coube na queue, segue o baile
		default:
			// queue cheia: esse cliente está 16 mensagens atrasado. Derruba.
			c.drop()
		}
	}
}
```

`select` com `default` significa *"tenta pôr na queue; se for ter que esperar, faz o `default`
em vez disso"*. É o envio que **nunca bloqueia**.

Por que 16: uma partida de Coup tem umas 120 mensagens do servidor. Estar 16 atrás é estar
~13 % da partida atrasado — não é lentidão, é cliente morto. Derruba, e ele reconecta com o
token e recebe o snapshot atual.

**O ponto estrutural:** não existe *outro lugar* de onde mandar mensagem, então é impossível
esquecer. A alternativa (mutex na sala) exige a disciplina de "nenhuma escrita de rede
segurando o lock", repetida em toda função nova que encostar na sala — e nenhum teste acusa se
você esquecer, porque em `localhost` escrever é instantâneo.

## Pedido que precisa de resposta

"Entrar na sala" precisa de um sim ou não. O plano previa um `chan error` embutido no próprio
pedido; **a 0.1 não fez assim**, e saiu mais barato: a resposta é uma mensagem como qualquer
outra, entregue no outbox de quem perguntou.

```go
type command struct {
	from    *connection
	message protocol.FromClient
}
```

`r.refuse(c, reason)` vira um `{"type":"error", ...}` no outbox de **uma** conexão, nunca
difundido. A regra de cima continua valendo — a goroutine dona só enfileira — e não existe
caminho síncrono onde alguém possa ficar esperando resposta sem querer.

## O clock

O motor **não sabe que horas são**. Ao abrir uma janela ele emite `Window{ID: 42, Pending:
[...]}` e para aí. Quem carimba o deadline é o servidor:

```go
timer := time.AfterFunc(windowDeadline, func() {
	room.inbox <- command{move: engine.Timeout{Window: 42}}
})
```

O timeout não é um caminho paralelo — é **mais um pedido na queue**, igual a um clique.

**Por isso o `ID` da janela é obrigatório.** Se o tester5 contestou aos 24,998 s, a janela 42
já fechou e a 43 abriu. Aos 25,000 s o timer velho dispara, o motor vê `Window: 42` contra a
corrente 43, e devolve o refusal `window_closed`. Jogado fora, sem drama, sem uma linha de trava.

Isso compra a suíte do motor rodando sem fake clock, sem `Sleep` e sem flake. E a duração vira
config do servidor (25 s na web, 0 nos testes de integração) sem tocar em regra nenhuma.

### Pausa

Quando a partida depende de quem caiu (ver Q12), o actor **cancela o `AfterFunc`** e marca a
sala como pausada, com um timer de grace de 30 s.

- **Reconectou dentro dos 30 s** → snapshot atual, e a janela **reinicia com os 25 s cheios**.
- **Estourou os 30 s** → auto-resolve na hora pelo default seguro, e o jogo segue com ele
  marcado desconectado, **sem pausar de novo**.
- **Ele volta no minuto 10** → o token devolve ele à partida em andamento, no snapshot atual.

Reiniciar o deadline cheio é menos bookkeeping do que guardar o resto congelado, e a proteção é
real — devolver 2 s pra quem acabou de reconectar não protege ninguém.

## O trace de uma ação, ponta a ponta

tester2 (5 moedas) assassina tester3. tester5 contesta no último instante.

| tempo | o que acontece |
|---|---|
| `0.000` | Navegador da tester2 envia `{"type":"play","action":"assassinate","target":"tester3"}` |
| `0.001` | Goroutine **leitora** da tester2 decodifica e faz `room.inbox <- command{...}` |
| `0.001` | **Actor** tira da queue. `game.Apply(...)`: tester2 tem 5 ≥ 3 → cobra 3, guarda a ação pendente, abre `Window{ID: 42}`, emite `action_declared` |
| `0.001` | Actor agenda `time.AfterFunc(25s, …)` publicando `Timeout{Window: 42}` no próprio inbox |
| `0.002` | Actor chama `ViewFor(game, name)` 4 vezes e larga nos 4 outbox. 4 goroutines **escritoras** mandam pela rede |
| `0.002`→`24.99` | **Zero tráfego.** Os 4 clientes animam o countdown localmente a partir do `closes_in_ms` que veio no snapshot |
| `24.998` | tester5 contesta → `room.inbox <- command{Respond, Window: 42}` |
| `24.999` | Actor aplica. tester2 **tinha** o Assassino: devolve a carta, embaralha, puxa outra; tester5 perdeu → fase vira `awaiting_influence_loss` com `losing: "tester5"`, abre `Window{ID: 43}` |
| `25.000` | O `AfterFunc` dispara e publica `Timeout{Window: 42}`. Actor tira da queue, motor vê que a corrente é a 43 → `window_closed`. Ignorado. |

Nessa trilha inteira não se escreveu uma linha de sincronização. Nem `Lock`, nem `Unlock`, nem
`atomic`, nem `-race` reclamando de nada. A queue e o `ID` de janela fizeram o trabalho.

## Layout do repositório

```
coup/
├── cmd/coup/main.go          # o único binário; switch em os.Args[1]
├── internal/
│   ├── engine/               # regras. Sem rede, sem clock, sem I/O.
│   ├── protocol/             # os tipos que viajam no fio
│   ├── server/               # http, websocket, registry, Room (o actor)
│   └── tui/                  # cliente Bubble Tea (só nasce na 0.3)
├── web/                      # Vite + React + TS
│   ├── embed.go              # o //go:embed mora aqui: a diretiva não aceita `..`
│   └── dist/                 # build; embutido via embed.FS
├── docs/plan/major/
├── go.mod                    # module github.com/gabrielmgaa/coup
├── LICENSE                   # MIT
└── README.md
```

`internal/` **não é convenção** — é regra do compilador: nada fora do módulo consegue importar
um pacote debaixo dele. Isso te deixa livre pra trocar a assinatura de `Apply` no dia 40 sem
quebrar ninguém. Quando a arena de bots exigir protocolo público, `internal/protocol` vira
`protocol/` — que é um rename. O caminho contrário é que é breaking change.

## O site dentro do binário

```go
// web/embed.go
//go:embed all:dist
var embedded embed.FS
```

Aquela linha **não é comentário** — é diretiva de compilador. No `go build`, o Go copia os
bytes daquela pasta pra dentro do executável. O `coup` vira um arquivo de ~12 MB que
**contém o site**. Sem pasta pra subir, sem CDN, sem nginx. É o que faz o modo LAN funcionar
num avião.

Servir são duas linhas:

```go
mux.Handle("/", http.FileServerFS(web.Dist()))
```

**O custo honesto:** em desenvolvimento ninguém quer rodar `vite build` a cada `ctrl+s`. Então
o dia a dia é o servidor Vite em `:5173` com proxy de `/ws` pro Go em `:8080`, e o `embed` só
vale na build de release. Dois processos em dev, um só em produção.
