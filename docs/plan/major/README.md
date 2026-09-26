# Coup — plano do core

Implementação open source do **Coup** (Rikki Tahta / La Mame Games; regras da 2ª edição
brasileira, Mandala Jogos), jogável por navegador e por CLI. Servidor autoritativo em Go,
motor de regras puro, protocolo de eventos tipado.

**Objetivos do projeto, nesta ordem:**

1. Aprender Go de verdade — concorrência, protocolo cliente/servidor, distribuição de binário.
2. Ter um projeto pequeno, terminado e com charme pra publicar.

O core **não pode inflar**. Tudo que não for necessário pra *"amigos entram numa sala e jogam
uma partida completa"* fica pra depois.

## Definição de pronto do core

2 a 6 pessoas abrem um link, entram numa sala por código, jogam uma partida completa de Coup
com as regras base (5 personagens, 7 ações, contestação, bloqueio, deadline por janela), com
reconexão se a conexão cair. A CLI instala como binário único e faz o mesmo como cliente.
Sem conta, sem histórico, sem ranking, sem bots.

## Índice

| Arquivo | O que tem dentro |
|---|---|
| [`01-regras.md`](01-regras.md) | O livreto destrinchado, as 11 armadilhas, a árvore de decisão de uma ação, as regras da casa |
| [`02-arquitetura.md`](02-arquitetura.md) | Goroutines e channels explicados, o actor por sala, backpressure, o trace de uma ação ponta a ponta, layout do repo |
| [`03-protocolo.md`](03-protocolo.md) | Toda mensagem cliente↔servidor, o snapshot completo, como a informação oculta é garantida pelo compilador |
| [`04-motor.md`](04-motor.md) | API do motor, a tabela de regras, a máquina de fases, RNG determinístico, estratégia de teste |
| [`05-fases.md`](05-fases.md) | Ordem de construção, com definição de pronto por fase |

## Decisões

Cada linha foi grelhada antes de entrar aqui. A coluna da direita é o motivo, não a descrição.

### Arquitetura

| # | Decisão | Por quê |
|---|---|---|
| Q1 | **Actor**: 1 owner goroutine por sala, inbox `chan command`, outbox bufferizado (16) por conexão com `select`/`default` | A corrida "alguém responde no mesmo instante em que o deadline estoura" deixa de existir em vez de ser resolvida. E cliente lento fica *estruturalmente* incapaz de congelar a mesa |
| Q2 | **Snapshot personalizado + eventos de narração**, juntos na mesma mensagem | Reconexão vira "manda o snapshot de novo", zero código de remontagem — que seria escrito duas vezes, em Go e em TypeScript |
| — | **Motor sem clock.** `Apply` é pura; deadlines são do servidor | A suíte do motor roda sem fake clock, sem `Sleep`, sem flake. E a duração vira config sem tocar em regra |
| Q4 | **Next.js (export estático) + Tailwind + TS**, embutido via `embed.FS` | Decidido pelo dono do projeto depois do core, trocando o Vite da decisão original. Continua um binário só: `output: 'export'` gera HTML e JS estáticos, sem SSR nem rota de servidor — tudo segue vivendo no WebSocket |
| Q5 | **Bubble Tea + Lipgloss** na CLI | `Model`/`Update`/`View` casa 1:1 com o snapshot da Q2. Alternativas reintroduzem na CLI a reconciliação que o protocolo eliminou |
| Q8 | **Tabela de dados** para as 7 ações e 3 contra-ações | A assimetria do dinheiro (contestação devolve o custo, bloqueio não) mora num lugar só |
| Q17 | **`cmd/coup` + `internal/{engine,protocol,server,tui}` + `web/`** | `internal/` é regra do compilador, não convenção: ninguém de fora importa. Sair de `internal/` é rename; entrar é breaking change |

### Regras de jogo

