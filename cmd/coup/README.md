# `cmd/coup` — o binário

O único executável do projeto. `main.go` faz `switch` em `os.Args[1]`: `serve` sobe o
servidor com o site dentro; `join` abre a mesa no terminal.

```sh
go run ./cmd/coup serve
go run ./cmd/coup serve -port 3000 -starting-coins 14
go run ./cmd/coup join -name tester1          # abre uma mesa nova
go run ./cmd/coup join K7QM                   # entra na mesa K7QM com o nome salvo
```

## Flags do `join`

| Flag | Default | Pra que serve |
|---|---|---|
| `-server` | `ws://localhost:8080/ws`, ou o último usado | endereço WebSocket do servidor |
| `-name` | o último usado | seu nome na mesa; obrigatório na primeira vez |
| `-reconnect` | desligado | volta ao assento salvo (código e token do último `welcome`) |

Sem código, `join` abre uma mesa nova — o mesmo que deixar o código vazio no navegador. Nome e
servidor ficam em `session.json` dentro de `coup/` na pasta de configuração do sistema
(`os.UserConfigDir`: `~/.config/` no Linux, `~/Library/Application Support/` no macOS), em modo
`0600`, junto com o código e o token do último assento, para o `-reconnect`.

## Flags do `serve`

| Flag | Default | Pra que serve |
|---|---|---|
| `-port` | `8080` | porta HTTP |
| `-starting-coins` | `0` | moedas iniciais. `0` significa **seguir o livreto**: 2 moedas, ou 1 num jogo de duas pessoas |

**`-starting-coins 14` é como se testa uma partida inteira na mão.** Com as moedas do livreto e
só a Renda disponível, chegar ao segundo Golpe leva ~29 turnos de clique. A flag some de vista
quando as outras ações existirem, mas não some do código: continua sendo o jeito de montar uma
posição específica sem mexer no motor.

## O que ele monta

No `serve`, três coisas e nada mais: o site embutido (`web.Dist()`), o seed do RNG
(`crypto/rand` → `rand.NewPCG`) e o handler de `internal/server`. No `join`, a sessão salva e
`tui.Play`. Regra de jogo, protocolo e
concorrência estão nos pacotes; aqui só se liga um no outro.

Se a entropia do sistema falhar, o processo morre em vez de embaralhar com seed previsível —
baralho previsível é um jogo de blefe quebrado.
