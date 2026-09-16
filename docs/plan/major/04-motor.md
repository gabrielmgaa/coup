# O motor

`internal/engine`. Sem rede, sem relógio, sem I/O, sem goroutine. Determinístico dado um
`*rand.Rand`. É o mesmo pacote que servidor, CLI e, um dia, bots vão usar.

## API pública

```go
func NovoJogo(nomes []string, regras Regras, rng *rand.Rand) *Jogo
func (j *Jogo) Aplicar(jogada Jogada) ([]Evento, error)
func (j *Jogo) Fase() Fase
func (j *Jogo) JanelaAberta() (id int, ok bool)
func Ver(j *Jogo, jogador string) Visao
```

`Ver` mora no motor, não no protocolo, porque **informação oculta é regra do Coup**, não
detalhe de transporte. `internal/protocolo` embute `engine.Visao` no envelope e cuida do resto.

## Jogada: união selada

```go
type Jogada interface{ jogada() }

type Agir struct {
	De   string
	Acao TipoAcao
	Alvo string // vazio quando a ação não tem alvo
}

type Responder struct {
	De         string
	Janela     int
	Resposta   TipoResposta // Contestar | Bloquear | Passar
	Personagem Personagem   // só com Bloquear
}

type PerderInfluencia struct {
	De    string
	Carta Personagem
}

type DevolverCartas struct {
	De     string
	Cartas [2]Personagem
}

type Timeout struct {
	Janela int
}

func (Agir) jogada()             {}
func (Responder) jogada()        {}
func (PerderInfluencia) jogada() {}
func (DevolverCartas) jogada()   {}
func (Timeout) jogada()          {}
```

Aquele método `jogada()` com inicial minúscula **sela** a interface: nenhum pacote de fora
consegue implementar `Jogada`, porque não consegue nomear o método. `Aplicar` faz um type
switch e o compilador garante que a lista é fechada.

Isto **não** contraria a Q8. Lá, "interface por ação" foi rejeitada pra *representar as
regras* — o estado de janela pertence ao jogo, não à ação, e espalhar a máquina de estados por
7 arquivos custa caro. Aqui a interface representa a **entrada**, que é genuinamente uma união
de formas diferentes: `Agir` carrega alvo, `Timeout` carrega só o ID da janela. Um struct
gordo com 8 campos, dos quais 6 sempre vazios, seria o sintoma que o CLAUDE.md descreve.

## A tabela de regras

```go
type Personagem uint8

const (
	NenhumPersonagem Personagem = iota // zero value explícito, senão Duque vira default
	Duque
	Assassino
	Capitao
	Embaixador
	Condessa
)

type Regra struct {
	Nome       string
	Custo      int
	Alega      Personagem   // NenhumPersonagem = ação geral, não contestável
	ExigeAlvo  bool
	Bloqueiam  []Personagem // vazio = imune a bloqueio
	AlvoValido func(alvo *jogador) bool // nil = qualquer adversário vivo
	Efeito     func(j *Jogo, de, alvo int)
}

var tabela = map[TipoAcao]Regra{
	Renda:        {Nome: "renda"},
	AjudaExterna: {Nome: "ajuda_externa", Bloqueiam: []Personagem{Duque}},
	Golpe:        {Nome: "golpe", Custo: 7, ExigeAlvo: true},
	Taxas:        {Nome: "taxas", Alega: Duque},
	Assassinar:   {Nome: "assassinar", Custo: 3, Alega: Assassino, ExigeAlvo: true,
	               Bloqueiam: []Personagem{Condessa}},
	Extorquir:    {Nome: "extorquir", Alega: Capitao, ExigeAlvo: true,
	               Bloqueiam: []Personagem{Capitao, Embaixador}, AlvoValido: temMoeda},
	Trocar:       {Nome: "trocar", Alega: Embaixador},
}
```

Duas coisas caem fora da tabela por serem **deriváveis**, e adicionar campo pra elas seria
duplicar informação:

- **Contestável** = `Alega != NenhumPersonagem`. Ajuda Externa não alega nada, então não é
  contestável — mas é bloqueável, porque `Bloqueiam` não está vazio. Golpe tem os dois vazios:
  imune. É exatamente o que o livreto diz, sem escrever nenhum dos dois.
- **Quem pode bloquear** = `ExigeAlvo ? só o alvo : qualquer um`. Assassinato e Extorsão têm
  alvo → só ele bloqueia. Ajuda Externa não tem alvo → qualquer Duque na mesa bloqueia. Bate
  com as três linhas de contra-ação do livreto, sem uma terceira coluna.

O pipeline (declara → cobra custo → abre janela → resolve contestação → abre bloqueio →
resolve → aplica efeito → passa turno) é escrito **uma vez** e consulta a tabela. É lá que a
assimetria do dinheiro vive, num lugar só:

