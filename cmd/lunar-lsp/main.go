package main

import (
	"flag"
	"fmt"
	"os"

	"lunar/internal/lsp"
	"lunar/internal/version"
)

func main() {
	showVersion := flag.Bool("version", false, "Show version information")
	flag.Parse()

	if *showVersion {
		fmt.Println(version.String())
		os.Exit(0)
	}

	server := lsp.NewServer()
	if err := server.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
