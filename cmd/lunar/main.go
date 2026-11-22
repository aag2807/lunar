package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"strings"
	"syscall"
	"time"

	"lunar/internal/ast"
	"lunar/internal/codegen"
	"lunar/internal/config"
	"lunar/internal/diag"
	"lunar/internal/lexer"
	"lunar/internal/parser"
	"lunar/internal/types"
	"lunar/internal/version"
)

// Exit codes are part of the contract with tools that run the compiler.
const (
	exitOK       = 0 // compiled (warnings allowed)
	exitErrors   = 1 // the source has errors; nothing was written
	exitUsage    = 2 // bad flags, missing input, bad configuration
	exitInternal = 3 // an I/O failure or a compiler bug
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
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run is the whole command: it returns the exit code instead of exiting, so
// every path out, a panic included, reports the same way.
func run(args []string, stdout, stderr io.Writer) (code int) {
	jsonMode := false
	for _, a := range args {
		if a == "--json" || a == "-json" {
			jsonMode = true
		}
	}
	rep := &reporter{json: jsonMode, stdout: stdout, stderr: stderr}

	defer func() {
		if r := recover(); r != nil {
			code = rep.internal(fmt.Sprintf("internal compiler error: %v\n%s", r, debug.Stack()))
		}
	}()

	flags := flag.NewFlagSet("lunar", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	outputFile := flags.String("o", "", "")
	sourceMap := flags.Bool("source-map", false, "")
	targetLua := flags.String("target", "", "")
	configFile := flags.String("config", "", "")
	watchMode := flags.Bool("watch", false, "")
	watchInterval := flags.Int("watch-interval", 500, "")
	watchClear := flags.Bool("watch-clear", false, "")
	flags.Bool("json", false, "")
	showVersion := flags.Bool("version", false, "")
	showHelp := flags.Bool("help", false, "")

	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printHelp(stdout)
			return exitOK
		}
		return rep.usage(err.Error())
	}

	if *showVersion {
		fmt.Fprintln(stdout, version.String())
		return exitOK
	}
	if *showHelp {
		printHelp(stdout)
		return exitOK
	}

	rest := flags.Args()
	if len(rest) < 1 {
		return rep.usage("no input file specified (usage: lunar [options] <input.lunar>)")
	}
	inputFile := rest[0]
	if _, err := os.Stat(inputFile); err != nil {
		return rep.usage(fmt.Sprintf("input file '%s' does not exist", inputFile))
	}
	if !strings.HasSuffix(inputFile, ".lunar") && !rep.json {
		fmt.Fprintf(stderr, "Warning: Input file '%s' does not have .lunar extension\n", inputFile)
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
		return rep.usage(fmt.Sprintf("loading config: %v", err))
	}
	cfg.Merge(false, *sourceMap, false, *targetLua)

	target := cfg.CompilerOptions.Target
	if target == "" {
		target = "lua53"
	}
	if !supportedTargets[target] {
		return rep.usage(fmt.Sprintf("unsupported target '%s' (supported: lua53, lua54)", target))
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

	opts := compileOptions{sourceMap: *sourceMap, target: target}
	if *watchMode {
		runWatchMode(rep, inputFile, output, opts, *watchInterval, *watchClear)
		return exitOK
	}
	return rep.build(inputFile, output, opts)
}

type compileOptions struct {
	sourceMap bool
	target    string
}

// compileResult is what one compile produced: the diagnostics, the source
// text of every file they may point into, and any I/O failure.
type compileResult struct {
	diags   diag.List
	sources map[string]string
	ioErr   error
}

// compile runs lexer -> parser -> checker -> codegen on one file and writes
// the Lua. Nothing is written unless the source has no errors, so the last
// good output stays in place.
func compile(inputFile, outputFile string, opts compileOptions) compileResult {
	res := compileResult{sources: map[string]string{}}

	source, err := os.ReadFile(inputFile)
	if err != nil {
		res.ioErr = fmt.Errorf("reading %s: %w", inputFile, err)
		return res
	}
	res.sources[inputFile] = string(source)

	// Declarations next to the input are checked first, so their names are
	// known to the script; they are never emitted.
	var declarations []ast.Statement
	declFiles, err := discoverDeclarationFiles(inputFile)
	if err != nil {
		res.ioErr = fmt.Errorf("finding declaration files: %w", err)
		return res
	}
	for _, declFile := range declFiles {
		text, err := os.ReadFile(declFile)
		if err != nil {
			res.ioErr = fmt.Errorf("reading %s: %w", declFile, err)
			return res
		}
		res.sources[declFile] = string(text)
		p := parser.New(lexer.NewFile(declFile, string(text)))
		statements := p.Parse()
		res.diags = append(res.diags, parseDiagnostics(p.Errors())...)
		declarations = append(declarations, statements...)
	}

	p := parser.New(lexer.NewFile(inputFile, string(source)))
	statements := p.Parse()
	res.diags = append(res.diags, parseDiagnostics(p.Errors())...)
	if res.diags.HasErrors() {
		return res
	}

	// The checker also decides ':' versus '.' and which calls construct, so
	// it always runs, and codegen depends on what it records.
	checker := types.NewChecker(getStdlibPath())
	res.diags = append(res.diags, typeDiagnostics(checker.Check(append(declarations, statements...)))...)
	if res.diags.HasErrors() {
		return res
	}

	var luaCode, mapJSON string
	if opts.sourceMap {
		code, sm := codegen.GenerateWithSourceMapAndTarget(statements, inputFile, outputFile, opts.target)
		luaCode = code + "\n" + sm.GenerateComment(filepath.Base(outputFile)+".map") + "\n"
		if mapJSON, err = sm.ToJSON(); err != nil {
			res.ioErr = fmt.Errorf("encoding the source map: %w", err)
			return res
		}
	} else {
		luaCode = codegen.GenerateWithTarget(statements, opts.target)
	}

	if err := os.MkdirAll(filepath.Dir(outputFile), 0o755); err != nil {
		res.ioErr = fmt.Errorf("creating the output directory: %w", err)
		return res
	}
	if opts.sourceMap {
		if err := os.WriteFile(outputFile+".map", []byte(mapJSON), 0o644); err != nil {
			res.ioErr = fmt.Errorf("writing %s.map: %w", outputFile, err)
			return res
		}
	}
	if err := os.WriteFile(outputFile, []byte(luaCode), 0o644); err != nil {
		res.ioErr = fmt.Errorf("writing %s: %w", outputFile, err)
	}
	return res
}

func parseDiagnostics(errs []*parser.ParseError) diag.List {
	out := make(diag.List, len(errs))
	for i, e := range errs {
		out[i] = diag.Diagnostic{File: e.File, Line: e.Line, Column: e.Column, EndLine: e.EndLine,
			EndColumn: e.EndColumn, Severity: diag.Error, Code: e.Code, Message: e.Message}
	}
	return out
}

func typeDiagnostics(errs []*types.TypeError) diag.List {
	out := make(diag.List, len(errs))
	for i, e := range errs {
		code := e.Code
		if code == "" {
			code = "type"
		}
		out[i] = diag.Diagnostic{File: e.File, Line: e.Line, Column: e.Column, EndLine: e.EndLine,
			EndColumn: e.EndColumn, Severity: diag.Error, Code: code, Message: e.Message}
	}
	return out
}

// discoverDeclarationFiles finds all .d.lunar files in the same directory as the input file.
func discoverDeclarationFiles(inputFile string) ([]string, error) {
	return filepath.Glob(filepath.Join(filepath.Dir(inputFile), "*.d.lunar"))
}

// reporter prints results for people (stderr, with source context) or, with
// --json, as one JSON object per line on stdout and nothing on stderr.
type reporter struct {
	json   bool
	stdout io.Writer
	stderr io.Writer
}

// build compiles once and reports, returning the exit code.
func (r *reporter) build(inputFile, outputFile string, opts compileOptions) int {
	res := compile(inputFile, outputFile, opts)
	res.diags.Sort()

	if res.ioErr != nil {
		r.emit(res, inputFile, outputFile, opts, false)
		return r.internalDiag("io", res.ioErr.Error())
	}
	written := !res.diags.HasErrors()
	r.emit(res, inputFile, outputFile, opts, written)
	if !written {
		return exitErrors
	}
	return exitOK
}

// emit prints a compile's diagnostics and, in JSON mode, its summary line.
func (r *reporter) emit(res compileResult, inputFile, outputFile string, opts compileOptions, written bool) {
	if r.json {
		for _, d := range res.diags {
			d.File = absolute(d.File)
			diag.WriteJSON(r.stdout, d)
		}
		if res.ioErr == nil {
			diag.WriteJSON(r.stdout, diag.Summary{Errors: res.diags.Errors(), Warnings: res.diags.Warnings(), Files: 1, Written: written})
		}
		return
	}
	if len(res.diags) > 0 {
		fmt.Fprint(r.stderr, formatDiagnostics(res.diags, res.sources))
	}
	if written {
		fmt.Fprintf(r.stdout, "Successfully compiled %s -> %s\n", inputFile, outputFile)
		if opts.sourceMap {
			fmt.Fprintf(r.stdout, "Source map: %s.map\n", outputFile)
		}
	}
}

// usage reports a problem with how lunar was invoked.
func (r *reporter) usage(message string) int {
	if !r.json {
		fmt.Fprintf(r.stderr, "Error: %s\nRun 'lunar --help' for more information\n", message)
		return exitUsage
	}
	diag.WriteJSON(r.stdout, diag.Diagnostic{Severity: diag.Error, Code: "usage", Message: message})
	diag.WriteJSON(r.stdout, diag.Summary{Errors: 1})
	return exitUsage
}

// internal reports a compiler failure that is not the source's fault.
func (r *reporter) internal(message string) int {
	return r.internalDiag("internal", message)
}

func (r *reporter) internalDiag(code, message string) int {
	if !r.json {
		fmt.Fprintf(r.stderr, "Error: %s\n", message)
		return exitInternal
	}
	diag.WriteJSON(r.stdout, diag.Diagnostic{Severity: diag.Error, Code: code, Message: message})
	diag.WriteJSON(r.stdout, diag.Summary{Errors: 1})
	return exitInternal
}

// absolute makes a path absolute for JSON output; names such as
// <stdlib>/lua.d.lunar are left alone.
func absolute(path string) string {
	if path == "" || strings.HasPrefix(path, "<") || filepath.IsAbs(path) {
		return path
	}
	if abs, err := filepath.Abs(path); err == nil {
		return abs
	}
	return path
}

// formatDiagnostics renders diagnostics with a line of source context on
// each side and a caret under the reported column.
func formatDiagnostics(diags diag.List, sources map[string]string) string {
	var sb strings.Builder
	sb.WriteString("Compilation failed:\n")
	for i, d := range diags {
		sb.WriteString("\n")
		fmt.Fprintf(&sb, "  Error %d: %s\n", i+1, d.Location())
		fmt.Fprintf(&sb, "  %s\n", d.Message)

		text, ok := sources[d.File]
		if !ok || d.Line < 1 {
			continue
		}
		sb.WriteString("\n")
		lines := strings.Split(text, "\n")
		start, end := d.Line-1, d.Line+1
		if start < 1 {
			start = 1
		}
		if end > len(lines) {
			end = len(lines)
		}
		for n := start; n <= end; n++ {
			content := lines[n-1]
			fmt.Fprintf(&sb, "  %4d | %s\n", n, content)
			if n == d.Line && d.Column > 0 && d.Column <= len(content)+1 {
				fmt.Fprintf(&sb, "       | %s^\n", strings.Repeat(" ", d.Column-1))
			}
		}
	}
	return sb.String()
}

// runWatchMode recompiles inputFile whenever its modification time changes.
// In JSON mode every rebuild ends with its own summary line.
func runWatchMode(rep *reporter, inputFile, outputFile string, opts compileOptions, intervalMs int, clearConsole bool) {
	status := func(format string, args ...any) {
		if !rep.json {
			fmt.Fprintf(rep.stdout, "[%s] "+format+"\n", append([]any{time.Now().Format("15:04:05")}, args...)...)
		}
	}
	status("Watching %s for changes (Ctrl+C to stop)", inputFile)
	rep.build(inputFile, outputFile, opts)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	ticker := time.NewTicker(time.Duration(intervalMs) * time.Millisecond)
	defer ticker.Stop()

	lastModTime := getFileModTime(inputFile)
	for {
		select {
		case <-sigChan:
			status("Stopping watch mode")
			return
		case <-ticker.C:
			if mod := getFileModTime(inputFile); !mod.Equal(lastModTime) {
				lastModTime = mod
				// Let the editor finish writing before reading the file.
				time.Sleep(200 * time.Millisecond)
				if clearConsole && !rep.json {
					fmt.Fprint(rep.stdout, "\033[H\033[2J\033[3J")
				}
				status("File changed, recompiling")
				rep.build(inputFile, outputFile, opts)
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

func printHelp(w io.Writer) {
	fmt.Fprintf(w, `%s - a statically-typed superset of Lua

Usage:
  lunar [options] <input.lunar>

Options:
  -o <file>             Output file (default: input with .lua extension)
  --target <version>    Target Lua version: lua53 (default) or lua54
  --source-map          Also write <output>.map
  --config <file>       Path to lunar.config.json (default: searched upward)
  --json                Report as JSON lines on stdout (see below)
  --watch               Recompile when the input file changes
  --watch-interval <ms> Watch polling interval (default 500)
  --watch-clear         Clear the console before each rebuild
  --version             Print the version
  --help                Print this help

Declaration files (*.d.lunar) next to the input file are loaded automatically.

Exit codes: 0 compiled, 1 the source has errors (nothing written),
2 usage or configuration error, 3 I/O failure or internal error.

With --json, stdout carries one JSON object per line and stderr stays
empty: a "diagnostic" line per problem (file, line, column, endLine,
endColumn, severity, code, message; lines and byte columns are 1-based and
the end is exclusive), then one "summary" line (errors, warnings, files,
written).
`, version.String())
}
