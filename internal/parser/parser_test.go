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

func TestSimpleImport(t *testing.T) {
	src := `syntax = "proto3"; import "other.proto";`
	file, err := parseSource(t, src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(file.Imports) != 1 {
		t.Fatalf("got %d imports, want 1", len(file.Imports))
	}
	if file.Imports[0].Path != "other.proto" || file.Imports[0].Public {
		t.Errorf("got %+v", file.Imports[0])
	}
}

func TestPublicImport(t *testing.T) {
	src := `syntax = "proto3"; import public "x.proto";`
	file, err := parseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	if !file.Imports[0].Public {
		t.Error("expected public=true")
	}
}

func TestMultipleImports(t *testing.T) {
	src := `syntax = "proto3";
import "foo.proto";
import "bar.proto";
import public "baz.proto";`
	file, err := parseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	if len(file.Imports) != 3 {
		t.Fatalf("got %d", len(file.Imports))
	}
	if file.Imports[2].Path != "baz.proto" || !file.Imports[2].Public {
		t.Errorf("got %+v", file.Imports[2])
	}
}

func TestSimplePackage(t *testing.T) {
	file, err := parseSource(t, `syntax = "proto3"; package mypackage;`)
	if err != nil {
		t.Fatal(err)
	}
	if file.Package != "mypackage" {
		t.Errorf("Package = %q", file.Package)
	}
}

func TestDottedPackage(t *testing.T) {
	file, err := parseSource(t, `syntax = "proto3"; package com.example.proto;`)
	if err != nil {
		t.Fatal(err)
	}
	if file.Package != "com.example.proto" {
		t.Errorf("Package = %q", file.Package)
	}
}

func TestFileOptionIdentifier(t *testing.T) {
	file, err := parseSource(t, `syntax = "proto3"; option optimize_for = SPEED;`)
	if err != nil {
		t.Fatal(err)
	}
	if file.Options["optimize_for"] != "SPEED" {
		t.Errorf("got %v", file.Options)
	}
}

func TestFileOptionInt(t *testing.T) {
	file, err := parseSource(t, `syntax = "proto3"; option max_size = 42;`)
	if err != nil {
		t.Fatal(err)
	}
	if v, ok := file.Options["max_size"].(int64); !ok || v != 42 {
		t.Errorf("got %v %T", file.Options["max_size"], file.Options["max_size"])
	}
}

func TestFileOptionBool(t *testing.T) {
	file, err := parseSource(t, `syntax = "proto3"; option deprecated = true;`)
	if err != nil {
		t.Fatal(err)
	}
	if file.Options["deprecated"] != true {
		t.Errorf("got %v", file.Options["deprecated"])
	}
}
