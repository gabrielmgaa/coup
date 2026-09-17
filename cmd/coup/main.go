package main

import (
	crand "crypto/rand"
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"math/rand/v2"
	"net/http"
	"os"

	"github.com/gabrielmgaa/coup/internal/engine"
	"github.com/gabrielmgaa/coup/internal/server"
	"github.com/gabrielmgaa/coup/web"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
	}
	switch os.Args[1] {
	case "serve":
		serve(os.Args[2:])
	default:
		printUsage()
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "usage: coup serve [-port 8080]")
	os.Exit(2)
}

func serve(args []string) {
	flags := flag.NewFlagSet("serve", flag.ExitOnError)
	port := flags.Int("port", 8080, "HTTP port")
	coins := flags.Int("starting-coins", engine.RulebookCoins,
		"coins each player starts with; 0 follows the rulebook (2, or 1 in a two-player game)")
	flags.Parse(args)

	address := fmt.Sprintf(":%d", *port)
	log.Printf("coup listening on http://localhost%s", address)
	log.Fatal(http.ListenAndServe(address, server.New(web.Dist(), rand.New(seed()), *coins)))
}

func seed() *rand.PCG {
	var bytes [16]byte
	if _, err := crand.Read(bytes[:]); err != nil {
		log.Fatalf("no entropy available to shuffle the deck: %v", err)
	}
	return rand.NewPCG(
		binary.NativeEndian.Uint64(bytes[:8]),
		binary.NativeEndian.Uint64(bytes[8:]),
	)
}
