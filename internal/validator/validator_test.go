package validator_test

import (
	"strings"
	"testing"

	"github.com/cafecito-games/gogdproto/internal/lexer"
	"github.com/cafecito-games/gogdproto/internal/parser"
	"github.com/cafecito-games/gogdproto/internal/validator"
)

// validate parses and validates source. Returns the error list.
func validate(t *testing.T, src string) []validator.ValidationError {
	t.Helper()
	tokens, err := lexer.Tokenize(src, "test.proto")
	if err != nil {
		t.Fatalf("lex: %v", err)
	}
	file, err := parser.Parse(tokens, "test.proto")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return validator.Validate(file, "test.proto")
}

func TestValidationErrorFormat(t *testing.T) {
	e := &validator.ValidationError{File: "x.proto", Line: 5, Column: 10, Message: "boom"}
	want := "x.proto:5:10: error: boom"
	if e.Error() != want {
		t.Fatalf("got %q, want %q", e.Error(), want)
	}
}

func TestValidationErrorDefaultFile(t *testing.T) {
	e := &validator.ValidationError{Line: 1, Column: 1, Message: "oops"}
	if !strings.Contains(e.Error(), "<input>") {
		t.Fatalf("got %q", e.Error())
	}
}

func TestProto3Valid(t *testing.T) {
	errs := validate(t, `syntax = "proto3"; message Foo {}`)
	if len(errs) != 0 {
		t.Fatalf("got %d errors: %+v", len(errs), errs)
	}
}

func TestProto2Rejected(t *testing.T) {
	errs := validate(t, `syntax = "proto2"; message Foo {}`)
	if len(errs) != 1 {
		t.Fatalf("got %d errors", len(errs))
	}
	if !strings.Contains(errs[0].Message, "Unsupported syntax version") || !strings.Contains(errs[0].Message, "proto2") {
		t.Errorf("got %q", errs[0].Message)
	}
}
