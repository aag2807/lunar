package integration

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"lunar/internal/testutil"
	"lunar/internal/version"
)

// compilerBin is the compiler built once for the whole package by TestMain, so
// the tests do not depend on a binary having been left in the project root.
var compilerBin string

func TestMain(m *testing.M) {
	code, err := runIntegrationTests(m)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(code)
}

func runIntegrationTests(m *testing.M) (int, error) {
	tmpDir, err := ioutil.TempDir("", "lunar-compiler")
	if err != nil {
		return 0, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	wd, err := os.Getwd()
	if err != nil {
		return 0, fmt.Errorf("failed to get working directory: %w", err)
	}
	projectRoot := filepath.Join(wd, "../..")

	compilerBin = filepath.Join(tmpDir, "lunar")
	if runtime.GOOS == "windows" {
		compilerBin += ".exe"
	}
	build := exec.Command("go", "build", "-o", compilerBin, "./cmd/lunar")
	build.Dir = projectRoot
	if output, err := build.CombinedOutput(); err != nil {
		return 0, fmt.Errorf("failed to build compiler: %w\n%s", err, output)
	}

	return m.Run(), nil
}

// TestCompileBasicTypes tests compilation of basic type annotations
func TestCompileBasicTypes(t *testing.T) {
	testCompile(t, "basic_types.lunar", false)
}

// TestCompileClasses tests compilation of class definitions
// TODO: Re-enable when class instantiation type checking is fixed
/*
func TestCompileClasses(t *testing.T) {
	testCompile(t, "classes.lunar", false)
}
*/

// TestCompileWithSourceMap tests source map generation
func TestCompileWithSourceMap(t *testing.T) {
	testCompile(t, "sourcemap_test.lunar", true)
}

// testCompile is a helper that compiles a Lunar file and checks for success
func testCompile(t *testing.T, filename string, withSourceMap bool) {
	// Get absolute paths
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	// Navigate up to project root
	projectRoot := filepath.Join(wd, "../..")
	inputFile := filepath.Join(projectRoot, "test/integration/testdata", filename)
	outputFile := strings.TrimSuffix(inputFile, ".lunar") + ".lua"

	// Clean up output files before test
	defer func() {
		os.Remove(outputFile)
		os.Remove(outputFile + ".map")
	}()

	// Build compile command
	args := []string{inputFile}
	if withSourceMap {
		args = []string{"-source-map", inputFile}
	}

	// Run compiler
	cmd := exec.Command(compilerBin, args...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Fatalf("Compilation failed for %s:\n%s", filename, string(output))
	}

	// Check that output file was created
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Fatalf("Output file %s was not created", outputFile)
	}

	// If source map was requested, check it was created
	if withSourceMap {
		mapFile := outputFile + ".map"
		if _, err := os.Stat(mapFile); os.IsNotExist(err) {
			t.Fatalf("Source map file %s was not created", mapFile)
		}

		// Verify source map is valid JSON
		mapContent, err := ioutil.ReadFile(mapFile)
		if err != nil {
			t.Fatalf("Failed to read source map: %v", err)
		}

		// Check for required source map fields
		mapStr := string(mapContent)
		requiredFields := []string{"version", "sources", "mappings"}
		for _, field := range requiredFields {
			if !strings.Contains(mapStr, field) {
				t.Errorf("Source map missing required field: %s", field)
			}
		}

		// Check that Lua file contains source map comment
		luaContent, err := ioutil.ReadFile(outputFile)
		if err != nil {
			t.Fatalf("Failed to read Lua output: %v", err)
		}

		if !strings.Contains(string(luaContent), "sourceMappingURL") {
			t.Error("Lua output missing source map comment")
		}
	}

	t.Logf("Successfully compiled %s", filename)
}

