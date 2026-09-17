package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var embedded embed.FS

func Dist() fs.FS {
	root, err := fs.Sub(embedded, "dist")
	if err != nil {
		panic(err)
	}
	return root
}
