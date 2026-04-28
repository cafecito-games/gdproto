# M3 — Validator + Importer Implementation Plan

> **For agentic workers:** Use superpowers-extended-cc:subagent-driven-development to execute task-by-task.

**Goal:** Port `gdproto/validator.py` (634 lines) to `internal/validator` and the import-resolution logic from `gdproto/cli.py` (`resolve_external_enum_types`) to `internal/importer`, with TDD red/green for every behavior.

**Architecture:** Validator runs on a `*ast.ProtoFile` and returns `[]ValidationError` (multi-error). Importer resolves `import` statements via an injectable `FS` interface, parses imported files, builds an external type registry, and annotates fields in the input AST with `SourceFile`/`FullTypePath`/`IsEnum`/etc. Both packages have no I/O of their own except via `FS` (importer only).

**Tech Stack:** Standard library + `internal/lexer`, `internal/parser`, `internal/ast`. The `FS` interface is small (read file, walk parent dirs).

**Reference:**
- Python validator: `~/foss/gdproto/src/gdproto/validator.py`
- Python tests: `~/foss/gdproto/tests/test_validator.py` (~30 tests in 12 classes), `tests/test_external_enums.py` (~15 tests)
- Python importer logic: `~/foss/gdproto/src/gdproto/cli.py:resolve_external_enum_types`

