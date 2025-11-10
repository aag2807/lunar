package parser

import (
	"lunar/internal/lexer"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			"simple variable",
			"local x = 5",
		},
		{
			"variable with type",
			"local x: number = 5",
		},
		{
			"function without types",
			`function add(a, b)
    return a + b
end`,
		},
		{
			"function with types",
			`function add(a: number, b: number): number
    return a + b
end`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := New(l)
			statements := p.Parse()

			if len(p.Errors()) > 0 {
				t.Errorf("Parser errors for %q:", tt.name)
				for _, err := range p.Errors() {
					t.Errorf("  %s", err)
				}
			}

			if len(statements) == 0 {
				t.Errorf("Parse() returned no statements for %q", tt.name)
			}
		})
	}
}
