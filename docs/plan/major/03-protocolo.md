# Protocolo

WebSocket em `/ws`. JSON, `encoding/json`, tagged union pelo campo `tipo`. Uma conexão por
jogador, aberta uma vez e mantida aberta.

## O princípio que rege tudo: o cliente não sabe as regras

O servidor manda **quais ações são legais agora** (`suas_acoes`) e **quais respostas cabem
nesta janela** (`suas_opcoes`). O React e o Bubble Tea são renderizadores puros: desenham
botões a partir de uma lista e mandam de volta o que foi clicado.

Sem isso, "Extorquir não pode mirar quem tem 0 moedas" precisaria ser escrito três vezes — em
Go no motor, em TypeScript no navegador, em Go de novo na TUI — e divergiria. Com isso, é
escrito uma vez.

Mesmo motivo do snapshot em vez de delta: **regra e estado moram no servidor, ponto.**

## Cliente → servidor

| `tipo` | Campos | Quando |
|---|---|---|
| `criar_sala` | `nome`, `regras?` | Sem sala ainda. Responde `bem_vindo` com o código gerado |
| `entrar` | `sala`, `nome` | Entrar numa sala existente |
| `reconectar` | `token` | Retomar a sessão de antes |
| `pronto` | `pronto: bool` | Só no lobby |
| `comecar` | — | Só o host, só com todos prontos e ≥2 jogadores |
| `jogar` | `acao`, `alvo?` | Só no seu turno |
| `responder` | `janela`, `resposta`, `personagem?` | `resposta` ∈ `contestar` / `bloquear` / `passar` |
| `perder_influencia` | `carta` | Quando a fase pede que você escolha |
| `devolver_cartas` | `cartas: [duas]` | Após Trocar (Embaixador) |
| `sair` | — | Sai da sala |

`personagem` só acompanha `bloquear`, porque Extorsão aceita dois bloqueadores diferentes
(Capitão ou Embaixador) e o motor precisa saber qual foi alegado pra resolver a contestação.

## Servidor → cliente

| `tipo` | Campos |
|---|---|
| `bem_vindo` | `token`, `voce`, `sala` |
| `atualizacao` | `estado`, `eventos` |
| `erro` | `codigo`, `mensagem`, `recebido`, `esperado` |

`estado` e `eventos` viajam **na mesma mensagem**, sempre. Nunca há estado sem a narração do
que causou ele, nem narração sem o estado resultante.

## O snapshot

Uma foto completa, recortada pro destinatário. ~700 bytes numa mesa de 6; uma partida inteira
gasta ~84 KB. Mandar tudo a cada ação é irrelevante.

```json
{
  "tipo": "atualizacao",
  "estado": {
    "sala": "K7QM",
    "fase": "aguardando_resposta",
    "voce": "pedro",
    "host": "marina",
    "regras": { "reacoes_independentes": false },
    "vez_de": "marina",
    "baralho_restante": 7,
    "pausada": null,
    "vencedor": null,
    "jogadores": [
      { "nome": "marina",  "moedas": 2, "ocultas": 2, "reveladas": [],
        "conectado": true,  "pronto": true, "eliminado": false },
      { "nome": "pedro",   "moedas": 3, "ocultas": 2, "reveladas": [],
        "conectado": true,  "pronto": true, "eliminado": false,
        "minhas_cartas": ["condessa", "capitao"] },
      { "nome": "sergio",  "moedas": 2, "ocultas": 1, "reveladas": ["assassino"],
        "conectado": true,  "pronto": true, "eliminado": false },
      { "nome": "vanessa", "moedas": 0, "ocultas": 2, "reveladas": [],
        "conectado": false, "pronto": true, "eliminado": false }
    ],
    "janela": {
      "id": 42,
      "acao": { "nome": "assassinar", "de": "marina", "alvo": "pedro", "alega": "assassino" },
      "bloqueio": null,
      "suas_opcoes": ["contestar", "bloquear_com_condessa", "passar"],
      "ja_responderam": ["vanessa"],
      "fecha_em_ms": 25000
    },
    "suas_acoes": null
  },
  "eventos": [
    { "n": 47, "tipo": "acao_declarada",
      "texto": "Marina pagou 3 e alegou Assassino contra Pedro.",
      "dados": { "de": "marina", "acao": "assassinar", "alvo": "pedro", "custo": 3 } }
  ]
}
```

### A mesma foto, mandada pro Sérgio

Muda em três lugares, e é aí que a informação oculta acontece:

```json
  "voce": "sergio",
    { "nome": "pedro",  "moedas": 3, "ocultas": 2, "reveladas": [] },
    { "nome": "sergio", "moedas": 2, "ocultas": 1, "reveladas": ["assassino"],
      "minhas_cartas": ["duque"] },
  "janela": { "suas_opcoes": ["contestar", "passar"] }
```

O Sérgio vê que o Pedro tem 2 cartas ocultas, **nunca quais são** — `minhas_cartas` só existe
na entrada dele mesmo. E ele não recebe `bloquear_com_condessa` porque bloquear Assassinato é
só do alvo.

### Quando é a sua vez

`janela` vem `null` e `suas_acoes` vem preenchido, já filtrado pelo que é legal:

