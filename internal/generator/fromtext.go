package generator

import (
	"strings"

	"github.com/cafecito-games/gogdproto/internal/ast"
	"github.com/cafecito-games/gogdproto/internal/gdast"
)

// generateFromText emits the `from_text` method that parses the protobuf text
// format produced by `to_text` and populates the message's fields. Each field
// is dispatched in a `match` block keyed on its declared name; unknown fields
// are skipped (with brace-balanced traversal of nested messages).
//
// The body is assembled as a single gdast.RawStatement because the per-field
// snippets are highly templated and byte-identity to the upstream Python
// generator is the primary correctness criterion for this method.
func (g *generator) generateFromText(m *ast.Message) gdast.Function {
	var b strings.Builder
	b.WriteString(`"""Deserialize message from protobuf text format."""` + "\n")
	b.WriteString("var pos: int = 0\n")
	b.WriteString("\n")
	b.WriteString("while pos < text.length():\n")
	b.WriteString("\t# Skip whitespace and comments\n")
	b.WriteString("\tpos = ProtoCoreUtils.skip_whitespace(text, pos)\n")
	b.WriteString("\tif pos >= text.length():\n")
	b.WriteString("\t\tbreak\n")
	b.WriteString("\n")
	b.WriteString("\t# Parse field name\n")
	b.WriteString("\tvar name_result := ProtoCoreUtils.parse_identifier(text, pos)\n")
	b.WriteString("\tif \"error\" in name_result:\n")
	b.WriteString("\t\tpush_error(name_result[\"error\"])\n")
	b.WriteString("\t\treturn ProtoCoreUtils.ProtobufError.UNDEFINED_STATE\n")
	b.WriteString("\tvar field_name: String = name_result[\"value\"]\n")
	b.WriteString("\tpos = name_result[\"pos\"]\n")
	b.WriteString("\n")
	b.WriteString("\t# Skip whitespace before colon\n")
	b.WriteString("\tpos = ProtoCoreUtils.skip_whitespace(text, pos)\n")
	b.WriteString("\n")
	b.WriteString("\t# Match field name and parse value\n")
	b.WriteString("\tmatch field_name:\n")

	// Field cases (declaration order: regular fields, oneof fields, map fields).
	for _, f := range m.Fields {
		b.WriteString(g.fromTextFieldCase(f, ""))
	}
	for _, oneof := range m.Oneofs {
		for _, f := range oneof.Fields {
			b.WriteString(g.fromTextFieldCase(f, oneof.Name))
		}
	}
	for _, mf := range m.Maps {
		b.WriteString(fromTextMapCase(mf))
	}

	// Default unknown-field case.
	b.WriteString("\t\t_:\n")
	b.WriteString("\t\t\t# Unknown field - skip it\n")
	b.WriteString("\t\t\tpush_warning(\"Unknown field: \" + field_name)\n")
	b.WriteString("\t\t\t# Try to skip the value\n")
	b.WriteString("\t\t\tpos = ProtoCoreUtils.skip_whitespace(text, pos)\n")
	b.WriteString("\t\t\tif pos < text.length() and text[pos] == \":\":\n")
	b.WriteString("\t\t\t\tpos = pos + 1\n")
	b.WriteString("\t\t\t\tpos = ProtoCoreUtils.skip_whitespace(text, pos)\n")
	b.WriteString("\t\t\t\tif pos < text.length() and text[pos] == \"{\":\n")
	b.WriteString("\t\t\t\t\t# Skip message body\n")
	b.WriteString("\t\t\t\t\tvar depth := 1\n")
	b.WriteString("\t\t\t\t\tpos = pos + 1\n")
	b.WriteString("\t\t\t\t\twhile pos < text.length() and depth > 0:\n")
	b.WriteString("\t\t\t\t\t\tif text[pos] == \"{\":\n")
	b.WriteString("\t\t\t\t\t\t\tdepth = depth + 1\n")
	b.WriteString("\t\t\t\t\t\telif text[pos] == \"}\":\n")
	b.WriteString("\t\t\t\t\t\t\tdepth = depth - 1\n")
	b.WriteString("\t\t\t\t\t\tpos = pos + 1\n")
	b.WriteString("\t\t\t\telse:\n")
	b.WriteString("\t\t\t\t\t# Skip to next field\n")
	b.WriteString("\t\t\t\t\twhile pos < text.length() and text[pos] not in [\"\\n\", \"#\"]:\n")
	b.WriteString("\t\t\t\t\t\tpos = pos + 1\n")
	b.WriteString("\n")
	b.WriteString("return ProtoCoreUtils.ProtobufError.NO_ERRORS")

	return gdast.Function{
		Name:       "from_text",
		Parameters: []gdast.Parameter{{Name: "text", TypeHint: "String"}},
		ReturnType: "ProtoCoreUtils.ProtobufError",
		Body:       []gdast.Statement{gdast.RawStatement{Code: b.String()}},
	}
}

