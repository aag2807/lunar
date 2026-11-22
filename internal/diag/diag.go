// Package diag is the one shape every compiler error takes on its way out:
// to a terminal, as JSON lines for an editor, or to the language server.
package diag

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"
)

// Severity orders how serious a diagnostic is.
type Severity int

const (
	Error Severity = iota
	Warning
	Info
)

func (s Severity) String() string {
	switch s {
	case Warning:
		return "warning"
	case Info:
		return "info"
	default:
		return "error"
	}
}

// Diagnostic is one message about one span of a file. Lines and columns are
// 1-based and columns count bytes; EndLine and EndColumn are just past the
// span. A diagnostic that is not about a place in a file (bad usage, an I/O
// failure) has an empty File and zero positions.
type Diagnostic struct {
	File      string
	Line      int
	Column    int
	EndLine   int
	EndColumn int
	Severity  Severity
	// Code is a stable kebab-case name for the kind of problem, such as
	// "field-type"; Message is the human text.
	Code    string
	Message string
}

// Location is "file:line:column".
func (d Diagnostic) Location() string {
	return fmt.Sprintf("%s:%d:%d", d.File, d.Line, d.Column)
}

// jsonDiagnostic fixes the field order of a JSON line; encoding/json writes
// struct fields in declaration order.
type jsonDiagnostic struct {
	Kind      string `json:"kind"`
	File      string `json:"file"`
	Line      int    `json:"line"`
	Column    int    `json:"column"`
	EndLine   int    `json:"endLine"`
	EndColumn int    `json:"endColumn"`
	Severity  string `json:"severity"`
	Code      string `json:"code"`
	Message   string `json:"message"`
}

func (d Diagnostic) MarshalJSON() ([]byte, error) {
	return marshalReadable(jsonDiagnostic{
		Kind: "diagnostic", File: d.File, Line: d.Line, Column: d.Column,
		EndLine: d.EndLine, EndColumn: d.EndColumn, Severity: d.Severity.String(),
		Code: d.Code, Message: d.Message,
	})
}

// Summary closes a build's JSON output: how many errors and warnings, how
// many source files were compiled, and whether any output was written.
type Summary struct {
	Errors   int
	Warnings int
	Files    int
	Written  bool
}

type jsonSummary struct {
	Kind     string `json:"kind"`
	Errors   int    `json:"errors"`
	Warnings int    `json:"warnings"`
	Files    int    `json:"files"`
	Written  bool   `json:"written"`
}

func (s Summary) MarshalJSON() ([]byte, error) {
	return marshalReadable(jsonSummary{Kind: "summary", Errors: s.Errors, Warnings: s.Warnings, Files: s.Files, Written: s.Written})
}

// marshalReadable is json.Marshal without escaping '<', '>' and '&', which
// messages use ('<table literal>', '<stdlib>/lua.d.lunar').
func marshalReadable(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}

// WriteJSON writes v as one line of JSON. HTML characters are not escaped,
// so messages stay readable: '<stdlib>' rather than '<stdlib>'.
func WriteJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

// List is a set of diagnostics.
type List []Diagnostic

// Sort orders the list by file, then position.
func (l List) Sort() {
	sort.SliceStable(l, func(i, j int) bool {
		a, b := l[i], l[j]
		if a.File != b.File {
			return a.File < b.File
		}
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		return a.Column < b.Column
	})
}

func (l List) count(s Severity) int {
	n := 0
	for _, d := range l {
		if d.Severity == s {
			n++
		}
	}
	return n
}

// Errors counts the error-severity diagnostics.
func (l List) Errors() int { return l.count(Error) }

// Warnings counts the warning-severity diagnostics.
func (l List) Warnings() int { return l.count(Warning) }

// HasErrors reports whether any diagnostic is an error.
func (l List) HasErrors() bool { return l.Errors() > 0 }
