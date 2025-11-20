package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"lunar/internal/ast"
	"lunar/internal/bundler"
	"lunar/internal/cache"
	"lunar/internal/codegen"
	"lunar/internal/config"
	"lunar/internal/docgen"
	"lunar/internal/formatter"
	"lunar/internal/lexer"
	"lunar/internal/linter"
	"lunar/internal/luarocks"
	"lunar/internal/migrate"
	"lunar/internal/minify"
	"lunar/internal/optimizer"
	"lunar/internal/parser"
	"lunar/internal/types"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

const version = "1.5.0"

func main() {
	// Define command-line flags
	outputFile := flag.String("o", "", "Output file (default: replaces .lunar with .lua)")
	noTypeCheck := flag.Bool("no-typecheck", false, "Skip type checking")
	sourceMap := flag.Bool("source-map", false, "Generate source map (.lua.map file)")
	showVersion := flag.Bool("version", false, "Show version information")
	showHelp := flag.Bool("help", false, "Show help message")
	replMode := flag.Bool("repl", false, "Start interactive REPL mode")
	watchMode := flag.Bool("watch", false, "Watch files for changes and recompile")
	watchInterval := flag.Int("watch-interval", 500, "Watch interval in milliseconds")
	watchClear := flag.Bool("watch-clear", false, "Clear console before each rebuild in watch mode")
	formatMode := flag.Bool("format", false, "Format the source file")
	formatInPlace := flag.Bool("format-write", false, "Format and write back to file")
	lintMode := flag.Bool("lint", false, "Run linter to check for issues")
	bundleMode := flag.Bool("bundle", false, "Bundle all dependencies into a single file")
	runMode := flag.Bool("run", false, "Run the compiled Lua file after compilation")
	testMode := flag.Bool("test", false, "Run tests in the specified directory")
	testFilter := flag.String("filter", "", "Run only tests matching this pattern (regex)")
	testWatch := flag.Bool("test-watch", false, "Watch test files and re-run on changes")
	testCoverage := flag.Bool("coverage", false, "Generate code coverage report")
	docsMode := flag.Bool("docs", false, "Generate documentation for source files")
	docsOutput := flag.String("docs-output", "", "Output directory for generated docs (default: ./docs)")
	targetLua := flag.String("target", "", "Target Lua version: lua51, lua52, lua53, lua54, luajit")
	configFile := flag.String("config", "", "Path to lunar.config.json (auto-detected if not specified)")

	// Optimization flags
	minifyMode := flag.Bool("minify", false, "Minify output (remove whitespace and comments)")
	optimizeMode := flag.Bool("optimize", false, "Enable optimizations (constant folding, dead code elimination)")
	buildMode := flag.String("mode", "", "Build mode: 'dev' (fast, debug info) or 'production' (optimized, minified)")
	noCache := flag.Bool("no-cache", false, "Disable incremental compilation cache")
	clearCache := flag.Bool("clear-cache", false, "Clear compilation cache and exit")
	cacheStats := flag.Bool("cache-stats", false, "Show cache statistics and exit")

	// LuaRocks integration flags
	rocksInstall := flag.String("rocks-install", "", "Install a LuaRocks package")
	rocksRemove := flag.String("rocks-remove", "", "Remove a LuaRocks package")
	rocksList := flag.Bool("rocks-list", false, "List installed LuaRocks packages")
	rocksSearch := flag.String("rocks-search", "", "Search for LuaRocks packages")
	rocksGenTypes := flag.String("rocks-types", "", "Generate type definitions for a package")
	rocksInit := flag.Bool("rocks-init", false, "Initialize lunarocks.json config file")
	rocksInstallDeps := flag.Bool("rocks-deps", false, "Install dependencies from lunarocks.json")

	// Migration tool flags
	migrateFile := flag.String("migrate", "", "Migrate Lua file to Lunar")
	migrateNoTypes := flag.Bool("migrate-no-types", false, "Don't add type annotations during migration")
	migrateUseConst := flag.Bool("migrate-const", true, "Use const for immutable variables")
	migrateOutput := flag.String("migrate-output", "", "Output file for migrated code (default: <input>.lunar)")

	flag.Parse()

	// Handle cache commands first
	compCache, _ := cache.New("")
	if *clearCache {
		if err := compCache.Clear(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to clear cache: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Cache cleared successfully")
		os.Exit(0)
	}

	if *cacheStats {
		stats := compCache.Stats()
		fmt.Println("Cache Statistics:")
		fmt.Printf("  Entries: %v\n", stats["entries"])
		fmt.Printf("  Cache Size: %v\n", stats["cachedSize"])
		fmt.Printf("  Cache Directory: %v\n", stats["cacheDir"])
		os.Exit(0)
	}

	// Handle LuaRocks commands
	if *rocksInstall != "" || *rocksRemove != "" || *rocksList || *rocksSearch != "" || *rocksGenTypes != "" || *rocksInit || *rocksInstallDeps {
		handleLuaRocksCommands(rocksInstall, rocksRemove, rocksList, rocksSearch, rocksGenTypes, rocksInit, rocksInstallDeps)
		os.Exit(0)
	}

	// Handle migration command
	if *migrateFile != "" {
		handleMigration(*migrateFile, *migrateOutput, *migrateNoTypes, *migrateUseConst)
		os.Exit(0)
	}

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

	// Handle REPL mode
	if *replMode {
		runREPL()
		os.Exit(0)
	}

	// Apply build mode presets (only if flags weren't explicitly set)
	if *buildMode != "" {
		switch *buildMode {
		case "dev", "development":
			// Development mode: fast builds, debugging support
			if !*optimizeMode {
				// Keep optimize off for dev mode
			}
			if !*minifyMode {
				// Keep minify off for dev mode
			}
			if !*sourceMap && !*noCache {
				// Enable source maps in dev mode if not explicitly disabled
				*sourceMap = true
			}
			fmt.Println("Build mode: Development (fast builds, source maps enabled)")
		case "prod", "production":
			// Production mode: optimized, minified, no debug info
			if !*optimizeMode {
				*optimizeMode = true
			}
			if !*minifyMode {
				*minifyMode = true
			}
			// Note: Don't disable source-map if user explicitly enabled it
			fmt.Println("Build mode: Production (optimized, minified)")
		default:
			fmt.Fprintf(os.Stderr, "Error: Unknown build mode '%s'. Use 'dev' or 'production'\n", *buildMode)
			os.Exit(1)
		}
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

	// Handle format mode
	if *formatMode || *formatInPlace {
		if err := formatFile(inputFile, *formatInPlace); err != nil {
			fmt.Fprintf(os.Stderr, "Format failed: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Handle lint mode
	if *lintMode {
		if err := lintFile(inputFile); err != nil {
			fmt.Fprintf(os.Stderr, "Lint failed: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Handle test mode
	if *testMode {
		if *testWatch {
			// Watch mode: run tests continuously on file changes
			if err := watchTests(inputFile, *testFilter, *testCoverage); err != nil {
				fmt.Fprintf(os.Stderr, "Watch mode failed: %v\n", err)
				os.Exit(1)
			}
		} else {
			// Regular mode: run tests once
			if err := runTests(inputFile, *testFilter, *testCoverage); err != nil {
				fmt.Fprintf(os.Stderr, "Tests failed: %v\n", err)
				os.Exit(1)
			}
		}
		os.Exit(0)
	}

	// Handle docs mode
	if *docsMode {
		if err := generateDocs(inputFile, *docsOutput); err != nil {
			fmt.Fprintf(os.Stderr, "Documentation generation failed: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Validate input file extension
	if !strings.HasSuffix(inputFile, ".lunar") {
		fmt.Fprintf(os.Stderr, "Warning: Input file '%s' does not have .lunar extension\n", inputFile)
	}

	// Load configuration
	var cfg *config.Config
	var err error
	var configDir string
	if *configFile != "" {
		cfg, err = config.LoadFromFile(*configFile)
		configDir = filepath.Dir(*configFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}
	} else {
		// Auto-discover config file
		configDir, cfg, err = config.FindConfig(filepath.Dir(inputFile))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}
		if configDir == "" {
			configDir = filepath.Dir(inputFile)
		}
	}

	// Merge command-line flags with config (CLI takes precedence)
	cfg.Merge(*noTypeCheck, *sourceMap, *bundleMode, *targetLua)

	// Determine output file
	output := *outputFile
	if output == "" {
		if cfg.OutFile != "" {
			output = filepath.Join(configDir, cfg.OutFile)
		} else if cfg.OutDir != "" {
			baseName := strings.TrimSuffix(filepath.Base(inputFile), ".lunar") + ".lua"
			output = filepath.Join(configDir, cfg.OutDir, baseName)
		} else {
			output = strings.TrimSuffix(inputFile, ".lunar") + ".lua"
		}
	}

	// Create output directory if it doesn't exist
	outDir := filepath.Dir(output)
	if err := os.MkdirAll(outDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	// Bundle mode
	if *bundleMode || cfg.CompilerOptions.Bundle {
		if *watchMode {
			runBundleWatchMode(inputFile, output, !*noTypeCheck, *runMode, *watchInterval, *watchClear, cfg)
		} else {
			if err := bundleFile(inputFile, output, !*noTypeCheck, cfg); err != nil {
				fmt.Fprintf(os.Stderr, "Bundle failed:\n%v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Successfully bundled %s -> %s\n", inputFile, output)

			// Run if requested
			if *runMode {
				if err := runLuaFile(output); err != nil {
					fmt.Fprintf(os.Stderr, "Run failed: %v\n", err)
					os.Exit(1)
				}
			}
		}
		os.Exit(0)
	}

	// Watch mode or single compilation
	if *watchMode {
		useCache := !*noCache
		runWatchMode(inputFile, output, !*noTypeCheck, *sourceMap, *watchInterval, *runMode, cfg.CompilerOptions.Target, useCache, *optimizeMode, *minifyMode, *watchClear)
	} else {
		// Compile the file
		useCache := !*noCache
		if err := compile(inputFile, output, !*noTypeCheck, *sourceMap, cfg.CompilerOptions.Target, useCache, *optimizeMode, *minifyMode); err != nil {
			fmt.Fprintf(os.Stderr, "Compilation failed:\n%v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Successfully compiled %s -> %s\n", inputFile, output)
		if *sourceMap {
			fmt.Printf("Source map: %s.map\n", output)
		}

		// Run if requested
		if *runMode {
			if err := runLuaFile(output); err != nil {
				fmt.Fprintf(os.Stderr, "Run failed: %v\n", err)
				os.Exit(1)
			}
		}
	}
}

// bundleFile bundles a Lunar file and all its dependencies
func bundleFile(inputFile, outputFile string, typeCheck bool, cfg *config.Config) error {
	_, err := bundleFileWithDeps(inputFile, outputFile, typeCheck, cfg)
	return err
}

// bundleFileWithDeps bundles a Lunar file and returns all dependency file paths
func bundleFileWithDeps(inputFile, outputFile string, typeCheck bool, cfg *config.Config) ([]string, error) {
	b := bundler.NewWithConfig(inputFile, typeCheck, cfg)
	bundledCode, err := b.Bundle()
	if err != nil {
		return nil, err
	}

	if err := ioutil.WriteFile(outputFile, []byte(bundledCode), 0644); err != nil {
		return nil, fmt.Errorf("failed to write output file: %w", err)
	}

	return b.GetAllFiles(), nil
}

// runLuaFile executes a Lua file
func runLuaFile(luaFile string) error {
	// Try lua5.4, lua5.3, lua5.1, luajit, then lua
	luaCommands := []string{"lua5.4", "lua5.3", "lua5.1", "luajit", "lua"}

	var cmd *exec.Cmd
	for _, luaCmd := range luaCommands {
		if _, err := exec.LookPath(luaCmd); err == nil {
			cmd = exec.Command(luaCmd, luaFile)
			break
		}
	}

	if cmd == nil {
		return fmt.Errorf("no Lua interpreter found. Please install lua")
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd.Run()
}

// runBundleWatchMode watches files and rebundles/reruns on changes
func runBundleWatchMode(inputFile, outputFile string, typeCheck, runAfter bool, intervalMs int, clearConsole bool, cfg *config.Config) {
	fmt.Printf("Watching %s and dependencies for changes (Ctrl+C to stop)\n", inputFile)

	// Track modification times for all files
	fileModTimes := make(map[string]time.Time)
	var watchedFiles []string

	// Initial bundle
	files, err := bundleFileWithDeps(inputFile, outputFile, typeCheck, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[%s] Bundle failed:\n%v\n", time.Now().Format("15:04:05"), err)
		// Still watch the entry file even if bundle fails
		watchedFiles = []string{inputFile}
	} else {
		watchedFiles = files
		fmt.Printf("[%s] Successfully bundled %s -> %s\n", time.Now().Format("15:04:05"), inputFile, outputFile)
		fmt.Printf("[%s] Watching %d file(s)\n", time.Now().Format("15:04:05"), len(watchedFiles))
		if runAfter {
			fmt.Printf("[%s] Running %s...\n", time.Now().Format("15:04:05"), outputFile)
			if err := runLuaFile(outputFile); err != nil {
				fmt.Fprintf(os.Stderr, "[%s] Run failed: %v\n", time.Now().Format("15:04:05"), err)
			}
		}
	}

	// Initialize modification times
	for _, file := range watchedFiles {
		fileModTimes[file] = getFileModTime(file)
	}

	// Handle interrupt signal for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Watch loop with smart debouncing
	ticker := time.NewTicker(time.Duration(intervalMs) * time.Millisecond)
	defer ticker.Stop()

	var debounceTimer *time.Timer
	var pendingCompile bool
	var lastChangedFile string

	for {
		select {
		case <-sigChan:
			fmt.Println("\nStopping watch mode...")
			return
		case <-ticker.C:
			// Check if any watched file has changed
			changed := false
			var changedFile string
			for _, file := range watchedFiles {
				currentModTime := getFileModTime(file)
				if lastModTime, exists := fileModTimes[file]; exists {
					if !currentModTime.Equal(lastModTime) {
						changed = true
						changedFile = file
						fileModTimes[file] = currentModTime
						break
					}
				}
			}

			if changed {
				pendingCompile = true
				lastChangedFile = changedFile

				// Reset debounce timer - wait for files to stabilize
				if debounceTimer != nil {
					debounceTimer.Stop()
				}
				debounceTimer = time.AfterFunc(200*time.Millisecond, func() {
					if pendingCompile {
						pendingCompile = false
						if clearConsole {
							clearScreen()
						}
						// Get relative path for cleaner output
						relPath, err := filepath.Rel(filepath.Dir(inputFile), lastChangedFile)
						if err != nil {
							relPath = lastChangedFile
						}
						fmt.Printf("[%s] %s changed, rebundling...\n", time.Now().Format("15:04:05"), relPath)

						files, err := bundleFileWithDeps(inputFile, outputFile, typeCheck, cfg)
						if err != nil {
							fmt.Fprintf(os.Stderr, "[%s] Bundle failed:\n%v\n", time.Now().Format("15:04:05"), err)
						} else {
							// Update watched files list (dependencies may have changed)
							watchedFiles = files
							// Update mod times for new files
							for _, file := range watchedFiles {
								if _, exists := fileModTimes[file]; !exists {
									fileModTimes[file] = getFileModTime(file)
								}
							}

							fmt.Printf("[%s] Successfully bundled %s -> %s\n", time.Now().Format("15:04:05"), inputFile, outputFile)
							if runAfter {
								fmt.Printf("[%s] Running %s...\n", time.Now().Format("15:04:05"), outputFile)
								if err := runLuaFile(outputFile); err != nil {
									fmt.Fprintf(os.Stderr, "[%s] Run failed: %v\n", time.Now().Format("15:04:05"), err)
								}
							}
						}
					}
				})
			}
		}
	}
}

// compile compiles a Lunar source file to Lua
func compile(inputFile, outputFile string, typeCheck, generateSourceMap bool, target string, useCache, useOptimize, useMinify bool) error {
	// Check cache if enabled
	var compCache *cache.Cache
	if useCache {
		var err error
		compCache, err = cache.New("")
		if err == nil {
			if entry, found := compCache.Get(inputFile); found {
				// Cache hit! Write cached output
				if err := ioutil.WriteFile(outputFile, []byte(entry.CompiledCode), 0644); err == nil {
					if entry.SourceMapData != "" && generateSourceMap {
						mapFile := outputFile + ".map"
						ioutil.WriteFile(mapFile, []byte(entry.SourceMapData), 0644)
					}
					fmt.Printf("✓ Using cached: %s\n", inputFile)
					return nil
				}
			}
		}
	}

	// Auto-load declaration files from the same directory
	declarationStatements := []ast.Statement{}
	if typeCheck {
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
	}

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
		return formatParserErrors(inputFile, string(source), p.Errors())
	}

	// Type Checker: Validate types (if enabled)
	if typeCheck {
		// Combine declaration statements with main file statements
		// Declarations first so they're registered before main code
		allStatements := append(declarationStatements, statements...)
		typeErrors := types.Check(allStatements)
		if len(typeErrors) > 0 {
			return formatTypeErrors(inputFile, string(source), typeErrors)
		}
	}

	// Optimizer: Apply AST-level optimizations (if enabled)
	if useOptimize {
		opts := optimizer.DefaultOptions()
		opt := optimizer.New(opts)
		statements = opt.Optimize(statements)
	}

	// Code Generator: Transpile to Lua (only main file, not declarations)
	var luaCode string
	if generateSourceMap {
		// Generate with source map
		sourceMapData, sourceMapObj := codegen.GenerateWithSourceMapAndTarget(statements, inputFile, outputFile, false, target)
		luaCode = sourceMapData

		// Add source map comment to Lua file
		mapFilename := outputFile + ".map"
		sourceMapComment := sourceMapObj.GenerateComment(filepath.Base(mapFilename))
		luaCode += "\n" + sourceMapComment + "\n"

		// Write source map file
		sourceMapJSON, err := sourceMapObj.ToJSON()
		if err != nil {
			return fmt.Errorf("failed to generate source map JSON: %w", err)
		}
		if err := ioutil.WriteFile(mapFilename, []byte(sourceMapJSON), 0644); err != nil {
			return fmt.Errorf("failed to write source map file: %w", err)
		}
	} else {
		// Generate without source map
		luaCode = codegen.GenerateWithTarget(statements, target)
	}

	// Minifier: Reduce code size (if enabled)
	sourceMapData := ""
	if generateSourceMap && strings.Contains(luaCode, "sourceMappingURL") {
		// Extract source map data if present
		lines := strings.Split(luaCode, "\n")
		for i := len(lines) - 1; i >= 0; i-- {
			if strings.Contains(lines[i], "sourceMappingURL") {
				sourceMapData = lines[i]
				break
			}
		}
	}

	if useMinify {
		minOpts := minify.DefaultOptions()
		luaCode = minify.Minify(luaCode, minOpts)

		// Re-add source map comment if it was present
		if sourceMapData != "" {
			luaCode += "\n" + sourceMapData
		}
	}

	// Write output file
	if err := ioutil.WriteFile(outputFile, []byte(luaCode), 0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	// Cache the compilation result (if enabled)
	if useCache && compCache != nil {
		dependencies := []string{} // TODO: Track actual dependencies
		mapData := ""
		if generateSourceMap {
			mapFile := outputFile + ".map"
			if mapBytes, err := ioutil.ReadFile(mapFile); err == nil {
				mapData = string(mapBytes)
			}
		}
		compCache.Put(inputFile, luaCode, dependencies, mapData)
	}

	return nil
}

// discoverDeclarationFiles finds all .d.lunar files in the same directory as the input file
func discoverDeclarationFiles(inputFile string) ([]string, error) {
	dir := filepath.Dir(inputFile)

	// Find all .d.lunar files in the directory
	pattern := filepath.Join(dir, "*.d.lunar")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}

	return matches, nil
}

// parseDeclarationFile parses a declaration file and returns its statements
func parseDeclarationFile(filename string) ([]ast.Statement, error) {
	source, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	l := lexer.New(string(source))
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		return nil, formatParserErrors(filename, string(source), p.Errors())
	}

	return statements, nil
}

// formatParserErrors formats parser errors for display with source context
func formatParserErrors(filename string, source string, errors []*parser.ParseError) error {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\n%s: Parse errors found:\n\n", filename))

	lines := strings.Split(source, "\n")

	for i, err := range errors {
		if i > 0 {
			sb.WriteString("\n")
		}

		// Error location header
		sb.WriteString(fmt.Sprintf("  Error %d: %s:%d:%d\n", i+1, filename, err.Line, err.Column))
		sb.WriteString(fmt.Sprintf("  %s\n\n", err.Message))

		// Show source context (line before, error line, line after)
		startLine := err.Line - 2
		endLine := err.Line + 1
		if startLine < 1 {
			startLine = 1
		}
		if endLine > len(lines) {
			endLine = len(lines)
		}

		for lineNum := startLine; lineNum <= endLine; lineNum++ {
			lineContent := lines[lineNum-1]

			// Highlight the error line
			if lineNum == err.Line {
				sb.WriteString(fmt.Sprintf("  %4d | %s\n", lineNum, lineContent))

				// Add caret pointing to error column
				if err.Column > 0 && err.Column <= len(lineContent)+1 {
					pointer := strings.Repeat(" ", err.Column-1) + "^"
					sb.WriteString(fmt.Sprintf("       | %s\n", pointer))
				}
			} else {
				sb.WriteString(fmt.Sprintf("  %4d | %s\n", lineNum, lineContent))
			}
		}
	}

	return fmt.Errorf("%s", sb.String())
}

// formatTypeErrors formats type errors for display with source context
func formatTypeErrors(filename string, source string, errors []*types.TypeError) error {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\n%s: Type errors found:\n\n", filename))

	lines := strings.Split(source, "\n")

	for i, err := range errors {
		if i > 0 {
			sb.WriteString("\n")
		}

		// Error location header
		sb.WriteString(fmt.Sprintf("  Error %d: %s:%d:%d\n", i+1, filename, err.Line, err.Column))
		sb.WriteString(fmt.Sprintf("  %s\n\n", err.Message))

		// Show source context (line before, error line, line after)
		startLine := err.Line - 2
		endLine := err.Line + 1
		if startLine < 1 {
			startLine = 1
		}
		if endLine > len(lines) {
			endLine = len(lines)
		}

		for lineNum := startLine; lineNum <= endLine; lineNum++ {
			lineContent := lines[lineNum-1]

			// Highlight the error line
			if lineNum == err.Line {
				sb.WriteString(fmt.Sprintf("  %4d | %s\n", lineNum, lineContent))

				// Add caret pointing to error column
				if err.Column > 0 && err.Column <= len(lineContent)+1 {
					pointer := strings.Repeat(" ", err.Column-1) + "^"
					sb.WriteString(fmt.Sprintf("       | %s\n", pointer))
				}
			} else {
				sb.WriteString(fmt.Sprintf("  %4d | %s\n", lineNum, lineContent))
			}
		}
	}

	return fmt.Errorf("%s", sb.String())
}

// formatFile formats a Lunar source file
func formatFile(inputFile string, writeBack bool) error {
	// Read input file
	source, err := ioutil.ReadFile(inputFile)
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}

	// Parse the file
	l := lexer.New(string(source))
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		return formatParserErrors(inputFile, string(source), p.Errors())
	}

	// Format
	f := formatter.New(formatter.DefaultOptions())
	formatted := f.Format(statements)

	if writeBack {
		// Write back to file
		if err := ioutil.WriteFile(inputFile, []byte(formatted), 0644); err != nil {
			return fmt.Errorf("failed to write formatted file: %w", err)
		}
		fmt.Printf("Formatted %s\n", inputFile)
	} else {
		// Print to stdout
		fmt.Print(formatted)
	}

	return nil
}

// lintFile runs the linter on a Lunar source file
func lintFile(inputFile string) error {
	// Read input file
	source, err := ioutil.ReadFile(inputFile)
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}

	// Run linter
	issues, err := linter.LintSource(string(source))
	if err != nil {
		return err
	}

	if len(issues) == 0 {
		fmt.Printf("No issues found in %s\n", inputFile)
		return nil
	}

	fmt.Printf("Found %d issue(s) in %s:\n\n", len(issues), inputFile)
	for _, issue := range issues {
		fmt.Println(issue)
	}

	return nil
}

// runWatchMode watches files for changes and recompiles automatically
func runWatchMode(inputFile, outputFile string, typeCheck, generateSourceMap bool, intervalMs int, runAfter bool, target string, useCache, useOptimize, useMinify, clearConsole bool) {
	fmt.Printf("Watching %s for changes (Ctrl+C to stop)\n", inputFile)

	// Get initial modification time
	lastModTime := getFileModTime(inputFile)

	// Initial compilation
	if err := compile(inputFile, outputFile, typeCheck, generateSourceMap, target, useCache, useOptimize, useMinify); err != nil {
		fmt.Fprintf(os.Stderr, "[%s] Compilation failed:\n%v\n", time.Now().Format("15:04:05"), err)
	} else {
		fmt.Printf("[%s] Successfully compiled %s -> %s\n", time.Now().Format("15:04:05"), inputFile, outputFile)
		if runAfter {
			fmt.Printf("[%s] Running %s...\n", time.Now().Format("15:04:05"), outputFile)
			if err := runLuaFile(outputFile); err != nil {
				fmt.Fprintf(os.Stderr, "[%s] Run failed: %v\n", time.Now().Format("15:04:05"), err)
			}
		}
	}

	// Handle interrupt signal for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Watch loop with smart debouncing
	ticker := time.NewTicker(time.Duration(intervalMs) * time.Millisecond)
	defer ticker.Stop()

	var debounceTimer *time.Timer
	var pendingCompile bool

	for {
		select {
		case <-sigChan:
			fmt.Println("\nStopping watch mode...")
			return
		case <-ticker.C:
			currentModTime := getFileModTime(inputFile)
			if !currentModTime.Equal(lastModTime) {
				lastModTime = currentModTime
				pendingCompile = true

				// Reset debounce timer - wait for file to stabilize
				if debounceTimer != nil {
					debounceTimer.Stop()
				}
				debounceTimer = time.AfterFunc(200*time.Millisecond, func() {
					if pendingCompile {
						pendingCompile = false
						if clearConsole {
							clearScreen()
						}
						fmt.Printf("[%s] File changed, recompiling...\n", time.Now().Format("15:04:05"))
						if err := compile(inputFile, outputFile, typeCheck, generateSourceMap, target, useCache, useOptimize, useMinify); err != nil {
							fmt.Fprintf(os.Stderr, "[%s] Compilation failed:\n%v\n", time.Now().Format("15:04:05"), err)
						} else {
							fmt.Printf("[%s] Successfully compiled %s -> %s\n", time.Now().Format("15:04:05"), inputFile, outputFile)
							if runAfter {
								fmt.Printf("[%s] Running %s...\n", time.Now().Format("15:04:05"), outputFile)
								if err := runLuaFile(outputFile); err != nil {
									fmt.Fprintf(os.Stderr, "[%s] Run failed: %v\n", time.Now().Format("15:04:05"), err)
								}
							}
						}
					}
				})
			}
		}
	}
}

// getFileModTime returns the modification time of a file
func getFileModTime(filename string) time.Time {
	info, err := os.Stat(filename)
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}

// clearScreen clears the console/terminal screen
func clearScreen() {
	// ANSI escape codes work on Unix/Linux/Mac and Windows 10+
	fmt.Print("\033[H\033[2J")
	fmt.Print("\033[3J") // Clear scrollback buffer on some terminals
}

// printHelp prints help information
func printHelp() {
	fmt.Println("Lunar - A statically-typed superset of Lua")
	fmt.Printf("Version: %s\n\n", version)
	fmt.Println("Usage:")
	fmt.Println("  lunar [options] <input.lunar>")
	fmt.Println("  lunar --repl")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -o <file>           Output file (default: replaces .lunar with .lua)")
	fmt.Println("  --no-typecheck      Skip type checking")
	fmt.Println("  --source-map        Generate source map (.lua.map file)")
	fmt.Println("  --target <version>  Target Lua version: lua51, lua52, lua53, lua54, luajit")
	fmt.Println("  --watch             Watch files for changes and recompile")
	fmt.Println("  --watch-interval    Watch interval in ms (default: 500)")
	fmt.Println("  --watch-clear       Clear console before each rebuild in watch mode")
	fmt.Println("  --format            Format source code and print to stdout")
	fmt.Println("  --format-write      Format source code and write back to file")
	fmt.Println()
	fmt.Println("Optimization Options:")
	fmt.Println("  --mode <mode>       Build mode: 'dev' (fast, debug info) or 'production' (optimized)")
	fmt.Println("  --optimize          Enable AST optimizations (constant folding, dead code elimination)")
	fmt.Println("  --minify            Minify output (remove whitespace and comments)")
	fmt.Println("  --no-cache          Disable incremental compilation cache")
	fmt.Println("  --clear-cache       Clear compilation cache and exit")
	fmt.Println("  --cache-stats       Show cache statistics and exit")
	fmt.Println("  --lint              Check code for potential issues")
	fmt.Println("  --bundle            Bundle all dependencies into a single file")
	fmt.Println("  --run               Run the compiled Lua file after compilation")
	fmt.Println("  --test              Run tests in directory (*_test.lunar files)")
	fmt.Println("  --filter <pattern>  Run only tests matching pattern (with --test)")
	fmt.Println("  --test-watch        Watch test files and re-run on changes")
	fmt.Println("  --docs              Generate documentation for source files")
	fmt.Println("  --docs-output <dir> Output directory for docs (default: ./docs)")
	fmt.Println("  --config <file>     Path to lunar.config.json")
	fmt.Println("  --repl              Start interactive REPL mode")
	fmt.Println("  --version           Show version information")
	fmt.Println("  --help              Show this help message")
	fmt.Println()
	fmt.Println("LuaRocks Integration:")
	fmt.Println("  --rocks-install <pkg>   Install a LuaRocks package (e.g., lfs@1.8.0)")
	fmt.Println("  --rocks-remove <pkg>    Remove a LuaRocks package")
	fmt.Println("  --rocks-list            List installed LuaRocks packages")
	fmt.Println("  --rocks-search <query>  Search for LuaRocks packages")
	fmt.Println("  --rocks-types <pkg>     Generate type definitions for a package")
	fmt.Println("  --rocks-init            Initialize lunarocks.json config file")
	fmt.Println("  --rocks-deps            Install dependencies from lunarocks.json")
	fmt.Println()
	fmt.Println("Migration Tool:")
	fmt.Println("  --migrate <file>        Migrate Lua file to Lunar with type annotations")
	fmt.Println("  --migrate-output <file> Specify output file (default: <input>.lunar)")
	fmt.Println("  --migrate-no-types      Don't add type annotations")
	fmt.Println("  --migrate-const         Use const for immutable variables (default: true)")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  lunar main.lunar")
	fmt.Println("  lunar main.lunar -o output.lua")
	fmt.Println("  lunar main.lunar --no-typecheck")
	fmt.Println("  lunar main.lunar --source-map")
	fmt.Println("  lunar main.lunar --target luajit")
	fmt.Println("  lunar main.lunar --watch")
	fmt.Println("  lunar main.lunar --bundle --run")
	fmt.Println("  lunar main.lunar --bundle --watch --run")
	fmt.Println("  lunar --test ./tests")
	fmt.Println("  lunar --test ./tests --filter \"Math\"")
	fmt.Println("  lunar --test ./tests --test-watch")
	fmt.Println("  lunar --docs ./src")
	fmt.Println("  lunar --docs ./src --docs-output ./api-docs")
	fmt.Println("  lunar --repl")
	fmt.Println()
	fmt.Println("For more information about the Lunar language:")
	fmt.Println("  See README.md in the repository")
}

// runREPL starts the interactive Read-Eval-Print-Loop
func runREPL() {
	fmt.Println("Lunar REPL - Interactive Mode")
	fmt.Printf("Version %s\n", version)
	fmt.Println("Type 'exit' or 'quit' to exit, 'help' for commands")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)
	var multiLineBuffer strings.Builder
	nestingLevel := 0
	allStatements := []ast.Statement{}
	checker := types.NewChecker()

	// Command history
	history := []string{}
	historyFile := filepath.Join(os.TempDir(), ".lunar_history")
	loadREPLHistory(&history, historyFile)
	defer saveREPLHistory(history, historyFile)

	for {
		// Print prompt
		if nestingLevel > 0 {
			fmt.Print("... ")
		} else {
			fmt.Print(">>> ")
		}

		// Read input
		if !scanner.Scan() {
			fmt.Println("\nGoodbye!")
			break
		}

		line := scanner.Text()

		// Handle special commands (only at top level)
		if nestingLevel == 0 {
			trimmed := strings.TrimSpace(line)
			switch trimmed {
			case "exit", "quit":
				fmt.Println("Goodbye!")
				return
			case "help":
				printREPLHelp()
				continue
			case "clear":
				allStatements = []ast.Statement{}
				checker = types.NewChecker()
				fmt.Println("Context cleared")
				continue
			case "context":
				fmt.Printf("Accumulated %d statements\n", len(allStatements))
				continue
			case "history":
				if len(history) == 0 {
					fmt.Println("No command history")
				} else {
					fmt.Println("Command history:")
					start := 0
					if len(history) > 20 {
						start = len(history) - 20
					}
					for i := start; i < len(history); i++ {
						fmt.Printf("  %d: %s\n", i+1, history[i])
					}
				}
				continue
			case "symbols":
				printREPLSymbols(allStatements, checker)
				continue
			}
		}

		// Calculate nesting level change
		words := strings.Fields(line)
		for _, word := range words {
			switch word {
			case "function", "if", "while", "for", "do", "class", "interface", "enum", "namespace":
				nestingLevel++
			case "end":
				if nestingLevel > 0 {
					nestingLevel--
				}
			}
		}

		// Accumulate input
		multiLineBuffer.WriteString(line)
		multiLineBuffer.WriteString("\n")

		// Continue if still in a block
		if nestingLevel > 0 {
			continue
		}

		// Build complete input
		input := multiLineBuffer.String()
		multiLineBuffer.Reset()

		// Skip empty input
		if strings.TrimSpace(input) == "" {
			continue
		}

		// Parse input
		l := lexer.New(input)
		p := parser.New(l)
		statements := p.Parse()

		// Check for parser errors
		if len(p.Errors()) > 0 {
			fmt.Println("Parse errors:")
			for _, err := range p.Errors() {
				fmt.Printf("  Line %d, Col %d: %s\n", err.Line, err.Column, err.Message)
			}
			continue
		}

		// Type check with accumulated context
		tempStatements := append(allStatements, statements...)
		typeErrors := checker.Check(tempStatements)
		if len(typeErrors) > 0 {
			fmt.Println("Type errors:")
			for _, err := range typeErrors {
				fmt.Printf("  Line %d: %s\n", err.Line, err.Message)
			}
			continue
		}

		// Generate Lua code for new statements only
		luaCode := codegen.Generate(statements)

		// Print generated Lua
		if strings.TrimSpace(luaCode) != "" {
			fmt.Println("-- Lua output:")
			fmt.Println(luaCode)
		}

		// Accumulate successful statements
		allStatements = append(allStatements, statements...)

		// Add to history
		trimmedInput := strings.TrimSpace(input)
		if trimmedInput != "" {
			history = append(history, trimmedInput)
			// Keep history limited to last 1000 entries
			if len(history) > 1000 {
				history = history[len(history)-1000:]
			}
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
	}
}

// printREPLHelp prints help for the REPL
func printREPLHelp() {
	fmt.Println("REPL Commands:")
	fmt.Println("  help     - Show this help")
	fmt.Println("  history  - Show command history")
	fmt.Println("  symbols  - Show available symbols (variables, functions, classes)")
	fmt.Println("  clear    - Clear accumulated context")
	fmt.Println("  context  - Show number of accumulated statements")
	fmt.Println("  exit     - Exit the REPL (or 'quit')")
	fmt.Println()
	fmt.Println("Multi-line input:")
	fmt.Println("  Blocks (function, if, class, etc.) automatically")
	fmt.Println("  continue until 'end' is reached")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  >>> local x: number = 42")
	fmt.Println("  >>> function add(a: number, b: number): number")
	fmt.Println("  ...   return a + b")
	fmt.Println("  ... end")
}

// printREPLSymbols prints all available symbols in the current REPL context
func printREPLSymbols(statements []ast.Statement, checker *types.Checker) {
	if len(statements) == 0 {
		fmt.Println("No symbols defined yet")
		return
	}

	fmt.Println("Available symbols:")

	variables := []string{}
	functions := []string{}
	classes := []string{}
	interfaces := []string{}
	enums := []string{}
	typeAliases := []string{}

	for _, stmt := range statements {
		switch s := stmt.(type) {
		case *ast.VariableDeclaration:
			for _, name := range s.Names {
				variables = append(variables, name.Value)
			}
		case *ast.FunctionDeclaration:
			if s.Name != nil {
				params := []string{}
				for _, param := range s.Parameters {
					paramStr := param.Name.Value
					if param.Type != nil {
						paramStr += ": " + param.Type.String()
					}
					params = append(params, paramStr)
				}
				returnType := "void"
				if s.ReturnType != nil {
					returnType = s.ReturnType.String()
				}
				functions = append(functions, fmt.Sprintf("%s(%s): %s", s.Name.Value, strings.Join(params, ", "), returnType))
			}
		case *ast.ClassDeclaration:
			if s.Name != nil {
				classes = append(classes, s.Name.Value)
			}
		case *ast.InterfaceDeclaration:
			if s.Name != nil {
				interfaces = append(interfaces, s.Name.Value)
			}
		case *ast.EnumDeclaration:
			if s.Name != nil {
				enums = append(enums, s.Name.Value)
			}
		case *ast.TypeDeclaration:
			if s.Name != nil {
				typeAliases = append(typeAliases, s.Name.Value)
			}
		}
	}

	if len(variables) > 0 {
		fmt.Println("\n  Variables:")
		for _, v := range variables {
			fmt.Printf("    - %s\n", v)
		}
	}

	if len(functions) > 0 {
		fmt.Println("\n  Functions:")
		for _, f := range functions {
			fmt.Printf("    - %s\n", f)
		}
	}

	if len(classes) > 0 {
		fmt.Println("\n  Classes:")
		for _, c := range classes {
			fmt.Printf("    - %s\n", c)
		}
	}

	if len(interfaces) > 0 {
		fmt.Println("\n  Interfaces:")
		for _, i := range interfaces {
			fmt.Printf("    - %s\n", i)
		}
	}

	if len(enums) > 0 {
		fmt.Println("\n  Enums:")
		for _, e := range enums {
			fmt.Printf("    - %s\n", e)
		}
	}

	if len(typeAliases) > 0 {
		fmt.Println("\n  Type Aliases:")
		for _, t := range typeAliases {
			fmt.Printf("    - %s\n", t)
		}
	}

	totalSymbols := len(variables) + len(functions) + len(classes) + len(interfaces) + len(enums) + len(typeAliases)
	fmt.Printf("\nTotal: %d symbols\n", totalSymbols)
}

// loadREPLHistory loads command history from file
func loadREPLHistory(history *[]string, filename string) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return // File doesn't exist yet, that's ok
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			*history = append(*history, line)
		}
	}
}

// saveREPLHistory saves command history to file
func saveREPLHistory(history []string, filename string) {
	content := strings.Join(history, "\n")
	ioutil.WriteFile(filename, []byte(content), 0644)
}

// runTests discovers and runs all test files in a directory
func runTests(path string, filter string, coverage bool) error {
	// Check if path is a directory or file
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("cannot access path: %w", err)
	}

	var testDir string
	if info.IsDir() {
		testDir = path
	} else {
		testDir = filepath.Dir(path)
	}

	// Discover test files
	testFiles, err := discoverTestFiles(testDir)
	if err != nil {
		return fmt.Errorf("failed to discover tests: %w", err)
	}

	if len(testFiles) == 0 {
		fmt.Println("No test files found")
		fmt.Println("Test files should match: *_test.lunar or *.test.lunar")
		return nil
	}

	// Apply filter if specified
	if filter != "" {
		filteredFiles := []string{}
		for _, f := range testFiles {
			relPath, _ := filepath.Rel(testDir, f)
			if strings.Contains(strings.ToLower(relPath), strings.ToLower(filter)) {
				filteredFiles = append(filteredFiles, f)
			}
		}
		testFiles = filteredFiles

		if len(testFiles) == 0 {
			fmt.Printf("No test files match filter: %s\n", filter)
			return nil
		}
	}

	fmt.Printf("Found %d test file(s)", len(testFiles))
	if filter != "" {
		fmt.Printf(" (filtered by: %s)", filter)
	}
	fmt.Println(":")
	for _, f := range testFiles {
		relPath, _ := filepath.Rel(testDir, f)
		fmt.Printf("  - %s\n", relPath)
	}
	fmt.Println()

	// Create a temporary test runner file
	runnerContent := generateTestRunner(testFiles, testDir, coverage)
	runnerFile := filepath.Join(testDir, "__test_runner__.lunar")

	if err := ioutil.WriteFile(runnerFile, []byte(runnerContent), 0644); err != nil {
		return fmt.Errorf("failed to create test runner: %w", err)
	}
	defer os.Remove(runnerFile)

	// Bundle the test runner
	b := bundler.New(runnerFile, false)
	bundledCode, err := b.Bundle()
	if err != nil {
		return fmt.Errorf("failed to bundle tests: %w", err)
	}

	// Write bundled output
	outputFile := filepath.Join(testDir, "__test_runner__.lua")
	if err := ioutil.WriteFile(outputFile, []byte(bundledCode), 0644); err != nil {
		return fmt.Errorf("failed to write bundled tests: %w", err)
	}
	defer os.Remove(outputFile)

	// Run the tests
	// Try lua5.4, lua5.3, lua5.1, luajit, then lua
	luaCommands := []string{"lua5.4", "lua5.3", "lua5.1", "luajit", "lua"}
	var cmd *exec.Cmd
	for _, luaCmd := range luaCommands {
		if _, err := exec.LookPath(luaCmd); err == nil {
			// Pass just the filename since we're running from testDir
			cmd = exec.Command(luaCmd, "__test_runner__.lua")
			break
		}
	}

	if cmd == nil {
		return fmt.Errorf("no Lua interpreter found. Please install lua")
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = testDir

	err = cmd.Run()

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("tests failed with exit code %d", exitErr.ExitCode())
		}
		return fmt.Errorf("failed to run tests: %w", err)
	}

	return nil
}

// watchTests watches test files and re-runs tests on changes
func watchTests(path string, filter string, coverage bool) error {
	fmt.Println("🔍 Watch mode: Monitoring for changes...")
	fmt.Println("Press Ctrl+C to exit")

	// Get initial file timestamps
	fileTimestamps := make(map[string]time.Time)

	// Determine directory to watch
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("cannot access path: %w", err)
	}

	var watchDir string
	if info.IsDir() {
		watchDir = path
	} else {
		watchDir = filepath.Dir(path)
	}

	// Function to get all .lunar files
	getLunarFiles := func() (map[string]time.Time, error) {
		timestamps := make(map[string]time.Time)
		err := filepath.Walk(watchDir, func(filePath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && strings.HasSuffix(filePath, ".lunar") {
				timestamps[filePath] = info.ModTime()
			}
			return nil
		})
		return timestamps, err
	}

	// Get initial timestamps
	fileTimestamps, err = getLunarFiles()
	if err != nil {
		return fmt.Errorf("failed to scan directory: %w", err)
	}

	// Run tests initially
	fmt.Println("Running initial test suite...")
	if err := runTests(path, filter, coverage); err != nil {
		fmt.Fprintf(os.Stderr, "\n❌ Tests failed: %v\n", err)
	}
	fmt.Println("\n👀 Watching for changes...")

	// Watch loop
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// Check for file changes
		currentTimestamps, err := getLunarFiles()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error scanning directory: %v\n", err)
			continue
		}

		// Check for changes
		changed := false
		changedFiles := []string{}

		// Check for modified or new files
		for file, modTime := range currentTimestamps {
			if oldTime, exists := fileTimestamps[file]; !exists || modTime.After(oldTime) {
				changed = true
				changedFiles = append(changedFiles, filepath.Base(file))
			}
		}

		// Check for deleted files
		for file := range fileTimestamps {
			if _, exists := currentTimestamps[file]; !exists {
				changed = true
				changedFiles = append(changedFiles, filepath.Base(file)+" (deleted)")
			}
		}

		if changed {
			// Clear screen
			fmt.Print("\033[2J\033[H")

			fmt.Println("🔄 Changes detected:")
			for _, file := range changedFiles {
				fmt.Printf("   - %s\n", file)
			}
			fmt.Println("\nRe-running tests...")

			// Update timestamps
			fileTimestamps = currentTimestamps

			// Re-run tests
			if err := runTests(path, filter, coverage); err != nil {
				fmt.Fprintf(os.Stderr, "\n❌ Tests failed: %v\n", err)
			}

			fmt.Println("\n👀 Watching for changes...")
		}
	}

	return nil
}

// discoverTestFiles finds all test files in a directory
func discoverTestFiles(dir string) ([]string, error) {
	var testFiles []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			// Skip vendor and hidden directories
			if info.Name() == "vendor" || strings.HasPrefix(info.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}

		// Check for test file patterns
		name := info.Name()
		if strings.HasSuffix(name, "_test.lunar") || strings.HasSuffix(name, ".test.lunar") {
			testFiles = append(testFiles, path)
		}

		return nil
	})

	return testFiles, err
}

