# Conexão WebSocket morta ou surda: como detectar e limpar

Pesquisa de 2026-09-26 para o servidor Go do coup (`internal/server`, `github.com/coder/websocket`
v1.8.15). A pergunta: qual o melhor jeito de perceber que um cliente morreu, sumiu da rede ou
parou de ler, e quais números de keepalive usar. Cada afirmação cita a fonte dona dela; a lista
numerada está em **Fontes**, no fim. Marcas: **[F]** a fonte diz, **[M]** medi eu mesmo, com scripts
locais que ficaram fora do repositório, **[I]** inferência minha. "Não encontrei" quer dizer isso mesmo.

## Resumo

**O que escolhi:** (a) prazo de escrita de 10 s **+** (c) ping do protocolo a cada 15 s com espera
de pong de 10 s, disparado pelo servidor. Mantive o keepalive TCP padrão do Go, não mexi em
`TCP_USER_TIMEOUT` e deixei o heartbeat de aplicação para depois. Pior caso até detectar: 25 s. O
custo é uma goroutine e um ticker por conexão, e ~10–18 bytes de WebSocket a cada 15 s.

| Opção | O que detecta | Tempo até detectar | Custo |
|---|---|---|---|
| **(a)** prazo por escrita (10 s) | cliente que não drena **e** já tem 0,55–1,85 MB parados nos buffers [M] | buffer cheio + 10 s. No ritmo do jogo (~700 B por jogada), isso é depois de ~790–2640 jogadas: na prática nunca. Consumidor lento: nunca, porque cada escrita volta em 0,5–1,3 s [M] | ~5 linhas em `write`; um timer por escrita |
| **(b)** (a) + keepalive TCP ajustado / `TCP_USER_TIMEOUT` | host que sumiu, com a conexão **ociosa** (keepalive); dado sem ACK ou janela zero (`TCP_USER_TIMEOUT`, só Linux) | keepalive padrão do Go: 15 s + 9 × 15 s = 150 s, e só se não houver dado pendente [F 6, 7] | só enxerga o salto até o proxy; só Linux; função `Control` + `golang.org/x/sys/unix` |
| **(c)** (a) + ping do protocolo (15 s / 10 s) | host sumido, meia-conexão, processo que não lê (cliente Go), navegador com buffer interno cheio, consumidor lento | ≤ 25 s. Medido com espera de 2 s: 1ª falha ~3 s depois de o cliente parar [M] | 1 goroutine + 1 ticker por conexão (~3,3 KB [M]); ~170–220 B no fio por par [I] |
| **(d)** heartbeat de aplicação | tudo acima **+** JS travado ou aba congelada; e deixa o *cliente* detectar servidor morto | período + espera | mensagem nova no protocolo, código no site e no terminal, falso positivo em aba escondida |

O que nenhuma opção de (a) a (c) pega: navegador com JS travado ou aba congelada, **com tráfego
baixo**. O Chromium responde ping fora do JS, então a conexão parece viva [M, F 26]. Nesse caso
quem protege a mesa é o prazo de 25 s de cada decisão, que já existe.

## 1. RFC 6455: ping e pong

- **Quem responde:** "Upon receipt of a Ping frame, an endpoint MUST send a Pong frame in response,
  unless it already received a Close frame. It SHOULD respond with Pong frame as soon as is
  practical." Qualquer lado pode pingar, a qualquer momento depois de aberta a conexão [F 1, §5.5.2].
- **Pong sem pedido:** "A Pong frame MAY be sent unsolicited. This serves as a unidirectional
  heartbeat. A response to an unsolicited Pong frame is not expected." O pong ecoa o payload do
  ping. Se vários pings chegarem antes de responder, basta o pong do último [F 1, §5.5.3].
