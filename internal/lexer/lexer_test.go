package lexer_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/cafecito-games/gogdproto/internal/lexer"
)

func TestTokenTypeString(t *testing.T) {
	cases := map[lexer.TokenType]string{
		lexer.TokenEOF:        "TokenEOF",
		lexer.TokenSyntax:     "TokenSyntax",
		lexer.TokenIdentifier: "TokenIdentifier",
		lexer.TokenIntLiteral: "TokenIntLiteral",
		lexer.TokenLBrace:     "TokenLBrace",
	}
	for tt, want := range cases {
		if got := tt.String(); got != want {
			t.Errorf("%d.String() = %q, want %q", tt, got, want)
		}
	}
}

func TestLexerErrorFormat(t *testing.T) {
	err := &lexer.LexerError{
		File:    "test.proto",
		Line:    5,
		Column:  12,
		Message: "Unexpected character",
	}
	got := err.Error()
	want := "test.proto:5:12: error: Unexpected character"
	if got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestLexerErrorDefaultFile(t *testing.T) {
	err := &lexer.LexerError{Line: 1, Column: 1, Message: "oops"}
	if !strings.Contains(err.Error(), "<input>") {
		t.Errorf("expected <input> in default error: %q", err.Error())
	}
}

func TestLexerErrorIsError(t *testing.T) {
	var e error = &lexer.LexerError{}
	var le *lexer.LexerError
	if !errors.As(e, &le) {
		t.Fatal("LexerError must implement error")
	}
}