| # | Decisão | Por quê |
|---|---|---|
| Q3 | **Janela first-responder**: o primeiro que reage resolve; fecha cedo se todos passaram | É o que acontece na mesa física, e ninguém espera deadline à toa |
| Q6 | **Uma janela combinada** (contestar e bloquear juntos), que **reabre só-bloqueio** se a ação sobreviver a uma contestação | Fiel ao livreto: quem falar primeiro fala. Duas janelas em sequência seriam convenção de casa, não regra |
| Q9/Q10 | **Uma reação por jogador** (contestar OU bloquear) por padrão; `independent_reactions` como opção da sala | O default é o jogo impresso na caixa. A divergência é opt-in e fica documentada como tal |
| — | **Extorquir** = `min(2, moedas do alvo)`; **alvo com 0 moedas não é alvo válido** | O livreto só cobre o caso "1 moeda" |
| — | **Tesouro infinito** | A caixa física tem 54 de valor em moedas; com 6 jogadores nunca seca. Modelar é overhead puro |
| Q16 | **25 s pra toda decisão**, valor único em um lugar só | Começa simples. Se incomodar, vira `map[Phase]time.Duration` sem tocar em regra |

### Sala

| # | Decisão | Por quê |
|---|---|---|
| Q14 | **Todos marcam pronto + host clica em começar** | Combina o sinal de presença com alguém dono da decisão |
| Q18 | **Desconectado é removido do lobby** automaticamente | Destrava o caso "entrou e fechou a aba", reusando o detector de queda que já vai existir |
| Q7/Q12 | **Pausa só quando o jogo depende de quem caiu** | A maioria das quedas numa partida real não bloqueia nada. Congelar por elas para a mesa à toa |
| Q11 | 30 s de pausa → **auto-resolve** pelo default seguro → segue sem pausar de novo; **reconexão a qualquer momento** | A partida sempre termina, e o grace period é tolerância a troca de wifi, não veículo pra sequestrar a mesa |
| Q15 | **Volta pro lobby**, host reinicia; **vencedor começa a próxima** (regra do livreto) | ~20 linhas, nenhum conceito novo, e remove 2 minutos de atrito entre cada partida da noite |
| — | **TTL de 30 min** sem conexão viva destrói a sala | |
| Q13 | **Fatia vertical fina primeiro** | Descobrir que a forma do snapshot está errada custa uma tarde no dia 3 e uma semana no dia 30 |

### Suposições pequenas, já fechadas

- **Transporte:** WebSocket, `github.com/coder/websocket` — API pequena, context-aware, idiomática com `net/http`.
- **Wire format:** `encoding/json`, tagged union com campo `type`.
- **Identidade:** token opaco de 128 bits gerado no join. `localStorage` na web, `~/.config/coup/session.json` na CLI.
- **Código de sala:** 4 chars de alfabeto sem ambiguidade (sem `O/0`, `I/1`).
- **URL da sala:** `/?room=K7QM` — query param, não rota. Não tem router no SPA e não precisa ter.
- **RNG:** `*rand.Rand` injetado no motor. Teste passa `rand.New(rand.NewPCG(1, 2))`; produção semeia do `crypto/rand`.
- **Nome do jogador:** 2–16 caracteres, duplicado na mesma sala é recusado.
- **Primeira partida** sorteia quem começa; da segunda em diante, quem venceu a anterior.
- **CLI:** `flag` da stdlib + `switch` em `os.Args[1]`. Três comandos não pagam `cobra`.
- **Módulo:** `github.com/gabrielmgaa/coup`. **Licença:** MIT.

## Fora do core, explicitamente

Nada disto entra antes do core fechar: modo LAN com mDNS, bots, Elo, histórico, estatísticas
de blefe, espectador, bot LLM, arena de bots, variantes (Inquisidor, Reforma), 7–10 jogadores,
temas, som, i18n. Descartado de vez: SSH, chat de voz, editor de cartas.

## Nota legal

Regras de jogo não são protegidas por copyright, então uma implementação limpa é legítima.
O que **não** entra no repositório é a arte da caixa. O README credita Rikki Tahta, La Mame
Games, Indie Boards & Cards e Mandala Jogos.
