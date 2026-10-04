package generator_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cafecito-games/gdproto/internal/ast"
	"github.com/cafecito-games/gdproto/internal/generator"
	"github.com/cafecito-games/gdproto/internal/importer"
	"github.com/cafecito-games/gdproto/internal/lexer"
	"github.com/cafecito-games/gdproto/internal/parser"
	"github.com/cafecito-games/gdproto/internal/validator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// parseProtoSource lexes and parses one proto source, failing the test on any
// error. It stops short of validation so a fixture may reference a type that
// only another file in the set declares.
func parseProtoSource(t *testing.T, source, filename string) *ast.ProtoFile {
	t.Helper()
	tokens, err := lexer.Tokenize(source, filename)
	require.NoError(t, err, "tokenize %s", filename)
	file, err := parser.Parse(tokens, filename)
	require.NoError(t, err, "parse %s", filename)
	return file
}

// generateWithImports writes sources to a temporary directory, resolves the
// named entry file's imports through the real importer, and generates it.
//
// Running the importer matters for any case about a reference that may bind to
// another file's declaration: it is the importer that rewrites the reference to
// the imported type's short name and records the SourceFile and FullTypePath
// the generator then has to honour.
func generateWithImports(t *testing.T, sources map[string]string, entry string) []generator.GeneratedFile {
	t.Helper()
	directory := t.TempDir()
	for name, source := range sources {
		require.NoError(t, os.WriteFile(filepath.Join(directory, name), []byte(source), 0o600))
	}
	entryFile := parseProtoSource(t, sources[entry], entry)
	imported, err := importer.ResolveExternalWithFiles(
		entryFile, filepath.Join(directory, entry), &importer.OSFS{BaseDir: directory},
	)
	require.NoError(t, err, "resolve imports")

	entries := make([]generator.FileEntry, 0, len(imported))
	for _, i := range imported {
		entries = append(entries, generator.FileEntry{File: i.File, Filename: i.Filename})
	}
	files, err := generator.Generate(entryFile, entry, entries)
	require.NoError(t, err, "generate")
	return files
}

// sourceOf generates from source and returns the rendered GDScript of the
// named generated class.
func sourceOf(t *testing.T, source, filename, className string) string {
	t.Helper()
	files := generateFromSource(t, source, filename)
	f := findFile(files, className)
	require.NotNil(t, f, "missing %s; got %v", className, classNames(files))
	return mustSource(t, *f)
}

// TestGenerateNestedMessageShadowsTopLevelEnum covers protobuf's rule that a
// reference binds to the nearest enclosing declaration: Outer.Mode is a
// message, so the field is a message field end to end. Binding it to the
// file-scope enum instead gives the field an enum type with a message default
// and codec, which Godot refuses to parse.
func TestGenerateNestedMessageShadowsTopLevelEnum(t *testing.T) {
	got := sourceOf(t, `syntax = "proto3";
package game.v1;
enum Mode {
  MODE_A = 0;
}
message Outer {
  message Mode {
    int32 x = 1;
  }
  Mode field = 1;
}
`, "shadow.proto", "ShadowOuter")

	assert.Contains(t, got, "var _field: ShadowOuterMode = null",
		"a nested message must shadow a same-named file-scope enum\n%s", got)
	assert.NotContains(t, got, "ShadowMode",
		"the reference must not resolve to the file-scope enum wrapper\n%s", got)
	assert.NotContains(t, got, "0 as ",
		"the field must not be annotated as an enum\n%s", got)
	assert.NotContains(t, got, "ProtoCoreUtils.encode_varint(_field)",
		"a message field must not use the varint codec\n%s", got)
	assert.Contains(t, got, "func new_field()",
		"a message field needs a constructor accessor\n%s", got)
	assert.Contains(t, got, "_field.to_bytes()",
		"a message field must serialize through to_bytes\n%s", got)
}

