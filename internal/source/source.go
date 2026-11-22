// Package source maps between byte offsets in a file and the positions
// people and editors use: 1-based lines and byte columns for messages, and
// the UTF-16 columns the Language Server Protocol counts in.
package source

import (
	"sort"
	"unicode/utf8"
)

// File is one source text and the offsets where its lines start.
type File struct {
	Path       string
	Text       string
	lineStarts []int
}

// NewFile indexes text's lines.
func NewFile(path, text string) *File {
	starts := []int{0}
	for i := 0; i < len(text); i++ {
		if text[i] == '\n' {
			starts = append(starts, i+1)
		}
	}
	return &File{Path: path, Text: text, lineStarts: starts}
}

// Position returns the 1-based line and byte column of offset. The offset
// of a newline is the last column of its line; len(Text) is valid too.
func (f *File) Position(offset int) (line, column int) {
	if offset < 0 {
		offset = 0
	}
	if offset > len(f.Text) {
		offset = len(f.Text)
	}
	// The last line start at or before offset.
	i := sort.Search(len(f.lineStarts), func(i int) bool { return f.lineStarts[i] > offset }) - 1
	return i + 1, offset - f.lineStarts[i] + 1
}

// Offset returns the byte offset of a 1-based line and byte column, clamped
// to the file.
func (f *File) Offset(line, column int) int {
	if line < 1 {
		return 0
	}
	if line > len(f.lineStarts) {
		return len(f.Text)
	}
	offset := f.lineStarts[line-1] + column - 1
	if offset > len(f.Text) {
		return len(f.Text)
	}
	if offset < 0 {
		return 0
	}
	return offset
}

// LineCount is the number of lines in the file.
func (f *File) LineCount() int {
	return len(f.lineStarts)
}

// lineText returns line's text without its newline.
func (f *File) lineText(line int) string {
	if line < 1 || line > len(f.lineStarts) {
		return ""
	}
	start := f.lineStarts[line-1]
	end := len(f.Text)
	if line < len(f.lineStarts) {
		end = f.lineStarts[line] - 1
	}
	return f.Text[start:end]
}

// UTF16Column converts a 1-based byte column on line to the 0-based count
// of UTF-16 code units before it, which is how LSP addresses characters.
func (f *File) UTF16Column(line, column int) int {
	text := f.lineText(line)
	if column-1 < len(text) {
		text = text[:max(column-1, 0)]
	}
	units := 0
	for _, r := range text {
		if r >= 0x10000 {
			units += 2
		} else {
			units++
		}
	}
	return units
}

// ByteColumnFromUTF16 converts a 0-based UTF-16 offset on line back to a
// 1-based byte column. An offset past the end of the line clamps to it.
func (f *File) ByteColumnFromUTF16(line, units int) int {
	text := f.lineText(line)
	count := 0
	for i := 0; i < len(text); {
		if count >= units {
			return i + 1
		}
		r, size := utf8.DecodeRuneInString(text[i:])
		if r >= 0x10000 {
			count += 2
		} else {
			count++
		}
		i += size
	}
	return len(text) + 1
}
