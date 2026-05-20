package parser_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/cafecito-games/gdproto/internal/ast"
	"github.com/cafecito-games/gdproto/internal/lexer"
	"github.com/cafecito-games/gdproto/internal/parser"
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

func TestSimpleEnum(t *testing.T) {
	src := `syntax = "proto3";
enum ColorEnum {
    RED = 0;
    GREEN = 1;
    BLUE = 2;
}`
	file, err := parseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	if len(file.Enums) != 1 {
		t.Fatalf("got %d enums", len(file.Enums))
	}
	e := file.Enums[0]
	if e.Name != "ColorEnum" || len(e.Values) != 3 {
		t.Fatalf("got %+v", e)
	}
	wantNames := []string{"RED", "GREEN", "BLUE"}
	for i, n := range wantNames {
		if e.Values[i].Name != n || e.Values[i].Number != i {
			t.Errorf("value[%d] = %+v", i, e.Values[i])
		}
	}
}

func TestEnumNegative(t *testing.T) {
	src := `syntax = "proto3";
enum Status { ERROR = -1; UNKNOWN = 0; OK = 1; }`
	file, err := parseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	if file.Enums[0].Values[0].Number != -1 {
		t.Errorf("ERROR number = %d", file.Enums[0].Values[0].Number)
	}
}

func TestEnumWithOption(t *testing.T) {
	src := `syntax = "proto3";
enum E {
    option deprecated = true;
    A = 0;
}`
	file, err := parseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	if file.Enums[0].Options["deprecated"] != true {
		t.Errorf("got %v", file.Enums[0].Options)
	}
}

func TestEnumHexValue(t *testing.T) {
	src := `syntax = "proto3";
enum X { Y = 0x1F; }`
	file, err := parseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	if file.Enums[0].Values[0].Number != 31 {
		t.Errorf("got %d", file.Enums[0].Values[0].Number)
	}
}

func TestEmptyMessage(t *testing.T) {
	file, err := parseSource(t, `syntax = "proto3"; message Empty {}`)
	if err != nil {
		t.Fatal(err)
	}
	if len(file.Messages) != 1 || file.Messages[0].Name != "Empty" {
		t.Fatalf("got %+v", file.Messages)
	}
	if len(file.Messages[0].Fields) != 0 {
		t.Errorf("got %d fields, want 0", len(file.Messages[0].Fields))
	}
}

func TestMessageScalarFields(t *testing.T) {
	src := `syntax = "proto3";
message Person {
    string name = 1;
    int32 age = 2;
    bool active = 3;
}`
	file, err := parseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	m := file.Messages[0]
	if m.Name != "Person" || len(m.Fields) != 3 {
		t.Fatalf("got %+v", m)
	}
	cases := []struct {
		name, ftype string
		number      int
	}{
		{"name", "string", 1},
		{"age", "int32", 2},
		{"active", "bool", 3},
	}
	for i, c := range cases {
		f := m.Fields[i]
		if f.Name != c.name || f.FieldType != c.ftype || f.Number != c.number {
			t.Errorf("field[%d] = %+v, want %+v", i, f, c)
		}
	}
}