// fromTextFieldCase emits the `match` case for a single field. Indentation is
// relative to the function body root (two tabs lead each line, since the case
// body lives inside `match` inside `while`).
func (g *generator) fromTextFieldCase(f *ast.Field, oneofGroup string) string {
	var b strings.Builder
	b.WriteString("\t\t\"" + f.Name + "\":\n")
	b.WriteString("\t\t\t# Parse colon\n")
	b.WriteString("\t\t\tif pos < text.length() and text[pos] == \":\":\n")
	b.WriteString("\t\t\t\tpos = pos + 1\n")
	b.WriteString("\t\t\tpos = ProtoCoreUtils.skip_whitespace(text, pos)\n")
	b.WriteString("\n")

	switch {
	case f.Repeated && isMessageType(f):
		b.WriteString(g.fromTextRepeatedMessageBody(f))
	case f.Repeated && f.FieldType == "string":
		b.WriteString(fromTextStringBody(f, oneofGroup, true))
	case f.Repeated:
		// Generic repeated scalar fallback (numbers).
		b.WriteString(fromTextScalarBody(f, oneofGroup, true))
	case f.FieldType == "string":
		b.WriteString(fromTextStringBody(f, oneofGroup, false))
	case f.FieldType == "float" || f.FieldType == "double":
		b.WriteString(fromTextFloatBody(f, oneofGroup))
	case f.FieldType == "bool":
		b.WriteString(fromTextBoolBody(f, oneofGroup))
	case isEnumType(f):
		b.WriteString(fromTextEnumBody(f, oneofGroup))
	case isMessageType(f):
		b.WriteString(g.fromTextMessageBody(f, oneofGroup))
	default:
		// Integer-like scalars: int32, int64, uint*, sint*, fixed*.
		b.WriteString(fromTextScalarBody(f, oneofGroup, false))
	}
	return b.String()
}

func isEnumType(f *ast.Field) bool {
	return f.IsEnum
}

func isMessageType(f *ast.Field) bool {
	if _, ok := scalarTypeMap[f.FieldType]; ok {
		return false
	}
	return !f.IsEnum
}

func oneofAssignment(oneofGroup, fieldName string) string {
	if oneofGroup == "" {
		return ""
	}
	return "\t\t\t" + oneofTrackingVar(oneofGroup) + " = " + oneofEnumQualified(oneofGroup, fieldName) + "\n"
}

func fromTextStringBody(f *ast.Field, oneofGroup string, repeated bool) string {
	var b strings.Builder
	b.WriteString("\t\t\tvar str_result := ProtoCoreUtils.parse_string_literal(text, pos)\n")
	b.WriteString("\t\t\tif \"error\" in str_result:\n")
	b.WriteString("\t\t\t\treturn ProtoCoreUtils.ProtobufError.UNDEFINED_STATE\n")
	if repeated {
		b.WriteString("\t\t\t_" + f.Name + ".append(str_result[\"value\"])\n")
	} else {
		b.WriteString("\t\t\t_" + f.Name + " = str_result[\"value\"]\n")
	}
	b.WriteString(oneofAssignment(oneofGroup, f.Name))
	b.WriteString("\t\t\tpos = str_result[\"pos\"]\n")
	return b.String()
}

