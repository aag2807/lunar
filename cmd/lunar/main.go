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
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"lunar/internal/diag"
	"lunar/internal/lsp"
	"lunar/internal/project"
	"lunar/internal/version"
)

// Exit codes are part of the contract with tools that run the compiler.
const (
	exitOK       = 0 // compiled (warnings allowed)
	exitErrors   = 1 // the source has errors; nothing was written
	exitUsage    = 2 // bad flags, missing input, bad configuration
	exitInternal = 3 // an I/O failure or a compiler bug
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// stringList is a repeatable flag: --decl a --decl b.
type stringList []string

func (s *stringList) String() string     { return strings.Join(*s, ",") }
func (s *stringList) Set(v string) error { *s = append(*s, v); return nil }

// run is the whole command: it returns the exit code instead of exiting, so
// every path out, a panic included, reports the same way.
func run(args []string, stdout, stderr io.Writer) (code int) {
	rep := &reporter{stdout: stdout, stderr: stderr}
	// JSON mode applies from the first line, so even a usage error comes
	// out as JSON. --json=false turns it back off, as the flag would.
	for _, a := range args {
		if a == "--" {
			break
		}
		name, value, hasValue := strings.Cut(strings.TrimLeft(a, "-"), "=")
		if strings.HasPrefix(a, "-") && name == "json" {
			on := true
			if hasValue {
				on, _ = strconv.ParseBool(value)
			}
			rep.json = on
		}
	}
	defer func() {
		if r := recover(); r != nil {
			code = rep.failure("internal", fmt.Sprintf("internal compiler error: %v\n%s", r, debug.Stack()))
		}
	}()

	// lunar lsp: the language server, on stdin and stdout.
	if len(args) > 0 && args[0] == "lsp" {
		if err := lsp.NewServer().Run(); err != nil {
			fmt.Fprintf(stderr, "lunar lsp: %v\n", err)
			return exitInternal
		}
		return exitOK
	}

	buildMode := len(args) > 0 && args[0] == "build"
	if buildMode {
		args = args[1:]
	}

	flags := flag.NewFlagSet("lunar", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var decls stringList
	flags.Var(&decls, "decl", "")
	target := flags.String("target", "", "")
	sourceMap := flags.Bool("source-map", false, "")
	watch := flags.Bool("watch", false, "")
	watchInterval := flags.Int("watch-interval", 500, "")
	watchClear := flags.Bool("watch-clear", false, "")
	flags.Bool("json", false, "")
	showVersion := flags.Bool("version", false, "")
	showHelp := flags.Bool("help", false, "")
	// build
	projectFile := flags.String("project", "", "")
	outDir := flags.String("out", "", "")
	// single file
	outputFile := flags.String("o", "", "")
	configFile := flags.String("config", "", "")

	// Flags may come before or after the file name: parse up to each
	// argument that is not a flag, keep it, and carry on after it.
	var positional []string
	for rest := args; ; {
		if err := flags.Parse(rest); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				printHelp(stdout)
				return exitOK
			}
			return rep.usage(err.Error())
		}
		left := flags.Args()
		if consumed := len(rest) - len(left); consumed > 0 && rest[consumed-1] == "--" {
			positional = append(positional, left...) // everything after -- is a file
			break
		}
		if len(left) == 0 {
			break
		}
		positional = append(positional, left[0])
		rest = left[1:]
	}
	if *showVersion {
		fmt.Fprintln(stdout, version.String())
		return exitOK
	}
	if *showHelp {
		printHelp(stdout)
		return exitOK
	}

	if *target != "" && *target != "lua53" && *target != "lua54" {
		return rep.usage(fmt.Sprintf("unsupported target '%s' (supported: lua53, lua54)", *target))
	}
	opts := project.Options{Decls: decls, Target: *target}
	if flagSet(flags, "source-map") {
		opts.SourceMap = sourceMap
	}

	var once func() int
	if buildMode {
		if len(positional) > 0 {
			return rep.usage("lunar build takes no file arguments; it compiles the whole project")
		}
		opts.ConfigPath, opts.OutDir = *projectFile, *outDir
		once = func() int { return rep.build(opts) }
	} else {
		if len(positional) < 1 {
			return rep.usage("no input file specified (usage: lunar [options] <input.lunar>, or lunar build)")
		}
		if len(positional) > 1 {
			return rep.usage(fmt.Sprintf("one input file at a time (got %s); use lunar build for a project", strings.Join(positional, ", ")))
		}
		input := positional[0]
		if _, err := os.Stat(input); err != nil {
			return rep.usage(fmt.Sprintf("input file '%s' does not exist", input))
		}
		output := *outputFile
		if output == "" {
			output = strings.TrimSuffix(input, ".lunar") + ".lua"
		}
		opts.ConfigPath, opts.Files = *configFile, []string{input}
		once = func() int { return rep.compileFile(opts, input, output) }
	}

	if *watch {
		watchLoop(rep, opts, once, *watchInterval, *watchClear)
		return exitOK
	}
	return once()
}