- **Tamanho:** "All control frames MUST have a payload length of 125 bytes or less and MUST NOT be
  fragmented." Controle pode ser intercalado no meio de uma mensagem fragmentada [F 1, §5.5]. A
  coder/websocket usa o mesmo limite, `maxControlPayload = 125` [F 9, frame.go#L104-L105].
- **Máscara:** o cliente "MUST mask all frames that it sends to the server"; o servidor "MUST NOT
  mask" [F 1, §5.1]. A chave tem 4 bytes [F 1, §5.2].
- **Bytes no fio:** um ping do servidor tem 2 bytes de cabeçalho + payload. A coder/websocket usa
  como payload um contador decimal (`strconv.FormatInt(p, 10)`, conn.go#L223-L230), então de 1 a
  5 bytes: **3–7 bytes**. O pong do navegador tem 2 + 4 (máscara) + o mesmo payload: **7–11
  bytes** [I, derivado de F 1 e F 9].

## 2. O que as bibliotecas maduras fazem

| Biblioteca | Quem pinga | Números padrão | Fonte |
|---|---|---|---|
| gorilla/websocket (exemplo do chat) | servidor, frame de protocolo | `writeWait` 10 s, `pongWait` 60 s, `pingPeriod` 54 s (9/10 de pongWait), `maxMessageSize` 512 B | [F 11] |
| coder/websocket | ninguém automaticamente | `Ping` manual; no navegador (Wasm) `Ping` é no-op | [F 9, F 10] |
| Socket.IO v4 / Engine.IO v4 | **servidor**, pacote de aplicação | `pingInterval` 25 s, `pingTimeout` 20 s, `maxHttpBufferSize` 1 MB | [F 12] |
| Phoenix Channels | **cliente** JS, mensagem de aplicação | heartbeat 30 s, o cliente espera 30 s pela resposta; servidor derruba após 60 s sem receber nada | [F 13] |
| ASP.NET Core SignalR | os dois lados, mensagem de aplicação | `KeepAliveInterval` 15 s, `ClientTimeoutInterval` 30 s, `HandshakeTimeout` 15 s; cliente JS: 15 s / 30 s | [F 14] |
| Node `ws` (README) | servidor, frame de protocolo | intervalo de 30 s, `terminate()` quem não respondeu ao ping anterior | [F 15] |
| Centrifuge / Centrifugo | servidor, mensagem de aplicação | ping 25 s; pong 10 s (biblioteca) / 8 s (Centrifugo) | [F 16] |
| Python `websockets` | servidor, frame de protocolo | `ping_interval` 20 s, `ping_timeout` 20 s | [F 17] |

Detalhes que importam para o coup:

- **gorilla** estende o prazo de leitura só quando chega pong (`SetPongHandler` →
  `SetReadDeadline(now+pongWait)`), e põe prazo de escrita em cada ping e em cada mensagem. A doc
  avisa: "The application must read the connection to process close, ping and pong messages sent
  from the peer" [F 11, doc.go#L108-L109].
- **coder/websocket** não tem ping automático; o "Ping pong heartbeat helper #267" segue sem marcar
  no README [F 9, README.md#L41]. A doc do `Ping`: "Ping must be called concurrently with Reader as
  it does not read from the connection but instead waits for a Reader call to read the pong. TCP
  Keepalives should suffice for most use cases." [F 9, conn.go#L216-L222]. O mantenedor repete
  isso ("Most of the time TCP keep alives are enough"), mas também diz "I'm quite honestly still
  not sure if it was the right decision to not build in some keep alive loop" [F 10]. A seção 7
  mostra que, para este caso, keepalive TCP **não** basta.
- **Por que `Ping` precisa de um leitor:** `ping` registra um channel em `activePings`, escreve o
  frame e espera nesse channel (conn.go#L233-L260). Quem entrega o pong é o `Read` de outra
  goroutine, ao tratar o opcode Pong (read.go#L324-L336). No coup isso já existe: `Room.read`
  fica em `conn.Read` o tempo todo (server.go#L207-L223).
- **Socket.IO inverteu o sentido na v4:** "The ping packets are now sent by the server, because the
  timers set in the browsers are not reliable enough" [F 12, engine.io-protocol README#L371-L373].
  O cliente considera a conexão fechada se não recebe ping em `pingInterval + pingTimeout` (45 s).
- **Por que os frameworks de navegador usam mensagem de aplicação:** o JS não vê ping nem pong (seção
  4). A doc do Python `websockets` diz: "the WebSocket API in browsers doesn't expose the native
  Ping and Pong functionality … You have to roll your own in the application layer" [F 17,
  keepalive.rst#L81-L83]. É isso que dá ao *cliente* um jeito de detectar servidor morto.
- **Padrão:** intervalo de 15–30 s (25 s é o mais comum) e detecção em intervalo + 8–20 s.

## 3. Timeouts de ociosidade no caminho

| Onde | Padrão | O que zera o relógio | Fonte |
|---|---|---|---|
| nginx `proxy_read_timeout` | 60 s | "if the proxied server does not transmit any data within 60 seconds" a conexão fecha; a doc sugere "periodically send WebSocket ping frames to reset the timeout" | [F 18] |
| Cloudflare | sem número público | "will close a WebSocket connection when no data is transmitted in either direction for a period of time"; deploys "terminates WebSockets connections" | [F 19] |
| AWS ALB | 60 s (1–4000) | "no data being sent or received"; "send at least 1 byte of data before each idle timeout period elapses" | [F 20] |
| AWS NLB (TCP) | 350 s | idem | [F 20] |
| AWS API Gateway WebSocket | 10 min ocioso, 2 h no máximo | não ajustáveis | [F 21] |
| Heroku | 55 s | "each byte sent (either from the client or from your app process) resets a rolling 55 second window" | [F 22] |
| Fly.io | não encontrei o padrão | `http_options.idle_timeout` configurável | [F 23] |
| GCP Load Balancer / Cloud Run | 30 s para WebSocket ocioso / 5 min por request | | [F 24] |
| Azure Front Door | 5 min | | [F 24] |
| NAT (RFC 5382 REQ-5) | ≥ 2 h 4 min para conexão estabelecida | na prática há menos: em 2011, 4 de 73 operadoras móveis cortavam em ≤ 5 min, uma em 255 s | [F 25] |
| Linux conntrack | 432000 s (5 dias) | | [F 25] |

- **nginx, pelo código-fonte:** depois do upgrade a conexão vira um túnel de bytes. O mesmo handler
  re-arma `read_timeout` para tráfego nos dois sentidos (`ngx_http_upstream.c#L3641-L3893`) [F 18,
  I]. Então o ping do servidor passa de ponta a ponta até o navegador, e o pong volta.
- **Cloudflare:** não encontrei se o ping atravessa até o navegador ou se a borda responde sozinha.
  Se a borda responder, o ping só prova que a borda está viva. É um limite real de (c) atrás da
  Cloudflare, e só um teste contra ela responde.
- **Número que manda:** o menor limite documentado é 30 s (GCP), depois 55 s (Heroku) e 60 s (nginx,
  ALB). Um ping a cada 15 s cobre todos com folga. Keepalive TCP não serve para isso, nem com os
  15 s do Go. A sonda é um segmento TCP vazio, que o kernel do proxy responde. O relógio de
  ociosidade de um proxy L7 conta bytes de aplicação, não segmentos TCP [I, F 18–22].

## 4. Navegadores

- **O JS não manda ping nem vê pong.** "These are not currently exposed in the API." O navegador pode
  pingar por conta própria ("User agents may send ping and unsolicited pong frames as desired"),
  mas "must not use pings or unsolicited pongs to aid the server" [F 2]. `bufferedAmount` só conta
  o que o JS enfileirou com `send()` e ainda não saiu, sem o buffer do SO [F 2].
- **Chromium responde o ping no network service, não no renderer.**
  `websocket_channel.cc#L766-L772`: `case kOpCodePing: … SendFrameInternal(true, kOpCodePong, …)`
  [F 26]. Por isso o pong sai mesmo com o main thread da página travado (medido na seção 7).
- **Controle de fluxo do Chromium.** O canal só lê o socket
  `while (!event_interface_->HasPendingDataFrames())` (`websocket_channel.cc#L633`). Os dados vão
  ao renderer por um data pipe de **131000 B** (65536 no Android) (`services/network/websocket.cc`
  `#L70-L71`). Quando o renderer não esvazia o pipe, o frame fica pendente, o canal para de ler, e
  o próximo ping fica **atrás dos dados** no buffer TCP, sem pong [F 26, I]. A seção 7 confirma: o
  RTT do pong cresce e depois falha.
- **Chromium não manda ping próprio.** Não há caminho que monte um frame Ping em `net/websockets`
  [F 26, ausência]. O Firefox tem a pref `network.websocket.timeout.ping.request`, com padrão `0`
  (desligada) [F 32]. Então o heartbeat tem que partir do servidor.
- **Firefox e Safari também respondem o ping fora do JS.** O Firefox responde na thread de socket
  (`WebSocketChannel.cpp#L1825-L1827`, `#L1497`) [F 32]. No Safari o socket é um
  `NSURLSessionWebSocketTask` no Network Process, que lê sem esperar a página [F 31]. Não
  encontrei doc da Apple dizendo que ele responde ping sozinho [I].
- **Timers em aba escondida.** O Chrome acorda timers 1 vez por segundo. O throttling intensivo
  passa para 1 vez por minuto. O blog de 2021 diz que isso começa depois de 5 min escondida [F 27].
  O código atual usa 60 s para página já carregada
  (`kIntensiveWakeUpThrottling_GracePeriodSecondsLoaded_Default = 60`, `features.h#L40-L41`)
  [F 27]. WebSocket aberto não isenta [F 27]. Um heartbeat de aplicação iniciado pelo cliente com
  `setInterval` cairia para 1 por minuto (medido abaixo).
- **Congelamento (frozen).** "JavaScript timers and fetch callbacks don't run" [F 28]. O Chrome
  congela abas quando o Energy Saver está ativo e a aba ficou escondida e silenciosa por mais de
  5 min, sendo CPU-intensiva; no Android, 1 min em background. WebSocket aberto **não** impede
  [F 28].
- **O que acontece com o socket no freeze.** No estado `kFrozen` o Blink falha o canal com a
  mensagem "Page entered Back-Forward Cache." (`dom_websocket.cc#L618-L627`, flag
  `DisconnectWebSocketOnBFCache` estável) [F 29]. O teardown é postado numa fila que também
  congela (`websocket_channel_impl.cc#L503-L506`, `frame_scheduler_impl.cc#L1557-L1560`) [F 29].
  Medido: o servidor continua recebendo pong durante o freeze, e o close 1001 só chega quando a
  página descongela (seção 7).
- **bfcache e discard.** "Chrome (as of 149) and Safari do no block on open WebSockets" (grafia
  original); eles fecham o socket ao entrar no cache [F 30, F 31]. Quando o renderer larga a
  conexão (discard, crash), o network service tenta um close 1001
  (`services/network/websocket.cc#L637-L640`) [F 26].
- **iOS.** "When the app is suspended, no code within the app's process executes", e o sistema pode
  retomar os recursos do socket, "thereby closing the network connection" (TN2277, arquivada)
  [F 31]. Não encontrei declaração da Apple ou do WebKit sobre WebSocket com o aparelho bloqueado.
  Com o processo suspenso não sai pong, e o servidor detecta pelo ping [I].
- **Consequência:** `web/lib/connection.ts` já reconecta no `onclose` com o token salvo. Isso cobre
  a volta de freeze, bfcache e discard. Pings do servidor não ajudam o *navegador* a notar um
  servidor morto, porque o JS não vê o pong. Isso só com (d).

## 5. TCP keepalive e `TCP_USER_TIMEOUT`

- **RFC:** o keepalive "MUST default to off" e o intervalo "MUST default to no less than two hours".
  Ele só é enviado "when no data or acknowledgement packets have been received" [F 3, §4.2.3.6].
- **Linux:** `tcp_keepalive_time` 7200 s, `tcp_keepalive_intvl` 75 s, `tcp_keepalive_probes` 9. Com
  keepalive ligado, a conexão cai "approximately an additional 11 minutes" depois das 2 h
  [F 5]. `tcp_retries2` = 15, "approximately between 13 to 30 minutes" retransmitindo antes de
  desistir [F 5].
- **macOS** (lido nesta máquina com `sysctl`): `keepidle` 7200000 ms, `keepintvl` 75000 ms,
  `keepcnt` 8 [M].
- **Go:** todo listener liga keepalive com 15 s de ocioso, 15 s de intervalo e 9 sondas
  (`net/dial.go#L18-L27`, `net/tcpsock.go#L289-L303`, `net/tcpsockopt_unix.go#L15-L55`).
  `http.ListenAndServe` usa `net.Listen`, então herda isso (`net/http/server.go#L3450-L3462`)
  [F 7]. Host sumido com a conexão ociosa cai em ~15 + 9 × 15 = **150 s** [I].
- **O que o keepalive não pega.** Primeiro, um processo vivo que não lê: o kernel dele continua
  mandando ACK. Segundo, qualquer conexão com dado pendente. No Linux,
  `if (tp->packets_out || !tcp_write_queue_empty(sk)) goto resched;`, comentado como "It is alive
  without keepalive 8)" (`tcp_timer.c#L821-L823`) [F 6]. Na janela zero, "RFC 1122 4.2.2.17
  requires the sender to stay open indefinitely as long as the receiver continues to respond
  probes" (`tcp_timer.c#L404-L420`) [F 6, F 3 §4.2.2.17]. Ou seja, o cliente surdo do coup fica
  aberto **para sempre** no nível TCP. Com jogadas indo para um host que sumiu, vale a
  retransmissão: 13–30 min no Linux [F 5, I].
- **`TCP_USER_TIMEOUT`** (RFC 5482; Linux ≥ 2.6.37): tempo máximo que um dado "may remain
  unacknowledged, or buffered data may remain untransmitted (due to zero window size)". Passado
  esse tempo, o kernel fecha com `ETIMEDOUT` e sobrepõe o keepalive [F 4, F 5]. O kernel confirma
  que ele mata a conexão em janela zero (`tcp_timer.c#L415-L420`) [F 6]. Existe só no Linux. O
  `net.ListenConfig` não tem campo para ele. O `syscall` padrão não tem a constante em
  linux/amd64. Seria preciso `golang.org/x/sys/unix.TCP_USER_TIMEOUT` (hoje dependência indireta
  do coup) numa função `Control` [F 7, F 8].
- **Por que (b) não entra:** ele dispara nas mesmas situações que (a) (precisa de dado parado) e no
  host sumido, que (c) já pega em ≤ 25 s. Atrás de nginx ou Cloudflare, o TCP do servidor termina
  no proxy, que continua lendo. E não funciona no macOS, onde o projeto é desenvolvido.

## 6. Custo

- **Rede** [I, calculado]: um par ping/pong tem 10–18 bytes de WebSocket. Cada segmento carrega ≥ 40
  bytes de IPv4+TCP (RFC 791, RFC 9293), ou 52 com timestamps (RFC 7323). Contando o ping, o pong
  e um ACK, são ~170 bytes; com TLS 1.3, ~22 bytes a mais por registro (RFC 8446 §5.2 + tag AEAD de
  16), ~215 bytes. A cada 15 s isso dá **~12–15 B/s por conexão**: ~90 B/s numa mesa de 6, ~150
  KB/s com 10 mil conexões.
- **Go** [M]: medi 10 mil goroutines paradas num `select` com um `time.Ticker` de 15 s cada,
  em 3 rodadas. Cada uma custou 2,2–2,4 KB de pilha + ~1,04 KB de heap, ou ~3,3
  KB por conexão: ~20 KB numa mesa de 6, ~33 MB com 10 mil conexões. O mínimo de pilha de uma goroutine é 2048 bytes
  (`runtime/stack.go#L78`) [F 7]. Cada `Ping` e cada escrita com prazo criam um
  `context.WithTimeout`, e a coder/websocket arma um `context.AfterFunc` por escrita
  (conn.go#L171-L182) [F 9]. É o mesmo custo que `Handshake` já paga na primeira leitura.
- **Números publicados de custo por conexão:** não encontrei fonte primária.

## 7. Experimento

**Montagem** [M]: escrevi um servidor Go só para isso (coder/websocket v1.8.15), em 127.0.0.1:18190
no macOS 26 (arm64). Ele pinga a cada 1 s com `conn.Ping` sob prazo de 2 s, com uma goroutine
leitora rodando. Inunda com mensagens de 700 B na taxa pedida e registra cada escrita que demora.
A página manda `{"received":N}` a cada 1 s por `setInterval`, e isso mede se o JS está vivo.
Rodei no Chromium 151.0.7922.34 e no Brave 154.0.8037.58 (headless via Playwright), e num cliente
Go que conecta e nunca chama `Read`. É loopback: numa WAN cabem ainda mais bytes em trânsito antes
do bloqueio.

| Cenário | Pongs durante o cenário | JS | Bytes aceitos antes de a escrita travar | Maior bloqueio de `Write` | Prazo de 10 s dispararia? |
|---|---|---|---|---|---|
| Cliente Go que nunca lê, sem tráfego | 0; 1ª falha em +3,0 s | — | nada a escrever | — | não |
| Cliente Go que nunca lê, 500 msg/s | 0; 1ª falha em +3,0 s | — | **550.900** (787 msgs), em 1,6 s | 23,4 s (até o cliente sair) | sim, em ~11,6 s |
| (i) página normal, 1 msg/s | 10/10 nos dois; RTT 0,14–0,32 ms | vivo | — | — | — |
| (ii) JS em laço de 30 s, 1 msg/s | **40/40** nos dois, sem falha; RTT ≤ 0,3 ms (Chromium) / ≤ 0,72 ms (Brave) | parado 30 s | nunca travou | — | não |
| (ii) JS em laço de 30 s, 500 msg/s | falham ~3 s depois de o JS parar; 14 falhas em 30 s nos dois | parado | **1.850.100** (2643 msgs) nos dois; ~800 KB depois de o JS parar | 27,77 s (Chromium) / 27,73 s (Brave) | sim, ~12,3 s depois de o JS parar |
| (iii) `onmessage` gasta 5 ms/msg (teto ~200 msg/s) contra 500 msg/s | RTT sobe 5 ms → 292 ms → 1,8 s, depois 13 (Chromium) / 14 (Brave) falhas | vivo, lento | 2.128.000 (Chromium), na 1ª escrita que passou de 0,5 s | **1,16 s** (Chromium) / **1,27 s** (Brave) | **não, nunca** |
| (iv) aba atrás de outra (`hidden`) + congelada 20 s, 1 msg/s | **24/24** nos dois, sem falha | parado | — | — | não; close 1001 só ao descongelar |
| (iv) aba atrás de outra 150 s, sem congelar (Chromium) | **157/157** | `setInterval` a 1/s por ~58 s, depois 1 por **60 s**; `onmessage` seguiu a 1/s | — | — | — |

Leituras:

1. **No ritmo do jogo, só o ping dispara.** Os buffers absorvem de 550 KB a 1,85 MB antes de a
   escrita travar, ou 787 a 2643 snapshots de 700 B. É mais que uma partida inteira. O `outbox`
   de 16 também não enche, porque a goroutine escritora despeja no kernel na hora. Hoje, portanto,
   um cliente surdo nunca é derrubado. Com (a) sozinho continuaria assim. O ping falhou em 3 s.
2. **"Surdo" num navegador não é "JS travado".** Com o JS parado 30 s e tráfego de jogo, o Chromium
   e o Brave responderam 40 de 40 pings, enquanto o heartbeat da página ficou mudo 30 s. O ping
   detecta transporte morto, processo morto e buffer do navegador cheio. Não detecta aba
   congelada nem JS travado com pouco tráfego.
3. **Consumidor lento derruba (a) como detector principal.** A escrita trava e solta em rajadas de
   0,5–1,3 s, então um prazo de 10 s nunca dispara. O pong degrada de 5 ms até falhar. Isso bate
   com o código do Chromium: o ping espera atrás dos dados (seção 4).
4. **Aba congelada parece viva para o servidor** até descongelar. Aí o navegador fecha com 1001, e
   o `connection.ts` do coup reconectaria com o token. Na aba só escondida o ping passa sempre
   (157/157). O `setInterval` cai para 1 por minuto ~60 s depois de esconder, como no
   `features.h` (seção 4). Um heartbeat que parte do cliente viraria falso positivo.
5. **O que a coder/websocket faz quando uma escrita está presa.** O `Ping` concorrente falha com
   "failed to write control frame opPing: failed to acquire lock", depois de
   `min(prazo do chamador, 5 s)`. O limite de 5 s vem de `writeControl`, em write.go#L276-L277.
   Essa falha **não** fecha a conexão (conn.go#L286-L292), então quem chamou precisa chamar
   `CloseNow`. Já se o prazo expira durante a escrita, a biblioteca fecha a conexão inteira
   (conn.go#L171-L182) [F 9, M].

**Limites do método.** Via Playwright, "aba escondida" e `Page.setWebLifecycleState` não tiveram
efeito: a página continuou `visible` e o JS continuou rodando. O motivo é que o Playwright liga
`--disable-renderer-backgrounding`, `--disable-backgrounding-occluded-windows`,
`--disable-background-timer-throttling` e `--disable-back-forward-cache` [F 33]. Os cenários (iv)
por isso usam CDP cru, com o Chrome aberto com janela e outra aba trazida para frente. O freeze é emulado por CDP, não o do Energy Saver nem o do Android. Não testei
Safari, Firefox nem celular. Nos logs, o campo `closed` da página vem sempre `false`: é o
`Window.closed` nativo, só de leitura, que o script não consegue sobrescrever. Quem mostra que o
socket fechou é `readyState: 3`.

## 8. O que escolhi para o coup

**(a) + (c).** Números: `PingEvery` 15 s, `PongWait` 10 s, `WriteWait` 10 s. Keepalive TCP no
padrão do Go.

- **Por que esses números.** 15 s fica abaixo do menor limite de proxy documentado (30 s), com
  folga para um ping perdido, e dentro da faixa dos frameworks (15–30 s). Uma espera de 10 s fica
  entre o Centrifugo (8 s) e o Python `websockets` (20 s); nenhuma rede sã tem RTT de 10 s. O pior
  caso de detecção é 15 + 10 = **25 s**, igual ao prazo de uma decisão. Se quem caiu tinha a
  decisão pendente, a mesa espera a detecção (≤ 25 s) + a carência (30 s) = ≤ 55 s até o piloto
  automático. Antes da mudança, esse jogador nunca era marcado como caído: cada decisão dele gastava
  25 s para sempre, e a sala nunca chegava ao `IdleTTL`.
- **O que cada peça pega que a outra não.**
  - (c) pega host sumido, meia-conexão, cliente que não lê e consumidor lento.
  - (a) limita o tempo que a goroutine escritora fica presa. Com (c) rodando, isso é quase
    redundante na coder/websocket: o ping falha em ≤ 5 s ao disputar o lock, e o `CloseNow`
    solta a escrita. (a) fica como seguro barato, igual ao `writeWait` do gorilla, e limita também
    o `conn.Close` final.
  - Falso positivo custa pouco: o navegador reconecta em ~1 s com o token (`retryDelayMs = 1000`),
    e o assento volta.
- **Por que não (b):** nada que (c) não pegue antes, só Linux, e cego atrás de proxy (seção 5).
- **Por que não (d) agora:** (d) pegaria JS travado e aba congelada. Mas marcaria como caído quem
  só trocou de aba (o timer de aba escondida cai para 1 por minuto) e muda o protocolo nos dois
  clientes. O prazo de 25 s já limita o estrago de um jogador de JS travado. Reabro isso se o
  *cliente* precisar notar servidor morto, por exemplo um celular trocando de rede com a mesa
  congelada na tela. Aí o desenho barato é o do Socket.IO v4 e do Centrifugo: o servidor manda a
  mensagem, e o cliente fecha e reconecta se passar `intervalo + folga` sem receber nada.
- **O que mudou no código** (~25 linhas de produção):
  - `Config` ganhou `PingEvery`, `PongWait` e `WriteWait`, com os valores em `DefaultConfig`, no
    mesmo padrão de `Deadline`, `Grace` e `Handshake`; os testes usam valores curtos.
  - `write` passou a escrever cada mensagem com `context.WithTimeout(ctx, WriteWait)`.
  - A função nova `keepAlive(ctx, conn, every, patience)` faz ticker → `conn.Ping` com prazo; em
    erro, `conn.CloseNow()` e retorna.
  - `accept` dispara `go keepAlive(...)` ao lado de `go write(...)`. O `ctx` de `accept` já é
    cancelado quando `read` volta, e isso encerra o pinger.
  - O caminho da queda é o que já existia: o `CloseNow` faz `conn.Read` errar, `read` entrega
    `leave`, e `disconnect` segue para pausa, carência, piloto automático e `IdleTTL`.
  - `internal/server/README.md` passou de "2 goroutines por jogador" para 3.
- **Os testes que provam:**
  - Um cliente coder/websocket que conecta, senta e nunca chama `Read`, com `PingEvery`/`PongWait`
    de 100 ms: o assento dele aparece em `disconnected` em até 500 ms.
  - Um lobby em que o tester2 para de ler enquanto o tester1 alterna "pronto" até encher o buffer,
    com o ping desligado na prática e `WriteWait` de 200 ms: o tester2 sai do lobby. Sem o prazo de
    escrita, ele fica sentado para sempre.
- **O que ainda não sei:**
  - Se a Cloudflare responde ping na borda (não encontrei).
  - Se a espera de 10 s gera falso positivo em rede móvel ruim (não medi; o Python `websockets`
    usa 20 s).
  - O comportamento real de iOS e Safari com o aparelho bloqueado (não testei).

## Fontes

1. RFC 6455, §5.1, §5.2, §5.5, §5.5.2, §5.5.3 — https://www.rfc-editor.org/rfc/rfc6455
2. WHATWG WebSockets Standard — https://websockets.spec.whatwg.org/#ping-and-pong-frames ·
   https://websockets.spec.whatwg.org/#dom-websocket-bufferedamount
3. RFC 1122, §4.2.3.6 e §4.2.2.17 — https://www.rfc-editor.org/rfc/rfc1122
4. RFC 5482 — https://www.rfc-editor.org/rfc/rfc5482
5. Linux `tcp(7)`, man-pages 6.19 — https://man7.org/linux/man-pages/man7/tcp.7.html
6. Linux `net/ipv4/tcp_timer.c` —
   https://github.com/torvalds/linux/blob/fd179f8a05be3ccae366b9b96e176b51fbe54aab/net/ipv4/tcp_timer.c#L404-L420 ·
   `#L821-L823`
7. Go 1.27.1, `$(go env GOROOT)/src`: `net/dial.go#L18-L27`, `net/tcpsock.go#L116-L148`,
   `net/tcpsock.go#L289-L303`, `net/tcpsockopt_unix.go#L15-L55`, `net/http/server.go#L3450-L3462`,
   `runtime/stack.go#L78`, `syscall/zerrors_linux_amd64.go` (sem `TCP_USER_TIMEOUT`)
8. `golang.org/x/sys` v0.47.0, `unix/zerrors_linux.go#L3848` (`TCP_USER_TIMEOUT = 0x12`)
9. coder/websocket v1.8.15, `~/go/pkg/mod/github.com/coder/websocket@v1.8.15/`: `conn.go#L171-L182`,
   `#L216-L260`, `#L286-L292`; `read.go#L303`, `#L316-L336`; `write.go#L276-L285`,
   `frame.go#L104-L105`; `doc.go#L29`; `README.md#L41`
10. coder/websocket issue #265 —
    https://github.com/coder/websocket/issues/265#issuecomment-733920987 ·
    `#issuecomment-733930861` · `#issuecomment-734499702`
11. gorilla/websocket —
    https://github.com/gorilla/websocket/blob/e064f32e3674d9d79a8fd417b5bc06fa5c6cad8f/examples/chat/client.go#L16-L28
    (`#L61-L63`, `#L114-L117`) · `…/doc.go#L99-L109`
12. Socket.IO —
    https://github.com/socketio/socket.io/blob/deee824c0e480c7590b74797e33d45a7ad993880/packages/engine.io/lib/server.ts#L201-L204 ·
    https://github.com/socketio/engine.io-protocol/blob/f21de7b00ed09b3bbad2807db718ea5d6bc36aba/README.md#L237-L241
    (`#L371-L373`) · https://socket.io/docs/v4/server-options/
13. Phoenix —
    https://github.com/phoenixframework/phoenix/blob/2380a61d5c7d008e01bc297a07ebadbeaf2baf67/assets/js/phoenix/socket.js#L166
    (`#L692-L694`) · `…/lib/phoenix/transports/websocket.ex#L31` · `…/lib/phoenix/endpoint.ex#L1018-L1019`
14. SignalR —
    https://github.com/dotnet/AspNetCore.Docs/blob/68566a558d284e7412c270737d2af524fda81b0c/aspnetcore/signalr/configuration.md#L73-L75
    (`#L336-L337`) ·
    https://github.com/dotnet/aspnetcore/blob/c7cef3bfadeb7416a67e6b06bf2a736dd540806b/src/SignalR/server/Core/src/HubOptionsSetup.cs#L15-L19
15. `ws` — https://github.com/websockets/ws/blob/297202cdae9b0590629821373b3b679c407f3431/README.md#L452-L509
16. Centrifuge/Centrifugo —
    https://github.com/centrifugal/centrifuge/blob/f70aaff9fb1f96d55f4a52dd996ee9008d2deb46/config.go#L493-L511 ·
    https://github.com/centrifugal/centrifugo/blob/76cfe5c52fa7eceb7729ed26f4efbefbcc21b0f3/internal/configtypes/types.go#L228-L231
17. Python `websockets` —
    https://github.com/python-websockets/websockets/blob/7db6ba3ead2bcfa7c914a9dfdba313b82a64b5dd/src/websockets/asyncio/connection.py#L49-L50 ·
    `…/docs/topics/keepalive.rst#L20-L83`
18. nginx — https://nginx.org/en/docs/http/websocket.html ·
    https://nginx.org/en/docs/http/ngx_http_proxy_module.html#proxy_read_timeout ·
    https://github.com/nginx/nginx/blob/939334efff3575ce52597cc8c13d55821044ac57/src/http/ngx_http_upstream.c#L3641-L3893
19. Cloudflare — https://developers.cloudflare.com/network/websockets/ (atualizada 2026-08-14)
20. AWS ELB —
    https://docs.aws.amazon.com/elasticloadbalancing/latest/application/edit-load-balancer-attributes.html#connection-idle-timeout ·
    https://docs.aws.amazon.com/elasticloadbalancing/latest/network/update-idle-timeout.html
21. AWS API Gateway —
    https://docs.aws.amazon.com/apigateway/latest/developerguide/apigateway-execution-service-websocket-limits-table.html
22. Heroku — https://devcenter.heroku.com/articles/http-routing · https://devcenter.heroku.com/articles/websockets
23. Fly.io — https://fly.io/docs/reference/configuration/
24. GCP/Azure — https://cloud.google.com/run/docs/triggering/websockets ·
    https://docs.cloud.google.com/load-balancing/docs/https/request-distribution ·
    https://learn.microsoft.com/en-us/azure/frontdoor/standard-premium/websocket
25. NAT — https://www.rfc-editor.org/rfc/rfc5382#section-5 · https://www.rfc-editor.org/rfc/rfc7857#section-2.1 ·
    https://docs.kernel.org/networking/nf_conntrack-sysctl.html ·
    https://conferences.sigcomm.org/sigcomm/2011/papers/sigcomm/p374.pdf
26. Chromium (commit `8b74927530da51629ee3dcdebf1f45499c6b0880`) —
    https://github.com/chromium/chromium/blob/8b74927530da51629ee3dcdebf1f45499c6b0880/net/websockets/websocket_channel.cc#L766-L772
    (`#L633`, `#L669-L670`) · `…/services/network/websocket.cc#L70-L71` (`#L431`, `#L637-L640`, `#L845-L854`)
27. Throttling — https://developer.chrome.com/blog/timer-throttling-in-chrome-88 ·
    `…/third_party/blink/renderer/platform/scheduler/common/features.h#L40-L41` (mesmo commit)
28. Page Lifecycle — https://developer.chrome.com/docs/web-platform/page-lifecycle-api ·
    https://developer.chrome.com/blog/freezing-on-energy-saver ·
    `…/components/performance_manager/public/freezing/cannot_freeze_reason.h` ·
    `…/third_party/blink/common/features.cc#L2236-L2245`
29. Freeze → falha do socket — `…/third_party/blink/renderer/modules/websockets/dom_websocket.cc#L618-L627` ·
    `…/websocket_channel_impl.cc#L503-L506` ·
    `…/platform/scheduler/main_thread/frame_scheduler_impl.cc#L1557-L1560` ·
    `…/platform/runtime_enabled_features.json5#L2437-L2438`
30. bfcache — https://web.dev/articles/bfcache
31. WebKit (commit `c39e3bbd953988fd6c963873ba3c6e53cde93d8f`) —
    https://github.com/WebKit/WebKit/blob/c39e3bbd953988fd6c963873ba3c6e53cde93d8f/Source/WebCore/Modules/websockets/WebSocket.cpp#L533-L535 ·
    `…/Source/WebKit/NetworkProcess/cocoa/WebSocketTaskCocoa.mm#L113-L118` ·
    https://developer.apple.com/library/archive/technotes/tn2277/_index.html
32. Firefox (commit `c32abda0190531351e22da36334f18f9e994d474`) —
    https://github.com/mozilla-firefox/firefox/blob/c32abda0190531351e22da36334f18f9e994d474/netwerk/protocol/websocket/WebSocketChannel.cpp#L1825-L1827 ·
    `…/modules/libpref/init/all.js#L1321-L1328`
33. playwright-core 1.63.0, `lib/coreBundle.js#L34852-L34876` (lista `chromiumSwitches`)
34. RFC 8446 §5.2 — https://www.rfc-editor.org/rfc/rfc8446#section-5.2 · RFC 7323 · RFC 791 · RFC 9293
35. Este repo: `internal/server/server.go#L136-L164` (`accept`), `#L207-L233` (`read`, `write`);
    `internal/server/broadcast.go#L70-L84` (`send`); `internal/server/seat.go#L19-L33`;
    `internal/server/room.go#L10-L13`; `web/lib/connection.ts#L14`, `#L85-L93`
