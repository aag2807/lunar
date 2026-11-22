package source

import "testing"

func TestPositionOfOffset(t *testing.T) {
	f := NewFile("a.lunar", "ab\ncd\n\nxyz")
	cases := []struct{ offset, line, col int }{
		{0, 1, 1}, {1, 1, 2}, {2, 1, 3}, // the newline itself ends line 1
		{3, 2, 1}, {4, 2, 2},
		{6, 3, 1},
		{7, 4, 1}, {9, 4, 3},
		{10, 4, 4}, // end of text
	}
	for _, c := range cases {
		line, col := f.Position(c.offset)
		if line != c.line || col != c.col {
			t.Errorf("Position(%d) = %d:%d, want %d:%d", c.offset, line, col, c.line, c.col)
		}
	}
}

func TestOffsetOfPosition(t *testing.T) {
	f := NewFile("a.lunar", "ab\ncd\n\nxyz")
	for offset := 0; offset <= len(f.Text); offset++ {
		line, col := f.Position(offset)
		if got := f.Offset(line, col); got != offset {
			t.Errorf("Offset(Position(%d)) = %d", offset, got)
		}
	}
}

// LSP positions count UTF-16 code units; byte columns do not.
func TestUTF16Column(t *testing.T) {
	// "é" is 2 bytes / 1 UTF-16 unit; "𝄞" is 4 bytes / 2 units.
	f := NewFile("a.lunar", "local s = \"é𝄞\" x")
	line, col := f.Position(len("local s = \"é𝄞\" "))
	if line != 1 {
		t.Fatalf("line %d", line)
	}
	if got := f.UTF16Column(line, col); got != 16 {
		t.Errorf("UTF16Column = %d, want 16 (0-based units before x)", got)
	}
	if got := f.ByteColumnFromUTF16(line, 16); got != col {
		t.Errorf("ByteColumnFromUTF16(16) = %d, want %d", got, col)
	}
}
