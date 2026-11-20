package generator

import (
	"fmt"
	"strings"

	"github.com/cafecito-games/gdproto/internal/ast"
	"github.com/cafecito-games/gdproto/internal/gdast"
)

// oneofEnumName converts a snake_case oneof group name into the PascalCase
// enum type name with the "OneOf" suffix used by the generated tracking enum.
// For example, "contact" becomes "ContactOneOf" and "oneof_test" becomes
// "OneofTestOneOf".
func oneofEnumName(name string) string {
	parts := strings.Split(name, "_")
	var builder strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}
		builder.WriteString(strings.ToUpper(part[:1]))
		if len(part) > 1 {
			builder.WriteString(part[1:])
		}
	}
	builder.WriteString("OneOf")
	return builder.String()
}

// oneofEnumValue returns the SCREAMING_SNAKE_CASE enum value name for a oneof
// field. The result is unqualified; callers prepend the enum type as needed.
func oneofEnumValue(fieldName string) string {
	return strings.ToUpper(fieldName)
}

// oneofEnumQualified returns the fully qualified enum value reference, i.e.
// "<Group>OneOf.<FIELD>".
func oneofEnumQualified(oneofName, fieldName string) string {
	return oneofEnumName(oneofName) + "." + oneofEnumValue(fieldName)
}

// oneofTrackingVar returns the name of the private tracking field associated
// with a oneof group.
func oneofTrackingVar(oneofName string) string {
	return "_oneof_" + oneofName
}

// oneofTransitionHelperPreferredName returns the preferred private helper name
// for transitioning a oneof group to a new active case.
func oneofTransitionHelperPreferredName(oneofName string) string {
	return "_select_" + oneofName
}

// generateOneofTransitionHelper emits the shared private helper that resets all
// backing arms of a oneof group to their generated defaults when switching to a
// different case, while leaving state untouched on a same-case transition.
func (g *generator) generateOneofTransitionHelper(
	messageName string,
	oneof *ast.Oneof,
	helperName string,
) gdast.Function {
	enumName := oneofEnumName(oneof.Name)
	trackingVar := oneofTrackingVar(oneof.Name)

	body := []gdast.Statement{
		gdast.IfStatement{
			Condition: gdast.Eq(gdast.V(trackingVar), gdast.V("next_case")),
			Body:      []gdast.Statement{gdast.Ret(nil)},
		},
	}
	for _, field := range oneof.Fields {
		defaultExpression, ok := g.oneofArmResetDefault(messageName, oneof.Name, field)
		if !ok {
			continue
		}
		body = append(body, gdast.Assign(
			"_"+field.Name,
			gdast.RawExpression{Code: defaultExpression},
		))
	}
	body = append(body, gdast.Assign(trackingVar, gdast.V("next_case")))

	return gdast.Function{
		Name:       helperName,
		Parameters: []gdast.Parameter{{Name: "next_case", TypeHint: enumName}},
		ReturnType: "void",
		Body:       body,
	}
}

// oneofArmResetDefault validates that a oneof arm has a supported default
// expression derived from fieldDefault and records a fail-closed generator
// error otherwise.
func (g *generator) oneofArmResetDefault(messageName, groupName string, field *ast.Field) (string, bool) {
	if field.Repeated {
		g.recordError(fmt.Errorf(
			"message %q oneof %q field %q: repeated field has no supported oneof reset default",
			messageName, groupName, field.Name,
		))
		return "", false
	}
	_, isScalar := scalarDefaultMap[field.FieldType]
	resolution, resolved := g.fieldResolution(field)
	supported := isScalar || field.IsEnum || (resolved && resolution.kind == declarationMessage)
	if !supported {
		g.recordError(fmt.Errorf(
			"message %q oneof %q field %q: unsupported type %q has no reset default expression",
			messageName, groupName, field.Name, field.FieldType,
		))
		return "", false
	}
	defaultExpression := g.fieldDefault(field)
	if defaultExpression == "" {
		g.recordError(fmt.Errorf(
			"message %q oneof %q field %q: empty reset default expression for type %q",
			messageName, groupName, field.Name, field.FieldType,
		))
		return "", false
	}
	return defaultExpression, true
}

// generateOneofEnum builds the `enum <Group>OneOf { UNSET = 0, F1 = 1, ... }`
// definition that backs a oneof group's type-safe tracking field. The UNSET
// member is always emitted first with value 0; subsequent members follow the
// declaration order of the oneof's fields starting at 1.
func generateOneofEnum(oneof *ast.Oneof) gdast.EnumDefinition {
	zero := 0
	values := []gdast.EnumValue{{Name: "UNSET", Value: &zero}}
	for i, f := range oneof.Fields {
		number := i + 1
		values = append(values, gdast.EnumValue{
			Name:  oneofEnumValue(f.Name),
			Value: &number,
		})
	}
	return gdast.EnumDefinition{
		Name:   oneofEnumName(oneof.Name),
		Values: values,
	}
}

// generateOneofCaseGetter emits `func get_<group>_case() -> <Group>OneOf` that
// exposes the current oneof tracking enum value to callers.
func generateOneofCaseGetter(oneof *ast.Oneof, methodName string) gdast.Function {
	enumName := oneofEnumName(oneof.Name)
	return gdast.Function{
		Name:       methodName,
		ReturnType: enumName,
		Body: []gdast.Statement{
			gdast.Ret(gdast.V(oneofTrackingVar(oneof.Name))),
		},
	}
}
