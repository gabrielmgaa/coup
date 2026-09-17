# `cmd/coup` — o binário

O único executável do projeto. `main.go` faz `switch` em `os.Args[1]`; hoje o único comando é
`serve`. `join` (cliente de terminal) chega na 0.3.

```sh
go run ./cmd/coup serve
go run ./cmd/coup serve -port 3000 -starting-coins 14
```

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

Três coisas e nada mais: o site embutido (`web.Dist()`), o seed do RNG
(`crypto/rand` → `rand.NewPCG`) e o handler de `internal/server`. Regra de jogo, protocolo e
concorrência estão nos pacotes; aqui só se liga um no outro.

Se a entropia do sistema falhar, o processo morre em vez de embaralhar com seed previsível —
baralho previsível é um jogo de blefe quebrado.
