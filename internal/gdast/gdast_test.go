package gdast

import (
	"math"
	"testing"
)

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

	_ Expression = Literal{}
	_ Expression = Variable{}
	_ Expression = BinaryOp{}
	_ Expression = UnaryOp{}
	_ Expression = CallExpr{}
	_ Expression = GetAttr{}
	_ Expression = Subscript{}
	_ Expression = Array{}
	_ Expression = Dictionary{}
	_ Expression = TypeCast{}
	_ Expression = TernaryOp{}

	_ Statement = DocString{}
	_ Statement = Comment{}
	_ Statement = DocumentationComment{}
	_ Statement = ExpressionStatement{}
	_ Statement = VarDeclaration{}
	_ Statement = Assignment{}
	_ Statement = ReturnStatement{}
)

func TestLiteralInt(t *testing.T) {
	if got := (Literal{Value: 42}).ToGDScript(0); got != "42" {
		t.Errorf("got %q", got)
	}
}

func TestLiteralFloat(t *testing.T) {
	if got := (Literal{Value: 3.14}).ToGDScript(0); got != "3.14" {
		t.Errorf("got %q", got)
	}
}

func TestLiteralFloatInf(t *testing.T) {
	if got := (Literal{Value: math.Inf(1)}).ToGDScript(0); got != "INF" {
		t.Errorf("got %q", got)
	}
	if got := (Literal{Value: math.Inf(-1)}).ToGDScript(0); got != "-INF" {
		t.Errorf("got %q", got)
	}
	if got := (Literal{Value: math.NaN()}).ToGDScript(0); got != "NAN" {
		t.Errorf("got %q", got)
	}
}

func TestLiteralBool(t *testing.T) {
	if got := (Literal{Value: true}).ToGDScript(0); got != "true" {
		t.Errorf("got %q", got)
	}
	if got := (Literal{Value: false}).ToGDScript(0); got != "false" {
		t.Errorf("got %q", got)
	}
}

func TestLiteralNil(t *testing.T) {
	if got := (Literal{Value: nil}).ToGDScript(0); got != "null" {
		t.Errorf("got %q", got)
	}
}

func TestLiteralStringDoubleQuoted(t *testing.T) {
	if got := (Literal{Value: "hello"}).ToGDScript(0); got != `"hello"` {
		t.Errorf("got %q", got)
	}
}

func TestLiteralStringWithDoubleQuoteOnly(t *testing.T) {
	if got := (Literal{Value: `say "hi"`}).ToGDScript(0); got != `'say "hi"'` {
		t.Errorf("got %q", got)
	}
}

func TestLiteralStringWithBothQuotes(t *testing.T) {
	if got := (Literal{Value: `it's "hot"`}).ToGDScript(0); got != `"it's \"hot\""` {
		t.Errorf("got %q", got)
	}
}

func TestVariable(t *testing.T) {
	if got := (Variable{Name: "x"}).ToGDScript(0); got != "x" {
		t.Errorf("got %q", got)
	}
}

func TestBinaryOp(t *testing.T) {
	b := BinaryOp{Left: Variable{Name: "a"}, Op: "+", Right: Variable{Name: "b"}}
	if got := b.ToGDScript(0); got != "a + b" {
		t.Errorf("got %q", got)
	}
}

func TestUnaryOpWord(t *testing.T) {
	u := UnaryOp{Op: "not", Operand: Variable{Name: "x"}}
	if got := u.ToGDScript(0); got != "not x" {
		t.Errorf("got %q", got)
	}
}

func TestUnaryOpSymbol(t *testing.T) {
	u := UnaryOp{Op: "-", Operand: Variable{Name: "x"}}
	if got := u.ToGDScript(0); got != "-x" {
		t.Errorf("got %q", got)
	}
}

func TestCallExprWithStringFunction(t *testing.T) {
	c := CallExpr{Function: "f", Arguments: []Expression{Variable{Name: "a"}, Variable{Name: "b"}}}
	if got := c.ToGDScript(0); got != "f(a, b)" {
		t.Errorf("got %q", got)
	}
}

func TestCallExprNoArgs(t *testing.T) {
	c := CallExpr{Function: "f"}
	if got := c.ToGDScript(0); got != "f()" {
		t.Errorf("got %q", got)
	}
}

func TestCallExprMethodCall(t *testing.T) {
	c := CallExpr{
		Function:  GetAttr{Object: Variable{Name: "obj"}, Attribute: "method"},
		Arguments: []Expression{Literal{Value: 1}},
	}
	if got := c.ToGDScript(0); got != "obj.method(1)" {
		t.Errorf("got %q", got)
	}
}

