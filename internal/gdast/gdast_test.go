package gdast

import "testing"

func TestGDScriptJoinsItemsWithNewline(t *testing.T) {
	s := GDScript{Items: []Node{
		PassStatement{},
		BreakStatement{},
		ContinueStatement{},
	}}
	want := "pass\nbreak\ncontinue"
	if got := s.ToGDScript(0); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestEmptyLine(t *testing.T) {
	if got := (EmptyLine{}).ToGDScript(0); got != "" {
		t.Errorf("got %q, want empty", got)
	}
	if got := (EmptyLine{}).ToGDScript(3); got != "" {
		t.Errorf("indented EmptyLine should still be empty, got %q", got)
	}
}

func TestPassStatementIndented(t *testing.T) {
	if got := (PassStatement{}).ToGDScript(2); got != "\t\tpass" {
		t.Errorf("got %q", got)
	}
	if got := (PassStatement{}).ToGDScript(0); got != "pass" {
		t.Errorf("got %q", got)
	}
}

func TestBreakStatementIndented(t *testing.T) {
	if got := (BreakStatement{}).ToGDScript(1); got != "\tbreak" {
		t.Errorf("got %q", got)
	}
}

func TestContinueStatementIndented(t *testing.T) {
	if got := (ContinueStatement{}).ToGDScript(1); got != "\tcontinue" {
		t.Errorf("got %q", got)
	}
}

func TestRawStatement(t *testing.T) {
	if got := (RawStatement{Code: "var x = 1"}).ToGDScript(0); got != "var x = 1" {
		t.Errorf("got %q", got)
	}
	if got := (RawStatement{Code: "var x = 1"}).ToGDScript(1); got != "\tvar x = 1" {
		t.Errorf("got %q", got)
	}
	multi := "if x:\n\treturn 1"
	want := "\tif x:\n\t\treturn 1"
	if got := (RawStatement{Code: multi}).ToGDScript(1); got != want {
		t.Errorf("multi-line: got %q, want %q", got, want)
	}
	multiWithBlank := "a\n\nb"
	wantBlank := "\ta\n\n\tb"
	if got := (RawStatement{Code: multiWithBlank}).ToGDScript(1); got != wantBlank {
		t.Errorf("blank-preserve: got %q, want %q", got, wantBlank)
	}
}

func TestRawExpression(t *testing.T) {
	if got := (RawExpression{Code: "1 + 2"}).ToGDScript(0); got != "1 + 2" {
		t.Errorf("got %q", got)
	}
	if got := (RawExpression{Code: "1 + 2"}).ToGDScript(2); got != "\t\t1 + 2" {
		t.Errorf("got %q", got)
	}
}

// Compile-time assertions that interfaces are satisfied.
var (
	_ Statement  = PassStatement{}
	_ Statement  = BreakStatement{}
	_ Statement  = ContinueStatement{}
	_ Statement  = EmptyLine{}
	_ Statement  = RawStatement{}
	_ Expression = RawExpression{}
	_ Node       = GDScript{}
)
