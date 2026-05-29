package generator

import (
	"fmt"
	"strings"

	"github.com/cafecito-games/gdproto/internal/ast"
	"github.com/cafecito-games/gdproto/internal/gdast"
)

// generateToText emits the `to_text` method that serializes a message instance
// into the protobuf text format produced by Python's reference generator. The
// body is built as a single RawStatement because the per-field templates are
// highly text-driven and byte-identity to the reference output is the primary
// correctness criterion.
func (g *generator) generateToText(m *ast.Message) gdast.Function {
	if isEmptyMessage(m) {
		return gdast.Function{
			Name:       "to_text",
			Parameters: []gdast.Parameter{{Name: "_indent_level", TypeHint: "int", Default: gdast.Lit(0)}},
			ReturnType: "String",
			Body:       []gdast.Statement{gdast.RawStatement{Code: `"""Serialize message to protobuf text format."""` + "\n" + `return ""`}},
		}
	}

	var b strings.Builder
	b.WriteString(`"""Serialize message to protobuf text format."""` + "\n")
	b.WriteString("var result: String = \"\"\n")
	hasBody := len(m.Fields) > 0 || len(m.Oneofs) > 0 || len(m.Maps) > 0
	if hasBody {
		b.WriteString("var indent: String = \"\\t\".repeat(indent_level)\n")
	}

	for _, f := range m.Fields {
		b.WriteString("\n")
		b.WriteString(g.toTextFieldBlock(f))
	}
	for _, oneof := range m.Oneofs {
		b.WriteString("\n")
		b.WriteString(g.toTextOneofBlock(oneof))
	}
	for _, mf := range m.Maps {
		b.WriteString("\n")
		b.WriteString(g.toTextMapBlock(mf))
	}

	b.WriteString("\n")
	b.WriteString("return result")

	return gdast.Function{
		Name:       "to_text",
		Parameters: []gdast.Parameter{{Name: "indent_level", TypeHint: "int", Default: gdast.Lit(0)}},
		ReturnType: "String",
		Body:       []gdast.Statement{gdast.RawStatement{Code: b.String()}},
	}
}

// toTextFieldBlock renders the section that emits a single regular field. The
// guard, formatting expression, and value rendering vary by field type.
func (g *generator) toTextFieldBlock(f *ast.Field) string {
	fieldVar := "_" + f.Name

	if f.Repeated {
		return g.toTextRepeatedField(f, fieldVar)
	}

	switch {
	case f.FieldType == "string":
		return fmt.Sprintf("# Field %s\nif %s != \"\":\n\tresult += indent + \"%s: \\\"\" + ProtoCoreUtils.escape_string_text_format(%s) + \"\\\"\\n\"\n",
			f.Name, fieldVar, f.Name, fieldVar)
	case f.FieldType == "bytes":
		return fmt.Sprintf("# Field %s\nif %s.size() > 0:\n\tresult += indent + \"%s: \\\"\" + ProtoCoreUtils.escape_bytes_text_format(%s) + \"\\\"\\n\"\n",
			f.Name, fieldVar, f.Name, fieldVar)
	case f.FieldType == "bool":
		return fmt.Sprintf("# Field %s\nif %s:\n\tresult += indent + \"%s: \" + str(%s).to_lower() + \"\\n\"\n",
			f.Name, fieldVar, f.Name, fieldVar)
	case f.FieldType == "float" || f.FieldType == "double":
		return toTextFloatField(f.Name, fieldVar)
	case g.isEnumField(f):
		return fmt.Sprintf("# Field %s\nif %s != 0:\n\tresult += indent + \"%s: \" + _get_enum_name_%s(%s) + \"\\n\"\n",
			f.Name, fieldVar, f.Name, f.Name, fieldVar)
	case isMessageType(f):
		return fmt.Sprintf("# Field %s\nif %s != null:\n\tresult += indent + \"%s {\\n\"\n\tresult += %s.to_text(indent_level + 1)\n\tresult += indent + \"}\\n\"\n",
			f.Name, fieldVar, f.Name, fieldVar)
	default:
		// Numeric scalar (int32, int64, uint*, sint*, fixed*, sfixed*).
		return fmt.Sprintf("# Field %s\nif %s != 0:\n\tresult += indent + \"%s: \" + str(%s) + \"\\n\"\n",
			f.Name, fieldVar, f.Name, fieldVar)
	}
}

// toTextRepeatedField emits the for-loop that serializes a repeated field.
func (g *generator) toTextRepeatedField(f *ast.Field, fieldVar string) string {
	var b strings.Builder
	b.WriteString("# Repeated field " + f.Name + "\n")
	b.WriteString("for item in " + fieldVar + ":\n")
	switch {
	case f.FieldType == "string":
		b.WriteString("\tresult += indent + \"" + f.Name + ": \\\"\" + ProtoCoreUtils.escape_string_text_format(item) + \"\\\"\\n\"\n")
	case f.FieldType == "bytes":
		b.WriteString("\tresult += indent + \"" + f.Name + ": \\\"\" + ProtoCoreUtils.escape_bytes_text_format(item) + \"\\\"\\n\"\n")
	case f.FieldType == "bool":
		b.WriteString("\tresult += indent + \"" + f.Name + ": \" + str(item).to_lower() + \"\\n\"\n")
	case f.FieldType == "float" || f.FieldType == "double":
		b.WriteString(toTextFloatRepeatedItem(f.Name))
	case g.isEnumField(f):
		b.WriteString("\tresult += indent + \"" + f.Name + ": \" + _get_enum_name_" + f.Name + "(item) + \"\\n\"\n")
	case isMessageType(f):
		b.WriteString("\tresult += indent + \"" + f.Name + " {\\n\"\n")
		b.WriteString("\tresult += item.to_text(indent_level + 1)\n")
		b.WriteString("\tresult += indent + \"}\\n\"\n")
	default:
		b.WriteString("\tresult += indent + \"" + f.Name + ": \" + str(item) + \"\\n\"\n")
	}
	return b.String()
}

