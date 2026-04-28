package generator

import "github.com/cafecito-games/gogdproto/internal/gdast"

// protobufCoreGDScript holds the static helper functions every generated proto
// file relies on for varint, zigzag, fixed-width, float, and string codecs as
// well as tag composition helpers. It is consumed by the PBCore class emitted
// at the bottom of the generated file in later milestones.
//
//nolint:unused // superseded by pbCoreClassGDScript; retained for reference.
const protobufCoreGDScript = "# Encode/decode varint (variable-length integer)\n" +
	"static func encode_varint(value: int) -> PackedByteArray:\n" +
	"\t\"\"\"Encode integer as varint.\"\"\"\n" +
	"\tvar result: PackedByteArray = PackedByteArray()\n" +
	"\t# Use unsigned right shift for proper varint encoding\n" +
	"\t# Negative values will be encoded as large unsigned values (10 bytes)\n" +
	"\tvar unsigned_value: int = value\n" +
	"\twhile unsigned_value > 0x7F or unsigned_value < 0:\n" +
	"\t\tresult.append((unsigned_value & 0x7F) | 0x80)\n" +
	"\t\tunsigned_value = (unsigned_value >> 7) & 0x01FFFFFFFFFFFFFF  # Unsigned right shift\n" +
	"\tresult.append(unsigned_value & 0x7F)\n" +
	"\treturn result\n" +
	"\n" +
	"static func decode_varint(data: PackedByteArray, offset: int) -> Dictionary[String, int]:\n" +
	"\t\"\"\"Decode varint from data.\n" +
	"\n" +
	"\tReturns:\n" +
	"\tDictionary with 'value' and 'size' keys\n" +
	"\t\"\"\"\n" +
	"\tvar result: int = 0\n" +
	"\tvar shift: int = 0\n" +
	"\tvar size: int = 0\n" +
	"\n" +
	"\twhile offset + size < data.size():\n" +
	"\t\tvar byte: int = data[offset + size]\n" +
	"\t\tresult |= (byte & 0x7F) << shift\n" +
	"\t\tsize += 1\n" +
	"\t\tif (byte & 0x80) == 0:\n" +
	"\t\t\treturn {\"value\": result, \"size\": size}\n" +
	"\t\tshift += 7\n" +
	"\t\tif shift > 63:\n" +
	"\t\t\tbreak\n" +
	"\n" +
	"\treturn {\"value\": 0, \"size\": -1}\n" +
	"\n" +
	"# Encode/decode zigzag (for sint32/sint64)\n" +
	"static func encode_zigzag32(value: int) -> int:\n" +
	"\t\"\"\"Encode signed int32 using zigzag encoding.\"\"\"\n" +
	"\t# Mask final result to 32 bits to prevent 64-bit sign extension issues\n" +
	"\treturn (((value << 1) & 0xFFFFFFFF) ^ (value >> 31)) & 0xFFFFFFFF\n" +
	"\n" +
	"static func encode_zigzag64(value: int) -> int:\n" +
	"\t\"\"\"Encode signed int64 using zigzag encoding.\"\"\"\n" +
	"\treturn (value << 1) ^ (value >> 63)\n" +
	"\n" +
	"static func decode_zigzag32(value: int) -> int:\n" +
	"\t\"\"\"Decode zigzag-encoded int32.\"\"\"\n" +
	"\t# Use conditional approach to avoid sign extension issues\n" +
	"\tif value & 0x01:\n" +
	"\t\treturn ~(value >> 1)\n" +
	"\telse:\n" +
	"\t\treturn value >> 1\n" +
	"\n" +
	"static func decode_zigzag64(value: int) -> int:\n" +
	"\t\"\"\"Decode zigzag-encoded int64.\"\"\"\n" +
	"\t# Need unsigned right shift for 64-bit values\n" +
	"\t# Simulate unsigned right shift by masking after shift\n" +
	"\tvar shifted: int = (value >> 1) & 0x7FFFFFFFFFFFFFFF\n" +
	"\tif value & 0x01:\n" +
	"\t\treturn ~shifted\n" +
	"\telse:\n" +
	"\t\treturn shifted\n" +
	"\n" +
	"# Encode/decode fixed-size integers\n" +
	"static func encode_fixed32(value: int) -> PackedByteArray:\n" +
	"\t\"\"\"Encode 32-bit fixed integer.\"\"\"\n" +
	"\tvar result: PackedByteArray = PackedByteArray()\n" +
	"\tresult.resize(4)\n" +
	"\tresult.encode_u32(0, value)\n" +
	"\treturn result\n" +
	"\n" +
	"static func encode_fixed64(value: int) -> PackedByteArray:\n" +
	"\t\"\"\"Encode 64-bit fixed integer.\"\"\"\n" +
	"\tvar result: PackedByteArray = PackedByteArray()\n" +
	"\tresult.resize(8)\n" +
	"\tresult.encode_u64(0, value)\n" +
	"\treturn result\n" +
	"\n" +
	"static func decode_fixed32(data: PackedByteArray, offset: int) -> int:\n" +
	"\t\"\"\"Decode 32-bit unsigned fixed integer.\"\"\"\n" +
	"\treturn data.decode_u32(offset)\n" +
	"\n" +
	"static func decode_fixed64(data: PackedByteArray, offset: int) -> int:\n" +
	"\t\"\"\"Decode 64-bit unsigned fixed integer.\"\"\"\n" +
	"\treturn data.decode_u64(offset)\n" +
	"\n" +
	"static func encode_sfixed32(value: int) -> PackedByteArray:\n" +
	"\t\"\"\"Encode 32-bit signed fixed integer.\"\"\"\n" +
	"\tvar result: PackedByteArray = PackedByteArray()\n" +
	"\tresult.resize(4)\n" +
	"\tresult.encode_s32(0, value)\n" +
	"\treturn result\n" +
	"\n" +
	"static func encode_sfixed64(value: int) -> PackedByteArray:\n" +
	"\t\"\"\"Encode 64-bit signed fixed integer.\"\"\"\n" +
	"\tvar result: PackedByteArray = PackedByteArray()\n" +
	"\tresult.resize(8)\n" +
	"\tresult.encode_s64(0, value)\n" +
	"\treturn result\n" +
	"\n" +
	"static func decode_sfixed32(data: PackedByteArray, offset: int) -> int:\n" +
	"\t\"\"\"Decode 32-bit signed fixed integer.\"\"\"\n" +
	"\treturn data.decode_s32(offset)\n" +
	"\n" +
	"static func decode_sfixed64(data: PackedByteArray, offset: int) -> int:\n" +
	"\t\"\"\"Decode 64-bit signed fixed integer.\"\"\"\n" +
	"\treturn data.decode_s64(offset)\n" +
	"\n" +
	"# Encode/decode float/double\n" +
	"static func encode_float(value: float) -> PackedByteArray:\n" +
	"\t\"\"\"Encode 32-bit float.\"\"\"\n" +
	"\tvar result: PackedByteArray = PackedByteArray()\n" +
	"\tresult.resize(4)\n" +
	"\tresult.encode_float(0, value)\n" +
	"\treturn result\n" +
	"\n" +
	"static func encode_double(value: float) -> PackedByteArray:\n" +
	"\t\"\"\"Encode 64-bit double.\"\"\"\n" +
	"\tvar result: PackedByteArray = PackedByteArray()\n" +
	"\tresult.resize(8)\n" +
	"\tresult.encode_double(0, value)\n" +
	"\treturn result\n" +
	"\n" +
	"static func decode_float(data: PackedByteArray, offset: int) -> float:\n" +
	"\t\"\"\"Decode 32-bit float.\"\"\"\n" +
	"\treturn data.decode_float(offset)\n" +
	"\n" +
	"static func decode_double(data: PackedByteArray, offset: int) -> float:\n" +
	"\t\"\"\"Decode 64-bit double.\"\"\"\n" +
	"\treturn data.decode_double(offset)\n" +
	"\n" +
	"# Encode/decode string\n" +
	"static func encode_string(value: String) -> PackedByteArray:\n" +
	"\t\"\"\"Encode string as UTF-8.\"\"\"\n" +
	"\treturn value.to_utf8_buffer()\n" +
	"\n" +
	"static func decode_string(data: PackedByteArray, offset: int, length: int) -> String:\n" +
	"\t\"\"\"Decode UTF-8 string.\"\"\"\n" +
	"\tvar slice: PackedByteArray = data.slice(offset, offset + length)\n" +
	"\treturn slice.get_string_from_utf8()\n" +
	"\n" +
	"# Make field tag\n" +
	"static func make_tag(field_number: int, wire_type: int) -> int:\n" +
	"\t\"\"\"Make protobuf field tag.\"\"\"\n" +
	"\treturn (field_number << 3) | wire_type\n" +
	"\n" +
	"# Get wire type from tag\n" +
	"static func get_wire_type(tag: int) -> int:\n" +
	"\t\"\"\"Extract wire type from tag.\"\"\"\n" +
	"\treturn tag & 0x7\n" +
	"\n" +
	"# Get field number from tag\n" +
	"static func get_field_number(tag: int) -> int:\n" +
	"\t\"\"\"Extract field number from tag.\"\"\"\n" +
	"\treturn tag >> 3\n"