```json
"fase": "aguardando_acao",
"vez_de": "pedro",
"suas_acoes": [
  { "nome": "renda" },
  { "nome": "ajuda_externa" },
  { "nome": "taxas" },
  { "nome": "trocar" },
  { "nome": "extorquir",  "alvos": ["marina", "sergio"] },
  { "nome": "assassinar", "alvos": ["marina", "sergio", "vanessa"], "custo": 3 }
]
```

Vanessa não aparece nos alvos de `extorquir` porque está com 0 moedas. `golpe` não aparece
porque Pedro tem menos de 7. Se Pedro tivesse 10+, a lista teria **só** `golpe`.

### Sala pausada

```json
"pausada": { "esperando": "pedro", "retoma_em_ms": 30000 }
```

Presente só quando a partida trava por queda de quem tem decisão pendente. Enquanto está
presente, `janela.fecha_em_ms` não corre.

## Eventos

```json
{ "n": 47, "tipo": "acao_declarada", "texto": "...", "dados": { ... } }
```

- `n` é sequencial por sala e nunca reseta durante a partida. É o log que o projeto quer desde
  o começo, mesmo hoje descartado no fim.
- `texto` é o que a CLI imprime no log e a web mostra no feed. Vem pronto do servidor, em
  pt-BR, porque montar a frase no cliente significaria montá-la **duas vezes**, em Go e em TS.
- `dados` é o que a interface usa pra animar (qual carta virou, quem perdeu, quanto de dinheiro
  andou).

Quando i18n entrar (fase futura), o cliente monta a string a partir de `tipo` + `dados` e
`texto` vira fallback. Até lá, `texto` é a fonte.

**Tipos de evento:** `partida_iniciada`, `acao_declarada`, `janela_aberta`, `passou`,
`contestou`, `contestacao_ganha`, `contestacao_perdida`, `bloqueou`, `bloqueio_valeu`,
`bloqueio_falhou`, `influencia_perdida`, `carta_trocada`, `moedas_devolvidas`, `acao_resolvida`,
`jogador_eliminado`, `turno_passou`, `partida_terminada`, `jogador_caiu`, `jogador_voltou`,
`sala_pausada`, `sala_retomada`, `auto_resolvido`.

## Erros

Formato fixo, sempre com o valor recebido e o esperado:

```json
{ "tipo": "erro", "codigo": "janela_fechada",
  "mensagem": "a janela 42 já fechou",
  "recebido": 42, "esperado": 43 }
```

| `codigo` | Significa |
|---|---|
| `sala_nao_encontrada` | código de sala não existe ou expirou |
| `sala_cheia` | já tem 6 |
| `nome_em_uso` | duplicado nesta sala |
| `nome_invalido` | fora de 2–16 caracteres |
| `nao_e_sua_vez` | jogou fora do turno |
| `acao_ilegal` | ação não está em `suas_acoes` |
| `alvo_invalido` | alvo não está na lista daquela ação |
| `moedas_insuficientes` | custo maior que o saldo |
| `golpe_obrigatorio` | tem 10+ moedas e tentou outra coisa |
| `janela_fechada` | respondeu a uma janela que já resolveu |
| `ja_respondeu` | segunda resposta na mesma janela |
| `nao_e_host` | tentou `comecar` sem ser host |
| `token_invalido` | reconexão com token desconhecido |

Um erro é sempre resposta a **uma** mensagem daquele cliente, e nunca é difundido.

## Informação oculta, garantida pelo compilador

São **dois tipos Go diferentes**, e não é convenção — é o compilador que impede o vazamento.

```go
// internal/engine/jogo.go — a VERDADE. Nunca sai do servidor.
type Jogo struct {
	jogadores []jogador   // minúsculo = não exportado
	baralho   []Carta     // minúsculo = não exportado
	fase      Fase
	janela    *Janela
}

// internal/protocolo/visao.go — o que VAI pro cliente.
type Visao struct {
	Voce      string         `json:"voce"`
	VezDe     string         `json:"vez_de"`
	Jogadores []JogadorVisto `json:"jogadores"`
	Janela    *JanelaVista   `json:"janela,omitempty"`
}
```

Maiúscula/minúscula **é** o sistema de visibilidade do Go: campo com inicial minúscula é
privado ao pacote. E `encoding/json` **só serializa campos exportados**. Então
`json.Marshal(jogo)` devolve literalmente `{}` — é *impossível* vazar a mão de alguém por
acidente, mesmo escrevendo o código errado.

Aquela `` `json:"vez_de"` `` é uma **struct tag**: metadado colado no campo dizendo ao
`encoding/json` como nomear ele no JSON.

A projeção é uma função pura:

```go
func Ver(j *Jogo, jogador string) protocolo.Visao
```

## Reconexão

1. No primeiro `entrar` ou `criar_sala`, o servidor responde `bem_vindo` com um token opaco de
   128 bits. Web guarda em `localStorage`, CLI em `~/.config/coup/sessao.json`.
2. Caiu a conexão → o cliente reabre o WebSocket e manda `reconectar {token}`.
3. O servidor associa o token ao assento, marca `conectado: true` e responde `atualizacao` com
   a foto atual. Acabou.

Não existe replay, não existe "me manda do evento 47 em diante", não existe histórico guardado
por sala pra isso. A foto **é** o estado.

O token vale enquanto a sala viver (TTL de 30 min sem nenhuma conexão). Reconexão depois disso
devolve `sala_nao_encontrada`.