```
contestação derrubou a ação → devolve AcaoPendente.Custo
bloqueio    derrubou a ação → NÃO devolve
```

## Máquina de fases

```go
type Fase uint8

const (
	Lobby Fase = iota
	AguardandoAcao
	AguardandoResposta
	AguardandoPerdaInfluencia
	AguardandoTrocaEmbaixador
	Terminado
)
```

Seis fases. `AguardandoResposta` cobre as três janelas diferentes — a combinada sobre a ação, a
que reabre só-bloqueio, e a que roda sobre um bloqueio declarado. O que diferencia não é a
fase, é o conteúdo da janela:

```go
type Janela struct {
	ID        int
	Acao      AcaoPendente
	Bloqueio  *BloqueioPendente // nil = janela sobre a ação
	Pendentes map[string]bool   // elegíveis que ainda não responderam
	Reagiram  map[string]bool   // quem já gastou a reação nesta ação
}

type AcaoPendente struct {
	Tipo  TipoAcao
	De    string
	Alvo  string
	Custo int // já cobrado; guardado aqui pra devolução ser uma subtração
}
```

`Reagiram` é o que implementa a invariante "uma reação por jogador por ação" (ver
[`01-regras.md`](01-regras.md)). Com `Regras.ReacoesIndependentes` ligado, ele é ignorado na
hora de montar `Pendentes` da janela reaberta.

A janela fecha por **três** motivos, não dois: alguém respondeu (first-responder), `Pendentes`
esvaziou, ou chegou um `Timeout` com o `ID` corrente.

## Configuração de regras

```go
type Regras struct {
	ReacoesIndependentes bool // false = livreto
}
```

Um campo hoje. Entra na fase 0.9, junto com os pontos de toque fora do motor (comando de criar
sala, campo na foto, checkbox no lobby, flag na CLI).

## Aleatoriedade sem interface

O motor precisa de aleatoriedade **durante** a partida, não só no setup: quem ganha uma
contestação devolve a carta, embaralha o Baralho da Corte e puxa outra.

```go
func NovoJogo(nomes []string, regras Regras, rng *rand.Rand) *Jogo
```

- **Produção:** `rand.New(rand.NewPCG(cryptoSeed(), cryptoSeed()))`
- **Teste:** `rand.New(rand.NewPCG(1, 2))` → a mesma partida, sempre

Nenhuma interface, nenhum mock, nenhum fake. `*rand.Rand` é um tipo concreto da stdlib
(`math/rand/v2`) e a semente é o ponto de injeção.

Pra tornar a mão inicial **legível** no teste, um construtor interno no mesmo pacote:

```go
func novoJogoComBaralho(nomes []string, regras Regras, baralho []Carta, rng *rand.Rand) *Jogo
```

O teste escreve o baralho na ordem que quer, então "Marina tem Duque e Assassino" está no
código do teste, não escondido atrás de uma semente que você descobriu rodando.

## Estratégia de teste

`go test ./internal/engine` fecha em milissegundos. Sem `Sleep`, sem rede, sem relógio, sem
`t.Parallel` precisando de cuidado.

Três camadas:

1. **Por ação** — table-driven: custo cobrado, alvo válido, efeito, turno passou.
2. **Por galho da árvore** — cada um dos 8 desfechos de [`01-regras.md`](01-regras.md) (A, B1,
   B2, B3, C1, C2, D1, D2) é um teste nomeado, afirmando moedas e influências **por número**.
   O galho C1 roda duas vezes, com `ReacoesIndependentes` ligado e desligado.
3. **Partida roteirizada** — uma sequência de `Jogada` do início ao vencedor, conferindo o
   estado final. O Exemplo de Jogo do livreto (Vanessa, Sérgio, Roberto) vira um desses,
   literalmente: é um roteiro de partida publicado com o resultado esperado impresso.

A **lista de pontos de teste** no formato de seis campos que o CLAUDE.md exige não é escrita
aqui — ela é escrita na hora de executar cada bloco, contra o código que vai existir, e
confirmada antes de qualquer linha. Este documento define o que testar; não como enunciar.

## Limite de arquivo

O pacote fica dividido por responsabilidade, nenhum arquivo passando de ~500 linhas:

```
internal/engine/
├── jogo.go        # Jogo, NovoJogo, Fase, Aplicar (o type switch e o pipeline)
├── jogada.go      # a união selada
├── tabela.go      # Regra, a tabela, os Efeito
├── janela.go      # Janela, abertura, fechamento, first-responder
├── contestacao.go # resolver contestação, troca de carta, devolução de custo
├── influencia.go  # perda, escolha, eliminação, condição de vitória
├── baralho.go     # Carta, embaralhar, comprar, devolver
└── visao.go       # Visao, Ver
```
