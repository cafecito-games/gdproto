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

func TestSimpleInteger(t *testing.T) {
	tokens, _ := lexer.Tokenize("123", "")
	if tokens[0].Type != lexer.TokenIntLiteral || tokens[0].Value != "123" {
		t.Errorf("got %+v, want IntLiteral 123", tokens[0])
	}
}

func TestNegativeInteger(t *testing.T) {
	tokens, _ := lexer.Tokenize("-456", "")
	if tokens[0].Type != lexer.TokenIntLiteral || tokens[0].Value != "-456" {
		t.Errorf("got %+v, want IntLiteral -456", tokens[0])
	}
}

func TestZero(t *testing.T) {
	tokens, _ := lexer.Tokenize("0", "")
	if tokens[0].Type != lexer.TokenIntLiteral || tokens[0].Value != "0" {
		t.Errorf("got %+v, want IntLiteral 0", tokens[0])
	}
}

func TestHexNumber(t *testing.T) {
	tokens, _ := lexer.Tokenize("0x1A2B", "")
	if tokens[0].Type != lexer.TokenIntLiteral || tokens[0].Value != "0x1A2B" {
		t.Errorf("got %+v, want IntLiteral 0x1A2B", tokens[0])
	}
}

func TestOctalNumber(t *testing.T) {
	tokens, _ := lexer.Tokenize("0755", "")
	if tokens[0].Type != lexer.TokenIntLiteral || tokens[0].Value != "0755" {
		t.Errorf("got %+v, want IntLiteral 0755", tokens[0])
	}
}

func TestFloatDecimal(t *testing.T) {
	tokens, _ := lexer.Tokenize("3.14", "")
	if tokens[0].Type != lexer.TokenFloatLiteral || tokens[0].Value != "3.14" {
		t.Errorf("got %+v, want FloatLiteral 3.14", tokens[0])
	}
}

func TestFloatExponent(t *testing.T) {
	tokens, _ := lexer.Tokenize("1.5e10", "")
	if tokens[0].Type != lexer.TokenFloatLiteral || tokens[0].Value != "1.5e10" {
		t.Errorf("got %+v, want FloatLiteral 1.5e10", tokens[0])
	}
}

func TestFloatNegativeExponent(t *testing.T) {
	tokens, _ := lexer.Tokenize("2.5e-3", "")
	if tokens[0].Type != lexer.TokenFloatLiteral || tokens[0].Value != "2.5e-3" {
		t.Errorf("got %+v, want FloatLiteral 2.5e-3", tokens[0])
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

func TestStringDoubleQuote(t *testing.T) {
	tokens, _ := lexer.Tokenize(`"hello world"`, "")
	if tokens[0].Type != lexer.TokenStringLiteral || tokens[0].Value != "hello world" {
		t.Errorf("got %+v", tokens[0])
	}
}

func TestStringSingleQuote(t *testing.T) {
	tokens, _ := lexer.Tokenize(`'hello world'`, "")
	if tokens[0].Type != lexer.TokenStringLiteral || tokens[0].Value != "hello world" {
		t.Errorf("got %+v", tokens[0])
	}
}

func TestStringEmpty(t *testing.T) {
	tokens, _ := lexer.Tokenize(`""`, "")
	if tokens[0].Type != lexer.TokenStringLiteral || tokens[0].Value != "" {
		t.Errorf("got %+v", tokens[0])
	}
}

func TestStringEscapes(t *testing.T) {
	tokens, _ := lexer.Tokenize(`"hello\nworld\t!"`, "")
	if tokens[0].Value != "hello\nworld\t!" {
		t.Errorf("got %q, want %q", tokens[0].Value, "hello\nworld\t!")
	}
}

func TestStringEscapedQuotes(t *testing.T) {
	tokens, _ := lexer.Tokenize(`"say \"hello\""`, "")
	if tokens[0].Value != `say "hello"` {
		t.Errorf("got %q", tokens[0].Value)
	}
}

func TestStringEscapedBackslash(t *testing.T) {
	tokens, _ := lexer.Tokenize(`"path\\to\\file"`, "")
	if tokens[0].Value != `path\to\file` {
		t.Errorf("got %q", tokens[0].Value)
	}
}

func TestStringHexEscape(t *testing.T) {
	tokens, _ := lexer.Tokenize(`"\x41\x42\x43"`, "")
	if tokens[0].Value != "ABC" {
		t.Errorf("got %q, want ABC", tokens[0].Value)
	}
}

func TestStringUnterminated(t *testing.T) {
	_, err := lexer.Tokenize(`"hello`, "")
	var le *lexer.LexerError
	if !errors.As(err, &le) || !strings.Contains(le.Message, "Unterminated string literal") {
		t.Errorf("got %v, want unterminated string error", err)
	}
}

func TestStringNewline(t *testing.T) {
	_, err := lexer.Tokenize("\"hello\nworld\"", "")
	var le *lexer.LexerError
	if !errors.As(err, &le) || !strings.Contains(le.Message, "Newline in string literal") {
		t.Errorf("got %v, want newline-in-string error", err)
	}
}

func TestStringInvalidEscape(t *testing.T) {
	_, err := lexer.Tokenize(`"\q"`, "")
	var le *lexer.LexerError
	if !errors.As(err, &le) || !strings.Contains(le.Message, "Invalid escape sequence") {
		t.Errorf("got %v, want invalid escape error", err)
	}
}
