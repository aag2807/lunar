package main

import (
	"flag"
	"fmt"
	"lunar/internal/lsp"
	"os"
)

var version = "1.3.0"

func main() {
	showVersion := flag.Bool("version", false, "Show version information")
	flag.Parse()

	if *showVersion {
		fmt.Printf("lunar-lsp version %s\n", version)
		os.Exit(0)
	}

	server := lsp.NewServer()
	if err := server.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
