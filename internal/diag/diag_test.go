package diag

import (
	"bytes"
	"testing"
)

// The engine's editor reads these lines with a small C parser, so the shape
// and field order are part of the contract.
func TestDiagnosticJSONLine(t *testing.T) {
	d := Diagnostic{
		File: "/game/scripts/enemy.lunar", Line: 12, Column: 7, EndLine: 12, EndColumn: 12,
		Severity: Error, Code: "field-type", Message: "Field 'hp' of 'Enemy' must be 'number', got 'string'",
	}
	var buf bytes.Buffer
	if err := WriteJSON(&buf, d); err != nil {
		t.Fatal(err)
	}
	want := `{"kind":"diagnostic","file":"/game/scripts/enemy.lunar","line":12,"column":7,"endLine":12,"endColumn":12,"severity":"error","code":"field-type","message":"Field 'hp' of 'Enemy' must be 'number', got 'string'"}` + "\n"
	if buf.String() != want {
		t.Errorf("got  %s\nwant %s", buf.String(), want)
	}
}

func TestSummaryJSONLine(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteJSON(&buf, Summary{Errors: 1, Files: 14}); err != nil {
		t.Fatal(err)
	}
	want := `{"kind":"summary","errors":1,"warnings":0,"files":14,"written":false}` + "\n"
	if buf.String() != want {
		t.Errorf("got  %s\nwant %s", buf.String(), want)
	}
}

func TestSeverityNames(t *testing.T) {
	for sev, name := range map[Severity]string{Error: "error", Warning: "warning", Info: "info"} {
		if sev.String() != name {
			t.Errorf("%d.String() = %q, want %q", sev, sev.String(), name)
		}
	}
}

func TestListSortAndCount(t *testing.T) {
	l := List{
		{File: "b.lunar", Line: 1, Column: 1, Severity: Error},
		{File: "a.lunar", Line: 9, Column: 1, Severity: Warning},
		{File: "a.lunar", Line: 2, Column: 5, Severity: Error},
		{File: "a.lunar", Line: 2, Column: 1, Severity: Error},
	}
	l.Sort()
	order := []string{"a.lunar:2:1", "a.lunar:2:5", "a.lunar:9:1", "b.lunar:1:1"}
	for i, d := range l {
		if got := d.Location(); got != order[i] {
			t.Errorf("position %d: %s, want %s", i, got, order[i])
		}
	}
	if l.Errors() != 3 || l.Warnings() != 1 || !l.HasErrors() {
		t.Errorf("counted %d errors, %d warnings", l.Errors(), l.Warnings())
	}
}
