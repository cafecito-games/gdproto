package parser_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/cafecito-games/gogdproto/internal/ast"
	"github.com/cafecito-games/gogdproto/internal/lexer"
	"github.com/cafecito-games/gogdproto/internal/parser"
)

// parseSource is a helper that lexes and parses a proto source.
func parseSource(t *testing.T, src string) (*ast.ProtoFile, error) {
	t.Helper()
	tokens, err := lexer.Tokenize(src, "test.proto")
	if err != nil {
		t.Fatalf("lex error: %v", err)
	}
	return parser.Parse(tokens, "test.proto")
}

func TestParserErrorFormat(t *testing.T) {
	tok := lexer.Token{Type: lexer.TokenIdentifier, Value: "Foo", Line: 3, Column: 7}
	err := &parser.ParserError{File: "x.proto", Token: tok, Message: "boom"}
	want := "x.proto:3:7: error: boom"
	if got := err.Error(); got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
}

func TestParserErrorDefaultFile(t *testing.T) {
	err := &parser.ParserError{Token: lexer.Token{Line: 1, Column: 1}, Message: "oops"}
	if !strings.Contains(err.Error(), "<input>") {
		t.Fatalf("expected <input>, got %q", err.Error())
	}
}

func TestSyntaxProto3(t *testing.T) {
	file, err := parseSource(t, `syntax = "proto3";`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if file.Syntax != "proto3" {
		t.Errorf("Syntax = %q, want %q", file.Syntax, "proto3")
	}
}

func TestSyntaxMissing(t *testing.T) {
	_, err := parseSource(t, `message Foo {}`)
	if err == nil {
		t.Fatal("expected error for missing syntax")
	}
	var pe *parser.ParserError
	if !errors.As(err, &pe) {
		t.Fatalf("expected *ParserError, got %T", err)
	}
	if !strings.Contains(pe.Message, "Expected TokenSyntax") {
		t.Errorf("Message = %q, want contains 'Expected TokenSyntax'", pe.Message)
	}
}
