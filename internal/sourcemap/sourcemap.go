package sourcemap

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

// SourceMap represents a source map following the Source Map v3 specification
// https://sourcemaps.info/spec.html
type SourceMap struct {
	Version    int      `json:"version"`
	File       string   `json:"file"`
	SourceRoot string   `json:"sourceRoot,omitempty"`
	Sources    []string `json:"sources"`
	Names      []string `json:"names,omitempty"`
	Mappings   string   `json:"mappings"`
}

// Builder helps construct source maps incrementally
type Builder struct {
	sourceFile    string
	generatedFile string
	mappings      []Mapping
	names         map[string]int
	namesList     []string
}

// Mapping represents a single position mapping
type Mapping struct {
	GeneratedLine   int
	GeneratedColumn int
	SourceLine      int
	SourceColumn    int
	Name            string
}

// NewBuilder creates a new source map builder
func NewBuilder(sourceFile, generatedFile string) *Builder {
	return &Builder{
		sourceFile:    sourceFile,
		generatedFile: generatedFile,
		mappings:      []Mapping{},
		names:         make(map[string]int),
		namesList:     []string{},
	}
}

// AddMapping adds a position mapping
func (b *Builder) AddMapping(genLine, genCol, srcLine, srcCol int, name string) {
	mapping := Mapping{
		GeneratedLine:   genLine,
		GeneratedColumn: genCol,
		SourceLine:      srcLine,
		SourceColumn:    srcCol,
		Name:            name,
	}
	b.mappings = append(b.mappings, mapping)

	// Track name if provided
	if name != "" {
		if _, exists := b.names[name]; !exists {
			b.names[name] = len(b.namesList)
			b.namesList = append(b.namesList, name)
		}
	}
}

// Build generates the final source map
func (b *Builder) Build() *SourceMap {
	return &SourceMap{
		Version:  3,
		File:     b.generatedFile,
		Sources:  []string{b.sourceFile},
		Names:    b.namesList,
		Mappings: b.encodeMappings(),
	}
}

// encodeMappings encodes mappings into VLQ (Variable Length Quantity) format
// For simplicity in v1, we'll use a simplified encoding
// Full VLQ encoding can be added later for production use
func (b *Builder) encodeMappings() string {
	if len(b.mappings) == 0 {
		return ""
	}

	var segments []string
	currentLine := 0

	var lineSegments []string
	for _, m := range b.mappings {
		// Start new line if needed
		for currentLine < m.GeneratedLine {
			if len(lineSegments) > 0 {
				segments = append(segments, strings.Join(lineSegments, ","))
				lineSegments = []string{}
			}
			currentLine++
			if currentLine < m.GeneratedLine {
				segments = append(segments, "")
			}
		}

		// For simplicity, encode as JSON for now
		// Production version should use VLQ base64 encoding
		segment := fmt.Sprintf("%d:%d:%d:%d",
			m.GeneratedColumn,
			0, // Source file index (always 0 for single source)
			m.SourceLine-1,
			m.SourceColumn,
		)
		lineSegments = append(lineSegments, segment)
	}

	if len(lineSegments) > 0 {
		segments = append(segments, strings.Join(lineSegments, ","))
	}

	return strings.Join(segments, ";")
}

// ToJSON converts the source map to JSON string
func (sm *SourceMap) ToJSON() (string, error) {
	data, err := json.MarshalIndent(sm, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ToBase64 converts the source map to base64-encoded data URL
func (sm *SourceMap) ToBase64() (string, error) {
	jsonData, err := json.Marshal(sm)
	if err != nil {
		return "", err
	}
	encoded := base64.StdEncoding.EncodeToString(jsonData)
	return fmt.Sprintf("data:application/json;base64,%s", encoded), nil
}

// GenerateComment generates a source map comment for embedding in Lua
func (sm *SourceMap) GenerateComment(mapFile string) string {
	if mapFile != "" {
		return fmt.Sprintf("--# sourceMappingURL=%s", mapFile)
	}

	// Inline source map
	encoded, err := sm.ToBase64()
	if err != nil {
		return ""
	}
	return fmt.Sprintf("--# sourceMappingURL=%s", encoded)
}
