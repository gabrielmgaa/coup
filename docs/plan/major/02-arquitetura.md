# Arquitetura

Escrito assumindo que você não conhece Go. Cada conceito é explicado onde aparece.

## O desenho, em uma frase

Um processo Go escutando numa porta. Ele serve o site (que está **dentro** do binário) e
aceita conexões WebSocket. Cada sala de jogo é uma goroutine que é dona absoluta daquela
partida; ninguém mexe no jogo direto, todo mundo manda pedido pra ela por uma fila. A cada
pedido aplicado, ela produz uma versão recortada do jogo pra cada jogador e larga na fila de
saída de cada um.

```
processo `coup serve`
│
├── goroutine principal ─ net/http escutando :8080
│     ├── GET /              → index.html (de dentro do binário)
│     ├── GET /assets/*      → JS e CSS (de dentro do binário)
│     └── GET /ws?sala=K7QM  → vira WebSocket
│
├── Registro ─ map[codigo]*Sala, com mutex (só pra criar/achar sala)
│
└── Sala K7QM
      ├── 1 goroutine DONA ("actor")
      │     inbox:  chan Comando     ← todo mundo publica aqui
      │     estado: *coup.Jogo       ← inalcançável de fora
      │
      └── por jogador conectado (4 jogadores = 8 goroutines):
            ├── goroutine leitora    ─ lê do socket → publica no inbox
            ├── goroutine escritora  ─ lê do outbox → escreve no socket
            └── outbox: chan Mensagem (com espaço pra 16)
```

Mesa de 4 = **9 goroutines**. Mesa de 6 = 13. Isso é nada: cada goroutine começa com 2 KB.

O `Registro` é o único lugar do servidor com mutex, e a seção crítica dele é uma busca em map.
Ele nunca toca em estado de partida.

## O que é uma goroutine

Uma goroutine **não é uma thread do sistema operacional**. O runtime do Go multiplexa milhares
delas em poucas threads reais. Criar uma custa nanossegundos e 2 KB.

Isso muda como se escreve I/O. Vindo de JavaScript:

```javascript
// Node: ler de um socket não bloqueia. Registra callback e sai.
socket.on('message', (dados) => { ... })
```

```go
// Go: ler de um socket BLOQUEIA. A linha para ali até chegar byte.
for {
	_, dados, err := conexao.Read(ctx)   // fica parado aqui
	if err != nil {
		return
	}
	sala.inbox <- decodificar(dados)
}
```

Parece ruim e não é: isso roda dentro de `go lerDoSocket(conexao)`, e o runtime **estaciona**
a goroutine enquanto não tem byte, rodando as outras. Você escreve código reto, de cima pra
baixo, sem `async`, sem `await`, sem callback — e ganha concorrência. É por isso que Go faz
"uma goroutine por conexão": é o modelo inteiro.

## O que é um channel

Um `chan T` é uma **fila tipada** entre goroutines.

```go
inbox := make(chan Comando)        // sem espaço: quem põe espera alguém tirar
outbox := make(chan Mensagem, 16)  // com espaço pra 16: só espera se encher

inbox <- Comando{...}   // põe  (seta apontando PRA dentro do canal)
cmd := <-inbox          // tira (seta saindo do canal)
```

E o laço que a goroutine dona roda pra sempre:

```go
for cmd := range sala.inbox {   // tira um, processa, tira o próximo, pra sempre
	...
}
```

`range` num channel é diferente de `range` num slice: não percorre uma coleção, **fica
esperando** o próximo valor chegar, e só termina quando o channel é fechado.

## Por que isso resolve a corrida sozinho

O cenário que mata implementações ingênuas: a janela fecha em 25 s; no segundo **24,998** o
Sérgio clica em contestar; no segundo **25,000** o relógio estoura. Duas coisas querem mexer
no jogo ao mesmo tempo.

Go garante que um channel entrega **um valor por vez, na ordem em que chegaram**. Então a
corrida não é "resolvida" — ela **não existe**. Só há uma mão, a goroutine dona, tirando um
item da fila de cada vez.

