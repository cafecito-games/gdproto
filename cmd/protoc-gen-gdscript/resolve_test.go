package main

import (
	"strings"
	"testing"
)

// TestRunResolvesReferencesLikeProtobuf drives the plugin path — protoc builds
// the descriptor set, the converter turns it into an AST, and run generates —
// for the shapes where a type reference can bind to more than one same-named
// declaration.
//
// The plugin path has to be covered separately from the direct-CLI path because
// the two populate the generator's inputs differently: the descriptor converter
// always records a SourceFile and the authoritative resolved type path, and it
// reduces a same-file reference to a scope-relative name, which discards the
// leading dot of an absolute reference. A binding that is right on one path can
// therefore be wrong on the other, and the halves that disagree produce
// GDScript that does not parse — an enum-typed field assigned null, or .new()
// called on an enum.
func TestRunResolvesReferencesLikeProtobuf(t *testing.T) {
	cases := []struct {
		name string
		// sources is the proto file set, keyed by filename.
		sources map[string]string
		// generate is the file protoc is asked to generate.
		generate string
		// class is the generated .pb.gd whose content is asserted on.
		class string
		// wantContains and wantAbsent are substrings of the generated
		// GDScript that must and must not appear.
		wantContains []string
		wantAbsent   []string
	}{
		{
			name: "nested message shadows a top-level enum",
			sources: map[string]string{"shadow.proto": `syntax = "proto3";
package game.v1;
enum Mode { MODE_A = 0; }
message Outer {
  message Mode { int32 x = 1; }
  Mode field = 1;
}
`},
			generate: "shadow.proto",
			class:    "ShadowOuter.pb.gd",
			wantContains: []string{
				"var _field: ShadowOuterMode = null",
				"func new_field()",
				"_field.to_bytes()",
			},
			wantAbsent: []string{"ShadowMode", "0 as ", "encode_varint(_field)"},
		},
		{
			name: "nested enum shadows a top-level message",
			sources: map[string]string{"shadow.proto": `syntax = "proto3";
package game.v1;
message Kind { int32 x = 1; }
message Outer {
  enum Kind { KIND_NESTED = 0; }
  Kind field = 1;
}
`},
			generate: "shadow.proto",
			class:    "ShadowOuter.pb.gd",
			wantContains: []string{
				"var _field: Kind = 0 as Kind",
				"Kind.KIND_NESTED",
			},
			wantAbsent: []string{"ShadowKind", "func new_field()"},
		},
		{
			name: "the walk stops at the nearest enclosing scope",
			sources: map[string]string{"shadow.proto": `syntax = "proto3";
package game.v1;
enum Mode { MODE_A = 0; }
message Outer {
  message Mode { int32 outer_value = 1; }
  message Middle {
    message Mode { bool middle_value = 1; }
    message Inner { Mode field = 1; }
  }
}
`},
			generate:     "shadow.proto",
			class:        "ShadowOuterMiddleInner.pb.gd",
			wantContains: []string{"var _field: ShadowOuterMiddleMode = null"},
			wantAbsent:   []string{"ShadowOuterMode", "0 as "},
		},
		{
			name: "an absolute same-file reference keeps the descriptor's binding",
			sources: map[string]string{"game.proto": `syntax = "proto3";
package game.v1;
enum Mode { MODE_A = 0; }
message Outer {
  message Mode { int32 y = 1; }
  .game.v1.Mode absolute = 1;
}
`},
			generate:     "game.proto",
			class:        "GameOuter.pb.gd",
			wantContains: []string{"var _absolute: GameMode.Mode = 0 as GameMode.Mode"},
			wantAbsent:   []string{"var _absolute: GameOuterMode", "func new_absolute()"},
		},
		{
			name: "enum value helpers follow a nested enum binding",
			sources: map[string]string{"shadow.proto": `syntax = "proto3";
package game.v1;
enum Mode { TOP_LEVEL_ZERO = 0; }
message Outer {
  enum Mode { NESTED_ZERO = 0; }
  Mode field = 1;
}
`},
			generate: "shadow.proto",
			class:    "ShadowOuter.pb.gd",
			wantContains: []string{
				"var _field: Mode = 0 as Mode",
				"Mode.NESTED_ZERO",
			},
			wantAbsent: []string{"ShadowMode", "TOP_LEVEL_ZERO"},
		},
		{
			name: "map values and oneof fields resolve like plain fields",
			sources: map[string]string{"shadow.proto": `syntax = "proto3";
package game.v1;
enum Mode { MODE_A = 0; }
message Outer {
  message Mode { int32 y = 1; }
  map<string, Mode> modes = 1;
  oneof choice { Mode one = 2; }
}
`},
			generate: "shadow.proto",
			class:    "ShadowOuter.pb.gd",
			wantContains: []string{
				"Dictionary[String, ShadowOuterMode]",
				"var _one: ShadowOuterMode = null",
				"_one = ShadowOuterMode.new()",
			},
			wantAbsent: []string{"ShadowMode", "= 0 as ShadowOuterMode"},
		},
		{
			name: "a nested enum referenced from a deeper message is qualified",
			sources: map[string]string{"sibling.proto": `syntax = "proto3";
package game.v1;
message Outer {
  enum Status { STATUS_OK = 0; }
  message Inner { Status s = 1; }
}
`},
			generate: "sibling.proto",
			class:    "SiblingOuterInner.pb.gd",
			wantContains: []string{
				"var _s: SiblingOuter.Status = 0 as SiblingOuter.Status",
				"SiblingOuter.Status.STATUS_OK",
			},
			wantAbsent: []string{": Status"},
		},
		{
			name: "nested message shadows a top-level message",
			sources: map[string]string{"shadow.proto": `syntax = "proto3";
package game.v1;
message Thing { int32 a = 1; }
message Outer {
  message Thing { int32 b = 1; }
  Thing field = 1;
}
`},
			generate:     "shadow.proto",
			class:        "ShadowOuter.pb.gd",
			wantContains: []string{"var _field: ShadowOuterThing = null"},
			wantAbsent:   []string{"ShadowThing"},
		},
		{
			name: "a reference to the parent package still resolves through the descriptor",
			sources: map[string]string{
				"base.proto": `syntax = "proto3";
package game;
enum Mode { MODE_A = 0; }
`,
				"leaf.proto": `syntax = "proto3";
package game.v1;
import "base.proto";
message Holder { Mode m = 1; }
`,
			},
			generate:     "leaf.proto",
			class:        "LeafHolder.pb.gd",
			wantContains: []string{"var _m: BaseMode.Mode = 0 as BaseMode.Mode"},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			request := buildRequestFromDescriptorSet(t, []string{testCase.generate}, testCase.sources)
			response := runPluginRequest(t, request)

			var got string
			for _, file := range response.GetFile() {
				if file.GetName() == testCase.class {
					got = file.GetContent()
				}
			}
			if got == "" {
				t.Fatalf("%s was not generated; got %v", testCase.class, responseFilenames(response))
			}
			for _, want := range testCase.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("missing %q\n%s", want, got)
				}
			}
			for _, absent := range testCase.wantAbsent {
				if strings.Contains(got, absent) {
					t.Errorf("unexpected %q\n%s", absent, got)
				}
			}
		})
	}
}
