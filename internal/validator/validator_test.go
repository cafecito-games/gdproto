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

func TestValidFieldNumbers(t *testing.T) {
	src := `syntax = "proto3"; message F { int32 a = 1; int32 b = 100; int32 c = 536870911; }`
	if errs := validate(t, src); len(errs) != 0 {
		t.Errorf("got %+v", errs)
	}
}

func TestDuplicateFieldNumber(t *testing.T) {
	src := `syntax = "proto3"; message F { int32 a = 1; int32 b = 1; }`
	errs := validate(t, src)
	if len(errs) != 1 || !strings.Contains(errs[0].Message, "Duplicate field number") {
		t.Errorf("got %+v", errs)
	}
}

func TestFieldNumberZero(t *testing.T) {
	src := `syntax = "proto3"; message F { int32 a = 0; }`
	errs := validate(t, src)
	if len(errs) != 1 || !strings.Contains(errs[0].Message, "out of valid range") {
		t.Errorf("got %+v", errs)
	}
}

func TestFieldNumberTooHigh(t *testing.T) {
	src := `syntax = "proto3"; message F { int32 a = 536870912; }`
	errs := validate(t, src)
	if len(errs) != 1 || !strings.Contains(errs[0].Message, "out of valid range") {
		t.Errorf("got %+v", errs)
	}
}

func TestFieldNumberInReservedRange(t *testing.T) {
	src := `syntax = "proto3"; message F { int32 a = 19000; }`
	errs := validate(t, src)
	if len(errs) != 1 || !strings.Contains(errs[0].Message, "in reserved range") {
		t.Errorf("got %+v", errs)
	}
}

func TestDuplicateFieldName(t *testing.T) {
	src := `syntax = "proto3"; message F { int32 a = 1; int32 a = 2; }`
	errs := validate(t, src)
	found := false
	for _, e := range errs {
		if strings.Contains(e.Message, "Duplicate field name") {
			found = true
		}
	}
	if !found {
		t.Errorf("got %+v", errs)
	}
}

func TestFieldNameKeyword(t *testing.T) {
	src := `syntax = "proto3"; message F { int32 Message = 1; }`
	errs := validate(t, src)
	found := false
	for _, e := range errs {
		if strings.Contains(e.Message, "reserved keyword") {
			found = true
		}
	}
	if !found {
		t.Errorf("got %+v", errs)
	}
}

func TestFieldConflictReservedNumber(t *testing.T) {
	src := `syntax = "proto3"; message F { reserved 5; int32 a = 5; }`
	errs := validate(t, src)
	found := false
	for _, e := range errs {
		if strings.Contains(e.Message, "is reserved") {
			found = true
		}
	}
	if !found {
		t.Errorf("got %+v", errs)
	}
}

func TestFieldConflictReservedRange(t *testing.T) {
	src := `syntax = "proto3"; message F { reserved 4 to 8; int32 a = 6; }`
	errs := validate(t, src)
	found := false
	for _, e := range errs {
		if strings.Contains(e.Message, "conflicts with reserved range") {
			found = true
		}
	}
	if !found {
		t.Errorf("got %+v", errs)
	}
}

func TestFieldConflictReservedName(t *testing.T) {
	src := `syntax = "proto3"; message F { reserved "foo"; int32 foo = 1; }`
	errs := validate(t, src)
	found := false
	for _, e := range errs {
		if strings.Contains(e.Message, "is reserved") {
			found = true
		}
	}
	if !found {
		t.Errorf("got %+v", errs)
	}
}

func TestMapValid(t *testing.T) {
	src := `syntax = "proto3"; message F { map<string, int32> m = 1; }`
	if errs := validate(t, src); len(errs) != 0 {
		t.Errorf("got %+v", errs)
	}
}

func TestMapInvalidKey(t *testing.T) {
	src := `syntax = "proto3"; message F { map<float, int32> m = 1; }`
	errs := validate(t, src)
	if len(errs) != 1 || !strings.Contains(errs[0].Message, "Invalid map key type") {
		t.Errorf("got %+v", errs)
	}
}
