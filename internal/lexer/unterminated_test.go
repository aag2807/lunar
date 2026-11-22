package lexer

import (
	"testing"
	"time"
)

// An unclosed string, template, long string or long comment used to loop
// forever at the end of input; in the language server that froze it for
// good the moment someone typed a quote.
func TestUnterminatedLiteralsStopAtEndOfInput(t *testing.T) {
	cases := map[string]string{
		`local s = "abc`:           "unterminated string",
		"local s = 'abc":           "unterminated string",
		"local s = `abc ${x}":      "unterminated template string",
		"local s = [[abc":          "unterminated long string",
		"local s = [==[abc]]":      "unterminated long string",
		"--[[ open\nlocal x = 1\n": "unterminated long comment",
	}
	for src, want := range cases {
		done := make(chan Token, 1)
		go func() {
			l := New(src)
			for i := 0; i < 100; i++ {
				tok := l.NextToken()
				if tok.Type == ILLEGAL || tok.Type == EOF {
					done <- tok
					return
				}
			}
			done <- Token{Type: "too many tokens"}
		}()
		select {
		case tok := <-done:
			if tok.Type != ILLEGAL || tok.Literal != want {
				t.Errorf("%q: got %s %q, want ILLEGAL %q", src, tok.Type, tok.Literal, want)
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("%q: lexer did not finish", src)
		}
	}
}