func TestGetAttr(t *testing.T) {
	g := GetAttr{Object: Variable{Name: "obj"}, Attribute: "x"}
	if got := g.ToGDScript(0); got != "obj.x" {
		t.Errorf("got %q", got)
	}
}

func TestSubscript(t *testing.T) {
	s := Subscript{Object: Variable{Name: "arr"}, Key: Literal{Value: 0}}
	if got := s.ToGDScript(0); got != "arr[0]" {
		t.Errorf("got %q", got)
	}
}

func TestArrayEmpty(t *testing.T) {
	if got := (Array{}).ToGDScript(0); got != "[]" {
		t.Errorf("got %q", got)
	}
}

func TestArrayWithElements(t *testing.T) {
	a := Array{Elements: []Expression{Literal{Value: 1}, Literal{Value: 2}, Literal{Value: 3}}}
	if got := a.ToGDScript(0); got != "[1, 2, 3]" {
		t.Errorf("got %q", got)
	}
}

func TestDictionaryEmpty(t *testing.T) {
	if got := (Dictionary{}).ToGDScript(0); got != "{}" {
		t.Errorf("got %q", got)
	}
}

func TestDictionaryWithPairs(t *testing.T) {
	d := Dictionary{Pairs: []DictPair{
		{Key: Literal{Value: "key"}, Value: Literal{Value: "value"}},
	}}
	if got := d.ToGDScript(0); got != `{"key": "value"}` {
		t.Errorf("got %q", got)
	}
}

func TestTypeCast(t *testing.T) {
	tc := TypeCast{Value: Variable{Name: "x"}, TypeName: "int"}
	if got := tc.ToGDScript(0); got != "x as int" {
		t.Errorf("got %q", got)
	}
}

func TestTernaryOp(t *testing.T) {
	tr := TernaryOp{
		Condition:  Variable{Name: "cond"},
		TrueValue:  Variable{Name: "true_val"},
		FalseValue: Variable{Name: "false_val"},
	}
	if got := tr.ToGDScript(0); got != "true_val if cond else false_val" {
		t.Errorf("got %q", got)
	}
}

func TestDocString(t *testing.T) {
	if got := (DocString{Text: "text"}).ToGDScript(0); got != `"""text"""` {
		t.Errorf("got %q", got)
	}
	if got := (DocString{Text: "text"}).ToGDScript(1); got != "\t"+`"""text"""` {
		t.Errorf("got %q", got)
	}
}

func TestCommentSingleLine(t *testing.T) {
	if got := (Comment{Text: "hello"}).ToGDScript(0); got != "# hello" {
		t.Errorf("got %q", got)
	}
	if got := (Comment{Text: "  hello  "}).ToGDScript(1); got != "\t# hello" {
		t.Errorf("got %q", got)
	}
}

