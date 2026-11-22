package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"lunar/internal/ast"
	"lunar/internal/codegen"
	"lunar/internal/config"
	"lunar/internal/lexer"
	"lunar/internal/parser"
	"lunar/internal/types"
	"lunar/internal/version"
)

// supportedTargets are the Lua versions the compiler is tested against.
var supportedTargets = map[string]bool{"lua53": true, "lua54": true}

// getStdlibPath returns the stdlib directory next to the executable, one
// level up (builds in dist/), or where `make install` puts it
// (<prefix>/share/lunar/stdlib); "" lets the checker search from cwd.
func getStdlibPath() string {
	exePath, err := os.Executable()
	if err != nil {
		return ""
	}
	exeDir := filepath.Dir(exePath)
	for _, candidate := range []string{
		filepath.Join(exeDir, "stdlib"),
		filepath.Join(exeDir, "..", "stdlib"),
		filepath.Join(exeDir, "..", "share", "lunar", "stdlib"),
	} {
		if _, err := os.Stat(candidate); err == nil {
			if abs, err := filepath.Abs(candidate); err == nil {
				return abs
			}
			return candidate
		}
	}
	return ""
}

func main() {
	outputFile := flag.String("o", "", "Output file (default: replaces .lunar with .lua)")
	sourceMap := flag.Bool("source-map", false, "Generate source map (.lua.map file)")
	targetLua := flag.String("target", "", "Target Lua version: lua53 (default) or lua54")
	configFile := flag.String("config", "", "Path to lunar.config.json (auto-detected if not specified)")
	watchMode := flag.Bool("watch", false, "Watch the input file and recompile on change")
	watchInterval := flag.Int("watch-interval", 500, "Watch polling interval in milliseconds")
	watchClear := flag.Bool("watch-clear", false, "Clear the console before each rebuild in watch mode")
	showVersion := flag.Bool("version", false, "Show version information")
	showHelp := flag.Bool("help", false, "Show help message")
	flag.Usage = printHelp
	flag.Parse()

	if *showVersion {
		fmt.Println(version.String())
		os.Exit(0)
	}
	if *showHelp {
		printHelp()
		os.Exit(0)
	}

	args := flag.Args()
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Error: No input file specified")
		fmt.Fprintln(os.Stderr, "Usage: lunar [options] <input.lunar>")
		fmt.Fprintln(os.Stderr, "Run 'lunar --help' for more information")
		os.Exit(1)
	}
	inputFile := args[0]
	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: Input file '%s' does not exist\n", inputFile)
		os.Exit(1)
	}
	if !strings.HasSuffix(inputFile, ".lunar") {
		fmt.Fprintf(os.Stderr, "Warning: Input file '%s' does not have .lunar extension\n", inputFile)
	}

	var cfg *config.Config
	var configDir string
	var err error
	if *configFile != "" {
		cfg, err = config.LoadFromFile(*configFile)
		configDir = filepath.Dir(*configFile)
	} else {
		configDir, cfg, err = config.FindConfig(filepath.Dir(inputFile))
		if configDir == "" {
			configDir = filepath.Dir(inputFile)
		}
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}
	cfg.Merge(false, *sourceMap, false, *targetLua)

	target := cfg.CompilerOptions.Target
	if target == "" {
		target = "lua53"
	}
	if !supportedTargets[target] {
		fmt.Fprintf(os.Stderr, "Error: Unsupported target '%s'. Supported targets: lua53, lua54\n", target)
		os.Exit(1)
	}

	output := *outputFile
	if output == "" {
		switch {
		case cfg.OutFile != "":
			output = filepath.Join(configDir, cfg.OutFile)
		case cfg.OutDir != "":
			output = filepath.Join(configDir, cfg.OutDir, strings.TrimSuffix(filepath.Base(inputFile), ".lunar")+".lua")
		default:
			output = strings.TrimSuffix(inputFile, ".lunar") + ".lua"
		}
	}
	if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	opts := compileOptions{sourceMap: *sourceMap, target: target}
	if *watchMode {
		runWatchMode(inputFile, output, opts, *watchInterval, *watchClear)
		return
	}

	if err := compile(inputFile, output, opts); err != nil {
		fmt.Fprintf(os.Stderr, "Compilation failed:\n%v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Successfully compiled %s -> %s\n", inputFile, output)
	if *sourceMap {
		fmt.Printf("Source map: %s.map\n", output)
	}
}

type compileOptions struct {
	sourceMap bool
	target    string
}

// compile runs lexer -> parser -> checker -> codegen on one file and writes the Lua.
func compile(inputFile, outputFile string, opts compileOptions) error {
	declarationStatements := []ast.Statement{}
	declFiles, err := discoverDeclarationFiles(inputFile)
	if err != nil {
		return fmt.Errorf("failed to discover declaration files: %w", err)
	}
	for _, declFile := range declFiles {
		declStatements, err := parseDeclarationFile(declFile)
		if err != nil {
			return fmt.Errorf("failed to parse declaration file %s: %w", declFile, err)
		}
		declarationStatements = append(declarationStatements, declStatements...)
	}

	source, err := os.ReadFile(inputFile)
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}

	p := parser.New(lexer.New(string(source)))
	statements := p.Parse()
	if len(p.Errors()) > 0 {
		return formatParserErrors(inputFile, string(source), p.Errors())
	}

	// Declarations first so they're registered before the main code. The
	// checker also decides ':' versus '.' and which calls construct, so it
	// always runs.
	allStatements := append(declarationStatements, statements...)
	checker := types.NewChecker(getStdlibPath())
	if typeErrors := checker.Check(allStatements); len(typeErrors) > 0 {
		return formatTypeErrors(inputFile, string(source), typeErrors)
	}

	// Only the main file is emitted, never its declarations.
	var luaCode string
	if opts.sourceMap {
		code, sm := codegen.GenerateWithSourceMapAndTarget(statements, inputFile, outputFile, opts.target)
		mapFilename := outputFile + ".map"
		luaCode = code + "\n" + sm.GenerateComment(filepath.Base(mapFilename)) + "\n"
		mapJSON, err := sm.ToJSON()
		if err != nil {
			return fmt.Errorf("failed to generate source map JSON: %w", err)
		}
		if err := os.WriteFile(mapFilename, []byte(mapJSON), 0644); err != nil {
			return fmt.Errorf("failed to write source map file: %w", err)
		}
	} else {
		luaCode = codegen.GenerateWithTarget(statements, opts.target)
	}

	if err := os.WriteFile(outputFile, []byte(luaCode), 0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}
	return nil
}