// generateTestRunner creates a Lunar file that imports and runs all test files
func generateTestRunner(testFiles []string, baseDir string, withCoverage bool) string {
	var sb strings.Builder

	sb.WriteString("-- Auto-generated test runner\n")
	sb.WriteString("import { runTests, printResults } from \"vendor/testing\"\n")

	if withCoverage {
		sb.WriteString("import * as coverage from \"vendor/testing/coverage\"\n\n")
		sb.WriteString("-- Start coverage tracking\n")
		sb.WriteString("coverage.start()\n\n")
	} else {
		sb.WriteString("\n")
	}

	// Import each test file (wildcard import to execute the file)
	for _, testFile := range testFiles {
		relPath, _ := filepath.Rel(baseDir, testFile)
		// Convert to import path (remove .lunar, use ./)
		importPath := "./" + strings.TrimSuffix(relPath, ".lunar")
		importPath = strings.ReplaceAll(importPath, "\\", "/")

		sb.WriteString(fmt.Sprintf("import * from \"%s\"\n", importPath))
	}

	sb.WriteString("\n-- Run all tests and print results\n")
	sb.WriteString("local results: any = runTests()\n")
	sb.WriteString("printResults(results)\n")

	if withCoverage {
		sb.WriteString("\n-- Print coverage report\n")
		sb.WriteString("coverage.stop()\n")
		sb.WriteString("coverage.report()\n")
	}

	sb.WriteString("\n")

	// Exit with appropriate code
	sb.WriteString("-- Exit with appropriate code\n")
	sb.WriteString("if results.failed > 0 then\n")
	sb.WriteString("    os.exit(1)\n")
	sb.WriteString("end\n")

	return sb.String()
}

