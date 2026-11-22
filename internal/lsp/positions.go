package lsp

import (
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"lunar/internal/source"
	"lunar/internal/version"
	"lunar/stdlib"
)

// uriToPath turns a file:// URI into an absolute path. On Windows the URI
// path starts with a slash before the drive letter, which is dropped.
func uriToPath(uri string) string {
	u, err := url.Parse(uri)
	if err != nil || u.Scheme != "file" {
		return uri
	}
	path := u.Path
	if runtime.GOOS == "windows" && len(path) > 2 && path[0] == '/' && path[2] == ':' {
		path = path[1:]
	}
	return filepath.Clean(filepath.FromSlash(path))
}

// pathToURI is the inverse of uriToPath.
func pathToURI(path string) string {
	if strings.HasPrefix(path, "<stdlib>/") {
		path = stdlibCopy(strings.TrimPrefix(path, "<stdlib>/"))
	}
	slashed := filepath.ToSlash(path)
	if !strings.HasPrefix(slashed, "/") {
		slashed = "/" + slashed // a Windows drive path
	}
	return (&url.URL{Scheme: "file", Path: slashed}).String()
}

// stdlibCopy writes a bundled declaration file to the user's cache folder
// (once per version) so an editor can open it, and returns its path.
func stdlibCopy(name string) string {
	dir, err := os.UserCacheDir()
	if err != nil {
		dir = os.TempDir()
	}
	path := filepath.Join(dir, "lunar", version.Version, "stdlib", name)
	if _, err := os.Stat(path); err == nil {
		return path
	}
	if data, err := stdlib.Files.ReadFile(name); err == nil {
		os.MkdirAll(filepath.Dir(path), 0o755)
		os.WriteFile(path, data, 0o644)
	}
	return path
}

// file returns the text a position in path refers to: the open buffer, the
// bundled stdlib, or the file on disk. The caller holds s.mu.
func (s *Server) file(path string) *source.File {
	if f, ok := s.files[path]; ok {
		return f
	}
	var text string
	if t, ok := s.docs[path]; ok {
		text = t
	} else if strings.HasPrefix(path, "<stdlib>/") {
		data, _ := stdlib.Files.ReadFile(strings.TrimPrefix(path, "<stdlib>/"))
		text = string(data)
	} else if data, err := os.ReadFile(path); err == nil {
		text = string(data)
	}
	f := source.NewFile(path, text)
	if s.files == nil {
		s.files = map[string]*source.File{}
	}
	s.files[path] = f
	return f
}

// position converts a 1-based line and byte column to an LSP position.
func (s *Server) position(path string, line, column int) Position {
	if line < 1 {
		return Position{}
	}
	return Position{Line: line - 1, Character: s.file(path).UTF16Column(line, column)}
}

// rangeOf converts a span to an LSP range.
func (s *Server) rangeOf(path string, line, column, endLine, endColumn int) Range {
	if endLine == 0 {
		endLine, endColumn = line, column
	}
	return Range{Start: s.position(path, line, column), End: s.position(path, endLine, endColumn)}
}

// offsetAt converts an LSP position in path to a byte offset.
func (s *Server) offsetAt(path string, pos Position) int {
	f := s.file(path)
	line := pos.Line + 1
	return f.Offset(line, f.ByteColumnFromUTF16(line, pos.Character))
}