func TestAllScalarTypes(t *testing.T) {
	src := `syntax = "proto3";
message AllTypes {
    double f1 = 1;
    float f2 = 2;
    int32 f3 = 3;
    int64 f4 = 4;
    uint32 f5 = 5;
    uint64 f6 = 6;
    sint32 f7 = 7;
    sint64 f8 = 8;
    fixed32 f9 = 9;
    fixed64 f10 = 10;
    sfixed32 f11 = 11;
    sfixed64 f12 = 12;
    bool f13 = 13;
    string f14 = 14;
    bytes f15 = 15;
}`
	file, err := parseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"double", "float", "int32", "int64", "uint32", "uint64", "sint32", "sint64",
		"fixed32", "fixed64", "sfixed32", "sfixed64", "bool", "string", "bytes"}
	got := make([]string, 0, 15)
	for _, f := range file.Messages[0].Fields {
		got = append(got, f.FieldType)
	}
	if len(got) != 15 {
		t.Fatalf("got %d fields", len(got))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("field[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestMessageTypedField(t *testing.T) {
	src := `syntax = "proto3";
message Inner {}
message Outer { Inner inner = 1; }`
	file, err := parseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	if file.Messages[1].Fields[0].FieldType != "Inner" {
		t.Errorf("got %q", file.Messages[1].Fields[0].FieldType)
	}
}

func TestNestedMessage(t *testing.T) {
	src := `syntax = "proto3";
message Outer {
    message Inner { string value = 1; }
    Inner inner = 1;
}`
	file, err := parseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	outer := file.Messages[0]
	if len(outer.NestedMessages) != 1 || outer.NestedMessages[0].Name != "Inner" {
		t.Errorf("got %+v", outer.NestedMessages)
	}
}

func TestNestedEnumInMessage(t *testing.T) {
	src := `syntax = "proto3";
message Foo { enum Bar { V = 0; } }`
	file, err := parseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	if len(file.Messages[0].NestedEnums) != 1 || file.Messages[0].NestedEnums[0].Name != "Bar" {
		t.Errorf("got %+v", file.Messages[0].NestedEnums)
	}
}

func TestDecimalFieldNumber(t *testing.T) {
	src := `syntax = "proto3"; message F { int32 v = 123; }`
	file, err := parseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	if file.Messages[0].Fields[0].Number != 123 {
		t.Errorf("got %d", file.Messages[0].Fields[0].Number)
	}
}

func TestHexFieldNumber(t *testing.T) {
	src := `syntax = "proto3"; message F { int32 v = 0x1F; }`
	file, err := parseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	if file.Messages[0].Fields[0].Number != 31 {
		t.Errorf("got %d", file.Messages[0].Fields[0].Number)
	}
}

func TestRepeatedScalar(t *testing.T) {
	src := `syntax = "proto3"; message F { repeated int32 numbers = 1; }`
	file, err := parseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	f := file.Messages[0].Fields[0]
	if !f.Repeated || f.FieldType != "int32" {
		t.Errorf("got %+v", f)
	}
}

func TestRepeatedMessage(t *testing.T) {
	src := `syntax = "proto3"; message Item {} message C { repeated Item items = 1; }`
	file, err := parseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	f := file.Messages[1].Fields[0]
	if !f.Repeated || f.FieldType != "Item" {
		t.Errorf("got %+v", f)
	}
}

func TestOptionalScalar(t *testing.T) {
	src := `syntax = "proto3"; message F { optional int32 v = 1; }`
	file, err := parseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	if !file.Messages[0].Fields[0].Optional {
		t.Errorf("Optional should be true")
	}
}

func TestOptionalMessage(t *testing.T) {
	src := `syntax = "proto3"; message Item {} message C { optional Item item = 1; }`
	file, err := parseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	f := file.Messages[1].Fields[0]
	if !f.Optional || f.FieldType != "Item" {
		t.Errorf("got %+v", f)
	}
}

func TestMapScalarToScalar(t *testing.T) {
	src := `syntax = "proto3"; message F { map<string, int32> my_map = 1; }`
	file, err := parseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	mp := file.Messages[0].Maps[0]
	if mp.KeyType != "string" || mp.ValueType != "int32" || mp.Name != "my_map" || mp.Number != 1 {
		t.Errorf("got %+v", mp)
	}
}

func TestMapScalarToMessage(t *testing.T) {
	src := `syntax = "proto3"; message Value {} message F { map<int32, Value> my_map = 1; }`
	file, err := parseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	mp := file.Messages[1].Maps[0]
	if mp.KeyType != "int32" || mp.ValueType != "Value" {
		t.Errorf("got %+v", mp)
	}
}

func TestDottedMessageType(t *testing.T) {
	src := `syntax = "proto3";
message Outer { message Inner {} }
message F { Outer.Inner v = 1; }`
	file, err := parseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	if file.Messages[1].Fields[0].FieldType != "Outer.Inner" {
		t.Errorf("got %q", file.Messages[1].Fields[0].FieldType)
	}
}

func TestAbsoluteMessageType(t *testing.T) {
	src := `syntax = "proto3"; message F { .pkg.Bar v = 1; }`
	file, err := parseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	if file.Messages[0].Fields[0].FieldType != ".pkg.Bar" {
		t.Errorf("got %q", file.Messages[0].Fields[0].FieldType)
	}
}

func TestAbsoluteMapValueType(t *testing.T) {
	src := `syntax = "proto3"; message F { map<string, .pkg.Bar> values = 1; }`
	file, err := parseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	if file.Messages[0].Maps[0].ValueType != ".pkg.Bar" {
		t.Errorf("got %q", file.Messages[0].Maps[0].ValueType)
	}
}

func TestSimpleOneof(t *testing.T) {
	src := `syntax = "proto3";
message F {
    oneof test {
        string name = 1;
        int32 value = 2;
    }
}`
	file, err := parseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	o := file.Messages[0].Oneofs[0]
	if o.Name != "test" || len(o.Fields) != 2 {
		t.Fatalf("got %+v", o)
	}
	if o.Fields[0].Name != "name" || o.Fields[0].OneofParent != "test" {
		t.Errorf("got %+v", o.Fields[0])
	}
}

func TestOneofRepeatedError(t *testing.T) {
	src := `syntax = "proto3";
message F { oneof t { repeated string ns = 1; } }`
	_, err := parseSource(t, src)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "Oneof fields cannot be repeated") {
		t.Errorf("got %v", err)
	}
}

