# O motor

`internal/engine`. Sem rede, sem clock, sem I/O, sem goroutine. Determinístico dado um
`*rand.Rand`. É o mesmo pacote que servidor, CLI e, um dia, bots vão usar.

## API pública

```go
func NewGame(names []string, rng *rand.Rand, initialCoins int) *Game
func (g *Game) Apply(move Move) ([]Event, error)
func ViewFor(g *Game, name string) View
func (g *Game) OpenWindow() (id int, ok bool)   // 0.4
```

As três primeiras existem hoje. Não há acessor de fase: a fase viaja no snapshot, em `View.Phase`,
porque quem pergunta "que fase é essa?" é sempre quem vai desenhar a tela.

`ViewFor` mora no motor, não no protocolo, porque **informação oculta é regra do Coup**, não
detalhe de transporte. `internal/protocol` embute `engine.View` no envelope e cuida do resto.

## `Move`: a união selada

```go
type Move interface{ sealedMove() }

type Act struct {
	By     string
	Action ActionType
	Target string // vazio quando a ação não tem alvo
}

type LoseInfluence struct {
	By   string
	Card Character
}

func (Act) sealedMove()           {}
func (LoseInfluence) sealedMove() {}
```

Essas duas existem hoje. As outras três entram com as fases que as pedem:

```go
type Respond struct {                 // 0.4
	By        string
	Window    int
	Answer    ResponseType // Challenge | Block | Pass
	Character Character    // só com Block
}

type ReturnCards struct {             // 0.7
	By    string
	Cards [2]Character
}

type Timeout struct {                 // 0.8
	Window int
}
```

Aquele método `sealedMove()` com inicial minúscula **sela** a interface: nenhum pacote de fora
consegue implementar `Move`, porque não consegue nomear o método. `Apply` faz um type
switch e o compilador garante que a lista é fechada.

Isto **não** contraria a Q8. Lá, "interface por ação" foi rejeitada pra *representar as
regras* — o estado de janela pertence ao jogo, não à ação, e espalhar a máquina de estados por
7 arquivos custa caro. Aqui a interface representa a **entrada**, que é genuinamente uma união
de formas diferentes: `Act` carrega alvo, `Timeout` carrega só o ID da janela. Um struct
gordo com 8 campos, dos quais 6 sempre vazios, seria o sintoma que o CLAUDE.md descreve.

## A tabela de regras

```go
type Character uint8

const (
	NoCharacter Character = iota // zero value explícito, senão Duke vira default
	Duke
	Assassin
	Captain
	Ambassador
	Contessa
)

type Rule struct {
	Name        string
	Cost        int
	Claims      Character // NoCharacter = ação geral, não contestável
	NeedsTarget bool
	BlockedBy   []Character              // vazio = imune a bloqueio
	ValidTarget func(target *player) bool // nil = qualquer adversário vivo
	Effect      func(g *Game, by, target int) []Event
}

var rules = map[ActionType]Rule{
	Income:      {Name: "income"},
	ForeignAid:  {Name: "foreign_aid", BlockedBy: []Character{Duke}},
	Coup:        {Name: "coup", Cost: 7, NeedsTarget: true},
	Tax:         {Name: "tax", Claims: Duke},
	Assassinate: {Name: "assassinate", Cost: 3, Claims: Assassin, NeedsTarget: true,
	              BlockedBy: []Character{Contessa}},
	Steal:       {Name: "steal", Claims: Captain, NeedsTarget: true,
	              BlockedBy: []Character{Captain, Ambassador}, ValidTarget: hasCoins},
	Exchange:    {Name: "exchange", Claims: Ambassador},
}
```

Hoje a tabela tem duas linhas, `Income` e `Coup`, e três campos: `Name`, `Cost`, `NeedsTarget`,
`Effect`. `Claims`, `BlockedBy` e `ValidTarget` nascem com as fases 0.4 a 0.7.

Duas coisas caem fora da tabela por serem **deriváveis**, e adicionar campo pra elas seria
duplicar informação:

- **Contestável** = `Claims != NoCharacter`. Ajuda Externa não alega nada, então não é
  contestável — mas é bloqueável, porque `BlockedBy` não está vazio. Golpe tem os dois vazios:
  imune. É exatamente o que o livreto diz, sem escrever nenhum dos dois.
- **Quem pode bloquear** = `NeedsTarget ? só o alvo : qualquer um`. Assassinato e Extorsão têm
  alvo → só ele bloqueia. Ajuda Externa não tem alvo → qualquer Duque na mesa bloqueia. Bate
  com as três linhas de contra-ação do livreto, sem uma terceira coluna.

O pipeline (declara → cobra custo → abre janela → resolve contestação → abre bloqueio →
resolve → aplica efeito → passa turno) é escrito **uma vez** e consulta a tabela. É lá que a
assimetria do dinheiro vive, num lugar só:

```
contestação derrubou a ação → devolve PendingAction.Cost
bloqueio    derrubou a ação → NÃO devolve
```