func fromTextScalarBody(f *ast.Field, oneofGroup string, repeated bool) string {
	var b strings.Builder
	b.WriteString("\t\t\tvar num_result := ProtoCoreUtils.parse_number(text, pos)\n")
	b.WriteString("\t\t\tif \"error\" in num_result:\n")
	b.WriteString("\t\t\t\treturn ProtoCoreUtils.ProtobufError.UNDEFINED_STATE\n")
	if repeated {
		b.WriteString("\t\t\t_" + f.Name + ".append(int(num_result[\"value\"]))\n")
	} else {
		b.WriteString("\t\t\t_" + f.Name + " = int(num_result[\"value\"])\n")
	}
	b.WriteString(oneofAssignment(oneofGroup, f.Name))
	b.WriteString("\t\t\tpos = num_result[\"pos\"]\n")
	return b.String()
}

func fromTextFloatBody(f *ast.Field, oneofGroup string) string {
	var b strings.Builder
	b.WriteString("\t\t\tvar float_result: Dictionary\n")
	b.WriteString("\t\t\t# Check for special values or identifiers\n")
	b.WriteString("\t\t\tif pos < text.length() and text[pos] in [\"i\", \"n\", \"-\", \"+\"] or not text[pos].is_valid_int():\n")
	b.WriteString("\t\t\t\tvar id_result := ProtoCoreUtils.parse_identifier(text, pos)\n")
	b.WriteString("\t\t\t\tif \"value\" in id_result:\n")
	b.WriteString("\t\t\t\t\tmatch id_result[\"value\"]:\n")
	b.WriteString("\t\t\t\t\t\t\"inf\":\n")
	b.WriteString("\t\t\t\t\t\t\tfloat_result = {\"value\": INF, \"pos\": id_result[\"pos\"]}\n")
	b.WriteString("\t\t\t\t\t\t\"nan\":\n")
	b.WriteString("\t\t\t\t\t\t\tfloat_result = {\"value\": NAN, \"pos\": id_result[\"pos\"]}\n")
	b.WriteString("\t\t\t\t\t\t_:\n")
	b.WriteString("\t\t\t\t\t\t\tfloat_result = ProtoCoreUtils.parse_number(text, pos)\n")
	b.WriteString("\t\t\t\telse:\n")
	b.WriteString("\t\t\t\t\tfloat_result = ProtoCoreUtils.parse_number(text, pos)\n")
	b.WriteString("\t\t\telse:\n")
	b.WriteString("\t\t\t\tfloat_result = ProtoCoreUtils.parse_number(text, pos)\n")
	b.WriteString("\t\t\tif \"error\" in float_result:\n")
	b.WriteString("\t\t\t\treturn ProtoCoreUtils.ProtobufError.UNDEFINED_STATE\n")
	b.WriteString("\t\t\t_" + f.Name + " = float(float_result[\"value\"])\n")
	b.WriteString(oneofAssignment(oneofGroup, f.Name))
	b.WriteString("\t\t\tpos = float_result[\"pos\"]\n")
	return b.String()
}

func fromTextBoolBody(f *ast.Field, oneofGroup string) string {
	var b strings.Builder
	b.WriteString("\t\t\tvar id_result := ProtoCoreUtils.parse_identifier(text, pos)\n")
	b.WriteString("\t\t\tif \"error\" in id_result:\n")
	b.WriteString("\t\t\t\treturn ProtoCoreUtils.ProtobufError.UNDEFINED_STATE\n")
	b.WriteString("\t\t\t_" + f.Name + " = id_result[\"value\"] == \"true\"\n")
	b.WriteString(oneofAssignment(oneofGroup, f.Name))
	b.WriteString("\t\t\tpos = id_result[\"pos\"]\n")
	return b.String()
}

