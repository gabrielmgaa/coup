package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gabrielmgaa/coup/web"
)

func main() {
	if len(os.Args) < 2 {
		imprimirUso()
	}
	switch os.Args[1] {
	case "serve":
		servir(os.Args[2:])
	default:
		imprimirUso()
	}
}

func imprimirUso() {
	fmt.Fprintln(os.Stderr, "uso: coup serve [-porta 8080]")
	os.Exit(2)
}

func servir(argumentos []string) {
	flags := flag.NewFlagSet("serve", flag.ExitOnError)
	porta := flags.Int("porta", 8080, "porta HTTP")
	flags.Parse(argumentos)

	mux := http.NewServeMux()
	mux.Handle("/", http.FileServerFS(web.Dist()))

	endereco := fmt.Sprintf(":%d", *porta)
	log.Printf("coup ouvindo em http://localhost%s", endereco)
	log.Fatal(http.ListenAndServe(endereco, mux))
}
