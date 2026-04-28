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

func TestEnumDuplicateNumber(t *testing.T) {
	src := `syntax = "proto3"; enum E { A = 0; B = 0; }`
	errs := validate(t, src)
	if len(errs) != 1 || !strings.Contains(errs[0].Message, "Duplicate enum value number") {
		t.Errorf("got %+v", errs)
	}
}

func TestEnumAllowAlias(t *testing.T) {
	src := `syntax = "proto3";
enum E {
    option allow_alias = true;
    A = 0;
    B = 0;
}`
	errs := validate(t, src)
	if len(errs) != 0 {
		t.Errorf("got %+v", errs)
	}
}

func TestEnumDuplicateName(t *testing.T) {
	src := `syntax = "proto3"; enum E { A = 0; A = 1; }`
	errs := validate(t, src)
	found := false
	for _, e := range errs {
		if strings.Contains(e.Message, "Duplicate enum value name") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected duplicate-name error, got %+v", errs)
	}
}

func TestEnumFirstNotZero(t *testing.T) {
	src := `syntax = "proto3"; enum E { A = 1; B = 2; }`
	errs := validate(t, src)
	if len(errs) != 1 || !strings.Contains(errs[0].Message, "First enum value in proto3 must be zero") {
		t.Errorf("got %+v", errs)
	}
}

func TestEnumNameKeyword(t *testing.T) {
	src := `syntax = "proto3"; enum Message { A = 0; }`
	errs := validate(t, src)
	found := false
	for _, e := range errs {
		if strings.Contains(e.Message, "reserved keyword") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected reserved-keyword error, got %+v", errs)
	}
}