func fromTextEnumBody(f *ast.Field, oneofGroup string) string {
	var b strings.Builder
	b.WriteString("\t\t\t# Parse enum value (name or number)\n")
	b.WriteString("\t\t\tvar enum_result: Dictionary\n")
	b.WriteString("\t\t\tif pos < text.length() and not text[pos].is_valid_int() and text[pos] != \"-\":\n")
	b.WriteString("\t\t\t\t# Parse as identifier (enum name)\n")
	b.WriteString("\t\t\t\tenum_result = ProtoCoreUtils.parse_identifier(text, pos)\n")
	b.WriteString("\t\t\t\tif \"error\" not in enum_result:\n")
	b.WriteString("\t\t\t\t\tvar enum_name: String = enum_result[\"value\"]\n")
	b.WriteString("\t\t\t\t\tvar enum_value: int = _parse_enum_value_" + f.Name + "(enum_name)\n")
	b.WriteString("\t\t\t\t\t_" + f.Name + " = enum_value\n")
	b.WriteString(oneofAssignmentExtraIndent(oneofGroup, f.Name, "\t\t\t\t\t"))
	b.WriteString("\t\t\t\t\tpos = enum_result[\"pos\"]\n")
	b.WriteString("\t\t\telse:\n")
	b.WriteString("\t\t\t\t# Parse as number\n")
	b.WriteString("\t\t\t\tenum_result = ProtoCoreUtils.parse_number(text, pos)\n")
	b.WriteString("\t\t\t\tif \"error\" in enum_result:\n")
	b.WriteString("\t\t\t\t\treturn ProtoCoreUtils.ProtobufError.UNDEFINED_STATE\n")
	b.WriteString("\t\t\t\t_" + f.Name + " = int(enum_result[\"value\"])\n")
	b.WriteString(oneofAssignmentExtraIndent(oneofGroup, f.Name, "\t\t\t\t"))
	b.WriteString("\t\t\t\tpos = enum_result[\"pos\"]\n")
	return b.String()
}

func oneofAssignmentExtraIndent(oneofGroup, fieldName, indent string) string {
	if oneofGroup == "" {
		return ""
	}
	return indent + oneofTrackingVar(oneofGroup) + " = " + oneofEnumQualified(oneofGroup, fieldName) + "\n"
}

func (g *generator) fromTextMessageBody(f *ast.Field, oneofGroup string) string {
	messageType := g.renderedFieldType(f)
	var b strings.Builder
	b.WriteString("\t\t\t# Parse message\n")
	b.WriteString("\t\t\tif pos < text.length() and text[pos] == \"{\":\n")
	b.WriteString("\t\t\t\tpos = pos + 1  # Skip opening brace\n")
	b.WriteString("\t\t\t\tpos = ProtoCoreUtils.skip_whitespace(text, pos)\n")
	b.WriteString("\n")
	b.WriteString("\t\t\t\t# Extract message body\n")
	b.WriteString("\t\t\t\tvar msg_start := pos\n")
	b.WriteString("\t\t\t\tvar depth := 1\n")
	b.WriteString("\t\t\t\twhile pos < text.length() and depth > 0:\n")
	b.WriteString("\t\t\t\t\tif text[pos] == \"{\":\n")
	b.WriteString("\t\t\t\t\t\tdepth = depth + 1\n")
	b.WriteString("\t\t\t\t\telif text[pos] == \"}\":\n")
	b.WriteString("\t\t\t\t\t\tdepth = depth - 1\n")
	b.WriteString("\t\t\t\t\t\tif depth == 0:\n")
	b.WriteString("\t\t\t\t\t\t\tbreak\n")
	b.WriteString("\t\t\t\t\tpos = pos + 1\n")
	b.WriteString("\n")
	b.WriteString("\t\t\t\tvar msg_text := text.substr(msg_start, pos - msg_start)\n")
	b.WriteString("\t\t\t\tpos = pos + 1  # Skip closing brace\n")
	b.WriteString("\n")
	b.WriteString("\t\t\t\t_" + f.Name + " = " + messageType + ".new()\n")
	b.WriteString("\t\t\t\tvar parse_result := _" + f.Name + ".from_text(msg_text)\n")
	b.WriteString("\t\t\t\tif parse_result != ProtoCoreUtils.ProtobufError.NO_ERRORS:\n")
	b.WriteString("\t\t\t\t\treturn parse_result\n")
	b.WriteString(oneofAssignmentExtraIndent(oneofGroup, f.Name, "\t\t\t\t"))
	return b.String()
}

