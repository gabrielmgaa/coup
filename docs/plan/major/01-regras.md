# As regras, destrinchadas

Fonte: *Coup — livreto de regras*, 2ª edição brasileira (Mandala Jogos). Tradução de Tatiane
Cappelari. Este documento é a referência do motor: se o código e o livreto discordarem, o
livreto ganha — **exceto** nas divergências deliberadas listadas ao final.

## Setup

| | |
|---|---|
| Baralho base | **15 cartas**: 3 de cada personagem (Duque, Assassino, Capitão, Embaixador, Condessa) |
| Por jogador | 2 cartas viradas para baixo + 2 moedas |
| Com 2 jogadores | 2 cartas + **1 moeda** |
| Resto | Baralho da Corte (viradas para baixo) |
| Tesouro | Modelado como infinito — ver *Divergências* |

> A caixa física tem 25 cartas de personagem (5 de cada) porque suporta as variantes de 7 a 10
> jogadores. O **jogo base usa 15**. O próprio livreto prova, no Exemplo de Jogo: *"Três
> jogadores. Cada um começa com 2 cartas de influência e 2 moedas. As 9 cartas de personagem
> restantes compõem o Baralho da Corte."* — 3 × 2 + 9 = 15.

## Influência

As cartas viradas para baixo são as influências do jogador. Ao perder uma influência, **o
jogador sempre escolhe qual carta revelar**. Carta revelada fica visível pra todos e não
fornece mais influência. Perdeu as duas → exilado, fora do jogo, **devolve todas as moedas ao
Tesouro**.

## O turno

Sentido horário. No seu turno o jogador escolhe **exatamente uma** ação. **Passar é proibido.**

Começou o turno com **10 ou mais moedas** → é **obrigado** a dar um Golpe, e esse é o turno
inteiro dele.

## As 7 ações

| Ação | Custo | Alega | Alvo | Contestável | Bloqueiam | Efeito |
|---|---|---|---|---|---|---|
| Renda | — | — | — | não | — | +1 moeda |
| Ajuda Externa | — | — | — | **não** | Duque | +2 moedas |
| Golpe de Estado | 7 | — | obrigatório | **não** | **ninguém** | alvo perde 1 influência |
| Taxas | — | Duque | — | sim | — | +3 moedas |
| Assassinar | 3 | Assassino | obrigatório | sim | Condessa | alvo perde 1 influência |
| Extorquir | — | Capitão | obrigatório | sim | Capitão, Embaixador | rouba `min(2, moedas do alvo)` |
| Trocar | — | Embaixador | — | sim | — | pega 2 do baralho, devolve 2 |

Ajuda Externa é a exceção que confunde: **não é contestável** (ninguém alegou personagem
nenhum), mas **é bloqueável** por qualquer jogador que alegue Duque.

Golpe é a única ação totalmente imune: não contesta, não bloqueia. Por isso custa 7.

## As 3 contra-ações

| Bloqueio | Quem pode | Contra |
|---|---|---|
| Duque bloqueia Ajuda Externa | **qualquer jogador** | Ajuda Externa |
| Condessa bloqueia Assassinato | **só o alvo** | Assassinar |
| Capitão **ou** Embaixador bloqueia Extorsão | **só o alvo** | Extorquir |

Contra-ações funcionam como ações de personagem: podem ser blefe, não exigem mostrar carta a
menos que contestadas, e **se não contestadas, são automaticamente bem-sucedidas**.

## Contestação

Qualquer ação de personagem **ou contra-ação** pode ser contestada, e **qualquer jogador pode
contestar, envolvido ou não**.

- Contestado, o jogador deve provar mostrando o personagem. Não pôde ou não quis → **perde a
  contestação**.
- **Quem perde a contestação perde uma influência imediatamente.**
- Quem **ganha** a contestação devolve a carta mostrada ao Baralho da Corte, **embaralha, e
  puxa uma nova aleatoriamente**. A mão dele muda no meio da partida.
- **"Contestações são resolvidas antes de qualquer ação ou ação contrária."** Isso é ordem de
  *resolução*, não sequência de janelas — ver Q6 em [`README.md`](README.md).

## As 11 armadilhas

Regras que um motor escrito "de cabeça" erra em silêncio.

1. **Baralho base é 15 cartas, não 25.**
2. **Contestação resolve antes do bloqueio**, sempre.
3. **Bloqueio bem-sucedido NÃO devolve o custo; contestação bem-sucedida DEVOLVE.** Condessa
   bloqueia o assassinato → as 3 moedas continuam gastas. Contestaram o Assassino e era blefe →
   as 3 voltam. Mesma ação derrubada, destino oposto do dinheiro.
4. **Quem ganha contestação troca de carta** (devolve, embaralha, puxa).
5. **Extorquir de quem tem 1 moeda pega 1.** O livreto não cobre o caso de 0 moedas — ver
   *Divergências*.
6. **Dá pra perder 2 influências no mesmo turno** ("perigo duplo do Assassino").
7. **Qualquer um contesta**, envolvido ou não. Mas bloquear Assassinato/Extorsão é **só o
   alvo**; bloquear Ajuda Externa é **qualquer um**.
8. **Partida de 2 começa com 1 moeda**, não 2.
9. **Jogador eliminado devolve as moedas** ao Tesouro.
10. **Passar a vez é proibido.**
11. **10+ moedas no início do turno = Golpe obrigatório**, e é a única ação do turno.

## A árvore completa de uma ação