// textFormatUtilsGDScript contains the text-format escape, parse, and helper
// functions used by the proto-text reflection implementation.
//
//nolint:unused // used by text-format milestones (M5-T5+).
const textFormatUtilsGDScript = "# ============================================================================\n" +
	"# Text Format Utilities\n" +
	"# ============================================================================\n" +
	"\n" +
	"# Text format escaping and unescaping\n" +
	"static func escape_string_text_format(value: String) -> String:\n" +
	"\t\"\"\"Escape string for text format output.\"\"\"\n" +
	"\tvar result: String = \"\"\n" +
	"\tfor i in range(value.length()):\n" +
	"\t\tvar ch = value[i]\n" +
	"\t\tmatch ch:\n" +
	"\t\t\t\"\\n\": result += \"\\\\n\"\n" +
	"\t\t\t\"\\r\": result += \"\\\\r\"\n" +
	"\t\t\t\"\\t\": result += \"\\\\t\"\n" +
	"\t\t\t\"\\\"\": result += \"\\\\\\\"\"\n" +
	"\t\t\t\"\\\\\": result += \"\\\\\\\\\"\n" +
	"\t\t\t_:\n" +
	"\t\t\t\tvar code = ch.unicode_at(0)\n" +
	"\t\t\t\tif code < 32:\n" +
	"\t\t\t\t\t# Non-printable control characters: use \\xHH escape\n" +
	"\t\t\t\t\tresult += \"\\\\x%02x\" % code\n" +
	"\t\t\t\telse:\n" +
	"\t\t\t\t\t# Preserve all other characters (including UTF-8)\n" +
	"\t\t\t\t\tresult += ch\n" +
	"\treturn result\n" +
	"\n" +
	"static func escape_bytes_text_format(value: PackedByteArray) -> String:\n" +
	"\t\"\"\"Escape bytes for text format output.\"\"\"\n" +
	"\tvar result: String = \"\"\n" +
	"\tfor byte in value:\n" +
	"\t\tif byte >= 32 and byte < 127 and byte != 92 and byte != 34:  # printable, not backslash or quote\n" +
	"\t\t\tresult += char(byte)\n" +
	"\t\telse:\n" +
	"\t\t\tresult += \"\\\\x%02x\" % byte\n" +
	"\treturn result\n" +
	"\n" +
	"static func unescape_string_text_format(value: String) -> String:\n" +
	"\t\"\"\"Unescape text format string.\"\"\"\n" +
	"\tvar result: String = \"\"\n" +
	"\tvar i: int = 0\n" +
	"\twhile i < value.length():\n" +
	"\t\tvar ch = value[i]\n" +
	"\t\tif ch == \"\\\\\":\n" +
	"\t\t\ti += 1\n" +
	"\t\t\tif i >= value.length():\n" +
	"\t\t\t\tbreak\n" +
	"\t\t\tvar next = value[i]\n" +
	"\t\t\tmatch next:\n" +
	"\t\t\t\t\"n\": result += \"\\n\"\n" +
	"\t\t\t\t\"r\": result += \"\\r\"\n" +
	"\t\t\t\t\"t\": result += \"\\t\"\n" +
	"\t\t\t\t\"\\\\\": result += \"\\\\\"\n" +
	"\t\t\t\t\"\\\"\": result += \"\\\"\"\n" +
	"\t\t\t\t\"x\":\n" +
	"\t\t\t\t\t# \\xHH hex escape\n" +
	"\t\t\t\t\tif i + 2 < value.length():\n" +
	"\t\t\t\t\t\tvar hex = value.substr(i + 1, 2)\n" +
	"\t\t\t\t\t\tresult += char(hex.hex_to_int())\n" +
	"\t\t\t\t\t\ti += 2\n" +
	"\t\t\t\t_:\n" +
	"\t\t\t\t\tresult += next\n" +
	"\t\t\ti += 1\n" +
	"\t\telse:\n" +
	"\t\t\tresult += ch\n" +
	"\t\t\ti += 1\n" +
	"\treturn result\n" +
	"\n" +
	"static func unescape_bytes_text_format(value: String) -> PackedByteArray:\n" +
	"\t\"\"\"Unescape text format string directly to bytes.\"\"\"\n" +
	"\tvar result := PackedByteArray()\n" +
	"\tvar i: int = 0\n" +
	"\twhile i < value.length():\n" +
	"\t\tvar ch = value[i]\n" +
	"\t\tif ch == \"\\\\\":\n" +
	"\t\t\ti += 1\n" +
	"\t\t\tif i >= value.length():\n" +
	"\t\t\t\tbreak\n" +
	"\t\t\tvar next = value[i]\n" +
	"\t\t\tmatch next:\n" +
	"\t\t\t\t\"n\": result.append(0x0A)  # \\n\n" +
	"\t\t\t\t\"r\": result.append(0x0D)  # \\r\n" +
	"\t\t\t\t\"t\": result.append(0x09)  # \\t\n" +
	"\t\t\t\t\"\\\\\": result.append(0x5C)  # \\\n" +
	"\t\t\t\t\"\\\"\": result.append(0x22)  # \"\n" +
	"\t\t\t\t\"x\":\n" +
	"\t\t\t\t\t# \\xHH hex escape - convert directly to byte\n" +
	"\t\t\t\t\tif i + 2 < value.length():\n" +
	"\t\t\t\t\t\tvar hex = value.substr(i + 1, 2)\n" +
	"\t\t\t\t\t\tresult.append(hex.hex_to_int())\n" +
	"\t\t\t\t\t\ti += 2\n" +
	"\t\t\t\t_:\n" +
	"\t\t\t\t\t# Unknown escape, just use the byte value of the character\n" +
	"\t\t\t\t\tresult.append(next.unicode_at(0))\n" +
	"\t\t\ti += 1\n" +
	"\t\telse:\n" +
	"\t\t\t# Regular character - append its byte value\n" +
	"\t\t\tresult.append(ch.unicode_at(0))\n" +
	"\t\t\ti += 1\n" +
	"\treturn result\n" +
	"\n" +
	"# Text format parsing utilities\n" +
	"static func skip_whitespace(text: String, pos: int) -> int:\n" +
	"\t\"\"\"Skip whitespace and comments.\"\"\"\n" +
	"\twhile pos < text.length():\n" +
	"\t\tvar ch = text[pos]\n" +
	"\t\tif ch in [\" \", \"\\t\", \"\\n\", \"\\r\"]:\n" +
	"\t\t\tpos += 1\n" +
	"\t\telif ch == \"#\":\n" +
	"\t\t\t# Skip comment until end of line\n" +
	"\t\t\twhile pos < text.length() and text[pos] != \"\\n\":\n" +
	"\t\t\t\tpos += 1\n" +
	"\t\telse:\n" +
	"\t\t\tbreak\n" +
	"\treturn pos\n" +
	"\n" +
	"static func parse_identifier(text: String, pos: int) -> Dictionary:\n" +
	"\t\"\"\"Parse identifier (field name or keyword).\"\"\"\n" +
	"\tvar start = pos\n" +
	"\twhile pos < text.length():\n" +
	"\t\tvar ch = text[pos]\n" +
	"\t\tif ch.is_valid_identifier() or ch == \"_\" or (pos > start and ch.is_valid_int()):\n" +
	"\t\t\tpos += 1\n" +
	"\t\telse:\n" +
	"\t\t\tbreak\n" +
	"\tif pos == start:\n" +
	"\t\treturn {\"error\": \"Expected identifier\"}\n" +
	"\treturn {\"value\": text.substr(start, pos - start), \"pos\": pos}\n" +
	"\n" +
	"static func parse_string_literal(text: String, pos: int) -> Dictionary:\n" +
	"\t\"\"\"Parse quoted string literal.\"\"\"\n" +
	"\tif pos >= text.length() or text[pos] != \"\\\"\":\n" +
	"\t\treturn {\"error\": \"Expected string literal\"}\n" +
	"\tpos += 1  # Skip opening quote\n" +
	"\tvar value: String = \"\"\n" +
	"\twhile pos < text.length():\n" +
	"\t\tvar ch = text[pos]\n" +
	"\t\tif ch == \"\\\"\":\n" +
	"\t\t\tpos += 1\n" +
	"\t\t\treturn {\"value\": value, \"pos\": pos}\n" +
	"\t\telif ch == \"\\\\\":\n" +
	"\t\t\tpos += 1\n" +
	"\t\t\tif pos >= text.length():\n" +
	"\t\t\t\treturn {\"error\": \"Unterminated string\"}\n" +
	"\t\t\tvar next = text[pos]\n" +
	"\t\t\tmatch next:\n" +
	"\t\t\t\t\"n\": value += \"\\n\"\n" +
	"\t\t\t\t\"r\": value += \"\\r\"\n" +
	"\t\t\t\t\"t\": value += \"\\t\"\n" +
	"\t\t\t\t\"\\\\\": value += \"\\\\\"\n" +
	"\t\t\t\t\"\\\"\": value += \"\\\"\"\n" +
	"\t\t\t\t\"x\":\n" +
	"\t\t\t\t\tif pos + 2 < text.length():\n" +
	"\t\t\t\t\t\tvar hex = text.substr(pos + 1, 2)\n" +
	"\t\t\t\t\t\tvalue += char(hex.hex_to_int())\n" +
	"\t\t\t\t\t\tpos += 2\n" +
	"\t\t\t\t_:\n" +
	"\t\t\t\t\tvalue += next\n" +
	"\t\t\tpos += 1\n" +
	"\t\telse:\n" +
	"\t\t\tvalue += ch\n" +
	"\t\t\tpos += 1\n" +
	"\treturn {\"error\": \"Unterminated string\"}\n" +
	"\n" +
	"static func parse_number(text: String, pos: int) -> Dictionary:\n" +
	"\t\"\"\"Parse number (int or float).\"\"\"\n" +
	"\tvar start = pos\n" +
	"\t# Handle negative sign\n" +
	"\tif pos < text.length() and text[pos] in [\"-\", \"+\"]:\n" +
	"\t\tpos += 1\n" +
	"\t# Check for special float values\n" +
	"\tif pos + 2 < text.length():\n" +
	"\t\tvar substr = text.substr(pos, 3)\n" +
	"\t\tif substr == \"inf\":\n" +
	"\t\t\treturn {\"value\": INF if text[start] != \"-\" else -INF, \"pos\": pos + 3, \"is_float\": true}\n" +
	"\t\tif substr == \"nan\":\n" +
	"\t\t\treturn {\"value\": NAN, \"pos\": pos + 3, \"is_float\": true}\n" +
	"\t# Parse digits\n" +
	"\tvar has_dot = false\n" +
	"\tvar has_exp = false\n" +
	"\twhile pos < text.length():\n" +
	"\t\tvar ch = text[pos]\n" +
	"\t\tif ch.is_valid_int():\n" +
	"\t\t\tpos += 1\n" +
	"\t\telif ch == \".\" and not has_dot and not has_exp:\n" +
	"\t\t\thas_dot = true\n" +
	"\t\t\tpos += 1\n" +
	"\t\telif ch in [\"e\", \"E\"] and not has_exp:\n" +
	"\t\t\thas_exp = true\n" +
	"\t\t\tpos += 1\n" +
	"\t\t\tif pos < text.length() and text[pos] in [\"+\", \"-\"]:\n" +
	"\t\t\t\tpos += 1\n" +
	"\t\telse:\n" +
	"\t\t\tbreak\n" +
	"\tif pos == start or (pos == start + 1 and text[start] in [\"-\", \"+\"]):\n" +
	"\t\treturn {\"error\": \"Expected number\"}\n" +
	"\tvar num_str = text.substr(start, pos - start)\n" +
	"\tif has_dot or has_exp:\n" +
	"\t\treturn {\"value\": float(num_str), \"pos\": pos, \"is_float\": true}\n" +
	"\telse:\n" +
	"\t\treturn {\"value\": int(num_str), \"pos\": pos, \"is_float\": false}"

