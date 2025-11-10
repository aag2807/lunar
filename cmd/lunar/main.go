package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"lunar/internal/codegen"
	"lunar/internal/lexer"
	"lunar/internal/parser"
	"lunar/internal/types"
	"os"
	"strings"
)

const version = "0.1.0"

func main() {
	// Define command-line flags
	outputFile := flag.String("o", "", "Output file (default: replaces .lunar with .lua)")
	noTypeCheck := flag.Bool("no-typecheck", false, "Skip type checking")
	showVersion := flag.Bool("version", false, "Show version information")
	showHelp := flag.Bool("help", false, "Show help message")

	flag.Parse()

	// Handle version flag
	if *showVersion {
		fmt.Printf("Lunar compiler version %s\n", version)
		os.Exit(0)
	}

	// Handle help flag
	if *showHelp {
		printHelp()
		os.Exit(0)
	}

	// Get input file
	args := flag.Args()
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Error: No input file specified")
		fmt.Fprintln(os.Stderr, "Usage: lunar [options] <input.lunar>")
		fmt.Fprintln(os.Stderr, "Run 'lunar --help' for more information")
		os.Exit(1)
	}

	inputFile := args[0]

	// Validate input file exists
	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: Input file '%s' does not exist\n", inputFile)
		os.Exit(1)
	}

	// Validate input file extension
	if !strings.HasSuffix(inputFile, ".lunar") {
		fmt.Fprintf(os.Stderr, "Warning: Input file '%s' does not have .lunar extension\n", inputFile)
	}

	// Determine output file
	output := *outputFile
	if output == "" {
		output = strings.TrimSuffix(inputFile, ".lunar") + ".lua"
	}

	// Compile the file
	if err := compile(inputFile, output, !*noTypeCheck); err != nil {
		fmt.Fprintf(os.Stderr, "Compilation failed:\n%v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully compiled %s -> %s\n", inputFile, output)
}

// compile compiles a Lunar source file to Lua
func compile(inputFile, outputFile string, typeCheck bool) error {
	// Read source file
	source, err := ioutil.ReadFile(inputFile)
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}

	// Lexer: Tokenize the source
	l := lexer.New(string(source))

	// Parser: Build AST
	p := parser.New(l)
	statements := p.Parse()

	// Check for parser errors
	if len(p.Errors()) > 0 {
		return formatParserErrors(inputFile, p.Errors())
	}

	// Type Checker: Validate types (if enabled)
	if typeCheck {
		typeErrors := types.Check(statements)
		if len(typeErrors) > 0 {
			return formatTypeErrors(inputFile, typeErrors)
		}
	}

	// Code Generator: Transpile to Lua
	luaCode := codegen.Generate(statements)

	// Write output file
	if err := ioutil.WriteFile(outputFile, []byte(luaCode), 0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	return nil
}

// formatParserErrors formats parser errors for display
func formatParserErrors(filename string, errors []string) error {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\n%s: Parse errors:\n", filename))
	for _, msg := range errors {
		sb.WriteString(fmt.Sprintf("  %s\n", msg))
	}
	return fmt.Errorf("%s", sb.String())
}

// formatTypeErrors formats type errors for display
func formatTypeErrors(filename string, errors []*types.TypeError) error {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\n%s: Type errors:\n", filename))
	for _, err := range errors {
		sb.WriteString(fmt.Sprintf("  Line %d, Column %d: %s\n", err.Line, err.Column, err.Message))
	}
	return fmt.Errorf("%s", sb.String())
}

// printHelp prints help information
func printHelp() {
	fmt.Println("Lunar - A statically-typed superset of Lua")
	fmt.Printf("Version: %s\n\n", version)
	fmt.Println("Usage:")
	fmt.Println("  lunar [options] <input.lunar>")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -o <file>        Output file (default: replaces .lunar with .lua)")
	fmt.Println("  --no-typecheck   Skip type checking")
	fmt.Println("  --version        Show version information")
	fmt.Println("  --help           Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  lunar main.lunar")
	fmt.Println("  lunar main.lunar -o output.lua")
	fmt.Println("  lunar main.lunar --no-typecheck")
	fmt.Println()
	fmt.Println("For more information about the Lunar language:")
	fmt.Println("  See README.md in the repository")
}
