# M5 — Generator + CLI Parity Implementation Plan

> **For agentic workers:** Use superpowers-extended-cc:subagent-driven-development.

**Goal:** Port `gdproto/generator/` (4609 lines in `gdscript.py` + 399 in `templates.py` + 398 in `gdproto_helpers.py`) to `internal/generator` and wire the CLI end-to-end (lex → parse → import-resolve → validate → generate → write). End state: `gogdproto example.proto -o out.gd` produces byte-identical output to the Python `gdproto` tool for the staged `examples/example.proto` fixture.

**Architecture:** `Generate(file *ast.ProtoFile, sourceName string) (*gdast.ClassDefinition, error)` — pure function. Templates (proto core helpers + ProtobufError enum) are embedded as `gdast.RawStatement`/string constants. Generator builds a gdast tree by traversing the proto AST and assembling sub-trees per message/enum/field. CLI runs the full pipeline.

**Tech Stack:** Standard library + `internal/{lexer,parser,ast,validator,importer,gdast}`.

**Reference:**
- `~/foss/gdproto/src/gdproto/generator/gdscript.py` (4609 lines, ~60 methods).
- `~/foss/gdproto/src/gdproto/generator/templates.py` (399 lines — embedded GDScript code as constants).
- `~/foss/gdproto/src/gdproto/generator/gdproto_helpers.py` (398 lines — gdast-builder helpers for common patterns like `encode_field_call`).
- `~/foss/gdproto/examples/example.proto` and `example.gd` (golden fixture, copied to `examples/`).
- `~/foss/gdproto/src/gdproto/cli.py` (250+ lines — CLI orchestration).

