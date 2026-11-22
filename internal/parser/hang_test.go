package parser

import (
	"testing"
	"time"

	"lunar/internal/lexer"
)

// Half-typed members must end in a syntax error, not a parser that never
// moves on: `get` alone in a class body used to loop forever, growing.
func TestHalfTypedMembersDoNotHang(t *testing.T) {
	for _, src := range []string{
		"class Box\n    v: number = 1\n    get\nend\n",
		"class Box\n    set\nend\n",
		"class Box\n    static get\nend\n",
		"class Box\n    private set\nend\n",
		"class Box\n    get .alive()\nend\n",
		"class Box\n    get<T> size()\nend\n",
		"declare class B\n    get\nend\n",
		"interface I\n    static\nend\n",
		"interface I\n    readonly\nend\n",
		"enum E\n    A =\nend\n",
		"class Box\n    @\nend\n",
		"class Box\n    constructor\nend\n",
	} {
		done := make(chan []*ParseError, 1)
		go func() {
			p := New(lexer.New(src))
			p.Parse()
			done <- p.Errors()
		}()
		select {
		case errs := <-done:
			if len(errs) == 0 {
				t.Errorf("%q: no syntax error", src)
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("%q: the parser did not finish", src)
		}
	}
}