**GitHub tracking:** [issue #4](https://github.com/cafecito-games/gogdproto/issues/4). One PR per milestone (`feat/m3-validator-importer`). **Auto-merge enabled only on the final task.**

---

## File Structure

| Path                                     | Responsibility                                              |
|------------------------------------------|-------------------------------------------------------------|
| `internal/validator/doc.go`              | Existing.                                                   |
| `internal/validator/error.go`            | `ValidationError` type.                                     |
| `internal/validator/constants.go`        | Field-number bounds, scalar-type set, map-key set, keywords.|
| `internal/validator/validator.go`        | Exported `Validate` entrypoint, validator struct.           |
| `internal/validator/messages.go`         | `validateMessage`, `validateField`, `validateMapField`.     |
| `internal/validator/enums.go`            | `validateEnum`, `validateReserved`, `checkReservedConflicts`.|
| `internal/validator/types.go`            | `buildTypeRegistry`, `validateFieldType`.                   |
| `internal/validator/oneofs.go`           | `validateOneof`.                                            |
| `internal/validator/validator_test.go`   | Tests mirroring Python's test_validator.py.                 |
| `internal/importer/doc.go`               | Existing.                                                   |
| `internal/importer/fs.go`                | `FS` interface + `OSFS`.                                    |
| `internal/importer/importer.go`          | `ResolveExternal(file, inputPath, fs FS) error`.            |
| `internal/importer/importer_test.go`     | Tests with in-memory FS.                                    |

Exported API:

```go
// internal/validator
type ValidationError struct {
    File    string
    Line    int
    Column  int
    Message string
}
func (e *ValidationError) Error() string

func Validate(file *ast.ProtoFile, filename string) []ValidationError

// internal/importer
type FS interface {
    Read(path string) ([]byte, error)
    Exists(path string) bool
}
type OSFS struct { Root string }   // implements FS using os
func ResolveExternal(file *ast.ProtoFile, inputPath string, fs FS) error
```

`ValidationError` follows the `<file>:<line>:<col>: error: <message>` format. Empty `File` falls back to `<input>`.

---

## Task 0: ValidationError + Validator skeleton + validate_syntax + buildTypeRegistry

**Goal:** `Validate` exists and runs on an empty `*ProtoFile`. Catches non-proto3 syntax. Builds the type registry (used by later tasks).

**Files:** `error.go`, `constants.go`, `validator.go`, `types.go`, `validator_test.go`.

**Acceptance:**
- [ ] `Validate(file, "")` returns `nil` for an empty `*ProtoFile{Syntax: "proto3"}`.
- [ ] `Validate` for `Syntax: "proto2"` returns 1 error with `Unsupported syntax version` and message contains `proto2`.
- [ ] `ValidationError.Error()` formats correctly.
- [ ] `buildTypeRegistry` correctly enumerates top-level and nested types into `validator.definedTypes` (a `map[string]struct{}`). Test by exposing a helper or verifying via `validateFieldType` later.

**Tests:**

```go
package validator_test

import (
	"strings"
	"testing"

	"github.com/cafecito-games/gogdproto/internal/lexer"
	"github.com/cafecito-games/gogdproto/internal/parser"
	"github.com/cafecito-games/gogdproto/internal/validator"
)

// validate parses and validates source. Returns the error list.
func validate(t *testing.T, src string) []validator.ValidationError {
	t.Helper()
	tokens, err := lexer.Tokenize(src, "test.proto")
	if err != nil {
		t.Fatalf("lex: %v", err)
	}
	file, err := parser.Parse(tokens, "test.proto")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return validator.Validate(file, "test.proto")
}

func TestValidationErrorFormat(t *testing.T) {
	e := &validator.ValidationError{File: "x.proto", Line: 5, Column: 10, Message: "boom"}
	want := "x.proto:5:10: error: boom"
	if e.Error() != want {
		t.Fatalf("got %q, want %q", e.Error(), want)
	}
}

func TestValidationErrorDefaultFile(t *testing.T) {
	e := &validator.ValidationError{Line: 1, Column: 1, Message: "oops"}
	if !strings.Contains(e.Error(), "<input>") {
		t.Fatalf("got %q", e.Error())
	}
}

func TestProto3Valid(t *testing.T) {
	errs := validate(t, `syntax = "proto3"; message Foo {}`)
	if len(errs) != 0 {
		t.Fatalf("got %d errors: %+v", len(errs), errs)
	}
}

func TestProto2Rejected(t *testing.T) {
	errs := validate(t, `syntax = "proto2"; message Foo {}`)
	if len(errs) != 1 {
		t.Fatalf("got %d errors", len(errs))
	}
	if !strings.Contains(errs[0].Message, "Unsupported syntax version") || !strings.Contains(errs[0].Message, "proto2") {
		t.Errorf("got %q", errs[0].Message)
	}
}
```

**Implementation notes:**
- `error.go`: pattern matches `LexerError`/`ParserError`. Empty `File` → `<input>`.
- `constants.go`: port `MIN_FIELD_NUMBER`, `MAX_FIELD_NUMBER`, `RESERVED_START`, `RESERVED_END`, `validMapKeyTypes` (set of strings → use `map[string]struct{}` or `map[string]bool`), `scalarTypes`, `reservedKeywords`. All unexported in the package.
- `validator.go`: define unexported `validator` struct holding `file *ast.ProtoFile`, `filename string`, `errors []ValidationError`, `definedTypes map[string]bool`. Method `validate()` orchestrates: `validateSyntax()`, `buildTypeRegistry()`, then iterate enums and messages. Exported `Validate(file, filename) []ValidationError` constructs the struct and returns `v.errors` (or `nil` if empty).
- `types.go`: `buildTypeRegistry` recursive over messages (matches Python).

**Steps:**
1. Read plan + reference Python validator.py.
2. Write failing tests.
3. Run tests, observe red.
4. Write `error.go`, `constants.go`, `validator.go` (with stubs for enum/message validation), `types.go`.
5. Run tests, verify green.
6. `task ci` green.
7. Commit + push:
   ```bash
   git add internal/validator/
   git commit -m "feat(validator): error type, skeleton, syntax validation, type registry"
   git push origin feat/m3-validator-importer
   ```

**DO NOT enable auto-merge** — this is task 0 of 7.

---

## Task 1: Enum validation

**Goal:** `validateEnum` enforces all Python rules: duplicate value numbers (unless `allow_alias`), duplicate value names, first value must be zero, reserved-keyword names.

**Files:** `enums.go`, modify `validator.go` (call `validateEnum` for top-level enums), `validator_test.go`.

**Acceptance:** All of these tests pass:

```go
func TestEnumDuplicateNumber(t *testing.T) {
	src := `syntax = "proto3"; enum E { A = 0; B = 0; }`
	errs := validate(t, src)
	if len(errs) != 1 || !strings.Contains(errs[0].Message, "Duplicate enum value number") {
		t.Errorf("got %+v", errs)
	}
}

func TestEnumAllowAlias(t *testing.T) {
	src := `syntax = "proto3";
enum E {
    option allow_alias = true;
    A = 0;
    B = 0;
}`
	errs := validate(t, src)
	if len(errs) != 0 {
		t.Errorf("got %+v", errs)
	}
}

func TestEnumDuplicateName(t *testing.T) {
	src := `syntax = "proto3"; enum E { A = 0; A = 1; }`
	errs := validate(t, src)
	found := false
	for _, e := range errs {
		if strings.Contains(e.Message, "Duplicate enum value name") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected duplicate-name error, got %+v", errs)
	}
}

func TestEnumFirstNotZero(t *testing.T) {
	src := `syntax = "proto3"; enum E { A = 1; B = 2; }`
	errs := validate(t, src)
	if len(errs) != 1 || !strings.Contains(errs[0].Message, "First enum value in proto3 must be zero") {
		t.Errorf("got %+v", errs)
	}
}

func TestEnumNameKeyword(t *testing.T) {
	src := `syntax = "proto3"; enum message { A = 0; }`
	// "message" is a keyword and won't even parse; use a case-insensitive variant.
	// Try "Message" — should pass parsing (TokenIdentifier) and fail validation
	// via the case-insensitive keyword check.
	src = `syntax = "proto3"; enum Message { A = 0; }`
	errs := validate(t, src)
	found := false
	for _, e := range errs {
		if strings.Contains(e.Message, "reserved keyword") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected reserved-keyword error, got %+v", errs)
	}
}
```

**Implementation notes:** Port `validate_enum` from `validator.py:223-278` line-by-line. Use `strings.EqualFold` or `strings.ToLower` for the keyword check.

**Steps:** standard TDD (red → write `enums.go` and wire into `validator.go` → green → commit + push).

---

## Task 2: Field validation + Map field validation + check_reserved_conflicts

**Goal:** `validateField` and `validateMapField` enforce field-number bounds, reserved 19000-19999 range, duplicate numbers/names, reserved-keyword names, and conflicts with reserved statements. Map fields additionally validate the key type.

**Files:** `messages.go` (will partially overlap with T4 — keep it for now), modify `validator.go`, modify `validator_test.go`.

**Acceptance:** ~12 tests covering:
- Valid field numbers (1, 100, 536870911).
- Duplicate field numbers within a message → 1 error per pair.
- Field number 0 → "out of valid range".
- Field number 536870912 → "out of valid range".
- Field number in 19000-19999 → "in reserved range".
- Duplicate field name → 1 error.
- Field name `message` (keyword) → "reserved keyword".
- Field number colliding with `reserved 5;` → "is reserved".
- Field number colliding with `reserved 4 to 8;` → "conflicts with reserved range".
- Field name colliding with `reserved "foo";` → "is reserved".
- Map valid: `map<string, int32> m = 1;`.
- Map invalid key: `map<float, int32> m = 1;` → "Invalid map key type".

Test source patterns from Python `tests/test_validator.py:TestFieldNumbers`, `TestFieldNames`, `TestMapValidation`, `TestReservedFields`. Port verbatim where possible.

**Implementation notes:** Port `validate_field`, `validate_map_field`, `check_reserved_conflicts`. Field-numbers/names must be tracked in passing-by-pointer `map`s so multiple field validations share state within a message. **Note about `validate_field_type`**: the type-resolution call exists in T3 (don't try to resolve types in this task; either skip the call or stub `validateFieldType` to succeed always). T3 lands real type resolution.

**Steps:** standard TDD. Commit message: `feat(validator): field and map-field validation with reserved conflicts`.

---

## Task 3: validateFieldType (type resolution)

**Goal:** Resolve type references against the `definedTypes` registry. Handle scalars, simple message types, dotted types, absolute (`.pkg.Foo`) types, package-qualified names, and nested-scope lookup. Mark imported types as resolved (`source_file != ""` in Python — translates to `field.SourceFile != ""` in our AST).

**Files:** `types.go`, `validator_test.go`.

**Acceptance:** ~6 tests:
- Scalar `int32` always valid.
- Simple message type `Inner` defined in same file → valid.
- Dotted type `Outer.Inner` defined → valid.
- Absolute type `.pkg.Inner` → strip leading `.`, look up in `defined_types`.
- Undefined type `MissingType` → error containing `Undefined type "MissingType"`.
- Field with `SourceFile != ""` (imported) skips local resolution → valid.

**Implementation notes:** Port `validate_field_type` from `validator.py:515-567`. Wire it into `validateField`/`validateMapField`'s type checks (replace any T2 stubs).

**Steps:** standard TDD. Commit: `feat(validator): field type resolution`.

---

## Task 4: validateOneof + validateReserved + validateMessage orchestration

**Goal:** `validateOneof` validates oneof fields (no repeated). `validateReserved` checks range validity (start ≤ end, within field-number bounds). `validateMessage` orchestrates: keyword check, nested enums/messages recursion, oneofs, fields, maps, reserved.

**Files:** `oneofs.go`, modify `enums.go` (`validateReserved`), modify `messages.go` (or move orchestration to `validator.go`).

**Acceptance:** ~6 tests:
- Oneof with repeated field → "Oneof field cannot be repeated" error (parser already rejects this; if so, the test source needs to bypass parser by hand-crafted AST, OR rely on the parser's rejection. **Easier:** skip the test if parser already rejects, and document it.)
- Reserved range with start > end → "Invalid reserved range".
- Reserved range exceeding bounds → "out of valid field number range".
- Nested message with all rules applies recursively (test a 2-level deep nest).
- Top-level message keyword name → "reserved keyword".

**Implementation notes:** Port `validate_message`, `validate_oneof`, `validate_reserved`. Make sure recursion on nested messages uses correct scope (`Outer.Inner`).

**Steps:** standard TDD. Commit: `feat(validator): oneof, reserved, message orchestration`.

---

## Task 5: Importer — FS interface + ResolveExternal

**Goal:** Port `resolve_external_enum_types` (from `cli.py:28-200`) into `internal/importer`. The function reads imported `.proto` files, parses them, builds a type registry mapping `Package.Type` → `(sourceFile, isEnum)`, then walks the input AST and annotates `Field`/`MapField` with `SourceFile`/`FullTypePath`/`IsEnum`/`ValueIsEnum`/etc.

**Files:** `internal/importer/fs.go`, `internal/importer/importer.go`, `internal/importer/importer_test.go`.

**Acceptance:** ~6 tests using an in-memory FS:
- File imports `other.proto` defining `enum E { ... }`. Field of type `E` in input gets `SourceFile = "other.proto"`, `IsEnum = true`.
- File imports `other.proto` defining `message M { ... }`. Field of type `M` gets `SourceFile = "other.proto"`, `IsEnum = false`.
- Package-qualified imported type: `package pkg; enum E { ... }` in `other.proto`; current file imports it; field type `pkg.E` resolves; field type `E` (unqualified, same package) resolves.
- Map field value type from imported file: `map<string, ImportedEnum>` gets `ValueIsEnum = true`, `ValueSourceFile = "other.proto"`.
- Nested type in import: `message M { enum N { ... } }` in `other.proto`; field type `M.N` resolves with `IsEnum = true`.
- Missing import: skipped silently (validator will catch missing types).

**FS interface:**

```go
type FS interface {
    Read(path string) ([]byte, error)
    Exists(path string) bool
}

// OSFS uses os.ReadFile and walks parent directories from a base path.
// (Mirrors the Python "walk up parent directories" strategy.)
type OSFS struct {
    BaseDir string
}

func (f *OSFS) Read(path string) ([]byte, error)
func (f *OSFS) Exists(path string) bool

// memFS for tests:
type memFS struct {
    files map[string][]byte
}
```

**Strategy:** the Python version uses three lookup strategies: relative-to-input-dir, walk-up parent directories, and fallback by basename. Mirror this in `OSFS` and provide a similar interface for tests.

**Implementation notes:** This is the most complex M3 task. The function:
1. For each `*ast.Import` in the file:
   a. Locate the file via FS.
   b. Lex + parse it (recurse into our own lexer/parser).
   c. Collect top-level + nested types into a registry.
2. Walk the input file's messages, fields, map fields, oneof fields. For each whose `FieldType` matches an imported type, set `SourceFile`, `FullTypePath`, `IsEnum`.
3. Same for `MapField.ValueType` → `ValueSourceFile`/`FullValueTypePath`/`ValueIsEnum`.

Reference Python source: `cli.py:28-200` (the `resolve_external_enum_types` function and its helpers).

**Steps:** standard TDD with focus on test scaffolding (in-memory FS makes unit tests fast). Commit: `feat(importer): import resolution and external type marking`.

---

## Task 6: Integration tests + coverage + push + close issue

**Goal:** Validator + importer integration tests using realistic proto fixtures. Coverage targets: validator ≥ 90%, importer ≥ 85%. Push, enable auto-merge on the PR (FIRST time for M3), watch CI, verify merge, close issue #4.

**Files:** Add a few integration tests in both `validator_test.go` and `importer_test.go` exercising paths not yet covered.

**Acceptance:**
- A complex example combining imports + enums + messages + maps validates with 0 errors.
- Coverage ≥ 90% (validator), ≥ 85% (importer).
- `task ci` green.
- PR auto-merges; issue #4 closed.

**Steps:**
1. Add integration tests.
2. Coverage check; fill gaps.
3. `task ci`.
4. Commit + push.
5. **Enable auto-merge:** `gh pr merge <PR#> --auto --squash`
6. Watch CI.
7. Verify PR merged.
8. Close issue #4.
9. Pull main locally.

---

## Self-Review

**Spec coverage:** All Python validator test classes are mapped to T0–T4. Importer covers `tests/test_external_enums.py`. Integration in T6.

**Type/identifier consistency:**
- `Validate` (exported, free function) returns `[]ValidationError`.
- `validator` (unexported struct) holds state.
- `definedTypes` map of string → bool (or `struct{}`).
- `ResolveExternal(file, inputPath, fs FS) error` — signature locked.
- `FS` interface with `Read`, `Exists`.

**Risks:**
1. The Python validator returns errors as a list in source order. We must preserve order for determinism. Append errors as encountered; don't sort.
2. `validateFieldType`'s scope-walking logic for nested types is subtle. Port carefully (see Python:515-567).
3. The importer's "walk up parent dirs" strategy is filesystem-dependent. The `OSFS` implementation must match Python's behavior; tests use `memFS` and don't exercise the walk-up logic. Add a small `OSFS` integration test that creates a temp dir tree and verifies lookup.
4. The importer mutates the input AST (annotates fields). Order matters: validator runs AFTER importer in the CLI pipeline. M5 will wire that ordering; M3's job is just to provide the building blocks.
