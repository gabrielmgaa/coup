# Coup

Implementação open source do **Coup**, jogável por navegador e por terminal. Servidor
autoritativo em Go, motor de regras puro, protocolo de eventos tipado. Sem conta, sem
instalação: um binário só, com o site dentro.

De 2 a 6 pessoas abrem um link, entram numa mesa por código de 4 letras e jogam uma partida
completa com as regras base: 5 personagens, 7 ações, contestação, bloqueio, prazo de 25 s por
decisão, pausa quando cai quem tem de decidir, e reconexão. No fim a mesa volta para o lobby e
quem venceu começa a próxima. O terminal joga a mesma mesa que o navegador.

O plano completo, com as regras destrinchadas e as decisões de cada fase, está em
[`docs/plan/major/`](docs/plan/major/README.md).

## Jogar

```sh
make build            # site + binário, um arquivo só
./coup serve          # http://localhost:8080
./coup join -name ana # a mesma mesa pelo terminal; sem código abre uma nova
./coup join K7QM      # entra na mesa K7QM
./coup join -reconnect
```

`make release` gera binários estáticos para Linux e macOS (amd64 e arm64) em `release/`. Eles
rodam numa máquina sem Go nem Node.

### Regra da casa

Ao abrir uma mesa dá para ligar **reações independentes** (checkbox no navegador,
`-independent-reactions` no terminal): quem contestou e perdeu ainda pode bloquear a mesma
ação. Desligado, vale o livreto — uma reação por jogador por ação. As outras duas divergências
deliberadas (alvo sem moedas não pode ser extorquido; tesouro infinito) estão em
[`01-regras.md`](docs/plan/major/01-regras.md).

## Desenvolver

Em desenvolvimento são **dois processos**, porque ninguém quer esperar `vite build` a cada
`ctrl+s`:

```sh
cd web && pnpm install && pnpm dev   # :5173, com hot reload
go run ./cmd/coup serve              # :8080
```

Em release é **um só**. O `go build` copia `web/dist` pra dentro do executável, então o
`pnpm build` precisa vir antes:

```sh
cd web && pnpm build
go build -o coup ./cmd/coup
./coup serve
```

O binário resultante contém o site. Não tem pasta pra subir, CDN nem nginx.

`make verify` roda o que um PR precisa passar: `go test -race`, `go vet`, `gofmt`, e o build e
o lint do site. A cobertura fica acima de 95% (`go test -cover ./...`).

### Testes

- `internal/engine` — cada ação, os oito galhos da árvore de uma ação afirmando moedas e cartas
  por número, uma partida roteirizada que usa todas as ações, e uma bateria que joga centenas de
  partidas aleatórias misturando jogadas legais com tentativas de roubo — carta que aparece ou
  some, moeda negativa, eliminado com moedas, mão alheia no snapshot, jogada recusada que muda a
  mesa. A mesma bateria roda como fuzz: `go test -fuzz=FuzzAnyMoveSequenceKeepsTheTableWhole ./internal/engine`.
- `internal/server` — ponta a ponta pelo WebSocket: lobby, partida, prazo, pausa, reconexão, TTL,
  revanche, e tentativas de trapaça (jogar fora da vez, falar por outro assento, forjar
  mensagem interna, reusar token de outra sala, sentar duas vezes pela mesma conexão).
- `internal/tui` — um terminal e um navegador jogando a mesma partida até o fim.

### Os dois `.gitkeep` que parecem lixo e não são

`go:embed` é erro de compilação quando o padrão não casa arquivo nenhum, então `web/dist/`
nunca pode ficar vazia. Quem segura isso são dois arquivos vazios: **`web/dist/.gitkeep`** está
no git e faz um clone recém-baixado compilar antes de qualquer build; **`web/public/.gitkeep`** é
copiado pra dentro do `dist` por todo `pnpm build`, repondo o primeiro, que o Vite apaga ao
esvaziar a pasta. Apagar qualquer um dos dois quebra `go build ./...`.

## Créditos

Coup é de **Rikki Tahta**, publicado por **La Mame Games** e **Indie Boards & Cards**; no
Brasil pela **Mandala Jogos**. Este repositório é uma implementação independente das regras,
sem nenhuma arte da caixa. Regras de jogo não são protegidas por copyright; a arte é.

Código sob licença [MIT](LICENSE).