func TestReservedNumbers(t *testing.T) {
	src := `syntax = "proto3"; message F { reserved 2, 15, 9; }`
	file, err := parseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	r := file.Messages[0].Reserved[0]
	want := []ast.ReservedRange{{Start: 2, End: 2}, {Start: 15, End: 15}, {Start: 9, End: 9}}
	if len(r.Numbers) != len(want) {
		t.Fatalf("got %d numbers", len(r.Numbers))
	}
	for i, w := range want {
		if r.Numbers[i] != w {
			t.Errorf("number[%d] = %+v, want %+v", i, r.Numbers[i], w)
		}
	}
	if len(r.Names) != 0 {
		t.Errorf("names not empty: %v", r.Names)
	}
}

func TestReservedRanges(t *testing.T) {
	src := `syntax = "proto3"; message F { reserved 2, 9 to 11, 15; }`
	file, err := parseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	r := file.Messages[0].Reserved[0]
	want := []ast.ReservedRange{{Start: 2, End: 2}, {Start: 9, End: 11}, {Start: 15, End: 15}}
	for i, w := range want {
		if r.Numbers[i] != w {
			t.Errorf("range[%d] = %+v, want %+v", i, r.Numbers[i], w)
		}
	}
}

func TestReservedNames(t *testing.T) {
	src := `syntax = "proto3"; message F { reserved "foo", "bar"; }`
	file, err := parseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	r := file.Messages[0].Reserved[0]
	if len(r.Numbers) != 0 || len(r.Names) != 2 || r.Names[0] != "foo" || r.Names[1] != "bar" {
		t.Errorf("got %+v", r)
	}
}

func TestFieldPackedOption(t *testing.T) {
	src := `syntax = "proto3"; message F { repeated int32 numbers = 1 [packed = false]; }`
	file, err := parseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	f := file.Messages[0].Fields[0]
	if f.Options["packed"] != false {
		t.Errorf("got %v", f.Options)
	}
}

func TestFieldMultipleOptions(t *testing.T) {
	src := `syntax = "proto3"; message F { int32 v = 1 [packed = true, deprecated = true]; }`
	file, err := parseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	opts := file.Messages[0].Fields[0].Options
	if opts["packed"] != true || opts["deprecated"] != true {
		t.Errorf("got %v", opts)
	}
}

func TestMessageOption(t *testing.T) {
	src := `syntax = "proto3"; message F { option deprecated = true; }`
	file, err := parseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	if file.Messages[0].Options["deprecated"] != true {
		t.Errorf("got %v", file.Messages[0].Options)
	}
}