func flagSet(flags *flag.FlagSet, name string) bool {
	set := false
	flags.Visit(func(f *flag.Flag) {
		if f.Name == name {
			set = true
		}
	})
	return set
}

// reporter prints results for people (stderr, with source context) or, with
// --json, as one JSON object per line on stdout and nothing on stderr.
type reporter struct {
	json   bool
	stdout io.Writer
	stderr io.Writer
}

// build compiles the whole project into its output folder.
func (r *reporter) build(opts project.Options) int {
	p, diags := project.Open(opts)
	if diags.HasErrors() {
		return r.failure("config", diags[0].Message)
	}
	res, diags := project.Build(p)
	diags.Sort()
	if code := r.ioOrConfig(diags); code != 0 {
		return code
	}
	r.report(diags, res.Files, res.Written)
	if res.Written && !r.json {
		fmt.Fprintf(r.stdout, "Compiled %d file(s) into %s\n", res.Files, p.OutDir)
	}
	if !res.Written {
		return exitErrors
	}
	return exitOK
}

// compileFile compiles one file (checking what it imports) to output.
func (r *reporter) compileFile(opts project.Options, input, output string) int {
	p, diags := project.Open(opts)
	if diags.HasErrors() {
		return r.failure("config", diags[0].Message)
	}
	prog, diags := p.Check()
	if !diags.HasErrors() {
		outs, emitDiags := prog.Emit()
		diags = append(diags, emitDiags...)
		if !diags.HasErrors() {
			if err := writeSingle(outs, output); err != nil {
				return r.failure("io", err.Error())
			}
		}
	}
	diags.Sort()
	if code := r.ioOrConfig(diags); code != 0 {
		return code
	}
	written := !diags.HasErrors()
	r.report(diags, 1, written)
	if !written {
		return exitErrors
	}
	if !r.json {
		fmt.Fprintf(r.stdout, "Successfully compiled %s -> %s\n", input, output)
		if p.SourceMap {
			fmt.Fprintf(r.stdout, "Source map: %s.map\n", output)
		}
	}
	return exitOK
}

// writeSingle writes a one-file build's Lua (and map) to output, staging it
// next to the target and renaming it over, so a reader never sees half.
func writeSingle(outs []project.Output, output string) error {
	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		return fmt.Errorf("creating the output directory: %w", err)
	}
	for _, o := range outs {
		target := output
		if strings.HasSuffix(o.Rel, ".map") {
			target = output + ".map"
		} else if !strings.HasSuffix(o.Rel, ".lua") {
			continue
		}
		tmp := target + ".lunar-tmp"
		if err := os.WriteFile(tmp, o.Content, 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", target, err)
		}
		if err := os.Rename(tmp, target); err != nil {
			os.Remove(tmp)
			return fmt.Errorf("writing %s: %w", target, err)
		}
	}
	return nil
}

// ioOrConfig turns a failure that is not about the source into its exit
// code, or returns 0.
func (r *reporter) ioOrConfig(diags diag.List) int {
	for _, d := range diags {
		switch d.Code {
		case "io":
			return r.failure("io", d.Message)
		case "config":
			return r.failure("config", d.Message)
		}
	}
	return 0
}

// report prints diagnostics and, in JSON mode, the closing summary.
func (r *reporter) report(diags diag.List, files int, written bool) {
	if r.json {
		for _, d := range diags {
			d.File = absolute(d.File)
			diag.WriteJSON(r.stdout, d)
		}
		diag.WriteJSON(r.stdout, diag.Summary{Errors: diags.Errors(), Warnings: diags.Warnings(), Files: files, Written: written})
		return
	}
	if len(diags) > 0 {
		fmt.Fprint(r.stderr, formatDiagnostics(diags))
	}
}

// usage reports a problem with how lunar was invoked.
func (r *reporter) usage(message string) int {
	return r.fail(exitUsage, "usage", message)
}

// failure reports an outcome that is not the source's fault: bad
// configuration (exit 2), I/O failure or compiler bug (exit 3).
func (r *reporter) failure(code, message string) int {
	if code == "config" {
		return r.fail(exitUsage, code, message)
	}
	return r.fail(exitInternal, code, message)
}