// generateDocs generates documentation for Lunar source files
func generateDocs(path string, outputDir string) error {
	// Set default output directory
	if outputDir == "" {
		outputDir = "./docs"
	}

	// Check if path is a file or directory
	fileInfo, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to access path: %v", err)
	}

	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	var files []string
	if fileInfo.IsDir() {
		// Walk directory and find all .lunar files
		err = filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && strings.HasSuffix(filePath, ".lunar") {
				files = append(files, filePath)
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to walk directory: %v", err)
		}
	} else {
		// Single file
		files = append(files, path)
	}

	if len(files) == 0 {
		return fmt.Errorf("no .lunar files found in %s", path)
	}

	fmt.Printf("Generating documentation for %d file(s)...\n", len(files))

	// Generate documentation for each file
	generator := docgen.New()
	for _, file := range files {
		// Read source
		source, err := ioutil.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %v", file, err)
		}

		// Generate documentation
		docs, err := generator.Generate(filepath.Base(file), string(source))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to generate docs for %s: %v\n", file, err)
			continue
		}

		// Determine output file path
		relPath, err := filepath.Rel(path, file)
		if err != nil {
			relPath = filepath.Base(file)
		}
		outputFile := filepath.Join(outputDir, strings.TrimSuffix(relPath, ".lunar")+".md")

		// Create subdirectories if needed
		if err := os.MkdirAll(filepath.Dir(outputFile), 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %v", err)
		}

		// Write documentation
		if err := ioutil.WriteFile(outputFile, []byte(docs), 0644); err != nil {
			return fmt.Errorf("failed to write documentation to %s: %v", outputFile, err)
		}

		fmt.Printf("  ✓ %s → %s\n", file, outputFile)
	}

	fmt.Printf("\n✓ Documentation generated in %s\n", outputDir)
	return nil
}

