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

func TestTokenizeEmpty(t *testing.T) {
	tokens, err := lexer.Tokenize("", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tokens) != 1 || tokens[0].Type != lexer.TokenEOF {
		t.Fatalf("got %+v, want single EOF", tokens)
	}
	if tokens[0].Line != 1 || tokens[0].Column != 1 {
		t.Errorf("EOF position = %d:%d, want 1:1", tokens[0].Line, tokens[0].Column)
	}
}

func TestTokenizeWhitespaceOnly(t *testing.T) {
	tokens, err := lexer.Tokenize("   \t\n\r\n   ", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tokens) != 1 || tokens[0].Type != lexer.TokenEOF {
		t.Fatalf("got %+v, want single EOF", tokens)
	}
}

func TestTokenizeAllSymbols(t *testing.T) {
	tokens, err := lexer.Tokenize("{}[]()<>;=,.", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []lexer.TokenType{
		lexer.TokenLBrace, lexer.TokenRBrace,
		lexer.TokenLBracket, lexer.TokenRBracket,
		lexer.TokenLParen, lexer.TokenRParen,
		lexer.TokenLT, lexer.TokenGT,
		lexer.TokenSemicolon, lexer.TokenEquals,
		lexer.TokenComma, lexer.TokenDot,
		lexer.TokenEOF,
	}
	if len(tokens) != len(want) {
		t.Fatalf("got %d tokens, want %d: %+v", len(tokens), len(want), tokens)
	}
	for i, w := range want {
		if tokens[i].Type != w {
			t.Errorf("token[%d].Type = %s, want %s", i, tokens[i].Type, w)
		}
	}
}

func TestSymbolPositionTracking(t *testing.T) {
	tokens, err := lexer.Tokenize("=\n=", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tokens[0].Line != 1 || tokens[0].Column != 1 {
		t.Errorf("first = at %d:%d, want 1:1", tokens[0].Line, tokens[0].Column)
	}
	if tokens[1].Line != 2 || tokens[1].Column != 1 {
		t.Errorf("second = at %d:%d, want 2:1", tokens[1].Line, tokens[1].Column)
	}
}

func TestUnexpectedCharacter(t *testing.T) {
	_, err := lexer.Tokenize("@", "test.proto")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var le *lexer.LexerError
	if !errors.As(err, &le) {
		t.Fatalf("expected *LexerError, got %T", err)
	}
	if !strings.Contains(le.Message, "Unexpected character") {
		t.Errorf("message = %q, want contains 'Unexpected character'", le.Message)
	}
	if le.File != "test.proto" {
		t.Errorf("file = %q, want %q", le.File, "test.proto")
	}
	if le.Line != 1 || le.Column != 1 {
		t.Errorf("position = %d:%d, want 1:1", le.Line, le.Column)
	}
}

func TestSimpleIdentifier(t *testing.T) {
	tokens, err := lexer.Tokenize("MyMessage", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tokens[0].Type != lexer.TokenIdentifier || tokens[0].Value != "MyMessage" {
		t.Errorf("got %+v, want Identifier 'MyMessage'", tokens[0])
	}
}

func TestIdentifierVariants(t *testing.T) {
	cases := []string{"field123", "my_field_name", "_private"}
	for _, src := range cases {
		tokens, err := lexer.Tokenize(src, "")
		if err != nil {
			t.Errorf("Tokenize(%q) error: %v", src, err)
			continue
		}
		if tokens[0].Type != lexer.TokenIdentifier || tokens[0].Value != src {
			t.Errorf("Tokenize(%q): got %+v, want Identifier %q", src, tokens[0], src)
		}
	}
}

func TestKeywordsAll(t *testing.T) {
	cases := map[string]lexer.TokenType{
		"syntax":   lexer.TokenSyntax,
		"message":  lexer.TokenMessage,
		"enum":     lexer.TokenEnum,
		"repeated": lexer.TokenRepeated,
		"map":      lexer.TokenMap,
		"oneof":    lexer.TokenOneof,
		"import":   lexer.TokenImport,
		"public":   lexer.TokenPublic,
		"option":   lexer.TokenOption,
		"packed":   lexer.TokenPacked,
		"reserved": lexer.TokenReserved,
		"package":  lexer.TokenPackage,
		"service":  lexer.TokenService,
		"rpc":      lexer.TokenRPC,
		"returns":  lexer.TokenReturns,
		"stream":   lexer.TokenStream,
		"int32":    lexer.TokenInt32,
		"int64":    lexer.TokenInt64,
		"string":   lexer.TokenString,
		"bool":     lexer.TokenBool,
		"bytes":    lexer.TokenBytes,
		"double":   lexer.TokenDouble,
		"float":    lexer.TokenFloat,
		"true":     lexer.TokenTrue,
		"false":    lexer.TokenFalse,
	}
	for word, want := range cases {
		tokens, err := lexer.Tokenize(word, "")
		if err != nil {
			t.Errorf("Tokenize(%q) error: %v", word, err)
			continue
		}
		if tokens[0].Type != want {
			t.Errorf("Tokenize(%q): got %s, want %s", word, tokens[0].Type, want)
		}
		if tokens[0].Value != word {
			t.Errorf("Tokenize(%q): value = %q, want %q", word, tokens[0].Value, word)
		}
	}
}

func TestMultipleKeywords(t *testing.T) {
	tokens, err := lexer.Tokenize("message enum repeated", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []lexer.TokenType{
		lexer.TokenMessage, lexer.TokenEnum, lexer.TokenRepeated, lexer.TokenEOF,
	}
	if len(tokens) != len(want) {
		t.Fatalf("got %d tokens, want %d", len(tokens), len(want))
	}
	for i, w := range want {
		if tokens[i].Type != w {
			t.Errorf("token[%d] = %s, want %s", i, tokens[i].Type, w)
		}
	}
}