## Máquina de fases

```go
type Phase uint8

const (
	AwaitingAction Phase = iota
	AwaitingResponse       // 0.4
	AwaitingInfluenceLoss
	AwaitingExchange       // 0.7
	Finished
)
```

Cinco fases, três delas já existindo: `AwaitingAction`, `AwaitingInfluenceLoss`, `Finished`.
**O `Lobby` que este documento previa não é fase do motor** — quem não começou não tem jogo, e
o lobby é `room.game == nil` do lado da sala. O motor só nasce com a partida.

`AwaitingResponse` cobre as três janelas diferentes — a combinada sobre a ação, a
que reabre só-bloqueio, e a que roda sobre um bloqueio declarado. O que diferencia não é a
fase, é o conteúdo da janela:

```go
type Window struct {
	ID      int
	Action  PendingAction
	Block   *PendingBlock   // nil = janela sobre a ação
	Pending map[string]bool // elegíveis que ainda não responderam
	Reacted map[string]bool // quem já gastou a reação nesta ação
}

type PendingAction struct {
	Type   ActionType
	By     string
	Target string
	Cost   int // já cobrado; guardado aqui pra devolução ser uma subtração
}
```

`Reacted` é o que implementa a invariante "uma reação por jogador por ação" (ver
[`01-regras.md`](01-regras.md)). Com `Options.IndependentReactions` ligado, ele é ignorado na
hora de montar `Pending` da janela reaberta.

A janela fecha por **três** motivos, não dois: alguém respondeu (first-responder), `Pending`
esvaziou, ou chegou um `Timeout` com o `ID` corrente.

## Configuração de regras

```go
type Options struct {
	IndependentReactions bool // false = livreto
}
```

Um campo hoje, e `NewGame` **não** recebe isso — a assinatura de hoje é
`NewGame(names, rng, initialCoins)`. Entra na fase 0.9, junto com os pontos de toque fora do motor (comando de criar
sala, campo no snapshot, checkbox no lobby, flag na CLI).

## Aleatoriedade sem interface

O motor precisa de aleatoriedade **durante** a partida, não só no setup: quem ganha uma
contestação devolve a carta, embaralha o Baralho da Corte e puxa outra.

```go
func NewGame(names []string, rng *rand.Rand, initialCoins int) *Game
```

- **Produção:** `rand.New(rand.NewPCG(cryptoSeed(), cryptoSeed()))`
- **Teste:** `rand.New(rand.NewPCG(1, 2))` → a mesma partida, sempre

Nenhuma interface, nenhum mock, nenhum fake. `*rand.Rand` é um tipo concreto da stdlib
(`math/rand/v2`) e o seed é o ponto de injeção.

Pra tornar a mão inicial **legível** no teste, um construtor interno no mesmo pacote:

```go
func newGameWithDeck(names []string, deck []Character) *Game
```

O teste escreve o baralho na ordem que quer, então "tester2 tem Duque e Assassino" está no
código do teste, não escondido atrás de um seed que você descobriu rodando.

## Estratégia de teste

`go test ./internal/engine` fecha em milissegundos. Sem `Sleep`, sem rede, sem clock, sem
`t.Parallel` precisando de cuidado.

Três camadas:

1. **Por ação** — table-driven: custo cobrado, alvo válido, efeito, turno passou.
2. **Por galho da árvore** — cada um dos 8 desfechos de [`01-regras.md`](01-regras.md) (A, B1,
   B2, B3, C1, C2, D1, D2) é um teste nomeado, afirmando moedas e influências **por número**.
   O galho C1 roda duas vezes, com `IndependentReactions` ligado e desligado.
3. **Partida roteirizada** — uma sequência de `Move` do início ao vencedor, conferindo o
   estado final. O Exemplo de Jogo do livreto (tester4, tester5, tester6) vira um desses,
   literalmente: é um roteiro de partida publicado com o resultado esperado impresso.

A **lista de pontos de teste** no formato de seis campos que o CLAUDE.md exige não é escrita
aqui — ela é escrita na hora de executar cada bloco, contra o código que vai existir, e
confirmada antes de qualquer linha. Este documento define o que testar; não como enunciar.

## Limite de arquivo

O pacote fica dividido por responsabilidade, nenhum arquivo passando de ~500 linhas:

```
internal/engine/
├── game.go        # Game, NewGame, Phase, Apply (o type switch e o pipeline)
├── move.go        # a união selada
├── rules.go       # Rule, a tabela, os Effect
├── window.go      # Window, abertura, fechamento, first-responder        (0.4)
├── challenge.go   # contestação, troca de carta, devolução de custo      (0.4)
├── influence.go   # perda, escolha, eliminação, condição de vitória
├── deck.go        # Character, embaralhar, comprar, devolver
├── event.go       # Event e narrate: `n` sequencial + texto pt-BR
├── refusal.go     # Refusal: code, message, received, expected
└── view.go        # View, ViewFor
```