func (g *generator) fromTextRepeatedMessageBody(f *ast.Field) string {
	messageType := g.renderedFieldType(f)
	var b strings.Builder
	b.WriteString("\t\t\t# Parse message\n")
	b.WriteString("\t\t\tif pos < text.length() and text[pos] == \"{\":\n")
	b.WriteString("\t\t\t\tpos = pos + 1  # Skip opening brace\n")
	b.WriteString("\t\t\t\tpos = ProtoCoreUtils.skip_whitespace(text, pos)\n")
	b.WriteString("\n")
	b.WriteString("\t\t\t\t# Extract message body\n")
	b.WriteString("\t\t\t\tvar msg_start := pos\n")
	b.WriteString("\t\t\t\tvar depth := 1\n")
	b.WriteString("\t\t\t\twhile pos < text.length() and depth > 0:\n")
	b.WriteString("\t\t\t\t\tif text[pos] == \"{\":\n")
	b.WriteString("\t\t\t\t\t\tdepth = depth + 1\n")
	b.WriteString("\t\t\t\t\telif text[pos] == \"}\":\n")
	b.WriteString("\t\t\t\t\t\tdepth = depth - 1\n")
	b.WriteString("\t\t\t\t\t\tif depth == 0:\n")
	b.WriteString("\t\t\t\t\t\t\tbreak\n")
	b.WriteString("\t\t\t\t\tpos = pos + 1\n")
	b.WriteString("\n")
	b.WriteString("\t\t\t\tvar msg_text := text.substr(msg_start, pos - msg_start)\n")
	b.WriteString("\t\t\t\tpos = pos + 1  # Skip closing brace\n")
	b.WriteString("\n")
	b.WriteString("\t\t\t\tvar msg_instance := " + messageType + ".new()\n")
	b.WriteString("\t\t\t\tvar parse_result := msg_instance.from_text(msg_text)\n")
	b.WriteString("\t\t\t\tif parse_result != ProtoCoreUtils.ProtobufError.NO_ERRORS:\n")
	b.WriteString("\t\t\t\t\treturn parse_result\n")
	b.WriteString("\t\t\t\t_" + f.Name + ".append(msg_instance)\n")
	return b.String()
}

func fromTextMapCase(mf *ast.MapField) string {
	var b strings.Builder
	b.WriteString("\t\t\"" + mf.Name + "\":\n")
	b.WriteString("\t\t\t# Parse map entry\n")
	b.WriteString("\t\t\tif pos < text.length() and text[pos] == \":\":\n")
	b.WriteString("\t\t\t\tpos += 1\n")
	b.WriteString("\t\t\tpos = ProtoCoreUtils.skip_whitespace(text, pos)\n")
	b.WriteString("\t\t\tif pos < text.length() and text[pos] == \"{\":\n")
	b.WriteString("\t\t\t\tpos += 1\n")
	b.WriteString("\t\t\t\tpos = ProtoCoreUtils.skip_whitespace(text, pos)\n")
	b.WriteString("\n")
	b.WriteString("\t\t\t\tvar map_key = null\n")
	b.WriteString("\t\t\t\tvar map_value = null\n")
	b.WriteString("\n")
	b.WriteString("\t\t\t\t# Parse key and value\n")
	b.WriteString("\t\t\t\twhile pos < text.length() and text[pos] != \"}\":\n")
	b.WriteString("\t\t\t\t\tvar entry_name_result := ProtoCoreUtils.parse_identifier(text, pos)\n")
	b.WriteString("\t\t\t\t\tif \"error\" in entry_name_result:\n")
	b.WriteString("\t\t\t\t\t\tbreak\n")
	b.WriteString("\t\t\t\t\tvar entry_field: String = entry_name_result[\"value\"]\n")
	b.WriteString("\t\t\t\t\tpos = entry_name_result[\"pos\"]\n")
	b.WriteString("\t\t\t\t\tpos = ProtoCoreUtils.skip_whitespace(text, pos)\n")
	b.WriteString("\t\t\t\t\tif pos < text.length() and text[pos] == \":\":\n")
	b.WriteString("\t\t\t\t\t\tpos += 1\n")
	b.WriteString("\t\t\t\t\tpos = ProtoCoreUtils.skip_whitespace(text, pos)\n")
	b.WriteString("\n")
	b.WriteString("\t\t\t\t\tif entry_field == \"key\":\n")
	b.WriteString(fromTextMapEntryParser(mf.KeyType, "map_key", "\t\t\t\t\t\t"))
	b.WriteString("\t\t\t\t\telif entry_field == \"value\":\n")
	b.WriteString(fromTextMapEntryParser(mf.ValueType, "map_value", "\t\t\t\t\t\t"))
	b.WriteString("\n")
	b.WriteString("\t\t\t\t\tpos = ProtoCoreUtils.skip_whitespace(text, pos)\n")
	b.WriteString("\n")
	b.WriteString("\t\t\t\tif map_key != null and map_value != null:\n")
	b.WriteString("\t\t\t\t\t_" + mf.Name + "[map_key] = map_value\n")
	b.WriteString("\n")
	b.WriteString("\t\t\t\tpos = ProtoCoreUtils.skip_whitespace(text, pos)\n")
	b.WriteString("\t\t\t\tif pos < text.length() and text[pos] == \"}\":\n")
	b.WriteString("\t\t\t\t\tpos += 1\n")
	return b.String()
}

