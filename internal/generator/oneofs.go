package generator

import (
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
func generateOneofCaseGetter(oneof *ast.Oneof) gdast.Function {
	enumName := oneofEnumName(oneof.Name)
	return gdast.Function{
		Name:       "get_" + oneof.Name + "_case",
		ReturnType: enumName,
		Body: []gdast.Statement{
			gdast.Ret(gdast.V(oneofTrackingVar(oneof.Name))),
		},
	}
}