func (r *reporter) fail(exit int, code, message string) int {
	if !r.json {
		fmt.Fprintf(r.stderr, "Error: %s\n", message)
		if code == "usage" {
			fmt.Fprintln(r.stderr, "Run 'lunar --help' for more information")
		}
		return exit
	}
	diag.WriteJSON(r.stdout, diag.Diagnostic{Severity: diag.Error, Code: code, Message: message})
	diag.WriteJSON(r.stdout, diag.Summary{Errors: 1})
	return exit
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

// displayPath shows a path relative to the working directory when it is
// inside it, as people typed it.
func displayPath(path string) string {
	if wd, err := os.Getwd(); err == nil && filepath.IsAbs(path) {
		if rel, err := filepath.Rel(wd, path); err == nil && !strings.HasPrefix(rel, "..") {
			return rel
		}
	}
	return path
}

// formatDiagnostics renders diagnostics with a line of source context on
// each side and a caret under the reported column.
func formatDiagnostics(diags diag.List) string {
	var sb strings.Builder
	sb.WriteString("Compilation failed:\n")
	sources := map[string][]string{}
	for i, d := range diags {
		sb.WriteString("\n")
		loc := d
		loc.File = displayPath(d.File)
		fmt.Fprintf(&sb, "  Error %d: %s\n", i+1, loc.Location())
		fmt.Fprintf(&sb, "  %s\n", d.Message)

		lines, ok := sources[d.File]
		if !ok {
			if data, err := os.ReadFile(d.File); err == nil {
				lines = strings.Split(string(data), "\n")
			}
			sources[d.File] = lines
		}
		if len(lines) == 0 || d.Line < 1 || d.Line > len(lines) {
			continue
		}
		sb.WriteString("\n")
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

// watchLoop runs once, then again whenever a file the project reads
// changes. In JSON mode every rebuild ends with its own summary line.
func watchLoop(rep *reporter, opts project.Options, once func() int, intervalMs int, clearConsole bool) {
	status := func(format string, args ...any) {
		if !rep.json {
			fmt.Fprintf(rep.stdout, "[%s] "+format+"\n", append([]any{time.Now().Format("15:04:05")}, args...)...)
		}
	}
	status("Watching for changes (Ctrl+C to stop)")
	once()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	ticker := time.NewTicker(time.Duration(intervalMs) * time.Millisecond)
	defer ticker.Stop()

	last := snapshot(opts)
	for {
		select {
		case <-sigChan:
			status("Stopping watch mode")
			return
		case <-ticker.C:
			if now := snapshot(opts); now != last {
				last = now
				// Let the editor finish writing before reading the files.
				time.Sleep(200 * time.Millisecond)
				if clearConsole && !rep.json {
					fmt.Fprint(rep.stdout, "\033[H\033[2J\033[3J")
				}
				status("Change detected, recompiling")
				once()
			}
		}
	}
}

// snapshot fingerprints every file the project reads: sources, plain Lua
// and declarations, by path, size and modification time.
func snapshot(opts project.Options) string {
	p, diags := project.Open(opts)
	if diags.HasErrors() {
		return "config error: " + diags[0].Message
	}
	files := append(append(append([]string{}, p.Sources...), p.LuaFiles...), p.DeclFiles...)
	// Modules reached only through imports are not in Sources; watching the
	// whole root covers them.
	filepath.WalkDir(p.Root, func(path string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(path, ".lunar") {
			files = append(files, path)
		}
		return nil
	})
	sort.Strings(files)
	var sb strings.Builder
	for _, f := range files {
		if info, err := os.Stat(f); err == nil {
			fmt.Fprintf(&sb, "%s %d %d\n", f, info.Size(), info.ModTime().UnixNano())
		}
	}
	return sb.String()
}

func printHelp(w io.Writer) {
	fmt.Fprintf(w, `%s - a statically-typed superset of Lua

Usage:
  lunar build [options]          Compile the whole project into its outDir
  lunar [options] <input.lunar>  Compile one file (and check what it imports)
  lunar lsp                      Run the language server on stdin/stdout

Options:
  --target <version>    Target Lua version: lua53 (default) or lua54
  --decl <path>         Also load declarations from this file or folder (repeatable)
  --source-map          Also write <output>.map
  --json                Report as JSON lines on stdout (see below)
  --watch               Recompile when a file changes
  --watch-interval <ms> Watch polling interval (default 500)
  --watch-clear         Clear the console before each rebuild
  --version             Print the version
  --help                Print this help

lunar build:
  --project <file>      The lunar.config.json to use (default: searched upward)
  --out <dir>           Output folder, overriding outDir

Single file:
  -o <file>             Output file (default: input with .lua extension)
  --config <file>       The lunar.config.json to use (default: searched upward)

lunar.config.json (paths relative to it):
  {"rootDir": "scripts", "outDir": "../build/scripts", "target": "lua54",
   "declarations": ["../engine/api"], "copyLua": true, "sourceMap": false}

Declarations are the bundled Lua standard library, every *.d.lunar under
rootDir, and the "declarations" and --decl paths. Imports starting with ./
or ../ are relative to the importing file, others to rootDir; require()
names are relative to rootDir ("ai/brain.lunar" is require("ai.brain")).

Exit codes: 0 compiled, 1 the source has errors (nothing written),
2 usage or configuration error, 3 I/O failure or internal error.

With --json, stdout carries one JSON object per line and stderr stays
empty: a "diagnostic" line per problem (file, line, column, endLine,
endColumn, severity, code, message; lines and byte columns are 1-based and
the end is exclusive), then one "summary" line (errors, warnings, files,
written).
`, version.String())
}