// TestErrorMessages tests that type errors produce helpful messages
func TestErrorMessages(t *testing.T) {
	// Create a temporary file with a type error
	tmpDir, err := ioutil.TempDir("", "lunar-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	errorFile := filepath.Join(tmpDir, "error.lunar")
	errorCode := `
local myVariable: number = 42
local result: number = myVariabl + 1
`
	if err := ioutil.WriteFile(errorFile, []byte(errorCode), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Run compiler (should fail)
	cmd := exec.Command(compilerBin, errorFile)
	output, err := cmd.CombinedOutput()

	if err == nil {
		t.Fatal("Expected compilation to fail, but it succeeded")
	}

	outputStr := string(output)

	// Check for "Did you mean?" suggestion
	if !strings.Contains(outputStr, "Did you mean") {
		t.Error("Error message missing 'Did you mean?' suggestion")
	}

	if !strings.Contains(outputStr, "myVariable") {
		t.Error("Error message missing correct suggestion 'myVariable'")
	}

	t.Log("Error messages include helpful suggestions")
}

// TestVersionFlagPrintsSingleSourcedVersion checks `lunar --version` prints the
// version from internal/version and nothing else on stdout.
func TestVersionFlagPrintsSingleSourcedVersion(t *testing.T) {
	cmd := exec.Command(compilerBin, "--version")
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("lunar --version failed: %v\nstderr: %s", err, stderr.String())
	}
	want := regexp.MustCompile(`^lunar ` + regexp.QuoteMeta(version.Version) + `( \([0-9a-f]{7,12}(-dirty)?\))?\n$`)
	if !want.MatchString(stdout.String()) {
		t.Fatalf("stdout = %q, want it to match %s", stdout.String(), want)
	}
}

// TestInstalledLayoutFindsStdlib mirrors `make install`: the binary in
// <prefix>/bin and the declarations in <prefix>/share/lunar/stdlib. Compiling
// from an unrelated directory must still know about print.
func TestInstalledLayoutFindsStdlib(t *testing.T) {
	prefix := t.TempDir()
	bin := filepath.Join(prefix, "bin", filepath.Base(compilerBin))
	stdlibDir := filepath.Join(prefix, "share", "lunar", "stdlib")
	for _, dir := range []string{filepath.Dir(bin), stdlibDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	copyFile(t, compilerBin, bin, 0o755)
	decls, err := filepath.Glob(filepath.Join("..", "..", "stdlib", "*.d.lunar"))
	if err != nil || len(decls) == 0 {
		t.Fatalf("no stdlib declarations found: %v", err)
	}
	for _, d := range decls {
		copyFile(t, d, filepath.Join(stdlibDir, filepath.Base(d)), 0o644)
	}

	work := t.TempDir()
	if err := os.WriteFile(filepath.Join(work, "hello.lunar"), []byte("print(\"hello\")\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(bin, "hello.lunar")
	cmd.Dir = work
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("installed lunar failed outside the repo: %v\n%s", err, out)
	}
}

func copyFile(t *testing.T, from, to string, mode os.FileMode) {
	t.Helper()
	data, err := os.ReadFile(from)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(to, data, mode); err != nil {
		t.Fatal(err)
	}
}

// TestNoTypecheckFlagIsGone: the checker decides ':' versus '.' and which
// calls construct, so compiling without it would silently change the Lua.
func TestNoTypecheckFlagIsGone(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "a.lunar")
	if err := os.WriteFile(src, []byte("local x = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := exec.Command(compilerBin, "--no-typecheck", src).CombinedOutput()
	if err == nil {
		t.Fatalf("--no-typecheck was accepted:\n%s", out)
	}
	if !strings.Contains(string(out), "no-typecheck") {
		t.Errorf("error should name the flag, got:\n%s", out)
	}
}

// TestCompilerSourcesAreTracked reads the compiler's sources so Go's test
// cache re-runs this package when they change; see TouchCompilerSources.
func TestCompilerSourcesAreTracked(t *testing.T) {
	if err := testutil.TouchCompilerSources(filepath.Join("..", "..")); err != nil {
		t.Fatal(err)
	}
}