```go
func (s *Sala) rodar() {
	for cmd := range s.inbox {
		eventos, err := s.jogo.Aplicar(cmd.Jogada)
		if err != nil {
			s.responderErro(cmd, err)
			continue
		}
		s.difundir(eventos)
	}
}
```

Esse `for` de 9 linhas **é a sala inteira**. Lê de cima a baixo e você sabe tudo que pode
acontecer com uma partida.

## Backpressure: por que tem `select`/`default`

A goroutine dona **nunca escreve na rede** — escrever na rede pode demorar. Ela larga a
mensagem na fila de saída de cada jogador e segue:

```go
func (s *Sala) difundir(eventos []coup.Evento) {
	for _, c := range s.conexoes {
		msg := Mensagem{Estado: coup.Ver(s.jogo, c.nome), Eventos: eventos}
		select {
		case c.outbox <- msg:
			// coube na fila, segue o baile
		default:
			// fila cheia: esse cliente está 16 mensagens atrasado. Derruba.
			s.derrubar(c)
		}
	}
}
```

`select` com `default` significa *"tenta pôr na fila; se for ter que esperar, faz o `default`
em vez disso"*. É o envio que **nunca bloqueia**.

Por que 16: uma partida de Coup tem umas 120 mensagens do servidor. Estar 16 atrás é estar
~13 % da partida atrasado — não é lentidão, é cliente morto. Derruba, e ele reconecta com o
token e recebe a foto atual.

**O ponto estrutural:** não existe *outro lugar* de onde mandar mensagem, então é impossível
esquecer. A alternativa (mutex na sala) exige a disciplina de "nenhuma escrita de rede com o
cadeado na mão", repetida em toda função nova que encostar na sala — e nenhum teste acusa se
você esquecer, porque em `localhost` escrever é instantâneo.

## Pedido que precisa de resposta

"Entrar na sala" precisa de um sim ou não. Com fila, a resposta volta por um channel embutido
no próprio pedido:

```go
type Comando struct {
	Jogada   coup.Jogada
	Resposta chan error   // quem pediu fica esperando aqui
}

// do lado de quem chama (a goroutine da conexão):
resposta := make(chan error, 1)
sala.inbox <- Comando{Jogada: jogada, Resposta: resposta}
if err := <-resposta; err != nil {   // bloqueia até a dona responder
	...
}
```

São ~6 linhas a mais, em duas operações do core: `entrar` e `reconectar`.

## O relógio

O motor **não sabe que horas são**. Ao abrir uma janela ele emite `Janela{ID: 42, Tipo:
Contestacao, Elegiveis: [...]}` e para aí. Quem carimba prazo é o servidor:

```go
timer := time.AfterFunc(janelaPadrao, func() {
	sala.inbox <- Comando{Jogada: coup.Jogada{Tipo: coup.Timeout, Janela: 42}}
})
```

O timeout não é um caminho paralelo — é **mais um pedido na fila**, igual a um clique.

**Por isso o `ID` da janela é obrigatório.** Se o Sérgio contestou aos 24,998 s, a janela 42
já fechou e a 43 abriu. Aos 25,000 s o timer velho dispara, o motor vê `Janela: 42` contra a
corrente 43, e devolve `ErrJanelaFechada`. Jogado fora, sem drama, sem uma linha de trava.

Isso compra a suíte do motor rodando sem fake clock, sem `Sleep` e sem flake. E a duração vira
config do servidor (25 s na web, 0 nos testes de integração) sem tocar em regra nenhuma.

### Pausa

Quando a partida depende de quem caiu (ver Q12), o actor **cancela o `AfterFunc`** e marca a
sala como pausada, com um timer de grace de 30 s.

- **Reconectou dentro dos 30 s** → foto atual, e a janela **reinicia com os 25 s cheios**.
- **Estourou os 30 s** → auto-resolve na hora pelo default seguro, e o jogo segue com ele
  marcado desconectado, **sem pausar de novo**.
- **Ele volta no minuto 10** → o token devolve ele à partida em andamento, na foto atual.

