package parser

import (
	"strings"
	"testing"

	"lunar/internal/lexer"
)

// A declaration is only a signature; a body after one is reported as
// such, not as a stray 'end'.
func TestDeclareFunctionWithABody(t *testing.T) {
	p := New(lexer.New("declare function f(): number\n    return 1\nend\n"))
	p.Parse()
	errs := p.Errors()
	if len(errs) == 0 || !strings.Contains(errs[len(errs)-1].Message, "'declare function f' cannot have a body") {
		t.Fatalf("got %v", errs)
	}
}
