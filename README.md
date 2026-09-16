# Coup

Implementação open source do **Coup**, jogável por navegador e por terminal. Servidor
autoritativo em Go, motor de regras puro, protocolo de eventos tipado. Sem conta, sem
instalação: um binário só, com o site dentro.

> **Em construção.** O esqueleto está de pé; o jogo ainda não. O plano completo, com as
> regras destrinchadas e a ordem de construção, está em
> [`docs/plan/major/`](docs/plan/major/README.md).

## Rodar

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

## Créditos

Coup é de **Rikki Tahta**, publicado por **La Mame Games** e **Indie Boards & Cards**; no
Brasil pela **Mandala Jogos**. Este repositório é uma implementação independente das regras,
sem nenhuma arte da caixa. Regras de jogo não são protegidas por copyright; a arte é.

Código sob licença [MIT](LICENSE).
