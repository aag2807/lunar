package lexer

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"lunar/internal/source"
)

// lunarFiles lists every .lunar file in the repository.
func lunarFiles(t *testing.T) []string {
	t.Helper()
	var files []string
	root := filepath.Join("..", "..")
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && (d.Name() == ".git" || d.Name() == "node_modules") {
			return filepath.SkipDir
		}
		if !d.IsDir() && strings.HasSuffix(path, ".lunar") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("no .lunar files found")
	}
	return files
}

// Every token's recorded position must agree with its byte offsets: the
// start line and column are where Offset is, the end is where End is, and
// the text between them is the token. Diagnostics and the language server
// are only as accurate as this.
func TestTokenPositionsMatchOffsets(t *testing.T) {
	for _, path := range lunarFiles(t) {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		text := string(data)
		file := source.NewFile(path, text)
		l := NewFile(path, text)

		for i := 0; ; i++ {
			tok := l.NextToken()
			if tok.Type == EOF {
				break
			}
			if tok.File != path {
				t.Fatalf("%s: token %d has File %q", path, i, tok.File)
			}
			if tok.Offset < 0 || tok.End <= tok.Offset || tok.End > len(text) {
				t.Fatalf("%s: token %d %s %q has offsets %d..%d", path, i, tok.Type, tok.Literal, tok.Offset, tok.End)
			}
			if line, col := file.Position(tok.Offset); line != tok.Line || col != tok.Column {
				t.Fatalf("%s: token %s %q starts at %d:%d but its offset is at %d:%d",
					path, tok.Type, tok.Literal, tok.Line, tok.Column, line, col)
			}
			if line, col := file.Position(tok.End); line != tok.EndLine || col != tok.EndColumn {
				t.Fatalf("%s: token %s %q ends at %d:%d but its end offset is at %d:%d",
					path, tok.Type, tok.Literal, tok.EndLine, tok.EndColumn, line, col)
			}
			lexeme := text[tok.Offset:tok.End]
			switch tok.Type {
			case STRING, TEMPLATE_STRING:
				if q := lexeme[0]; lexeme[len(lexeme)-1] != q && q != '[' {
					t.Fatalf("%s: string token text %q is not delimited", path, lexeme)
				}
			default:
				if lexeme != tok.Literal {
					t.Fatalf("%s: token %s literal %q but source text %q", path, tok.Type, tok.Literal, lexeme)
				}
			}
		}
	}
}