Reiniciar o prazo cheio é menos bookkeeping do que guardar o resto congelado, e a proteção é
real — devolver 2 s pra quem acabou de reconectar não protege ninguém.

## O trace de uma ação, ponta a ponta

Marina (5 moedas) assassina Pedro. Sérgio contesta no último instante.

| tempo | o que acontece |
|---|---|
| `0.000` | Navegador da Marina envia `{"tipo":"jogar","acao":"assassinar","alvo":"pedro"}` |
| `0.001` | Goroutine **leitora** da Marina decodifica e faz `sala.inbox <- Comando{...}` |
| `0.001` | **Actor** tira da fila. `jogo.Aplicar(...)`: Marina tem 5 ≥ 3 → cobra 3, guarda a ação pendente, abre `Janela{ID: 42}`, emite `AcaoDeclarada` |
| `0.001` | Actor agenda `time.AfterFunc(25s, …)` publicando `Timeout{Janela: 42}` no próprio inbox |
| `0.002` | Actor chama `Ver(jogo, nome)` 4 vezes e larga nos 4 outbox. 4 goroutines **escritoras** mandam pela rede |
| `0.002`→`24.99` | **Zero tráfego.** Os 4 clientes animam o countdown localmente a partir do `fecha_em_ms` que veio na foto |
| `24.998` | Sérgio contesta → `sala.inbox <- Comando{Contestar, Janela: 42}` |
| `24.999` | Actor aplica. Marina **tinha** o Assassino: devolve a carta, embaralha, puxa outra; Sérgio perdeu → fase vira `AguardandoPerdaInfluencia{De: "sergio"}`, abre `Janela{ID: 43}` |
| `25.000` | O `AfterFunc` dispara e publica `Timeout{Janela: 42}`. Actor tira da fila, motor vê que a corrente é a 43 → `ErrJanelaFechada`. Ignorado. |

Nessa trilha inteira não se escreveu uma linha de sincronização. Nem `Lock`, nem `Unlock`, nem
`atomic`, nem `-race` reclamando de nada. A fila e o `ID` de janela fizeram o trabalho.

## Layout do repositório

```
coup/
├── cmd/coup/main.go          # o único binário; switch em os.Args[1]
├── internal/
│   ├── engine/               # regras. Sem rede, sem relógio, sem I/O.
│   ├── protocolo/            # os tipos que viajam no fio
│   ├── servidor/             # http, websocket, Registro, Sala (o actor)
│   └── tui/                  # cliente Bubble Tea
├── web/                      # Vite + React + TS
│   └── dist/                 # build; embutido via embed.FS
├── docs/plan/major/
├── go.mod                    # module github.com/gabrielmgaa/coup
├── LICENSE                   # MIT
└── README.md
```

`internal/` **não é convenção** — é regra do compilador: nada fora do módulo consegue importar
um pacote debaixo dele. Isso te deixa livre pra trocar a assinatura de `Aplicar` no dia 40 sem
quebrar ninguém. Quando a arena de bots exigir protocolo público, `internal/protocolo` vira
`protocolo/` — que é um rename. O caminho contrário é que é breaking change.

## O site dentro do binário

```go
//go:embed all:web/dist
var siteEstatico embed.FS
```

Aquela linha **não é comentário** — é diretiva de compilador. No `go build`, o Go copia os
bytes daquela pasta pra dentro do executável. O `coup` vira um arquivo de ~12 MB que
**contém o site**. Sem pasta pra subir, sem CDN, sem nginx. É o que faz o modo LAN funcionar
num avião.

Servir são duas linhas:

```go
sub, _ := fs.Sub(siteEstatico, "web/dist")
mux.Handle("/", http.FileServerFS(sub))
```

**O custo honesto:** em desenvolvimento ninguém quer rodar `vite build` a cada `ctrl+s`. Então
o dia a dia é o servidor Vite em `:5173` com proxy de `/ws` pro Go em `:8080`, e o `embed` só
vale na build de release. Dois processos em dev, um só em produção.