// headerCommentText returns the file header comment block emitted at the top
// of every generated GDScript file.
func headerCommentText(sourceName string) string {
	return "Generated by gdproto\nSource: " + sourceName + "\nDO NOT EDIT"
}

// pbCoreClassGDScript is the verbatim source of the trailing `class PBCore:`
// block that historically shipped inside every generated proto file. It is no
// longer emitted from the wrapper (M7-T0); subsequent tasks reuse this content
// when generating the sibling `proto_core_utils.gd` file.
//
//nolint:unused // referenced again starting in M7-T1 for the sibling file.
const pbCoreClassGDScript = "class PBCore:\n" +
	"\t\"\"\"Core protobuf encoding/decoding utilities.\"\"\"\n" +
	"\t\n" +
	"\t# Encode/decode varint (variable-length integer)\n" +
	"\tstatic func encode_varint(value: int) -> PackedByteArray:\n" +
	"\t\t\"\"\"Encode integer as varint.\"\"\"\n" +
	"\t\tvar result: PackedByteArray = PackedByteArray()\n" +
	"\t\twhile value > 0x7F:\n" +
	"\t\t\tresult.append((value & 0x7F) | 0x80)\n" +
	"\t\t\tvalue >>= 7\n" +
	"\t\tresult.append(value & 0x7F)\n" +
	"\t\treturn result\n" +
	"\t\n" +
	"\tstatic func decode_varint(data: PackedByteArray, offset: int) -> Dictionary[String, int]:\n" +
	"\t\t\"\"\"Decode varint from data.\n" +
	"\t\t\n" +
	"\t\tReturns:\n" +
	"\t\t\tDictionary with 'value' and 'size' keys\n" +
	"\t\t\"\"\"\n" +
	"\t\tvar result: int = 0\n" +
	"\t\tvar shift: int = 0\n" +
	"\t\tvar size: int = 0\n" +
	"\t\t\n" +
	"\t\twhile offset + size < data.size():\n" +
	"\t\t\tvar byte: int = data[offset + size]\n" +
	"\t\t\tresult |= (byte & 0x7F) << shift\n" +
	"\t\t\tsize += 1\n" +
	"\t\t\tif (byte & 0x80) == 0:\n" +
	"\t\t\t\treturn {\"value\": result, \"size\": size}\n" +
	"\t\t\tshift += 7\n" +
	"\t\t\tif shift > 63:\n" +
	"\t\t\t\tbreak\n" +
	"\t\t\n" +
	"\t\treturn {\"value\": 0, \"size\": -1}\n" +
	"\t\n" +
	"\t# Encode/decode zigzag (for sint32/sint64)\n" +
	"\tstatic func encode_zigzag32(value: int) -> int:\n" +
	"\t\t\"\"\"Encode signed int32 using zigzag encoding.\"\"\"\n" +
	"\t\treturn (value << 1) ^ (value >> 31)\n" +
	"\t\n" +
	"\tstatic func encode_zigzag64(value: int) -> int:\n" +
	"\t\t\"\"\"Encode signed int64 using zigzag encoding.\"\"\"\n" +
	"\t\treturn (value << 1) ^ (value >> 63)\n" +
	"\t\n" +
	"\tstatic func decode_zigzag32(value: int) -> int:\n" +
	"\t\t\"\"\"Decode zigzag-encoded int32.\"\"\"\n" +
	"\t\treturn (value >> 1) ^ (-(value & 1))\n" +
	"\t\n" +
	"\tstatic func decode_zigzag64(value: int) -> int:\n" +
	"\t\t\"\"\"Decode zigzag-encoded int64.\"\"\"\n" +
	"\t\treturn (value >> 1) ^ (-(value & 1))\n" +
	"\t\n" +
	"\t# Encode/decode fixed-size integers\n" +
	"\tstatic func encode_fixed32(value: int) -> PackedByteArray:\n" +
	"\t\t\"\"\"Encode 32-bit fixed integer.\"\"\"\n" +
	"\t\tvar result: PackedByteArray = PackedByteArray()\n" +
	"\t\tresult.resize(4)\n" +
	"\t\tresult.encode_u32(0, value)\n" +
	"\t\treturn result\n" +
	"\t\n" +
	"\tstatic func encode_fixed64(value: int) -> PackedByteArray:\n" +
	"\t\t\"\"\"Encode 64-bit fixed integer.\"\"\"\n" +
	"\t\tvar result: PackedByteArray = PackedByteArray()\n" +
	"\t\tresult.resize(8)\n" +
	"\t\tresult.encode_u64(0, value)\n" +
	"\t\treturn result\n" +
	"\t\n" +
	"\tstatic func decode_fixed32(data: PackedByteArray, offset: int) -> int:\n" +
	"\t\t\"\"\"Decode 32-bit fixed integer.\"\"\"\n" +
	"\t\treturn data.decode_u32(offset)\n" +
	"\t\n" +
	"\tstatic func decode_fixed64(data: PackedByteArray, offset: int) -> int:\n" +
	"\t\t\"\"\"Decode 64-bit fixed integer.\"\"\"\n" +
	"\t\treturn data.decode_u64(offset)\n" +
	"\t\n" +
	"\t# Encode/decode float/double\n" +
	"\tstatic func encode_float(value: float) -> PackedByteArray:\n" +
	"\t\t\"\"\"Encode 32-bit float.\"\"\"\n" +
	"\t\tvar result: PackedByteArray = PackedByteArray()\n" +
	"\t\tresult.resize(4)\n" +
	"\t\tresult.encode_float(0, value)\n" +
	"\t\treturn result\n" +
	"\t\n" +
	"\tstatic func encode_double(value: float) -> PackedByteArray:\n" +
	"\t\t\"\"\"Encode 64-bit double.\"\"\"\n" +
	"\t\tvar result: PackedByteArray = PackedByteArray()\n" +
	"\t\tresult.resize(8)\n" +
	"\t\tresult.encode_double(0, value)\n" +
	"\t\treturn result\n" +
	"\t\n" +
	"\tstatic func decode_float(data: PackedByteArray, offset: int) -> float:\n" +
	"\t\t\"\"\"Decode 32-bit float.\"\"\"\n" +
	"\t\treturn data.decode_float(offset)\n" +
	"\t\n" +
	"\tstatic func decode_double(data: PackedByteArray, offset: int) -> float:\n" +
	"\t\t\"\"\"Decode 64-bit double.\"\"\"\n" +
	"\t\treturn data.decode_double(offset)\n" +
	"\t\n" +
	"\t# Encode/decode string\n" +
	"\tstatic func encode_string(value: String) -> PackedByteArray:\n" +
	"\t\t\"\"\"Encode string as UTF-8.\"\"\"\n" +
	"\t\treturn value.to_utf8_buffer()\n" +
	"\t\n" +
	"\tstatic func decode_string(data: PackedByteArray, offset: int, length: int) -> String:\n" +
	"\t\t\"\"\"Decode UTF-8 string.\"\"\"\n" +
	"\t\tvar slice: PackedByteArray = data.slice(offset, offset + length)\n" +
	"\t\treturn slice.get_string_from_utf8()\n" +
	"\t\n" +
	"\t# Make field tag\n" +
	"\tstatic func make_tag(field_number: int, wire_type: int) -> int:\n" +
	"\t\t\"\"\"Make protobuf field tag.\"\"\"\n" +
	"\t\treturn (field_number << 3) | wire_type\n" +
	"\t\n" +
	"\t# Get wire type from tag\n" +
	"\tstatic func get_wire_type(tag: int) -> int:\n" +
	"\t\t\"\"\"Extract wire type from tag.\"\"\"\n" +
	"\t\treturn tag & 0x7\n" +
	"\t\n" +
	"\t# Get field number from tag\n" +
	"\tstatic func get_field_number(tag: int) -> int:\n" +
	"\t\t\"\"\"Extract field number from tag.\"\"\"\n" +
	"\t\treturn tag >> 3"

// textFormatUtilsStatement returns the text-format helper functions as a
// RawStatement.
//
//nolint:unused // used by text-format milestones (M5-T5+).
func textFormatUtilsStatement() gdast.RawStatement {
	return gdast.RawStatement{Code: textFormatUtilsGDScript}
}
