package ast_test

import (
	"testing"

	"github.com/cafecito-games/gdproto/internal/ast"
)

func TestProtoFileZero(t *testing.T) {
	f := &ast.ProtoFile{}
	if f.Imports != nil || f.Messages != nil || f.Enums != nil {
		t.Fatal("zero-value collections should be nil (caller appends)")
	}
}

func TestPositionEmbedded(t *testing.T) {
	m := &ast.Message{Position: ast.Position{Line: 5, Column: 12}, Name: "Foo"}
	if m.Line != 5 || m.Column != 12 {
		t.Fatalf("position not accessible via embedding: got %d:%d", m.Line, m.Column)
	}
}

func TestReservedRangeSingle(t *testing.T) {
	r := ast.ReservedRange{Start: 7, End: 7}
	if r.Start != r.End {
		t.Fatal("single-number reserved range should have Start == End")
	}
}

func TestFieldDefaults(t *testing.T) {
	f := &ast.Field{}
	if f.Repeated || f.Optional || f.IsEnum {
		t.Fatal("zero-value Field should have all bool fields false")
	}
	if f.OneofParent != "" {
		t.Fatal("zero-value Field should have empty OneofParent")
	}
}
