# Ordem de construção

Fatia vertical primeiro: a fase 0.1 entrega um jogo **jogável** com duas ações, atravessando
todas as camadas. Da 0.4 em diante, cada regra nova entra num jogo que já funciona.

O motivo é econômico. Se a forma da foto estiver errada, você descobre no dia 3 — quando custa
uma tarde — e não no dia 30, quando o motor inteiro já assumiu ela.

Cada fase tem um **pronto quando** verificável. Nenhuma fecha por "deveria funcionar".

---

## 0.1 — Fatia vertical

**Entrega:** dois navegadores entram na mesma sala por código e jogam uma partida completa com
**Renda** e **Golpe de Estado**, até alguém vencer.

**Obriga a existir:** `go.mod`; `cmd/coup` com `serve`; servidor HTTP; `embed.FS` com o build
do Vite; upgrade de WebSocket; `Registro` de salas; `Sala` com a goroutine dona, inbox e outbox
com backpressure; `engine` com `Jogo`, `Fase`, `Aplicar`, união de jogadas, tabela com duas
linhas, baralho, distribuição inicial, rodízio de turno, condição de vitória; `Ver`; mensagens
`entrar` / `jogar` / `atualizacao` / `erro`; React que desenha a mesa a partir do snapshot e
manda de volta o que foi clicado.

**Não tem:** CLI, contestação, bloqueio, prazo, reconexão, pausa, lobby com pronto, revanche.
A partida começa no segundo jogador que entrar.

**Flag de desenvolvimento:** `--moedas-iniciais=6`. Golpe custa 7; começando com 2 você clica
Renda cinco vezes antes de ver qualquer coisa, toda vez que testar. Duas linhas, e some quando
as outras ações existirem.

**Pronto quando:** duas abas jogam do começo ao fim e a tela diz quem venceu; `go test
./internal/engine` passa; `go vet ./...` limpo.

---

## 0.2 — Lobby de verdade

**Entrega:** 2 a 6 jogadores, nomes digitados, marcar pronto, host começa.

**Obriga a existir:** `criar_sala` com código de 4 chars sem ambiguidade; validação de nome
(2–16, sem duplicado na sala); campo `host` com sucessão pro assento mais antigo conectado;
`pronto`; `comecar` habilitado só com todos prontos e ≥2; remoção automática de quem
desconectar **no lobby**; erros `sala_cheia`, `nome_em_uso`, `nome_invalido`, `nao_e_host`.

**Pronto quando:** quatro abas entram, uma fecha antes de marcar pronto e some da lista
sozinha, e as três restantes começam a partida.

---

## 0.3 — A CLI

**Entrega:** `coup join K7QM` joga a mesma partida que o navegador, na mesma sala.

**Obriga a existir:** `internal/tui` com Bubble Tea; snapshot chegando como `tea.Msg`; `View()`
desenhando a mesa inteira a partir dele; Lipgloss para caixa, cor e alinhamento; `bubbles`
para o viewport com scroll do log; `~/.config/coup/sessao.json`; `flag` + `switch` em
`os.Args[1]`.

Entra **agora** e não no fim de propósito: com o protocolo ainda de duas ações, a TUI cresce
junto e cada mensagem nova já nasce com as duas pontas. Deixar pro fim é escrever a TUI inteira
de uma vez contra um protocolo pronto.

**Pronto quando:** um jogador no terminal e outro no navegador jogam a mesma partida até o fim,
e os dois veem o mesmo resultado.

---

## 0.4 — Contestação

A fase mais pesada. **Entrega:** **Taxas** (Duque) com toda a maquinaria de janela.

**Obriga a existir:** `Janela` com `ID`, `Pendentes` e `Reagiram`; fase `AguardandoResposta`;
first-responder; fechamento quando `Pendentes` esvazia; `responder` com `contestar` e `passar`;
resolução de contestação nas duas direções; **quem ganha devolve a carta, embaralha e puxa
outra**; fase `AguardandoPerdaInfluencia` com escolha, e resolução automática quando só resta
uma carta; eliminação com devolução de moedas; `suas_opcoes` computado no servidor.

**Pronto quando:** os testes dos galhos C2 e D2 (contestação derruba a ação, custo volta) e do
caso "contestou e perdeu" passam afirmando moedas e influências **por número**; e três abas
jogam uma partida onde alguém blefa Duque, é pego, e perde influência.

