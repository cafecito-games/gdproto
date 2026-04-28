package lexer

import "fmt"

// Tokenize converts .proto source code into a stream of tokens.
// The filename is used only in error messages; pass "" for "<input>".
// The returned slice always ends with a TokenEOF entry.
func Tokenize(source, filename string) ([]Token, error) {
	l := &lexer{source: source, filename: filename, line: 1, column: 1}
	return l.run()
}

type lexer struct {
	source   string
	filename string
	pos      int
	line     int
	column   int
	tokens   []Token
}

func (l *lexer) run() ([]Token, error) {
	for l.pos < len(l.source) {
		l.skipWhitespace()
		if l.pos >= len(l.source) {
			break
		}

		ch := l.source[l.pos]
		line, col := l.line, l.column

		if t, ok := singleCharSymbol(ch); ok {
			l.tokens = append(l.tokens, Token{Type: t, Value: string(ch), Line: line, Column: col})
			l.advance()
			continue
		}

		if isIdentStart(ch) {
			l.tokens = append(l.tokens, l.readIdentifier())
			continue
		}

		return nil, &LexerError{
			File:    l.filename,
			Line:    line,
			Column:  col,
			Message: fmt.Sprintf("Unexpected character: %q", ch),
		}
	}

	l.tokens = append(l.tokens, Token{Type: TokenEOF, Line: l.line, Column: l.column})
	return l.tokens, nil
}

func (l *lexer) advance() {
	if l.pos >= len(l.source) {
		return
	}
	ch := l.source[l.pos]
	l.pos++
	if ch == '\n' {
		l.line++
		l.column = 1
	} else {
		l.column++
	}
}

func (l *lexer) skipWhitespace() {
	for l.pos < len(l.source) {
		ch := l.source[l.pos]
		if ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' {
			l.advance()
			continue
		}
		return
	}
}

func isIdentStart(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_'
}

func isIdentContinue(ch byte) bool {
	return isIdentStart(ch) || (ch >= '0' && ch <= '9')
}

func (l *lexer) readIdentifier() Token {
	startLine, startCol := l.line, l.column
	start := l.pos
	for l.pos < len(l.source) && isIdentContinue(l.source[l.pos]) {
		l.advance()
	}
	value := l.source[start:l.pos]
	tt := TokenIdentifier
	if kw, ok := keywords[value]; ok {
		tt = kw
	}
	return Token{Type: tt, Value: value, Line: startLine, Column: startCol}
}

func singleCharSymbol(ch byte) (TokenType, bool) {
	switch ch {
	case '{':
		return TokenLBrace, true
	case '}':
		return TokenRBrace, true
	case '[':
		return TokenLBracket, true
	case ']':
		return TokenRBracket, true
	case '(':
		return TokenLParen, true
	case ')':
		return TokenRParen, true
	case '<':
		return TokenLT, true
	case '>':
		return TokenGT, true
	case ';':
		return TokenSemicolon, true
	case '=':
		return TokenEquals, true
	case ',':
		return TokenComma, true
	case '.':
		return TokenDot, true
	}
	return 0, false
}