// toTextOneofBlock renders the `match _oneof_<group>:` block that serializes
// whichever member of the oneof is currently set.
func (g *generator) toTextOneofBlock(oneof *ast.Oneof) string {
	var b strings.Builder
	b.WriteString("# Oneof group: " + oneof.Name + "\n")
	b.WriteString("match " + oneofTrackingVar(oneof.Name) + ":\n")
	for _, f := range oneof.Fields {
		b.WriteString("\t" + oneofEnumQualified(oneof.Name, f.Name) + ":\n")
		fieldVar := "_" + f.Name
		switch {
		case f.FieldType == "string":
			b.WriteString("\t\tresult += indent + \"" + f.Name + ": \\\"\" + ProtoCoreUtils.escape_string_text_format(" + fieldVar + ") + \"\\\"\\n\"\n")
		case f.FieldType == "bytes":
			b.WriteString("\t\tresult += indent + \"" + f.Name + ": \\\"\" + ProtoCoreUtils.escape_bytes_text_format(" + fieldVar + ") + \"\\\"\\n\"\n")
		case f.FieldType == "bool":
			b.WriteString("\t\tresult += indent + \"" + f.Name + ": \" + str(" + fieldVar + ").to_lower() + \"\\n\"\n")
		case f.FieldType == "float" || f.FieldType == "double":
			b.WriteString(indentLines(toTextFloatField(f.Name, fieldVar), "\t\t"))
		case g.isEnumField(f):
			b.WriteString("\t\tresult += indent + \"" + f.Name + ": \" + _get_enum_name_" + f.Name + "(" + fieldVar + ") + \"\\n\"\n")
		case isMessageType(f):
			b.WriteString("\t\tresult += indent + \"" + f.Name + " {\\n\"\n")
			b.WriteString("\t\tresult += " + fieldVar + ".to_text(indent_level + 1)\n")
			b.WriteString("\t\tresult += indent + \"}\\n\"\n")
		default:
			b.WriteString("\t\tresult += indent + \"" + f.Name + ": \" + str(" + fieldVar + ") + \"\\n\"\n")
		}
	}
	return b.String()
}

// toTextMapBlock renders the for-loop that serializes a map field as a series
// of `name { key: ..., value: ... }` entries.
func (g *generator) toTextMapBlock(mf *ast.MapField) string {
	fieldVar := "_" + mf.Name
	var b strings.Builder
	b.WriteString("# Map field " + mf.Name + "\n")
	b.WriteString("for key in " + fieldVar + ":\n")
	b.WriteString("\tvar value: " + g.renderedMapValueType(mf) + " = " + fieldVar + "[key]\n")
	b.WriteString("\tresult += indent + \"" + mf.Name + " {\\n\"\n")
	b.WriteString("\tvar inner_indent: String = \"\\t\".repeat(indent_level + 1)\n")
	b.WriteString("\n")
	b.WriteString("\tresult += inner_indent + " + toTextMapEntryRender("key", mf.KeyType) + "\n")
	b.WriteString("\tresult += inner_indent + " + toTextMapEntryRender("value", mf.ValueType) + "\n")
	b.WriteString("\n")
	b.WriteString("\tresult += indent + \"}\\n\"\n")
	return b.String()
}

// toTextMapEntryRender returns the GDScript expression that produces the
// rendered text-format string for a single map entry field (key or value).
func toTextMapEntryRender(varName, protoType string) string {
	label := varName // "key" or "value"
	switch protoType {
	case "string":
		return "\"" + label + ": \\\"\" + ProtoCoreUtils.escape_string_text_format(" + varName + ") + \"\\\"\\n\""
	case "bytes":
		return "\"" + label + ": \\\"\" + ProtoCoreUtils.escape_bytes_text_format(" + varName + ") + \"\\\"\\n\""
	case "bool":
		return "\"" + label + ": \" + str(" + varName + ").to_lower() + \"\\n\""
	default:
		return "\"" + label + ": \" + str(" + varName + ") + \"\\n\""
	}
}

// toTextFloatField renders the inf/nan-aware text encoding for a single float
// or double field.
func toTextFloatField(name, fieldVar string) string {
	return fmt.Sprintf(`# Field %s
if %s != 0.0:
	if is_inf(%s):
		result += indent + "%s: " + ("inf" if %s > 0 else "-inf") + "\n"
	elif is_nan(%s):
		result += indent + "%s: nan\n"
	else:
		result += indent + "%s: " + str(%s) + "\n"
`, name, fieldVar, fieldVar, name, fieldVar, fieldVar, name, name, fieldVar)
}

// toTextFloatRepeatedItem renders the per-item body of a repeated float field's
// for-loop with inf/nan handling.
func toTextFloatRepeatedItem(name string) string {
	return fmt.Sprintf(`	if is_inf(item):
		result += indent + "%s: " + ("inf" if item > 0 else "-inf") + "\n"
	elif is_nan(item):
		result += indent + "%s: nan\n"
	else:
		result += indent + "%s: " + str(item) + "\n"
`, name, name, name)
}

// indentLines prepends prefix to every non-empty line of s.
func indentLines(s, prefix string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if line == "" {
			continue
		}
		lines[i] = prefix + line
	}
	return strings.Join(lines, "\n")
}
