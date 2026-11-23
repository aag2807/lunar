package lsp

import (
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"lunar/internal/source"
	"lunar/internal/version"
	"lunar/stdlib"
)

// uriToPath turns a file:// URI into an absolute path. On Windows the URI
// path starts with a slash before the drive letter, which is dropped.
func uriToPath(uri string) string { return uriToPathOn(runtime.GOOS, uri) }

// uriToPathOn converts for the given OS. On Windows a drive letter comes
// out upper case whichever way the editor wrote it (VS Code sends
// file:///c%3A/...), so one file always has one path, and a URI with a host
// (file://wsl$/Ubuntu/home/...) is a \\host\share path. The authority is
// taken apart by hand: Go's url.Parse refuses the escaped host wsl%24.
func uriToPathOn(goos, uri string) string {
	// A new buffer not saved yet (untitled:Untitled-1) is checked as a
	// file of its own in a folder no project owns.
	if name, ok := strings.CutPrefix(uri, "untitled:"); ok {
		return filepath.Join(untitledDir(), untitledName.ReplaceAllString(name, "_")+".lunar")
	}
	rest, ok := strings.CutPrefix(uri, "file://")
	if !ok {
		return uri
	}
	host, p, _ := strings.Cut(rest, "/")
	if i := strings.IndexAny(p, "?#"); i >= 0 {
		p = p[:i]
	}
	p = "/" + p
	if unescaped, err := url.PathUnescape(p); err == nil {
		p = unescaped
	}
	if unescaped, err := url.PathUnescape(host); err == nil {
		host = unescaped
	}
	if goos != "windows" {
		// path, not filepath: the result is a Unix path whichever system
		// this runs on (filepath would use '\' on Windows).
		return path.Clean(p)
	}
	if host != "" && host != "localhost" {
		return `\\` + host + strings.ReplaceAll(path.Clean(p), "/", `\`)
	}
	if len(p) > 2 && p[0] == '/' && p[2] == ':' {
		p = p[1:]
	}
	p = path.Clean(p)
	if len(p) >= 2 && p[1] == ':' {
		p = strings.ToUpper(p[:1]) + p[1:]
		if len(p) == 2 {
			p += "/" // the drive's root
		}
	}
	return strings.ReplaceAll(p, "/", `\`)
}

// pathToURI is the inverse of uriToPath.
func pathToURI(path string) string { return pathToURIOn(runtime.GOOS, path) }

func pathToURIOn(goos, p string) string {
	if strings.HasPrefix(p, "<stdlib>/") {
		p = stdlibCopy(strings.TrimPrefix(p, "<stdlib>/"))
	}
	if goos != "windows" {
		return (&url.URL{Scheme: "file", Path: filepath.ToSlash(p)}).String()
	}
	slashed := strings.ReplaceAll(p, `\`, "/")
	if share, ok := strings.CutPrefix(slashed, "//"); ok {
		host, rest, _ := strings.Cut(share, "/")
		return (&url.URL{Scheme: "file", Host: host, Path: "/" + rest}).String()
	}
	if !strings.HasPrefix(slashed, "/") {
		slashed = "/" + slashed // C:/... becomes /C:/...
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

// untitledName is what may not appear in the file name of an untitled
// buffer.
var untitledName = regexp.MustCompile(`[^A-Za-z0-9_-]`)

// untitledDir is where untitled buffers are placed for checking; it exists
// once one has been opened.
func untitledDir() string { return filepath.Join(os.TempDir(), "lunar-untitled") }
