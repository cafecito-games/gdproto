package generator_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cafecito-games/gdproto/internal/ast"
	"github.com/cafecito-games/gdproto/internal/generator"
)

func TestOneofTransitionHelperLinearGrowth(t *testing.T) {
	buildSchema := func(armCount int) string {
		var builder strings.Builder
		builder.WriteString("syntax = \"proto3\";\nmessage Payload { int32 id = 1; }\n")
		builder.WriteString("enum Kind { KIND_UNSPECIFIED = 0; KIND_ONE = 1; }\n")
		builder.WriteString("message Container {\n  oneof choice {\n")
		types := []string{"int32", "string", "bool", "float", "bytes", "Kind", "Payload"}
		for i := 1; i <= armCount; i++ {
			fieldType := types[(i-1)%len(types)]
			fmt.Fprintf(&builder, "    %s arm_%d = %d;\n", fieldType, i, i)
		}
		builder.WriteString("  }\n}\n")
		return builder.String()
	}

	for _, armCount := range []int{4, 16} {
		t.Run(fmt.Sprintf("%d_arms", armCount), func(t *testing.T) {
			files := generateFromSource(t, buildSchema(armCount), "linear.proto")
			container := findFile(files, "LinearContainer")
			require.NotNil(t, container)
			source := mustSource(t, *container)

			assert.Equal(t, 1, strings.Count(source, "func _select_choice(next_case: ChoiceOneOf) -> void:"))

			helperBody := extractFunctionBody(t, source, "func _select_choice(next_case: ChoiceOneOf) -> void:")
			assert.Contains(t, helperBody, "if _oneof_choice == next_case:\n\t\treturn")
			assert.Contains(t, helperBody, "\t_oneof_choice = next_case")

			helperResets := 0
			for i := 1; i <= armCount; i++ {
				prefix := fmt.Sprintf("\t_arm_%d = ", i)
				assert.Equal(t, 1, strings.Count(helperBody, prefix), "helper must reset _arm_%d once", i)
				helperResets += strings.Count(helperBody, prefix)
			}
			assert.Equal(t, armCount, helperResets, "helper must emit exactly N reset assignments")

			for i := 1; i <= armCount; i++ {
				header := fmt.Sprintf("func set_arm_%d(", i)
				setterBody := extractFunctionBodyByPrefix(t, source, header)
				wantCall := fmt.Sprintf("\t_select_choice(ChoiceOneOf.ARM_%d)", i)
				assert.Contains(t, setterBody, wantCall)
				assert.Contains(t, setterBody, fmt.Sprintf("\t_arm_%d = value", i))
				// Setters must never inline sibling resets.
				for sibling := 1; sibling <= armCount; sibling++ {
					if sibling == i {
						continue
					}
					assert.NotContains(t, setterBody, fmt.Sprintf("_arm_%d =", sibling))
				}
			}
		})
	}
}

func TestOneofTransitionHelperCoversAllDefaultCategories(t *testing.T) {
	const source = `syntax = "proto3";

enum TopMode {
  TOP_MODE_UNSPECIFIED = 0;
  TOP_MODE_ACTIVE = 1;
}

message SubPayload {
  string note = 1;
}

message AllTypes {
  enum LocalState {
    LOCAL_STATE_UNSPECIFIED = 0;
    LOCAL_STATE_ON = 1;
  }

  oneof choice {
    int32 int_arm = 1;
    int64 long_arm = 2;
    float float_arm = 3;
    double double_arm = 4;
    bool bool_arm = 5;
    string string_arm = 6;
    bytes bytes_arm = 7;
    TopMode top_enum_arm = 8;
    LocalState local_enum_arm = 9;
    SubPayload message_arm = 10;
  }
}
`
	files := generateFromSource(t, source, "all_types.proto")
	allTypes := findFile(files, "AllTypesAllTypes")
	require.NotNil(t, allTypes)
	rendered := mustSource(t, *allTypes)

	helperBody := extractFunctionBody(t, rendered, "func _select_choice(next_case: ChoiceOneOf) -> void:")
	for _, want := range []string{
		"\tif _oneof_choice == next_case:\n\t\treturn",
		"\t_int_arm = 0",
		"\t_long_arm = 0",
		"\t_float_arm = 0.0",
		"\t_double_arm = 0.0",
		"\t_bool_arm = false",
		"\t_string_arm = \"\"",
		"\t_bytes_arm = PackedByteArray()",
		"\t_top_enum_arm = 0 as AllTypesTopMode.TopMode",
		"\t_local_enum_arm = 0 as LocalState",
		"\t_message_arm = null",
		"\t_oneof_choice = next_case",
	} {
		assert.Contains(t, helperBody, want)
	}
}

func TestOneofMultipleGroupsRemainIndependent(t *testing.T) {
	const source = `syntax = "proto3";

message Dual {
  oneof primary {
    int32 a = 1;
    string b = 2;
  }
  oneof secondary {
    bool flag = 3;
    float ratio = 4;
  }
}
`
	files := generateFromSource(t, source, "dual.proto")
	dual := findFile(files, "DualDual")
	require.NotNil(t, dual)
	rendered := mustSource(t, *dual)

	primaryHelper := extractFunctionBody(t, rendered, "func _select_primary(next_case: PrimaryOneOf) -> void:")
	assert.Contains(t, primaryHelper, "_a = 0")
	assert.Contains(t, primaryHelper, `_b = ""`)
	assert.Contains(t, primaryHelper, "_oneof_primary = next_case")
	assert.NotContains(t, primaryHelper, "_flag")
	assert.NotContains(t, primaryHelper, "_ratio")
	assert.NotContains(t, primaryHelper, "_oneof_secondary")

	secondaryHelper := extractFunctionBody(t, rendered, "func _select_secondary(next_case: SecondaryOneOf) -> void:")
	assert.Contains(t, secondaryHelper, "_flag = false")
	assert.Contains(t, secondaryHelper, "_ratio = 0.0")
	assert.Contains(t, secondaryHelper, "_oneof_secondary = next_case")
	assert.NotContains(t, secondaryHelper, "_a =")
	assert.NotContains(t, secondaryHelper, "_b =")
	assert.NotContains(t, secondaryHelper, "_oneof_primary")
}