// TestGenerateNestedEnumShadowsTopLevelMessage is the mirror image: a nested
// enum shadows a same-named top-level message. The nested enum is declared by
// the class being generated, so its bare name is in scope.
func TestGenerateNestedEnumShadowsTopLevelMessage(t *testing.T) {
	got := sourceOf(t, `syntax = "proto3";
package game.v1;
message Mode {
  int32 x = 1;
}
message Outer {
  enum Mode {
    MODE_A = 0;
  }
  Mode field = 1;
}
`, "shadow.proto", "ShadowOuter")

	assert.Contains(t, got, "var _field: Mode = 0 as Mode",
		"a nested enum must shadow a same-named top-level message\n%s", got)
	assert.NotContains(t, got, "ShadowMode",
		"the reference must not resolve to the top-level message class\n%s", got)
	assert.NotContains(t, got, "func new_field()",
		"an enum field must not get a message constructor accessor\n%s", got)
}

// TestGenerateDeepNestingResolvesNearestScope checks that the outward walk
// stops at the first enclosing scope that declares the name, rather than at the
// outermost one or at file scope.
func TestGenerateDeepNestingResolvesNearestScope(t *testing.T) {
	got := sourceOf(t, `syntax = "proto3";
package game.v1;
enum Mode {
  MODE_A = 0;
}
message Outer {
  message Mode {
    int32 outer_value = 1;
  }
  message Middle {
    message Mode {
      bool middle_value = 1;
    }
    message Inner {
      Mode field = 1;
    }
  }
}
`, "shadow.proto", "ShadowOuterMiddleInner")

	assert.Contains(t, got, "var _field: ShadowOuterMiddleMode = null",
		"the walk must stop at the nearest enclosing declaration\n%s", got)
	assert.NotContains(t, got, "ShadowOuterMode",
		"the walk must not skip past the intermediate scope\n%s", got)
	assert.NotContains(t, got, "0 as ",
		"the field must not be annotated as an enum\n%s", got)
}

// TestGenerateAbsoluteReferenceIsNotScopeResolved covers protobuf's rule that a
// leading dot makes a reference absolute: it names a fully qualified type
// outright and is never resolved against an enclosing scope. The same path
// without the dot is scope-resolved and binds to the nested message instead.
func TestGenerateAbsoluteReferenceIsNotScopeResolved(t *testing.T) {
	got := sourceOf(t, `syntax = "proto3";
package game.v1;
enum Mode {
  MODE_A = 0;
}
message Outer {
  message game {
    message v1 {
      message Mode {
        int32 y = 1;
      }
    }
  }
  .game.v1.Mode absolute = 1;
  game.v1.Mode relative = 2;
}
`, "game.proto", "GameOuter")

	assert.Contains(t, got, "var _absolute: GameMode.Mode = 0 as GameMode.Mode",
		"an absolute reference must resolve at file scope, not against Outer\n%s", got)
	assert.NotContains(t, got, "func new_absolute()",
		"the absolute reference names an enum, which has no constructor\n%s", got)
	assert.Contains(t, got, "var _relative: GameOutergamev1Mode = null",
		"a relative reference must resolve against the nearest enclosing scope\n%s", got)
}

// TestGenerateAbsoluteImportedRootTypeIsNotCapturedLocally covers the one shape
// where a reference's definitive name is identical to the name as written: an
// absolute reference to a root-package type arrives as "Type" in both the field
// type and the resolved path, since neither emission path keeps the leading dot.
// The definitive name therefore has to win even when it equals the written name,
// or the package-qualified candidate is tried first and a same-named local
// declaration captures the imported type.
func TestGenerateAbsoluteImportedRootTypeIsNotCapturedLocally(t *testing.T) {
	files := generateWithImports(t, map[string]string{
		"rooted.proto": `syntax = "proto3";
enum Type {
  TYPE_A = 0;
}
`,
		"consumer.proto": `syntax = "proto3";
package game;
import "rooted.proto";
message Type {
  int32 n = 1;
}
message Holder {
  .Type imported_root = 1;
}
`,
	}, "consumer.proto")

	f := findFile(files, "ConsumerHolder")
	require.NotNil(t, f, "missing ConsumerHolder; got %v", classNames(files))
	got := mustSource(t, *f)

	assert.Contains(t, got, "var _imported_root: RootedType.Type = 0 as RootedType.Type",
		"the absolute reference names the imported root-package enum\n%s", got)
	assert.NotContains(t, got, "var _imported_root: ConsumerType",
		"a same-named local message captured the imported root type\n%s", got)
}