// handleLuaRocksCommands handles all LuaRocks-related commands
func handleLuaRocksCommands(install, remove *string, list *bool, search, genTypes *string, init, installDeps *bool) {
	mgr, err := luarocks.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Initialize lunarocks.json
	if *init {
		config := &luarocks.Config{
			Dependencies:    make(map[string]string),
			DevDependencies: make(map[string]string),
		}
		if err := mgr.SaveConfig("lunarocks.json", config); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create config: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✓ Created lunarocks.json")
		return
	}

	// Install dependencies
	if *installDeps {
		if err := mgr.InstallDependencies("lunarocks.json"); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to install dependencies: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Install package
	if *install != "" {
		parts := strings.SplitN(*install, "@", 2)
		packageName := parts[0]
		version := ""
		if len(parts) > 1 {
			version = parts[1]
		}

		if err := mgr.Install(packageName, version); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to install package: %v\n", err)
			os.Exit(1)
		}

		// Auto-generate type definitions
		if err := mgr.GenerateTypeDefinitions(packageName); err != nil {
			fmt.Printf("Warning: Failed to generate type definitions: %v\n", err)
		}
		return
	}

	// Remove package
	if *remove != "" {
		if err := mgr.Remove(*remove); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to remove package: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// List installed packages
	if *list {
		packages, err := mgr.List()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to list packages: %v\n", err)
			os.Exit(1)
		}

		if len(packages) == 0 {
			fmt.Println("No packages installed")
			return
		}

		fmt.Println("Installed packages:")
		for _, pkg := range packages {
			fmt.Printf("  %s@%s\n", pkg.Name, pkg.Version)
		}
		return
	}

	// Search packages
	if *search != "" {
		packages, err := mgr.Search(*search)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to search packages: %v\n", err)
			os.Exit(1)
		}

		if len(packages) == 0 {
			fmt.Printf("No packages found matching '%s'\n", *search)
			return
		}

		fmt.Printf("Search results for '%s':\n", *search)
		for _, pkg := range packages {
			fmt.Printf("  %s@%s\n", pkg.Name, pkg.Version)
		}
		return
	}

	// Generate type definitions
	if *genTypes != "" {
		if err := mgr.GenerateTypeDefinitions(*genTypes); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to generate type definitions: %v\n", err)
			os.Exit(1)
		}
		return
	}
}

