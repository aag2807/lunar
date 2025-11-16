package integration

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

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
	compilerPath := filepath.Join(projectRoot, "lunar")

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
	cmd := exec.Command(compilerPath, args...)
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
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	projectRoot := filepath.Join(wd, "../..")
	compilerPath := filepath.Join(projectRoot, "lunar")

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
	cmd := exec.Command(compilerPath, errorFile)
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
