package sourcemap

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestNewBuilder(t *testing.T) {
	builder := NewBuilder("test.lunar", "test.lua")

	if builder.sourceFile != "test.lunar" {
		t.Errorf("Expected sourceFile 'test.lunar', got '%s'", builder.sourceFile)
	}

	if builder.generatedFile != "test.lua" {
		t.Errorf("Expected generatedFile 'test.lua', got '%s'", builder.generatedFile)
	}

	if len(builder.mappings) != 0 {
		t.Errorf("Expected empty mappings, got %d", len(builder.mappings))
	}
}

func TestAddMapping(t *testing.T) {
	builder := NewBuilder("test.lunar", "test.lua")

	builder.AddMapping(1, 0, 1, 0, "")
	builder.AddMapping(1, 10, 1, 8, "myVar")

	if len(builder.mappings) != 2 {
		t.Errorf("Expected 2 mappings, got %d", len(builder.mappings))
	}

	// Check first mapping
	m1 := builder.mappings[0]
	if m1.GeneratedLine != 1 || m1.GeneratedColumn != 0 {
		t.Errorf("Mapping 1: expected gen (1,0), got (%d,%d)", m1.GeneratedLine, m1.GeneratedColumn)
	}
	if m1.SourceLine != 1 || m1.SourceColumn != 0 {
		t.Errorf("Mapping 1: expected src (1,0), got (%d,%d)", m1.SourceLine, m1.SourceColumn)
	}

	// Check second mapping with name
	m2 := builder.mappings[1]
	if m2.Name != "myVar" {
		t.Errorf("Mapping 2: expected name 'myVar', got '%s'", m2.Name)
	}

	// Check that name was tracked
	if len(builder.namesList) != 1 {
		t.Errorf("Expected 1 name in namesList, got %d", len(builder.namesList))
	}
	if builder.namesList[0] != "myVar" {
		t.Errorf("Expected namesList[0] = 'myVar', got '%s'", builder.namesList[0])
	}
}

func TestBuild(t *testing.T) {
	builder := NewBuilder("test.lunar", "test.lua")
	builder.AddMapping(1, 0, 1, 0, "")

	sourceMap := builder.Build()

	if sourceMap.Version != 3 {
		t.Errorf("Expected version 3, got %d", sourceMap.Version)
	}

	if sourceMap.File != "test.lua" {
		t.Errorf("Expected file 'test.lua', got '%s'", sourceMap.File)
	}

	if len(sourceMap.Sources) != 1 || sourceMap.Sources[0] != "test.lunar" {
		t.Errorf("Expected sources ['test.lunar'], got %v", sourceMap.Sources)
	}
}

func TestEncodeMappings(t *testing.T) {
	builder := NewBuilder("test.lunar", "test.lua")

	// Add a simple mapping: line 1, col 0 -> line 1, col 0
	builder.AddMapping(1, 0, 1, 0, "")

	encoded := builder.encodeMappings()

	// Should produce valid VLQ encoding
	if encoded == "" {
		t.Error("Expected non-empty encoding")
	}

	// The encoding should contain valid base64 characters
	validChars := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/;,"
	for _, char := range encoded {
		if !strings.ContainsRune(validChars, char) {
			t.Errorf("Invalid character in encoding: %c", char)
		}
	}
}

func TestEncodeMappingsMultipleLines(t *testing.T) {
	builder := NewBuilder("test.lunar", "test.lua")

	// Add mappings on different lines
	builder.AddMapping(1, 0, 1, 0, "")
	builder.AddMapping(2, 0, 2, 0, "")
	builder.AddMapping(3, 5, 3, 3, "")

	encoded := builder.encodeMappings()

	// Should have semicolons separating lines
	semicolonCount := strings.Count(encoded, ";")
	if semicolonCount < 2 {
		t.Errorf("Expected at least 2 semicolons for 3 lines, got %d", semicolonCount)
	}
}