func fromTextMapEntryParser(protoType, target, indent string) string {
	var b strings.Builder
	switch protoType {
	case "string", "bytes":
		b.WriteString(indent + "var str_result := ProtoCoreUtils.parse_string_literal(text, pos)\n")
		b.WriteString(indent + "if \"value\" in str_result:\n")
		b.WriteString(indent + "\t" + target + " = str_result[\"value\"]\n")
		b.WriteString(indent + "\tpos = str_result[\"pos\"]\n")
	case "float", "double":
		b.WriteString(indent + "var num_result := ProtoCoreUtils.parse_number(text, pos)\n")
		b.WriteString(indent + "if \"value\" in num_result:\n")
		b.WriteString(indent + "\t" + target + " = float(num_result[\"value\"])\n")
		b.WriteString(indent + "\tpos = num_result[\"pos\"]\n")
	case "bool":
		b.WriteString(indent + "var id_result := ProtoCoreUtils.parse_identifier(text, pos)\n")
		b.WriteString(indent + "if \"value\" in id_result:\n")
		b.WriteString(indent + "\t" + target + " = id_result[\"value\"] == \"true\"\n")
		b.WriteString(indent + "\tpos = id_result[\"pos\"]\n")
	default:
		b.WriteString(indent + "var num_result := ProtoCoreUtils.parse_number(text, pos)\n")
		b.WriteString(indent + "if \"value\" in num_result:\n")
		b.WriteString(indent + "\t" + target + " = int(num_result[\"value\"])\n")
		b.WriteString(indent + "\tpos = num_result[\"pos\"]\n")
	}
	return b.String()
}

// generateEnumNameAndParserHelpers emits both `_get_enum_name_<field>` and
// `_parse_enum_value_<field>` helpers for every enum-typed field on the
// message. to_text uses the former to render enum values as their declared
// names; from_text uses the latter to parse textual enum names back to their
// integer values.
func (g *generator) generateEnumNameAndParserHelpers(m *ast.Message) []gdast.Node {
	var out []gdast.Node
	enumFields := g.collectEnumFields(m)
	for i, f := range enumFields {
		out = append(out, g.generateGetEnumNameHelper(f), g.generateParseEnumValueHelper(f))
		if i < len(enumFields)-1 {
			out = append(out, gdast.EmptyLine{})
		}
	}
	return out
}

