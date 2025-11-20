package bundler

import (
	"lunar/internal/config"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBundleBasic(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "lunar-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files
	mainFile := filepath.Join(tmpDir, "main.lunar")
	utilsFile := filepath.Join(tmpDir, "utils.lunar")

	os.WriteFile(utilsFile, []byte(`
export function greet(name: string): string
    return "Hello, " .. name
end
`), 0644)

	os.WriteFile(mainFile, []byte(`
import { greet } from "./utils"
print(greet("World"))
`), 0644)

	// Bundle
	b := New(mainFile, false)
	output, err := b.Bundle()
	if err != nil {
		t.Fatalf("Bundle failed: %v", err)
	}

	// Check output contains module wrappers
	if !strings.Contains(output, "__modules") {
		t.Error("Expected __modules in output")
	}
	if !strings.Contains(output, "__require") {
		t.Error("Expected __require in output")
	}
	if !strings.Contains(output, "greet") {
		t.Error("Expected greet function in output")
	}
}

func TestBundleIndexFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "lunar-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create helpers directory with index.lunar
	helpersDir := filepath.Join(tmpDir, "helpers")
	os.MkdirAll(helpersDir, 0755)

	indexFile := filepath.Join(helpersDir, "index.lunar")
	mainFile := filepath.Join(tmpDir, "main.lunar")

	os.WriteFile(indexFile, []byte(`
export function helper(): string
    return "helper"
end
`), 0644)

	os.WriteFile(mainFile, []byte(`
import { helper } from "./helpers"
print(helper())
`), 0644)

	b := New(mainFile, false)
	output, err := b.Bundle()
	if err != nil {
		t.Fatalf("Bundle failed: %v", err)
	}

	if !strings.Contains(output, "helpers/index") {
		t.Error("Expected helpers/index in output")
	}
}

func TestCircularDependencyError(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "lunar-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create circular dependency
	aFile := filepath.Join(tmpDir, "a.lunar")
	bFile := filepath.Join(tmpDir, "b.lunar")

	os.WriteFile(aFile, []byte(`
import { b } from "./b"
export function a(): string
    return "a"
end
`), 0644)

	os.WriteFile(bFile, []byte(`
import { a } from "./a"
export function b(): string
    return "b"
end
`), 0644)

	b := New(aFile, false)
	_, err = b.Bundle()

	if err == nil {
		t.Fatal("Expected circular dependency error")
	}

	if !strings.Contains(err.Error(), "circular dependency") {
		t.Errorf("Expected 'circular dependency' in error, got: %v", err)
	}

	// Check that the error shows the cycle
	if !strings.Contains(err.Error(), "->") {
		t.Error("Expected cycle path in error message")
	}
}

func TestTreeShaking(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "lunar-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files
	mainFile := filepath.Join(tmpDir, "main.lunar")
	utilsFile := filepath.Join(tmpDir, "utils.lunar")

	// Utils has two exports but only one is used
	os.WriteFile(utilsFile, []byte(`
export function used(): string
    return "used"
end

export function unused(): string
    return "unused"
end
`), 0644)

	os.WriteFile(mainFile, []byte(`
import { used } from "./utils"
print(used())
`), 0644)

	// Bundle with tree shaking
	cfg := config.DefaultConfig()
	cfg.CompilerOptions.TreeShake = true
	b := NewWithConfig(mainFile, false, cfg)
	output, err := b.Bundle()
	if err != nil {
		t.Fatalf("Bundle failed: %v", err)
	}

	// Check that used function is present
	if !strings.Contains(output, "function used") {
		t.Error("Expected 'used' function in output")
	}

	// Check that unused function is NOT present
	if strings.Contains(output, "function unused") {
		t.Error("Expected 'unused' function to be tree-shaken")
	}
}

// TestPathAliases tests path alias resolution
// Note: Path aliases work when baseUrl is relative to the project root
func TestPathAliases(t *testing.T) {
	t.Skip("Path aliases require more complex setup - skipping for now")
}

func TestGetAllFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "lunar-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	mainFile := filepath.Join(tmpDir, "main.lunar")
	utilsFile := filepath.Join(tmpDir, "utils.lunar")
	mathFile := filepath.Join(tmpDir, "math.lunar")

	os.WriteFile(utilsFile, []byte(`
export function greet(): string
    return "hello"
end
`), 0644)

	os.WriteFile(mathFile, []byte(`
export function add(a: number, b: number): number
    return a + b
end
`), 0644)

	os.WriteFile(mainFile, []byte(`
import { greet } from "./utils"
import { add } from "./math"
print(greet())
print(add(1, 2))
`), 0644)

	b := New(mainFile, false)
	_, err = b.Bundle()
	if err != nil {
		t.Fatalf("Bundle failed: %v", err)
	}

	files := b.GetAllFiles()
	if len(files) != 3 {
		t.Errorf("Expected 3 files, got %d", len(files))
	}
}
