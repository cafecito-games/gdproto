package gdast

// Lit creates a Literal node from any Go value. Strings, booleans, numbers,
// and nil are formatted using GDScript conventions when rendered.
func Lit(value any) Literal {
	return Literal{Value: value}
}

// V creates a Variable reference by name.
func V(name string) Variable {
	return Variable{Name: name}
}

// Call creates a CallExpr. The function may be a string (bare callee) or any
// Expression (for method chains and other computed callees).
func Call(function any, args ...Expression) CallExpr {
	return CallExpr{Function: function, Arguments: args}
}

// Attr creates a GetAttr node for dot-notation attribute access.
func Attr(object Expression, attribute string) GetAttr {
	return GetAttr{Object: object, Attribute: attribute}
}

// Var creates a VarDeclaration. Pass an empty string for typeHint to omit the
// type annotation, and nil for value to omit the initializer.
func Var(name, typeHint string, value Expression) VarDeclaration {
	return VarDeclaration{Name: name, TypeHint: typeHint, InitialValue: value}
}

// Const creates a constant VarDeclaration. Pass an empty string for typeHint
// to omit the type annotation.
func Const(name string, value Expression, typeHint string) VarDeclaration {
	return VarDeclaration{Name: name, TypeHint: typeHint, InitialValue: value, IsConst: true}
}

// Assign creates an Assignment. The target may be a string (auto-wrapped as a
// Variable) or any Expression. The optional operator defaults to "=" when not
// provided.
func Assign(target any, value Expression, operator ...string) Assignment {
	var targetExpression Expression
	switch t := target.(type) {
	case string:
		targetExpression = Variable{Name: t}
	case Expression:
		targetExpression = t
	}
	op := "="
	if len(operator) > 0 {
		op = operator[0]
	}
	return Assignment{Target: targetExpression, Value: value, Operator: op}
}

// Ret creates a ReturnStatement. Pass nil to render a bare `return`.
func Ret(value Expression) ReturnStatement {
	return ReturnStatement{Value: value}
}

// If creates an IfStatement without elif branches. Pass nil for elseBody to
// omit the else clause.
func If(condition Expression, body, elseBody []Statement) IfStatement {
	return IfStatement{Condition: condition, Body: body, ElseBody: elseBody}
}

// While creates a WhileStatement.
func While(condition Expression, body []Statement) WhileStatement {
	return WhileStatement{Condition: condition, Body: body}
}

// For creates a ForStatement. Pass an empty string for typeHint to omit the
// loop variable type annotation.
func For(variable string, iterable Expression, body []Statement, typeHint string) ForStatement {
	return ForStatement{Variable: variable, Iterable: iterable, Body: body, TypeHint: typeHint}
}

// Eq creates a `==` comparison.
func Eq(left, right Expression) BinaryOp { return BinaryOp{Left: left, Op: "==", Right: right} }

// Ne creates a `!=` comparison.
func Ne(left, right Expression) BinaryOp { return BinaryOp{Left: left, Op: "!=", Right: right} }

// Lt creates a `<` comparison.
func Lt(left, right Expression) BinaryOp { return BinaryOp{Left: left, Op: "<", Right: right} }

// Gt creates a `>` comparison.
func Gt(left, right Expression) BinaryOp { return BinaryOp{Left: left, Op: ">", Right: right} }

// Le creates a `<=` comparison.
func Le(left, right Expression) BinaryOp { return BinaryOp{Left: left, Op: "<=", Right: right} }

// Ge creates a `>=` comparison.
func Ge(left, right Expression) BinaryOp { return BinaryOp{Left: left, Op: ">=", Right: right} }

// And creates a logical `and` operation.
func And(left, right Expression) BinaryOp { return BinaryOp{Left: left, Op: "and", Right: right} }

// Or creates a logical `or` operation.
func Or(left, right Expression) BinaryOp { return BinaryOp{Left: left, Op: "or", Right: right} }

// Not creates a logical `not` unary operation.
func Not(operand Expression) UnaryOp { return UnaryOp{Op: "not", Operand: operand} }

// Add creates an addition (`+`) binary operation.
func Add(left, right Expression) BinaryOp { return BinaryOp{Left: left, Op: "+", Right: right} }

// Sub creates a subtraction (`-`) binary operation.
func Sub(left, right Expression) BinaryOp { return BinaryOp{Left: left, Op: "-", Right: right} }

// Mul creates a multiplication (`*`) binary operation.
func Mul(left, right Expression) BinaryOp { return BinaryOp{Left: left, Op: "*", Right: right} }

// Div creates a division (`/`) binary operation.
func Div(left, right Expression) BinaryOp { return BinaryOp{Left: left, Op: "/", Right: right} }