**GitHub tracking:** [issue #6](https://github.com/cafecito-games/gogdproto/issues/6). One PR per milestone (`feat/m5-generator`). **Auto-merge enabled only on the final task.**

---

## Strategy

Because the generator is huge, the **golden-file diff is the primary verifier**. Each task incrementally extends the generator and shrinks the diff against `examples/golden.gd`. We don't enumerate test cases per method — we use the golden as ground truth and target byte-identical output.

The implementer must read `gdscript.py`, `templates.py`, and `gdproto_helpers.py` and **port verbatim**. Where Python uses a method, write the equivalent Go function/method. Where Python embeds raw GDScript strings, embed identical strings in Go constants.

**Key idiomatic translations:**
- Python `"\t" * n` → `strings.Repeat("\t", n)` or `gdast.indent(n)`.
- Python f-strings → `fmt.Sprintf` or string concatenation.
- Python `dict[str, Any]` → `map[string]any`.
- Python class with methods + state → Go struct with methods.
- Python optional kwargs → Go method overloads or option structs.
- Python tuple unpacking → multiple return values.

---

## File Structure

| Path                                       | Responsibility                                              |
|--------------------------------------------|-------------------------------------------------------------|
| `internal/generator/doc.go`                | Existing.                                                   |
| `internal/generator/templates.go`          | Constants for ProtobufError enum body, protobuf_core helper definitions, all literal GDScript snippets from Python's `templates.py`. |
| `internal/generator/helpers.go`            | Helper functions porting `gdproto_helpers.py` (encode_field_call, decode_field_call, check_decode_error, etc.). |
| `internal/generator/types.go`              | Type-mapping tables (proto type → GDScript type, wire types). Possibly shared with `internal/prototypes`. |
| `internal/generator/generator.go`          | Generator struct, exported `Generate`, top-level orchestration: `_collect_dependencies`, `_collect_enum_types`, `_to_snake_case`, `_get_qualified_type_name`, `is_enum_type`. |
| `internal/generator/messages.go`           | Per-message generation: `generate_message`, `generate_field_declarations`, `_get_field_default_value`, `_build_oneof_clear_statements`. |
| `internal/generator/accessors.go`          | All accessor methods: scalar, message, repeated, map, oneof. Maps to `generate_*_accessors` family in Python. |
| `internal/generator/serialize.go`          | `generate_to_bytes`, `generate_field_serialization`, `generate_oneof_serialization`, `generate_value_serialization`, `generate_map_serialization`. |
| `internal/generator/deserialize.go`        | `generate_from_bytes`, `generate_field_deserialization`, `_generate_single_field_deser`, `_generate_repeated_field_deser`, `_generate_map_field_deser`, `_build_tag_reading_statements`. |
| `internal/generator/tostring.go`           | `generate_to_string`, `generate_enum_name_helpers`. |
| `internal/generator/generator_test.go`     | Unit tests + golden-file diff. |
| `examples/example.proto` (already copied)  | Test fixture.                                               |
| `examples/golden.gd` (already copied)      | Expected output (from Python tool).                         |
| `cmd/gogdproto/main.go` (modify)           | Wire end-to-end pipeline.                                   |
| `internal/cli/root.go` (modify)            | Add `INPUT` positional + `-o/--output` flag handling, call into pipeline. |

The exported API:

```go
package generator

func Generate(file *ast.ProtoFile, sourceName string) (*gdast.ClassDefinition, error)
```

The `sourceName` is the input filename (stem used to derive the class_name directive — e.g., `example.proto` → `Example`).

---

## Task 0: Generator skeleton + templates + CLI wiring

**Goal:** Stand up `Generate` returning a minimal `*gdast.ClassDefinition` for the empty case. Embed `ProtobufError` enum + `protobuf_core` helpers as constants. Wire the CLI to run the full pipeline. Output won't be correct yet — that's later tasks. **End state: `gogdproto example.proto -o out.gd` runs without error and produces a partial file.**

**Files:** `templates.go`, `generator.go`, `messages.go` (stubs), `cmd/gogdproto/main.go`, `internal/cli/root.go`, `generator_test.go`.

**Acceptance:**
- [ ] `Generate(emptyFile, "example.proto")` returns a `*gdast.ClassDefinition` with `ClassNameDirective == "Example"`, `Extends == "RefCounted"`, statements containing the ProtobufError enum and protobuf_core helpers.
- [ ] CLI `gogdproto example.proto -o /tmp/out.gd` runs without error.
- [ ] Output file starts with `class_name Example\n\nextends RefCounted\n\nenum ProtobufError {`.

**Verify:** Run end-to-end CLI; cmp expected prefix.

**Steps:**

1. Read `gdproto/generator/gdscript.py` lines 1-330 (init, generate, generate_proto_core_utils, helpers).
2. Read `gdproto/generator/templates.py` end-to-end. Each function returns a `RawStatement` with embedded GDScript.
3. Read `gdproto/cli.py` for the orchestration.
4. Write `internal/generator/templates.go` with constants:
   ```go
   const errorEnumGDScript = "enum ProtobufError {\n\tNO_ERRORS = 0,\n\tVARINT_NOT_FOUND = -1,\n\t...\n}"
   const protobufCoreGDScript = "..."  // from templates.py:protobuf_core
   const textFormatHelpersGDScript = "..."  // from templates.py:text_format_helpers (if present)
   ```
   Or use `gdast.RawStatement{Code: "..."}` directly if the constant is consumed once.
5. Write `internal/generator/generator.go`:
   - `type generator struct { file *ast.ProtoFile; sourceName string; enumTypes map[string]bool; ... }`
   - `Generate` constructs the generator, calls `g.generate()`, returns the class definition.
   - `g.generate()` builds the top-level `*gdast.ClassDefinition`:
     - `ClassNameDirective: derivedFromStem(sourceName)`
     - `Extends: "RefCounted"`
     - `Statements`: starts with ProtobufError enum (as `RawStatement`), proto_core (as `RawStatement`), enum definitions, message classes (stub).
6. Stub `messages.go` with `generateMessage` returning a minimal `*gdast.ClassDefinition` with just the message name (no fields/methods yet). T2-T6 fill in real content.
7. Wire CLI: in `internal/cli/root.go`, replace `RunE: cmd.Help()` with logic that:
   - Reads input file
   - Lexes, parses, resolves imports (using OSFS), validates
   - If validator errors: print all to stderr, return error
   - Calls `generator.Generate`
   - Writes `class.ToGDScript(0)` to output file
   - Print `✓ Generated <output>` on success
   - Add `INPUT` positional arg and `-o/--output` flag (required).
   - Mimic Python CLI exit codes (0 success, 1 error, 130 interrupt).
8. Write a CLI test that invokes `cli.Execute([]string{"example.proto", "-o", tempFile}, ...)` and verifies the output starts with the expected prefix.
9. Commit + push:
   ```bash
   git add internal/generator/ cmd/ internal/cli/ examples/
   git commit -m "feat(generator): skeleton, templates, CLI end-to-end pipeline"
   git push origin feat/m5-generator
   ```

**DO NOT enable auto-merge.**

---

## Task 1: Enum generation + field declarations + simple accessor stubs

**Goal:** Generate proper enum definitions (top-level + nested). Generate field declarations (private `_field` vars with default values per type). Stub accessor methods (set_/get_) so the output structure matches but bodies are TODO placeholders.

**Files:** `messages.go`, `accessors.go` (stubs), `generator_test.go`.

**Acceptance:** Output for `example.proto` contains:
- `enum PlayerStatus { OFFLINE = 0, ONLINE = 1, AWAY = 2, IN_GAME = 3 }` at correct nesting.
- `class Player extends RefCounted:` declaration.
- Private vars per field: `var _username: String = ""`, `var _level: int = 0`, etc.
- Nested `Position` class with its private vars.
- Stub set_/get_ method signatures (bodies can be `pass` or returns).

The diff against golden.gd should now show only missing accessor bodies and serialization methods.

**Reference Python:** `gdscript.py:332-545` (generate_enum, generate_message, generate_field_declarations).

---

## Task 2: Scalar/message/repeated/map field accessors

**Goal:** Implement all `generate_*_accessors` methods. Output matches golden.gd for set_/get_/has_/clear_/add_/remove_ methods.

**Files:** `accessors.go`, plus helpers in `helpers.go`.

**Reference Python:** `gdscript.py:642-995` (scalar, message, repeated, map accessors), `gdproto_helpers.py:140-240` (helper builders).

**Acceptance:** Diff against golden.gd is small — only `to_bytes` and `from_bytes` and `to_string` missing.

---

## Task 3: to_bytes serialization

**Goal:** Implement `generate_to_bytes`, `generate_field_serialization`, `generate_oneof_serialization`, `generate_value_serialization`, `generate_map_serialization`. Output `to_bytes` method matches golden.

**Files:** `serialize.go` + additional helpers in `helpers.go`.

**Reference Python:** `gdscript.py:996-1453`.

**Acceptance:** `to_bytes` block in output is byte-identical to golden.

---

## Task 4: from_bytes deserialization

**Goal:** Implement `generate_from_bytes`, `_build_tag_reading_statements`, `generate_field_deserialization`, `_generate_single_field_deser`, `_generate_repeated_field_deser`, `_generate_map_field_deser`. Output `from_bytes` matches golden.

**Files:** `deserialize.go`.

**Reference Python:** `gdscript.py:1454-2831`.

**Acceptance:** `from_bytes` block matches golden.

---

## Task 5: to_string + enum_name_helpers + oneof handling

**Goal:** Implement `generate_to_string`, `generate_enum_name_helpers`, oneof-related code.

**Files:** `tostring.go` + finishing touches in `messages.go`.

**Reference Python:** `gdscript.py:2832-end`.

**Acceptance:** Diff against golden.gd is empty.

---

## Task 6: Golden-file diff test + push + close issue

**Goal:** Lock in byte-identical output via a `TestGoldenExample` test. Coverage ≥ 80% (the generator is huge; most error paths are difficult to exercise). Push, enable auto-merge, watch CI, close issue #6.

**Files:** `generator_test.go`.

**Test:**

```go
func TestGoldenExample(t *testing.T) {
    src, err := os.ReadFile("../../examples/example.proto")
    if err != nil { t.Fatal(err) }
    tokens, err := lexer.Tokenize(string(src), "example.proto")
    if err != nil { t.Fatal(err) }
    file, err := parser.Parse(tokens, "example.proto")
    if err != nil { t.Fatal(err) }
    if errs := validator.Validate(file, "example.proto"); len(errs) != 0 {
        t.Fatalf("validation errors: %+v", errs)
    }
    class, err := generator.Generate(file, "example.proto")
    if err != nil { t.Fatal(err) }
    got := class.ToGDScript(0)
    want, err := os.ReadFile("../../examples/golden.gd")
    if err != nil { t.Fatal(err) }
    if got != string(want) {
        // Compute a diff for nicer output.
        gotLines := strings.Split(got, "\n")
        wantLines := strings.Split(string(want), "\n")
        for i := 0; i < len(gotLines) && i < len(wantLines); i++ {
            if gotLines[i] != wantLines[i] {
                t.Errorf("first diff at line %d:\n  got:  %q\n  want: %q", i+1, gotLines[i], wantLines[i])
                break
            }
        }
        if len(gotLines) != len(wantLines) {
            t.Errorf("line counts differ: got %d, want %d", len(gotLines), len(wantLines))
        }
    }
}
```

**Steps:**

1. Run the test — it should pass after T0-T5.
2. If diff appears, fix the generator. Iterate.
3. Coverage check.
4. `task ci` green.
5. Commit + push.
6. Enable auto-merge: `gh pr merge <PR#> --auto --squash`.
7. Watch CI; verify merge.
8. Close issue #6.
9. Sync local main.

---

## Self-Review

**Risks:**

1. **Pure size.** The generator is the biggest component by far. Each task is hours of careful translation. Don't try to fast-path; read the Python and port faithfully.
2. **Embedded GDScript strings.** Tabs and newlines must match exactly. Use Go raw strings (backticks) for multi-line snippets to avoid escaping hell.
3. **Method signatures.** Python uses kwargs heavily; Go must use either separate methods or option structs. Default to separate methods named for the variation.
4. **State threading.** Python's `Generator` has a lot of instance state (`enum_types`, `defined_types`, etc.). Mirror as struct fields.
5. **Snake-case conversion** (`_to_snake_case`). Port the algorithm exactly.
6. **Naming convention for nested types.** Python uses `Outer.Inner` in qualified names but the actual GDScript class output uses nested `class Inner:` blocks. Match Python's behavior.