// discoverDeclarationFiles finds all .d.lunar files in the same directory as the input file.
func discoverDeclarationFiles(inputFile string) ([]string, error) {
	return filepath.Glob(filepath.Join(filepath.Dir(inputFile), "*.d.lunar"))
}

// parseDeclarationFile parses a declaration file and returns its statements.
func parseDeclarationFile(filename string) ([]ast.Statement, error) {
	source, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	p := parser.New(lexer.New(string(source)))
	statements := p.Parse()
	if len(p.Errors()) > 0 {
		return nil, formatParserErrors(filename, string(source), p.Errors())
	}
	return statements, nil
}

type located struct {
	line, column int
	message      string
}

func formatParserErrors(filename, source string, errors []*parser.ParseError) error {
	locs := make([]located, len(errors))
	for i, e := range errors {
		locs[i] = located{e.Line, e.Column, e.Message}
	}
	return formatErrors(filename, source, "Parse errors found", locs)
}

func formatTypeErrors(filename, source string, errors []*types.TypeError) error {
	locs := make([]located, len(errors))
	for i, e := range errors {
		locs[i] = located{e.Line, e.Column, e.Message}
	}
	return formatErrors(filename, source, "Type errors found", locs)
}

// formatErrors renders errors with a line of source context on each side and
// a caret under the reported column.
func formatErrors(filename, source, heading string, errs []located) error {
	var sb strings.Builder
	fmt.Fprintf(&sb, "\n%s: %s:\n\n", filename, heading)
	lines := strings.Split(source, "\n")
	for i, e := range errs {
		if i > 0 {
			sb.WriteString("\n")
		}
		fmt.Fprintf(&sb, "  Error %d: %s:%d:%d\n", i+1, filename, e.line, e.column)
		fmt.Fprintf(&sb, "  %s\n\n", e.message)
		start, end := e.line-2, e.line+1
		if start < 1 {
			start = 1
		}
		if end > len(lines) {
			end = len(lines)
		}
		for n := start; n <= end; n++ {
			content := lines[n-1]
			fmt.Fprintf(&sb, "  %4d | %s\n", n, content)
			if n == e.line && e.column > 0 && e.column <= len(content)+1 {
				fmt.Fprintf(&sb, "       | %s^\n", strings.Repeat(" ", e.column-1))
			}
		}
	}
	return fmt.Errorf("%s", sb.String())
}

// runWatchMode recompiles inputFile whenever its modification time changes.
func runWatchMode(inputFile, outputFile string, opts compileOptions, intervalMs int, clearConsole bool) {
	fmt.Printf("Watching %s for changes (Ctrl+C to stop)\n", inputFile)
	rebuild := func(reason string) {
		stamp := time.Now().Format("15:04:05")
		if reason != "" {
			fmt.Printf("[%s] %s\n", stamp, reason)
		}
		if err := compile(inputFile, outputFile, opts); err != nil {
			fmt.Fprintf(os.Stderr, "[%s] Compilation failed:\n%v\n", stamp, err)
			return
		}
		fmt.Printf("[%s] Successfully compiled %s -> %s\n", stamp, inputFile, outputFile)
	}

	lastModTime := getFileModTime(inputFile)
	rebuild("")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	ticker := time.NewTicker(time.Duration(intervalMs) * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-sigChan:
			fmt.Println("\nStopping watch mode...")
			return
		case <-ticker.C:
			if mod := getFileModTime(inputFile); !mod.Equal(lastModTime) {
				lastModTime = mod
				// Let the editor finish writing before reading the file.
				time.Sleep(200 * time.Millisecond)
				if clearConsole {
					fmt.Print("\033[H\033[2J\033[3J")
				}
				rebuild("File changed, recompiling...")
			}
		}
	}
}

func getFileModTime(filename string) time.Time {
	info, err := os.Stat(filename)
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}

func printHelp() {
	fmt.Printf(`%s - a statically-typed superset of Lua

Usage:
  lunar [options] <input.lunar>

Options:
  -o <file>             Output file (default: input with .lua extension)
  --target <version>    Target Lua version: lua53 (default) or lua54
  --source-map          Also write <output>.map
  --config <file>       Path to lunar.config.json (default: searched upward)
  --watch               Recompile when the input file changes
  --watch-interval <ms> Watch polling interval (default 500)
  --watch-clear         Clear the console before each rebuild
  --version             Print the version
  --help                Print this help

Declaration files (*.d.lunar) next to the input file are loaded automatically.
`, version.String())
}