// TestGeneratePackagedReferenceIsNotCapturedByAnImport covers the interaction
// between the innermost-first walk and the shared declaration index: that index
// holds imported declarations too, so a scope-relative candidate spelled without
// the package could match another file's type. Protobuf would never resolve a
// packaged file's reference that way.
func TestGeneratePackagedReferenceIsNotCapturedByAnImport(t *testing.T) {
	importedFile := parseProtoSource(t, `syntax = "proto3";
message Outer {
  message Mode {
    int32 z = 1;
  }
}
`, "imported.proto")
	mainFile := parseProtoSource(t, `syntax = "proto3";
package game.v1;
import "imported.proto";
enum Mode {
  MODE_A = 0;
}
message Outer {
  Mode field = 1;
}
`, "main.proto")

	files, err := generator.Generate(mainFile, "main.proto", []generator.FileEntry{
		{File: importedFile, Filename: "imported.proto"},
	})
	require.NoError(t, err, "generate")

	f := findFile(files, "MainOuter")
	require.NotNil(t, f, "missing MainOuter; got %v", classNames(files))
	got := mustSource(t, *f)

	assert.Contains(t, got, "var _field: MainMode.Mode = 0 as MainMode.Mode",
		"the reference must resolve to the local enum, not an import\n%s", got)
	assert.NotContains(t, got, "ImportedOuterMode",
		"an imported declaration captured a reference to a local type\n%s", got)
}

// TestGenerateFullyQualifiedMessageIsNotAnEnum checks that an explicitly
// package-qualified reference keeps the kind of the declaration it names, so a
// qualified message does not pick up enum defaults and a qualified enum does not
// pick up a message constructor.
func TestGenerateFullyQualifiedMessageIsNotAnEnum(t *testing.T) {
	got := sourceOf(t, `syntax = "proto3";
package game.v1;
enum Mode {
  MODE_A = 0;
}
message Outer {
  message Mode {
    int32 x = 1;
  }
  game.v1.Outer.Mode qualified = 1;
  game.v1.Mode explicit = 2;
}
`, "shadow.proto", "ShadowOuter")

	assert.Contains(t, got, "var _qualified: ShadowOuterMode = null",
		"a fully qualified message reference must stay a message\n%s", got)
	assert.Contains(t, got, "func new_qualified()",
		"a fully qualified message reference needs a constructor accessor\n%s", got)
	assert.Contains(t, got, "var _explicit: ShadowMode.Mode = 0 as ShadowMode.Mode",
		"a fully qualified enum reference must stay an enum\n%s", got)
	assert.NotContains(t, got, "func new_explicit()",
		"an enum field must not get a message constructor accessor\n%s", got)
}

// nestedEnumShadowsTopLevelEnumSource is Repro B: a nested enum shadowing a
// same-named top-level enum. It is the shape that proves the text-format
// helpers follow the binding, because the two enums declare different value
// names.
const nestedEnumShadowsTopLevelEnumSource = `syntax = "proto3";
package game.v1;
enum Mode {
  TOP_LEVEL_ZERO = 0;
}
message Outer {
  enum Mode {
    NESTED_ZERO = 0;
  }
  Mode field = 1;
}
`

// TestGenerateEnumValueHelpersFollowTheBinding covers the text-format helpers,
// which select an enum's values independently of the field's declared type.
// Selecting by bare name searches top-level enums first, so the field could be
// typed as the nested enum while _get_enum_name_field matched on the top-level
// enum's value names — names the nested enum does not declare.
func TestGenerateEnumValueHelpersFollowTheBinding(t *testing.T) {
	got := sourceOf(t, nestedEnumShadowsTopLevelEnumSource, "shadow.proto", "ShadowOuter")

	assert.Contains(t, got, "var _field: Mode = 0 as Mode",
		"the field must bind to the nested enum\n%s", got)
	assert.NotContains(t, got, "ShadowMode",
		"the reference must not resolve to the top-level enum wrapper\n%s", got)
	assert.Contains(t, got, "Mode.NESTED_ZERO",
		"the enum helpers must emit the bound enum's value names\n%s", got)
	assert.NotContains(t, got, "TOP_LEVEL_ZERO",
		"the enum helpers must not emit the shadowed enum's value names\n%s", got)
}

