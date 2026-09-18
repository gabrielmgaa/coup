# Ordem de construção

Fatia vertical primeiro: a fase 0.1 entrega um jogo **jogável** com duas ações, atravessando
todas as camadas. Da 0.4 em diante, cada regra nova entra num jogo que já funciona.

O motivo é econômico. Se a forma do snapshot estiver errada, você descobre no dia 3 — quando custa
uma tarde — e não no dia 30, quando o motor inteiro já assumiu ela.

Cada fase tem um **pronto quando** verificável. Nenhuma fecha por "deveria funcionar".

---

## 0.1 — Fatia vertical — **pronta** (commits `41506cb`…`f5b3612`)

**Entrega:** dois navegadores entram na mesma sala e jogam uma partida completa com
**Renda** e **Golpe de Estado**, até alguém vencer.

**Existe:** `go.mod`; `cmd/coup` com `serve`; servidor HTTP; `embed.FS` com o build do Vite;
upgrade de WebSocket; `Room` com a goroutine dona, inbox e outbox com backpressure; `engine`
com `Game`, `Phase`, `Apply`, união selada de jogadas, `rules` com duas linhas, baralho,
distribuição inicial, rodízio de turno, condição de vitória; `ViewFor`; mensagens `join` /
`play` / `lose_influence` / `leave` → `update` / `error`; React que desenha a mesa a partir do
snapshot e manda de volta o que foi clicado.

**Desvios decididos na execução, contra o que está escrito acima:**

- **Sem `registry` e sem código de sala.** Uma sala fixa, criada uma vez em `server.New`. O
  `map[code]*Room` e o `create_room` nascem na 0.2, e é lá que este andaime sai.
- **Escolha de qual carta perder puxada da 0.4 pra cá** — sem ela o Golpe não tem o que
  perguntar. Trouxe junto a fase `awaiting_influence_loss` e a mensagem `lose_influence`.
- **`coup_required` (10+ moedas) entrou aqui**, nas duas metades: refusal no motor e `your_actions`
  já vindo só com `coup`.
- **`name_taken` já recusa nome duplicado**, embora o lobby só nasça na 0.2.

**Não tem:** CLI, contestação, bloqueio, deadline, reconexão, pausa, lobby com pronto, revanche.
A partida começa no segundo jogador que entrar.

**Flag de desenvolvimento:** `-starting-coins`. O default é `0` = livreto (2 moedas, 1 no duelo);
`-starting-coins 14` é como se testa uma partida inteira na mão, porque com as moedas do livreto
e só a Renda disponível o segundo Golpe leva ~29 turnos de clique.

**Pronto quando:** duas abas jogam do começo ao fim e a tela diz quem venceu; `go test
./internal/engine` passa; `go vet ./...` limpo. ✅ Verificado também com partida completa pelo
binário e em duas abas de navegador.

---

## 0.2 — Lobby de verdade — **pronta**

**Entrega:** 2 a 6 jogadores, nomes digitados, marcar pronto, host começa.

**Obriga a existir:** `create_room` com código de 4 chars sem ambiguidade; o `map[code]*Room` no
lugar da sala única da 0.1; validação de nome (2–16 caracteres, sem duplicado na sala); campo
`host` com sucessão pro assento mais antigo conectado; `ready`; `start` habilitado só com todos
prontos e ≥2; remoção automática de quem desconectar **no lobby**; erros `room_full`,
`name_taken` (já existe), `invalid_name`, `not_host`.

**Dívida herdada da 0.1, paga aqui:** a validação de nome no `join` e o ponto de teste que
protege os quatro eventos do Golpe contra alvo de uma carta.

**Decisões tomadas na execução, que este documento não previa:**

- **Mensagem `lobby` nova**, separada do `update`. O lobby é estado de sala e o motor não sabe
  o que é host nem código de sala; misturar os dois colocaria conceito de sala dentro do motor.
- **O código da sala viaja na mensagem**, não em `/ws?room=`. O `02-arquitetura.md` mostrava as
  duas coisas; ficou a da tabela do `03-protocolo.md`.
- **`game_started` é código próprio**, separado de `room_full`: "sala lotada" e "chegou tarde"
  são informações diferentes para quem está do outro lado.
- **Sem `welcome` e sem token.** O código chega no primeiro `lobby`; token só com reconexão, na 0.8.
- **`engine.MaxPlayers`**, lido pelo motor e pela sala. `NewGame` recusa mais de 6 em vez de
  estourar o slice do baralho no oitavo jogador.
- **O host é o assento mais antigo conectado**, derivado da ordem da lista — não é campo guardado.

**Pronto quando:** quatro abas entram, uma fecha antes de marcar pronto e some da lista
sozinha, e as três restantes começam a partida. ✅ Verificado no navegador, com as três
mesas mostrando 3 jogadores e 2 moedas cada.

**Fica para a 0.8:** sala vazia continua viva até o processo morrer (o TTL de 30 min é de lá), e
quem cai no meio de uma partida é removido sem a mesa ser avisada — decidir o que os outros veem
nesse instante é o assunto da pausa.

---

## 0.3 — A CLI

**Entrega:** `coup join K7QM` joga a mesma partida que o navegador, na mesma sala.

**Obriga a existir:** `internal/tui` com Bubble Tea; snapshot chegando como `tea.Msg`; `View()`
desenhando a mesa inteira a partir dele; Lipgloss para caixa, cor e alinhamento; `bubbles`
para o viewport com scroll do log; `~/.config/coup/session.json`; `flag` + `switch` em
`os.Args[1]`.