// collectEnumFields returns every enum-typed field on the message, scanning
// regular and oneof fields in declaration order.
func (g *generator) collectEnumFields(m *ast.Message) []*ast.Field {
	var fields []*ast.Field
	for _, f := range m.Fields {
		if isEnumType(f) {
			fields = append(fields, f)
		}
	}
	for _, oneof := range m.Oneofs {
		for _, f := range oneof.Fields {
			if isEnumType(f) {
				fields = append(fields, f)
			}
		}
	}
	return fields
}

// generateGetEnumNameHelper emits the `_get_enum_name_<field>` function that
// maps an enum integer value back to its declared symbolic name. Unknown
// values fall through to `str(value)` so the text output remains stable.
func (g *generator) generateGetEnumNameHelper(f *ast.Field) gdast.Function {
	values := g.enumValuesFor(f)
	cases := make([]gdast.MatchCase, 0, len(values)+1)
	for _, v := range values {
		cases = append(cases, gdast.MatchCase{
			Pattern: g.renderedFieldType(f) + "." + v.Name,
			Body:    []gdast.Statement{gdast.Ret(gdast.Lit(v.Name))},
		})
	}
	cases = append(cases, gdast.MatchCase{
		Pattern: "_",
		Body:    []gdast.Statement{gdast.Ret(gdast.Call("str", gdast.V("value")))},
	})

	return gdast.Function{
		Name:       "_get_enum_name_" + f.Name,
		Parameters: []gdast.Parameter{{Name: "value", TypeHint: "int"}},
		ReturnType: "String",
		Body: []gdast.Statement{
			gdast.DocString{Text: "Get enum name for " + f.Name + " value."},
			gdast.MatchStatement{
				Expression: gdast.V("value"),
				Cases:      cases,
			},
		},
	}
}

func (g *generator) generateParseEnumValueHelper(f *ast.Field) gdast.Function {
	values := g.enumValuesFor(f)
	cases := make([]gdast.MatchCase, 0, len(values)+1)
	for _, v := range values {
		cases = append(cases, gdast.MatchCase{
			Pattern: `"` + v.Name + `"`,
			Body:    []gdast.Statement{gdast.Ret(gdast.RawExpression{Code: g.renderedFieldType(f) + "." + v.Name})},
		})
	}
	cases = append(cases, gdast.MatchCase{
		Pattern: "_",
		Body:    []gdast.Statement{gdast.Ret(gdast.Lit(0))},
	})

	return gdast.Function{
		Name:       "_parse_enum_value_" + f.Name,
		Parameters: []gdast.Parameter{{Name: "name", TypeHint: "String"}},
		ReturnType: "int",
		Body: []gdast.Statement{
			gdast.DocString{Text: "Parse enum value from name for " + f.Name + "."},
			gdast.MatchStatement{
				Expression: gdast.V("name"),
				Cases:      cases,
			},
		},
	}
}

// enumValuesFor returns the enum values associated with an enum-typed field.
// Same-file references resolve through the in-file enum AST; cross-file
// references rely on the EnumValues snapshot attached during descriptor or
// import resolution.
func (g *generator) enumValuesFor(f *ast.Field) []*ast.EnumValue {
	if enum := g.findEnum(f.FieldType); enum != nil {
		return enum.Values
	}
	return f.EnumValues
}

// findEnum locates an enum AST node by name, searching top-level enums then
// nested enums of every message. Returns nil if not found (shouldn't occur for
// well-formed input).
func (g *generator) findEnum(name string) *ast.Enum {
	for _, e := range g.file.Enums {
		if e.Name == name {
			return e
		}
	}
	for _, m := range g.file.Messages {
		if e := findNestedEnum(m, name, m.Name); e != nil {
			return e
		}
	}
	return nil
}

func findNestedEnum(m *ast.Message, name, prefix string) *ast.Enum {
	for _, e := range m.NestedEnums {
		fullName := prefix + "." + e.Name
		if e.Name == name || fullName == name {
			return e
		}
	}
	for _, nested := range m.NestedMessages {
		nestedPrefix := prefix + "." + nested.Name
		if e := findNestedEnum(nested, name, nestedPrefix); e != nil {
			return e
		}
	}
	return nil
}