// TestGenerateNestedEnumReferencedFromDeeperMessage covers a reference to
// another message's nested enum, which needs no shadowing to go wrong: nested
// messages are flattened into sibling classes, so the enum's bare name is not in
// the referring class's scope and has to be qualified with the declaring class.
func TestGenerateNestedEnumReferencedFromDeeperMessage(t *testing.T) {
	got := sourceOf(t, `syntax = "proto3";
package game.v1;
message Outer {
  enum Status {
    STATUS_OK = 0;
  }
  message Inner {
    Status s = 1;
  }
}
`, "sibling.proto", "SiblingOuterInner")

	assert.Contains(t, got, "var _s: SiblingOuter.Status = 0 as SiblingOuter.Status",
		"a nested enum must be qualified with its declaring class\n%s", got)
	assert.NotContains(t, got, ": Status",
		"a bare nested-enum name is not in the referring class's scope\n%s", got)
	assert.Contains(t, got, "SiblingOuter.Status.STATUS_OK",
		"the enum helpers must use the same qualified name\n%s", got)
}

// TestGenerateMapValueAndOneofFieldResolveLikeFields covers map values and
// oneof fields, which carry their own type metadata and so can disagree with a
// plain field's binding for the same reference.
func TestGenerateMapValueAndOneofFieldResolveLikeFields(t *testing.T) {
	got := sourceOf(t, `syntax = "proto3";
package game.v1;
enum Mode {
  MODE_A = 0;
}
message Outer {
  message Mode {
    int32 y = 1;
  }
  map<string, Mode> modes = 1;
  oneof choice {
    Mode one = 2;
  }
}
`, "shadow.proto", "ShadowOuter")

	assert.Contains(t, got, "Dictionary[String, ShadowOuterMode]",
		"a map value must bind to the nested message\n%s", got)
	assert.Contains(t, got, "var _one: ShadowOuterMode = null",
		"a oneof field must bind to the nested message\n%s", got)
	assert.Contains(t, got, "func set_one(value: ShadowOuterMode)",
		"a oneof message field's setter must take the bound message class\n%s", got)
	assert.Contains(t, got, "_one = ShadowOuterMode.new()",
		"a oneof message field must be constructed, not assigned an enum value\n%s", got)
	assert.NotContains(t, got, "ShadowMode",
		"neither reference may resolve to the file-scope enum wrapper\n%s", got)
	assert.NotContains(t, got, "= 0 as ShadowOuterMode",
		"a message-typed value must not get an enum default\n%s", got)
}

// TestGenerateNestedMessageShadowsTopLevelMessage covers two messages, where a
// wrong binding stays valid GDScript and so shows up only as the wrong wire
// shape and accessors.
func TestGenerateNestedMessageShadowsTopLevelMessage(t *testing.T) {
	got := sourceOf(t, `syntax = "proto3";
package game.v1;
message Thing {
  int32 a = 1;
}
message Outer {
  message Thing {
    int32 b = 1;
  }
  Thing field = 1;
}
`, "shadow.proto", "ShadowOuter")

	assert.Contains(t, got, "var _field: ShadowOuterThing = null",
		"a nested message must shadow a same-named top-level message\n%s", got)
	assert.NotContains(t, got, "ShadowThing",
		"the reference must not resolve to the top-level message class\n%s", got)
}

// TestGenerateUnresolvableReferenceIsAnError covers the fail-closed contract: a
// reference that names no declaration must fail the build rather than emit a
// plausible-looking name that Godot rejects at load time with no mention of the
// generator.
func TestGenerateUnresolvableReferenceIsAnError(t *testing.T) {
	file := &ast.ProtoFile{
		Syntax:  "proto3",
		Package: "game.v1",
		Messages: []*ast.Message{{
			Name: "Uses",
			Fields: []*ast.Field{{
				FieldType: "Stranger",
				Name:      "x",
				Number:    1,
			}},
		}},
	}

	files, err := generator.Generate(file, "main.proto", nil)
	require.Error(t, err, "an unresolvable reference must fail the build")
	assert.Nil(t, files, "no files may be returned when resolution failed")
	for _, want := range []string{"Stranger", "Uses", "main.proto"} {
		assert.Contains(t, err.Error(), want,
			"the error must name the reference, its scope and the file")
	}
}

