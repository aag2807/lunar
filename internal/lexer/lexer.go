package lexer

type Lexer struct {
	input        string
	position     int
	readPosition int
	ch           byte
	line         int
	column       int
}

func New(input string) *Lexer {
	l := &Lexer{input: input, line: 1}
	l.readChar()

	return l
}