func TestOneofTransitionHelperCollisionAvoidance(t *testing.T) {
	t.Run("field backing variable collides with preferred helper name", func(t *testing.T) {
		const source = `syntax = "proto3";
message Collide {
  string select_choice = 1;
  oneof choice {
    int32 number = 2;
    string text = 3;
  }
}
`
		files := generateFromSource(t, source, "collide.proto")
		rendered := mustSource(t, *findFile(files, "CollideCollide"))
		assert.Contains(t, rendered, `var _select_choice: String = ""`)
		assert.Contains(t, rendered, "func _select_choice_2(next_case: ChoiceOneOf) -> void:")
		assert.NotContains(t, rendered, "func _select_choice(")
		assert.Contains(t, rendered, "\t_select_choice_2(ChoiceOneOf.NUMBER)")
		assert.Contains(t, rendered, "\t_select_choice_2(ChoiceOneOf.TEXT)")
	})

	t.Run("avoids stealing another oneof group preferred helper name", func(t *testing.T) {
		const source = `syntax = "proto3";
message CollideSibling {
  string select_choice = 1;
  oneof choice {
    int32 first = 2;
  }
  oneof choice_2 {
    string second = 3;
  }
}
`
		files := generateFromSource(t, source, "collide_sibling.proto")
		rendered := mustSource(t, *findFile(files, "CollideSiblingCollideSibling"))
		assert.Contains(t, rendered, "func _select_choice_3(next_case: ChoiceOneOf) -> void:")
		assert.Contains(t, rendered, "func _select_choice_2(next_case: Choice2OneOf) -> void:")
		assert.Contains(t, rendered, "\t_select_choice_3(ChoiceOneOf.FIRST)")
		assert.Contains(t, rendered, "\t_select_choice_2(Choice2OneOf.SECOND)")
	})

	t.Run("resource message with native accessor collision and helper collision", func(t *testing.T) {
		const source = `syntax = "proto3";
message StoredCollide {
  option (gdproto.resource) = true;
  string select_choice = 1;
  oneof choice {
    string name = 2;
    int32 count = 3;
  }
}
`
		files := generateFromSource(t, source, "stored_collide.proto")
		rendered := mustSource(t, *findFile(files, "StoredCollideStoredCollide"))
		assert.Contains(t, rendered, "extends Resource")
		assert.Contains(t, rendered, `@export_storage var _select_choice: String = ""`)
		assert.Contains(t, rendered, "func _select_choice_2(next_case: ChoiceOneOf) -> void:")
		assert.Contains(t, rendered, "func set_name_pb(value: String) -> void:\n\t_select_choice_2(ChoiceOneOf.NAME)\n\t_name = value")
		assert.Contains(t, rendered, "func set_count(value: int) -> void:\n\t_select_choice_2(ChoiceOneOf.COUNT)\n\t_count = value")
	})
}

func TestOneofTransitionHelperFailClosed(t *testing.T) {
	t.Run("repeated field in oneof fails with message group and field", func(t *testing.T) {
		file := &ast.ProtoFile{
			Syntax: "proto3",
			Messages: []*ast.Message{{
				Name: "Broken",
				Oneofs: []*ast.Oneof{{
					Name: "choice",
					Fields: []*ast.Field{{
						Name:      "items",
						FieldType: "string",
						Number:    1,
						Repeated:  true,
					}},
				}},
			}},
		}
		_, err := generator.Generate(file, "broken.proto", nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Broken")
		assert.Contains(t, err.Error(), "choice")
		assert.Contains(t, err.Error(), "items")
	})

	t.Run("empty oneof group name fails closed", func(t *testing.T) {
		file := &ast.ProtoFile{
			Syntax: "proto3",
			Messages: []*ast.Message{{
				Name: "Unnamed",
				Oneofs: []*ast.Oneof{{
					Name: "",
					Fields: []*ast.Field{{
						Name:      "value",
						FieldType: "int32",
						Number:    1,
					}},
				}},
			}},
		}
		_, err := generator.Generate(file, "unnamed.proto", nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Unnamed")
	})
}

func extractFunctionBody(t *testing.T, source, header string) string {
	t.Helper()
	index := strings.Index(source, header)
	require.NotEqual(t, -1, index, "missing function header %q in:\n%s", header, source)
	return sliceUntilNextTopLevel(source[index:])
}

func extractFunctionBodyByPrefix(t *testing.T, source, prefix string) string {
	t.Helper()
	index := strings.Index(source, prefix)
	require.NotEqual(t, -1, index, "missing function prefix %q in:\n%s", prefix, source)
	return sliceUntilNextTopLevel(source[index:])
}

func sliceUntilNextTopLevel(rest string) string {
	lines := strings.Split(rest, "\n")
	end := len(lines)
	for i := 1; i < len(lines); i++ {
		line := lines[i]
		if line != "" && !strings.HasPrefix(line, "\t") {
			end = i
			break
		}
	}
	return strings.Join(lines[:end], "\n")
}