---

## 0.5 — Bloqueio

**Entrega:** **Ajuda Externa** e o bloqueio do Duque.

**Obriga a existir:** `Bloqueiam` na tabela; janela sobre um bloqueio declarado
(`Janela.Bloqueio != nil`); a derivação "sem alvo → qualquer um bloqueia"; `responder` com
`bloquear` + `personagem`; a regra de que **bloqueio bem-sucedido não devolve custo**.

**Pronto quando:** os galhos B1 e B3 passam por número, e uma Ajuda Externa bloqueada por um
terceiro jogador (não o alvo, porque não há alvo) funciona na tela.

---

## 0.6 — Assassinar

Onde as três regras mais traiçoeiras se encontram.

**Entrega:** **Assassinar**, Condessa, e a reabertura da janela.

**Obriga a existir:** custo cobrado na declaração e guardado em `AcaoPendente.Custo`; a
assimetria completa (contestação devolve, bloqueio não); a derivação "com alvo → só o alvo
bloqueia"; a **reabertura só-bloqueio** do galho C1; o perigo duplo do galho B3; a invariante
`Reagiram` separando C1 de D1.

**Pronto quando:** os oito galhos da árvore de [`01-regras.md`](01-regras.md) estão cobertos
por teste nomeado, cada um afirmando o saldo de moedas e a contagem de influências por número.

---

## 0.7 — Extorquir e Trocar

**Entrega:** as duas últimas ações. O jogo está completo em regras.

**Obriga a existir:** `min(2, moedas do alvo)`; `AlvoValido` recusando alvo com 0 moedas e
`suas_acoes` já vindo sem ele; dois bloqueadores possíveis (Capitão **ou** Embaixador) com o
`personagem` decidindo a contestação; fase `AguardandoTrocaEmbaixador` com compra de 2 e
devolução de 2, inclusive o caso de quem tem uma só influência (1 + 2 = 3, devolve 2, fica com
1).

**Pronto quando:** uma partida de 4 pessoas usa as 7 ações e as 3 contra-ações e termina; e o
Exemplo de Jogo do livreto roda como teste roteirizado, batendo com o resultado impresso lá.

---

## 0.8 — Tempo e rede real

Até aqui ninguém esperou nada e ninguém caiu.

**Entrega:** prazos, queda, pausa, reconexão.

**Obriga a existir:** `janelaPadrao = 25 * time.Second`; `time.AfterFunc` publicando `Timeout`
no inbox; descarte de timeout com `ID` velho; `fecha_em_ms` na foto e countdown animado nos
dois clientes; detecção de queda; **pausa só quando a partida depende de quem caiu**, com
cancelamento do timer e `pausada` na foto; 30 s de grace, depois auto-resolve pelo default
seguro (janela → passar; turno → Renda; perda de influência → primeira carta); token de 128
bits; `reconectar` devolvendo a foto atual; TTL de 30 min.

**Pronto quando:** você fecha uma aba no meio de uma janela, vê a mesa pausar nas outras,
reabre dentro dos 30 s e volta ao jogo com 25 s cheios; repete deixando estourar e vê a
partida seguir sem você; e `go test -race ./...` passa limpo.

---

## 0.9 — Revanche e acabamento

**Entrega:** o core fechado.

**Obriga a existir:** volta pro lobby no fim da partida, com os mesmos jogadores e o mesmo
código; **o vencedor começa a próxima**, como manda o livreto; `Regras.ReacoesIndependentes` e
seus seis pontos de toque (comando de criar sala, campo `regras` na foto, checkbox no lobby,
flag na CLI, e o galho C1 testado nos dois modos); animações da TUI; README; `LICENSE`;
`goreleaser` ou um `Makefile` produzindo binários pra macOS e Linux.

**Pronto quando:** seis pessoas jogam duas partidas seguidas sem recriar sala, uma delas pela
CLI; o binário roda numa máquina sem Go instalado; e a definição de pronto do core em
[`README.md`](README.md) está satisfeita inteira.

---

## Depois do core

Modo LAN com mDNS — já aprovado como primeira adição, e o `embed.FS` da fase 0.1 já preparou o
terreno. Tudo o mais fica em [`README.md`](README.md), na seção *Fora do core*.
