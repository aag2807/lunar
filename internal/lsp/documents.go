package lsp

import (
	"strings"
	"sync"
)

// DocumentManager manages open text documents
type DocumentManager struct {
	documents map[string]*Document
	mu        sync.RWMutex
}

// Document represents an open text document
type Document struct {
	URI     string
	Version int
	Content string
	Lines   []string
}

// NewDocumentManager creates a new document manager
func NewDocumentManager() *DocumentManager {
	return &DocumentManager{
		documents: make(map[string]*Document),
	}
}

// Open opens a document
func (dm *DocumentManager) Open(uri string, version int, content string) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	dm.documents[uri] = &Document{
		URI:     uri,
		Version: version,
		Content: content,
		Lines:   splitLines(content),
	}
}

// Update updates a document with full content
func (dm *DocumentManager) Update(uri string, version int, content string) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	if doc, ok := dm.documents[uri]; ok {
		doc.Version = version
		doc.Content = content
		doc.Lines = splitLines(content)
	}
}

// Close closes a document
func (dm *DocumentManager) Close(uri string) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	delete(dm.documents, uri)
}

// Get retrieves a document
func (dm *DocumentManager) Get(uri string) *Document {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	return dm.documents[uri]
}

// GetContent retrieves document content
func (dm *DocumentManager) GetContent(uri string) string {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	if doc, ok := dm.documents[uri]; ok {
		return doc.Content
	}
	return ""
}

// GetAll returns all open documents
func (dm *DocumentManager) GetAll() []*Document {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	docs := make([]*Document, 0, len(dm.documents))
	for _, doc := range dm.documents {
		docs = append(docs, doc)
	}
	return docs
}

// PositionToOffset converts a Position to a byte offset
func (d *Document) PositionToOffset(pos Position) int {
	if pos.Line >= len(d.Lines) {
		return len(d.Content)
	}

	offset := 0
	for i := 0; i < pos.Line; i++ {
		offset += len(d.Lines[i]) + 1 // +1 for newline
	}

	if pos.Character > len(d.Lines[pos.Line]) {
		offset += len(d.Lines[pos.Line])
	} else {
		offset += pos.Character
	}

	return offset
}

// OffsetToPosition converts a byte offset to a Position
func (d *Document) OffsetToPosition(offset int) Position {
	line := 0
	col := 0
	currentOffset := 0

	for i, l := range d.Lines {
		lineLen := len(l) + 1 // +1 for newline
		if currentOffset+lineLen > offset {
			line = i
			col = offset - currentOffset
			break
		}
		currentOffset += lineLen
		line = i + 1
	}

	return Position{Line: line, Character: col}
}

// GetWordAtPosition returns the word at the given position
func (d *Document) GetWordAtPosition(pos Position) string {
	if pos.Line >= len(d.Lines) {
		return ""
	}

	line := d.Lines[pos.Line]
	if pos.Character >= len(line) {
		return ""
	}

	// Find word boundaries
	start := pos.Character
	end := pos.Character

	for start > 0 && isWordChar(line[start-1]) {
		start--
	}

	for end < len(line) && isWordChar(line[end]) {
		end++
	}

	return line[start:end]
}

// GetLineContent returns the content of a specific line
func (d *Document) GetLineContent(line int) string {
	if line < 0 || line >= len(d.Lines) {
		return ""
	}
	return d.Lines[line]
}

func splitLines(content string) []string {
	return strings.Split(content, "\n")
}

func isWordChar(c byte) bool {
	return (c >= 'a' && c <= 'z') ||
		(c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') ||
		c == '_'
}