func TestCompleteExample(t *testing.T) {
	src := `syntax = "proto3";

import "other.proto";
import public "base.proto";

package com.example;

enum Status {
    UNKNOWN = 0;
    ACTIVE = 1;
    INACTIVE = 2;
}

message Person {
    string name = 1;
    int32 age = 2;
    Status status = 3;
    repeated string emails = 4;
    map<string, string> metadata = 5;

    message PhoneNumber {
        string number = 1;
        string type = 2;
    }

    repeated PhoneNumber phones = 6;

    oneof contact_preference {
        string email = 7;
        string phone = 8;
    }
}`
	file, err := parseSource(t, src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if file.Syntax != "proto3" {
		t.Errorf("syntax = %q", file.Syntax)
	}
	if len(file.Imports) != 2 {
		t.Errorf("imports = %d, want 2", len(file.Imports))
	}
	if file.Package != "com.example" {
		t.Errorf("package = %q", file.Package)
	}
	if len(file.Enums) != 1 || file.Enums[0].Name != "Status" || len(file.Enums[0].Values) != 3 {
		t.Errorf("enum: %+v", file.Enums)
	}
	if len(file.Messages) != 1 {
		t.Fatalf("messages = %d, want 1", len(file.Messages))
	}
	m := file.Messages[0]
	if m.Name != "Person" {
		t.Errorf("name = %q", m.Name)
	}
	if len(m.Fields) != 5 {
		t.Errorf("fields = %d, want 5", len(m.Fields))
	}
	if len(m.Maps) != 1 {
		t.Errorf("maps = %d, want 1", len(m.Maps))
	}
	if len(m.NestedMessages) != 1 || m.NestedMessages[0].Name != "PhoneNumber" {
		t.Errorf("nested = %+v", m.NestedMessages)
	}
	if len(m.Oneofs) != 1 || m.Oneofs[0].Name != "contact_preference" {
		t.Errorf("oneofs = %+v", m.Oneofs)
	}
}

func TestParenthesizedOptionName(t *testing.T) {
	src := `syntax = "proto3"; option (my.custom_option) = "value";`
	file, err := parseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	if file.Options["(my.custom_option)"] != "value" {
		t.Errorf("got %v", file.Options)
	}
}

func TestFileOptionString(t *testing.T) {
	file, err := parseSource(t, `syntax = "proto3"; option go_package = "example.com/foo";`)
	if err != nil {
		t.Fatal(err)
	}
	if file.Options["go_package"] != "example.com/foo" {
		t.Errorf("got %v", file.Options)
	}
}

func TestFileOptionFloat(t *testing.T) {
	file, err := parseSource(t, `syntax = "proto3"; option scaling = 1.5;`)
	if err != nil {
		t.Fatal(err)
	}
	if file.Options["scaling"] != 1.5 {
		t.Errorf("got %v", file.Options)
	}
}

func TestFileOptionFalse(t *testing.T) {
	file, err := parseSource(t, `syntax = "proto3"; option enabled = false;`)
	if err != nil {
		t.Fatal(err)
	}
	if file.Options["enabled"] != false {
		t.Errorf("got %v", file.Options)
	}
}

func TestEnumValueWithOptions(t *testing.T) {
	src := `syntax = "proto3"; enum E { UNKNOWN = 0 [deprecated = true]; ACTIVE = 1; }`
	file, err := parseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	v := file.Enums[0].Values[0]
	if v.Options["deprecated"] != true {
		t.Errorf("got %v", v.Options)
	}
}

func TestOneofWithOption(t *testing.T) {
	src := `syntax = "proto3"; message M { oneof choice { option deprecated = true; string a = 1; } }`
	file, err := parseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	if len(file.Messages[0].Oneofs) != 1 {
		t.Fatalf("oneofs = %+v", file.Messages[0].Oneofs)
	}
}

func TestOneofRepeatedRejected(t *testing.T) {
	src := `syntax = "proto3"; message M { oneof choice { repeated string a = 1; } }`
	_, err := parseSource(t, src)
	if err == nil {
		t.Fatal("expected error for repeated oneof field")
	}
}

func TestAbsoluteType(t *testing.T) {
	src := `syntax = "proto3"; message M { .foo.Bar baz = 1; }`
	file, err := parseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	if file.Messages[0].Fields[0].FieldType != ".foo.Bar" {
		t.Errorf("got %q", file.Messages[0].Fields[0].FieldType)
	}
}

func TestMapWithOptions(t *testing.T) {
	src := `syntax = "proto3"; message M { map<string, int32> m = 1 [deprecated = true]; }`
	file, err := parseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	if file.Messages[0].Maps[0].Options["deprecated"] != true {
		t.Errorf("got %v", file.Messages[0].Maps[0].Options)
	}
}

func TestReservedRange(t *testing.T) {
	src := `syntax = "proto3"; message M { reserved 9 to 11, 15; }`
	file, err := parseSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	r := file.Messages[0].Reserved[0]
	if len(r.Numbers) != 2 || r.Numbers[0].Start != 9 || r.Numbers[0].End != 11 || r.Numbers[1].Start != 15 || r.Numbers[1].End != 15 {
		t.Errorf("got %+v", r.Numbers)
	}
}

func TestParseErrorMissingSyntax(t *testing.T) {
	_, err := parseSource(t, `package foo;`)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseErrorBadFieldNumber(t *testing.T) {
	_, err := parseSource(t, `syntax = "proto3"; message M { string a = ; }`)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseErrorBadType(t *testing.T) {
	_, err := parseSource(t, `syntax = "proto3"; message M { = a = 1; }`)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseErrorUnexpectedTopLevel(t *testing.T) {
	_, err := parseSource(t, `syntax = "proto3"; foo bar;`)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseErrorOptionMissingValue(t *testing.T) {
	_, err := parseSource(t, `syntax = "proto3"; option foo = ;`)
	if err == nil {
		t.Fatal("expected error")
	}
}

// errorCases exercises many parser error paths for coverage.
func TestParserErrorCases(t *testing.T) {
	cases := []string{
		// missing semicolons / equals
		`syntax "proto3";`,
		`syntax = proto3;`,
		`syntax = "proto3"`,
		`syntax = "proto3"; import;`,
		`syntax = "proto3"; import "foo.proto"`,
		`syntax = "proto3"; package ;`,
		`syntax = "proto3"; package foo`,
		`syntax = "proto3"; option = 1;`,
		`syntax = "proto3"; option foo 1;`,
		`syntax = "proto3"; option foo = 1`,
		`syntax = "proto3"; option (foo = 1;`,
		`syntax = "proto3"; option (.foo) = 1;`,
		`syntax = "proto3"; enum;`,
		`syntax = "proto3"; enum E ;`,
		`syntax = "proto3"; enum E { = 0; }`,
		`syntax = "proto3"; enum E { A 0; }`,
		`syntax = "proto3"; enum E { A = ; }`,
		`syntax = "proto3"; enum E { A = 0 }`,
		`syntax = "proto3"; message;`,
		`syntax = "proto3"; message M ;`,
		`syntax = "proto3"; message M { string = 1; }`,
		`syntax = "proto3"; message M { string a 1; }`,
		`syntax = "proto3"; message M { string a = 1 }`,
		`syntax = "proto3"; message M { string a = 1 [deprecated true]; }`,
		`syntax = "proto3"; message M { string a = 1 [deprecated = true; }`,
		`syntax = "proto3"; message M { map<string int32> m = 1; }`,
		`syntax = "proto3"; message M { map string, int32> m = 1; }`,
		`syntax = "proto3"; message M { map<string, int32 m = 1; }`,
		`syntax = "proto3"; message M { map<string, int32> = 1; }`,
		`syntax = "proto3"; message M { map<string, int32> m 1; }`,
		`syntax = "proto3"; message M { map<string, int32> m = ; }`,
		`syntax = "proto3"; message M { map<string, int32> m = 1 }`,
		`syntax = "proto3"; message M { oneof; }`,
		`syntax = "proto3"; message M { oneof o ; }`,
		`syntax = "proto3"; message M { reserved; }`,
		`syntax = "proto3"; message M { reserved 1 to ; }`,
		`syntax = "proto3"; message M { reserved 1 to 5 }`,
		`syntax = "proto3"; message M { reserved "a" }`,
		`syntax = "proto3"; package foo.;`,
	}
	for _, src := range cases {
		if _, err := parseSource(t, src); err == nil {
			t.Errorf("expected error for: %q", src)
		}
	}
}
