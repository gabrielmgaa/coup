// Package web embute o build do Vite dentro do binário.
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var embutido embed.FS

// Dist entrega o conteúdo de web/dist com a raiz em "/".
//
// go:embed é erro de compilação quando o padrão não casa arquivo nenhum, então
// dist/ nunca pode ficar vazia — nem num clone recém-baixado, nem entre um
// `pnpm build` e outro. Quem segura isso são dois arquivos vazios que parecem
// lixo e não são: web/dist/.gitkeep está no git (faz o clone compilar antes de
// qualquer build) e web/public/.gitkeep é copiado pra dentro do dist por todo
// `pnpm build` (repõe o primeiro, que o vite apaga ao esvaziar a pasta).
// Apagar qualquer um dos dois quebra `go build ./...`.
func Dist() fs.FS {
	raiz, err := fs.Sub(embutido, "dist")
	if err != nil {
		panic(err) // "dist" é literal; fs.Sub só recusa caminho malformado
	}
	return raiz
}