func TestEncodeMappingsWithNames(t *testing.T) {
	builder := NewBuilder("test.lunar", "test.lua")

	builder.AddMapping(1, 0, 1, 0, "foo")
	builder.AddMapping(1, 10, 1, 8, "bar")

	sourceMap := builder.Build()

	// Check that names were included
	if len(sourceMap.Names) != 2 {
		t.Errorf("Expected 2 names, got %d", len(sourceMap.Names))
	}

	if sourceMap.Names[0] != "foo" {
		t.Errorf("Expected name[0] = 'foo', got '%s'", sourceMap.Names[0])
	}

	if sourceMap.Names[1] != "bar" {
		t.Errorf("Expected name[1] = 'bar', got '%s'", sourceMap.Names[1])
	}
}

func TestToJSON(t *testing.T) {
	builder := NewBuilder("test.lunar", "test.lua")
	builder.AddMapping(1, 0, 1, 0, "")

	sourceMap := builder.Build()
	jsonStr, err := sourceMap.ToJSON()

	if err != nil {
		t.Errorf("ToJSON failed: %v", err)
	}

	// Verify it's valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Errorf("Generated invalid JSON: %v", err)
	}

	// Check required fields
	if version, ok := parsed["version"].(float64); !ok || int(version) != 3 {
		t.Error("JSON missing or invalid 'version' field")
	}

	if file, ok := parsed["file"].(string); !ok || file != "test.lua" {
		t.Error("JSON missing or invalid 'file' field")
	}

	if _, ok := parsed["sources"].([]interface{}); !ok {
		t.Error("JSON missing or invalid 'sources' field")
	}

	if _, ok := parsed["mappings"].(string); !ok {
		t.Error("JSON missing or invalid 'mappings' field")
	}
}

func TestToBase64(t *testing.T) {
	builder := NewBuilder("test.lunar", "test.lua")
	builder.AddMapping(1, 0, 1, 0, "")

	sourceMap := builder.Build()
	base64URL, err := sourceMap.ToBase64()

	if err != nil {
		t.Errorf("ToBase64 failed: %v", err)
	}

	// Should start with data URL prefix
	expectedPrefix := "data:application/json;base64,"
	if !strings.HasPrefix(base64URL, expectedPrefix) {
		t.Errorf("Expected base64 URL to start with '%s'", expectedPrefix)
	}

	// Should have base64-encoded content after prefix
	if len(base64URL) <= len(expectedPrefix) {
		t.Error("Base64 URL has no content after prefix")
	}
}

func TestGenerateComment(t *testing.T) {
	builder := NewBuilder("test.lunar", "test.lua")
	builder.AddMapping(1, 0, 1, 0, "")
	sourceMap := builder.Build()

	// Test with external map file
	comment := sourceMap.GenerateComment("test.lua.map")
	expected := "--# sourceMappingURL=test.lua.map"

	if comment != expected {
		t.Errorf("Expected '%s', got '%s'", expected, comment)
	}

	// Test with inline source map
	inlineComment := sourceMap.GenerateComment("")

	if !strings.HasPrefix(inlineComment, "--# sourceMappingURL=data:application/json;base64,") {
		t.Error("Inline comment should have base64 data URL")
	}
}

func TestEmptyMappings(t *testing.T) {
	builder := NewBuilder("test.lunar", "test.lua")
	sourceMap := builder.Build()

	if sourceMap.Mappings != "" {
		t.Errorf("Expected empty mappings string, got '%s'", sourceMap.Mappings)
	}
}

func TestMappingOrder(t *testing.T) {
	builder := NewBuilder("test.lunar", "test.lua")

	// Add mappings in non-sequential order
	builder.AddMapping(2, 0, 2, 0, "")
	builder.AddMapping(1, 0, 1, 0, "")
	builder.AddMapping(3, 0, 3, 0, "")

	// Build should handle mappings regardless of order
	sourceMap := builder.Build()

	if sourceMap.Mappings == "" {
		t.Error("Expected non-empty mappings")
	}
}