// TestGenerateNestedShadowsNestedWithoutFileScopeName pins behavior that is
// already correct today: with no file-scope declaration of the name, the
// innermost nested declaration already won, and it must keep winning.
func TestGenerateNestedShadowsNestedWithoutFileScopeName(t *testing.T) {
	got := sourceOf(t, `syntax = "proto3";
package game.v1;
message Outer {
  message Thing {
    int32 a = 1;
  }
  message Inner {
    message Thing {
      int32 b = 1;
    }
    Thing field = 1;
  }
}
`, "shadow.proto", "ShadowOuterInner")

	assert.Contains(t, got, "var _field: ShadowOuterInnerThing = null",
		"the innermost nested declaration must win\n%s", got)
	assert.NotContains(t, got, "ShadowOuterThing = null",
		"the enclosing message's nested declaration must not capture it\n%s", got)
}

// TestGenerateNestedDeclarationVersusDifferentPackageImport pins behavior that
// is already correct today: an import in a different package is referenced
// through its package, so it collides with no scope candidate and a same-named
// local nested declaration stays separate.
func TestGenerateNestedDeclarationVersusDifferentPackageImport(t *testing.T) {
	files := generateWithImports(t, map[string]string{
		"ext.proto": `syntax = "proto3";
package other;
enum Mode {
  MODE_X = 0;
}
`,
		"main.proto": `syntax = "proto3";
package game.v1;
import "ext.proto";
message Outer {
  message Mode {
    int32 q = 1;
  }
  other.Mode imported = 1;
  Mode local = 2;
}
`,
	}, "main.proto")

	f := findFile(files, "MainOuter")
	require.NotNil(t, f, "missing MainOuter; got %v", classNames(files))
	got := mustSource(t, *f)

	assert.Contains(t, got, "var _imported: ExtMode.Mode = 0 as ExtMode.Mode",
		"the qualified reference names the imported enum\n%s", got)
	assert.Contains(t, got, "var _local: MainOuterMode = null",
		"the unqualified reference names the local nested message\n%s", got)
}

// TestValidateIntermediatePackageReferenceStillRejected pins the deviation this
// change deliberately leaves alone: the candidate set has no intermediate
// package-scope form, so a reference from game.v1 to a declaration in the
// parent package game is rejected by the CLI validator exactly as before. The
// plugin path resolves the same schema through the descriptor's own binding,
// which the companion plugin test covers.
func TestValidateIntermediatePackageReferenceStillRejected(t *testing.T) {
	directory := t.TempDir()
	sources := map[string]string{
		"base.proto": `syntax = "proto3";
package game;
enum Mode {
  MODE_A = 0;
}
`,
		"leaf.proto": `syntax = "proto3";
package game.v1;
import "base.proto";
message Holder {
  Mode m = 1;
}
`,
	}
	for name, source := range sources {
		require.NoError(t, os.WriteFile(filepath.Join(directory, name), []byte(source), 0o600))
	}
	leaf := parseProtoSource(t, sources["leaf.proto"], "leaf.proto")
	_, err := importer.ResolveExternalWithFiles(
		leaf, filepath.Join(directory, "leaf.proto"), &importer.OSFS{BaseDir: directory},
	)
	require.NoError(t, err, "resolve imports")

	errs := validator.Validate(leaf, "leaf.proto")
	require.NotEmpty(t, errs, "an intermediate-package reference is still rejected")
	joined := make([]string, 0, len(errs))
	for _, e := range errs {
		joined = append(joined, e.Error())
	}
	assert.Contains(t, strings.Join(joined, "\n"), `Undefined type "Mode"`,
		"the rejection message must be unchanged")
}