Entra **agora** e não no fim de propósito: com o protocolo ainda de duas ações, a TUI cresce
junto e cada mensagem nova já nasce com as duas pontas. Deixar pro fim é escrever a TUI inteira
de uma vez contra um protocolo pronto.

**Pronto quando:** um jogador no terminal e outro no navegador jogam a mesma partida até o fim,
e os dois veem o mesmo resultado.

---

## 0.4 — Contestação

A fase mais pesada. **Entrega:** **Taxas** (Duque) com toda a maquinaria de janela.

**Obriga a existir:** `Window` com `ID`, `Pending` e `Reacted`; fase `awaiting_response`;
first-responder; fechamento quando `Pending` esvazia; `respond` com `challenge` e `pass`;
resolução de contestação nas duas direções; **quem ganha devolve a carta, embaralha e puxa
outra**; a fase `awaiting_influence_loss` já existe desde a 0.1, com escolha e com resolução automática
quando só resta uma carta; eliminação com devolução de moedas; `your_options` computado no
servidor.

**Pronto quando:** os testes dos galhos C2 e D2 (contestação derruba a ação, custo volta) e do
caso "contestou e perdeu" passam afirmando moedas e influências **por número**; e três abas
jogam uma partida onde alguém blefa Duque, é pego, e perde influência.

---

## 0.5 — Bloqueio

**Entrega:** **Ajuda Externa** e o bloqueio do Duque.

**Obriga a existir:** `BlockedBy` na tabela; janela sobre um bloqueio declarado
(`Window.Block != nil`); a derivação "sem alvo → qualquer um bloqueia"; `respond` com
`block` + `character`; a regra de que **bloqueio bem-sucedido não devolve custo**.

**Pronto quando:** os galhos B1 e B3 passam por número, e uma Ajuda Externa bloqueada por um
terceiro jogador (não o alvo, porque não há alvo) funciona na tela.

---

## 0.6 — Assassinar

Onde as três regras mais traiçoeiras se encontram.

**Entrega:** **Assassinar**, Condessa, e a reabertura da janela.

**Obriga a existir:** custo cobrado na declaração e guardado em `PendingAction.Cost`; a
assimetria completa (contestação devolve, bloqueio não); a derivação "com alvo → só o alvo
bloqueia"; a **reabertura só-bloqueio** do galho C1; o perigo duplo do galho B3; a invariante
`Reacted` separando C1 de D1.

**Pronto quando:** os oito galhos da árvore de [`01-regras.md`](01-regras.md) estão cobertos
por teste nomeado, cada um afirmando o saldo de moedas e a contagem de influências por número.

---

## 0.7 — Extorquir e Trocar

**Entrega:** as duas últimas ações. O jogo está completo em regras.

**Obriga a existir:** `min(2, moedas do alvo)`; `ValidTarget` recusando alvo com 0 moedas e
`your_actions` já vindo sem ele; dois bloqueadores possíveis (Capitão **ou** Embaixador) com o
`character` decidindo a contestação; fase `awaiting_exchange` com compra de 2 e
devolução de 2, inclusive o caso de quem tem uma só influência (1 + 2 = 3, devolve 2, fica com
1).

**Pronto quando:** uma partida de 4 pessoas usa as 7 ações e as 3 contra-ações e termina; e o
Exemplo de Jogo do livreto roda como teste roteirizado, batendo com o resultado impresso lá.

---

## 0.8 — Tempo e rede real

Até aqui ninguém esperou nada e ninguém caiu.

**Entrega:** deadlines, queda, pausa, reconexão.

**Obriga a existir:** `windowDeadline = 25 * time.Second`; `time.AfterFunc` publicando `Timeout`
no inbox; descarte de timeout com `ID` velho; `closes_in_ms` no snapshot e countdown animado nos
dois clientes; detecção de queda; **pausa só quando a partida depende de quem caiu**, com
cancelamento do timer e `paused` no snapshot; 30 s de grace, depois auto-resolve pelo default
seguro (janela → passar; turno → Renda; perda de influência → primeira carta); token de 128
bits; `reconnect` devolvendo o snapshot atual; TTL de 30 min.

**Pronto quando:** você fecha uma aba no meio de uma janela, vê a mesa pausar nas outras,
reabre dentro dos 30 s e volta ao jogo com 25 s cheios; repete deixando estourar e vê a
partida seguir sem você; e `go test -race ./...` passa limpo.

---

## 0.9 — Revanche e acabamento

**Entrega:** o core fechado.

**Obriga a existir:** volta pro lobby no fim da partida, com os mesmos jogadores e o mesmo
código; **o vencedor começa a próxima**, como manda o livreto; `Options.IndependentReactions` e
seus seis pontos de toque (comando de criar sala, campo `options` no snapshot, checkbox no lobby,
flag na CLI, e o galho C1 testado nos dois modos); animações da TUI; README; `LICENSE`;
`goreleaser` ou um `Makefile` produzindo binários pra macOS e Linux.

**Pronto quando:** seis pessoas jogam duas partidas seguidas sem recriar sala, uma delas pela
CLI; o binário roda numa máquina sem Go instalado; e a definição de pronto do core em
[`README.md`](README.md) está satisfeita inteira.

---

## Depois do core

Modo LAN com mDNS — já aprovado como primeira adição, e o `embed.FS` da fase 0.1 já preparou o
terreno. Tudo o mais fica em [`README.md`](README.md), na seção *Fora do core*.