func TestCommentMultiLine(t *testing.T) {
	c := Comment{Text: "line1\n\nline2"}
	want := "# line1\n#\n# line2"
	if got := c.ToGDScript(0); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestDocumentationCommentSingleLine(t *testing.T) {
	if got := (DocumentationComment{Text: "doc"}).ToGDScript(0); got != "## doc" {
		t.Errorf("got %q", got)
	}
}

func TestDocumentationCommentMultiLine(t *testing.T) {
	d := DocumentationComment{Text: "a\n\nb"}
	want := "## a\n##\n## b"
	if got := d.ToGDScript(0); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestExpressionStatement(t *testing.T) {
	es := ExpressionStatement{Expression: Variable{Name: "x"}}
	if got := es.ToGDScript(0); got != "x" {
		t.Errorf("got %q", got)
	}
	if got := es.ToGDScript(2); got != "\t\tx" {
		t.Errorf("got %q", got)
	}
}

func TestVarDeclarationBare(t *testing.T) {
	if got := (VarDeclaration{Name: "x"}).ToGDScript(0); got != "var x" {
		t.Errorf("got %q", got)
	}
}

func TestVarDeclarationTypeOnly(t *testing.T) {
	if got := (VarDeclaration{Name: "x", TypeHint: "int"}).ToGDScript(0); got != "var x: int" {
		t.Errorf("got %q", got)
	}
}

func TestVarDeclarationTypeAndValue(t *testing.T) {
	v := VarDeclaration{Name: "x", TypeHint: "int", InitialValue: Literal{Value: 5}}
	if got := v.ToGDScript(0); got != "var x: int = 5" {
		t.Errorf("got %q", got)
	}
}

func TestVarDeclarationInferred(t *testing.T) {
	v := VarDeclaration{Name: "x", InitialValue: Literal{Value: 5}}
	if got := v.ToGDScript(0); got != "var x := 5" {
		t.Errorf("got %q", got)
	}
}

func TestVarDeclarationConst(t *testing.T) {
	v := VarDeclaration{Name: "X", TypeHint: "int", InitialValue: Literal{Value: 5}, IsConst: true}
	if got := v.ToGDScript(0); got != "const X: int = 5" {
		t.Errorf("got %q", got)
	}
}

func TestAssignmentSimple(t *testing.T) {
	a := Assignment{Target: Variable{Name: "x"}, Value: Literal{Value: 5}}
	if got := a.ToGDScript(0); got != "x = 5" {
		t.Errorf("got %q", got)
	}
}

func TestAssignmentCompound(t *testing.T) {
	a := Assignment{Target: Variable{Name: "x"}, Value: Literal{Value: 1}, Operator: "+="}
	if got := a.ToGDScript(1); got != "\tx += 1" {
		t.Errorf("got %q", got)
	}
}

func TestReturnStatementBare(t *testing.T) {
	if got := (ReturnStatement{}).ToGDScript(1); got != "\treturn" {
		t.Errorf("got %q", got)
	}
}

func TestReturnStatementWithValue(t *testing.T) {
	r := ReturnStatement{Value: Literal{Value: 42}}
	if got := r.ToGDScript(1); got != "\treturn 42" {
		t.Errorf("got %q", got)
	}
}

func TestIfSimple(t *testing.T) {
	s := IfStatement{
		Condition: Variable{Name: "x"},
		Body:      []Statement{ReturnStatement{Value: Literal{Value: 1}}},
	}
	want := "if x:\n\treturn 1"
	if got := s.ToGDScript(0); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestIfWithElifAndElse(t *testing.T) {
	s := IfStatement{
		Condition: Variable{Name: "a"},
		Body:      []Statement{ReturnStatement{Value: Literal{Value: 1}}},
		ElifBranches: []ElifBranch{
			{Condition: Variable{Name: "b"}, Body: []Statement{ReturnStatement{Value: Literal{Value: 2}}}},
		},
		ElseBody: []Statement{ReturnStatement{Value: Literal{Value: 3}}},
	}
	want := "if a:\n\treturn 1\nelif b:\n\treturn 2\nelse:\n\treturn 3"
	if got := s.ToGDScript(0); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestWhileEmptyBodyEmitsPass(t *testing.T) {
	s := WhileStatement{Condition: Variable{Name: "running"}}
	want := "while running:\n\tpass"
	if got := s.ToGDScript(0); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestWhileWithBody(t *testing.T) {
	s := WhileStatement{
		Condition: Variable{Name: "running"},
		Body:      []Statement{BreakStatement{}},
	}
	want := "while running:\n\tbreak"
	if got := s.ToGDScript(0); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestForWithTypeHint(t *testing.T) {
	s := ForStatement{
		Variable: "i",
		Iterable: Variable{Name: "arr"},
		Body:     []Statement{ContinueStatement{}},
		TypeHint: "int",
	}
	want := "for var i: int in arr:\n\tcontinue"
	if got := s.ToGDScript(0); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestForWithoutTypeHint(t *testing.T) {
	s := ForStatement{
		Variable: "item",
		Iterable: Variable{Name: "items"},
	}
	want := "for item in items:\n\tpass"
	if got := s.ToGDScript(0); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestMatchWithWildcard(t *testing.T) {
	s := MatchStatement{
		Expression: Variable{Name: "v"},
		Cases: []MatchCase{
			{Pattern: Literal{Value: 1}, Body: []Statement{ReturnStatement{Value: Literal{Value: "one"}}}},
			{Pattern: "_", Body: []Statement{ReturnStatement{Value: Literal{Value: "other"}}}},
		},
	}
	want := "match v:\n\t1:\n\t\treturn \"one\"\n\t_:\n\t\treturn \"other\""
	if got := s.ToGDScript(0); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestMatchCaseWithInlineComment(t *testing.T) {
	c := MatchCase{
		Pattern: Literal{Value: 1},
		Body:    []Statement{PassStatement{}},
		Comment: "first",
	}
	want := "1:  # first\n\tpass"
	if got := c.ToGDScript(0); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