tester2 tem 5 moedas. Declara **Assassinar** contra tester3, alegando Assassino. Paga 3 → fica
com 2. Abre a **janela 42**: tester3 recebe `[contestar, bloquear_condessa, passar]`; tester5 e
tester4 recebem `[contestar, passar]`.

```
A — ninguém reage (ou estoura o deadline)
    └─ assassinato resolve. tester3 escolhe carta pra revelar.
       tester2: 2 moedas.                                         1 janela

B — tester3 bloqueia com Condessa (foi o primeiro)
    │  janela 42 fecha; o Assassino da tester2 JÁ NÃO PODE ser contestado
    └─ janela 43 abre sobre a Condessa do tester3
       ├─ B1 ninguém contesta → bloqueio vale, assassinato falha.
       │                        As 3 moedas NÃO voltam. tester2: 2.      2 janelas
       ├─ B2 tester2 contesta, tester3 TINHA Condessa → tester2 perde influência.
       │     tester3 devolve a Condessa, embaralha, puxa outra.
       │     Assassinato falha. tester2: 2.                              2 janelas
       └─ B3 tester2 contesta, tester3 BLEFOU → tester3 perde influência (contestação),
             bloqueio falha, assassinato resolve → tester3 perde OUTRA.
             Perigo duplo. Se tinha 2 cartas, está fora.                2 janelas

C — tester5 contesta primeiro
    ├─ C1 tester2 TINHA Assassino → tester5 perde influência.
    │     tester2 devolve o Assassino, embaralha, puxa outra.
    │     A ação sobreviveu, mas o tester3 nunca reagiu
    │     └─ REABRE janela 44, só pro tester3, só [bloquear_condessa, passar]
    │        └─ se ele bloquear, abre a 45 sobre a Condessa dele…     até 4 janelas
    └─ C2 tester2 BLEFOU → tester2 perde influência, ação falha inteira.
          As 3 moedas VOLTAM. tester2: 5.                                1 janela

D — tester3 contesta primeiro
    ├─ D1 tester2 TINHA → tester3 perde influência, assassinato resolve,
    │     tester3 perde OUTRA. Perigo duplo — o caso que o livreto
    │     descreve nominalmente. tester3 NÃO reabre bloqueio: gastou a
    │     reação dele contestando.                                      1 janela
    └─ D2 tester2 BLEFOU → tester2 perde influência, ação falha,
          3 moedas voltam. tester2: 5.                                   1 janela
```

Repare onde o dinheiro diverge: **C2/D2 devolvem as 3 moedas** (contestação derrubou a ação);
**B1/B2 não devolvem** (bloqueio derrubou a ação). Mesmo resultado na mesa, destino oposto do
dinheiro. É a armadilha nº 3, e a tabela de regras do motor concentra ela num lugar só.

## A invariante que o livreto não escreve

Compare **C1** e **D1**: nos dois a tester2 provou o Assassino, mas em C1 o bloqueio reabre pro
tester3 e em D1 não.

**Cada jogador tem uma reação por ação declarada: contestar OU bloquear, nunca as duas.** Em
D1 o tester3 gastou a dele. Em C1 não gastou nada.

O livreto sustenta o lado D1 nominalmente, na nota *Perigo duplo do Assassino*:

> *"se você contestar o Assassino usando contra você e perder, você vai perder uma influência
> pela contestação perdida **e, em seguida, perder uma influência pelo assassinato de
> sucesso**."*

Sobre C1 ele é silencioso. Esta é a interpretação adotada.

## Divergências deliberadas

Cada uma é desvio consciente, não bug. Quem clonar o repositório precisa ver isto.

### 1. `reacoes_independentes` — opção da sala, **desligada por padrão**

Ligada, o tester3 pode contestar, perder uma influência, **e ainda bloquear com Condessa**. O
desfecho que o livreto descreve na nota do Perigo Duplo deixa de existir naquele caminho.

| tester3 tem Condessa | Padrão (livreto) | `reacoes_independentes` |
|---|---|---|
| Bloqueia direto | perde 0 cartas | perde 0 cartas |
| Contesta e tester2 blefou | tester2 perde 1, tester3 0 | igual |
| Contesta e tester2 tinha | perde 1 **e morre** | perde 1, bloqueia, **sobrevive com 1** |

O custo de errar a leitura cai de *morre* pra *perde uma carta*.

### 2. Alvo com 0 moedas não pode ser extorquido

O livreto cobre "se ele só tiver uma moeda, pegue apenas uma" e cala sobre zero. Aqui,
Extorquir contra alvo com 0 moedas é **jogada ilegal**: o motor recusa e a lista de alvos no
snapshot já vem sem ele.

Consequência: se **todos** os outros estiverem com 0 moedas, Extorquir some das opções daquele
turno. Não trava nada — Renda, Ajuda Externa, Taxas, Assassinar, Trocar e Golpe continuam, e
passar continua proibido.

### 3. Tesouro infinito

A caixa tem 24 moedas mais 6 de ouro (valendo 5 cada) = 54 de valor. Com 6 jogadores saem 12
no setup. Nunca seca. O campo `treasury` não existe no estado nem no snapshot.

## Fora do escopo do core

Variante do Inquisidor, variante de 7 a 10 jogadores, e o setup alternativo de 2 jogadores
(dividir em 3 conjuntos de 5, escolher uma carta e descartar o resto). O jogo de 2 jogadores
**está** no core, mas com o setup normal — só a moeda inicial muda.
