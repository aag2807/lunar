package parser

import (
	"lunar/internal/lexer"
	"testing"
)

func TestParseErrorNamesFileAndSpan(t *testing.T) {
	p := New(lexer.NewFile("scripts/bad.lunar", "local x = = 1\n"))
	p.Parse()
	errs := p.Errors()
	if len(errs) == 0 {
		t.Fatal("expected a parse error")
	}
	e := errs[0]
	if e.File != "scripts/bad.lunar" || e.Line != 1 || e.Column != 11 || e.EndColumn != 12 {
		t.Errorf("got %+v", e)
	}
}