// handleMigration handles the Lua to Lunar migration process
func handleMigration(inputFile, outputFile string, noTypes, useConst bool) {
	// Read input file
	source, err := ioutil.ReadFile(inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}

	// Configure migration options
	options := migrate.Options{
		AddTypes:           !noTypes,
		UseConst:           useConst,
		GenerateInterfaces: true,
		PreserveComments:   true,
		Modernize:          true,
	}

	// Create migrator
	migrator := migrate.New(options)

	// Migrate code
	fmt.Printf("Migrating %s to Lunar...\n", inputFile)
	lunarCode, err := migrator.Migrate(string(source))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Migration failed: %v\n", err)
		os.Exit(1)
	}

	// Determine output file
	output := outputFile
	if output == "" {
		// Replace .lua extension with .lunar or add .lunar
		if strings.HasSuffix(inputFile, ".lua") {
			output = strings.TrimSuffix(inputFile, ".lua") + ".lunar"
		} else {
			output = inputFile + ".lunar"
		}
	}

	// Write output
	if err := ioutil.WriteFile(output, []byte(lunarCode), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing output: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Successfully migrated to %s\n", output)

	// Print summary
	fmt.Println("\nMigration Summary:")
	fmt.Printf("  Input:  %s\n", inputFile)
	fmt.Printf("  Output: %s\n", output)
	fmt.Printf("  Type annotations: %v\n", !noTypes)
	fmt.Printf("  Use const: %v\n", useConst)
	fmt.Println("\nPlease review the migrated code and adjust type annotations as needed.")
}
