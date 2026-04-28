# Code Review Findings Fix Design

## Goal

Fix the code generation defects identified in `CODE_REVIEW.md` so that:

- CLI and `protoc` plugin outputs agree on type resolution and wrapper naming behavior
- nested and imported type references generate valid GDScript
- enum handling is driven by resolved type metadata instead of short-name heuristics
- map fields with enum values serialize and deserialize correctly

## Scope

This work covers the five findings from the review:

1. imported type references are resolved but not emitted correctly
2. plugin descriptor conversion loses nested qualification
3. enum detection is based on short names and misclassifies some message fields
4. `map<scalar, enum>` uses the wrong encode/decode path
5. CLI and plugin wrapper names diverge for nested proto paths

This work does not change the public feature set beyond making the advertised behavior actually work.

## Design

### 1. Canonical type identity

The current pipeline mixes short names and full names:

- parser preserves dotted source names
- descriptor conversion truncates them to the last segment
- generator infers enum-ness from a global short-name set

That makes correctness depend on naming accidents.

The fix is to treat full type identity as the source of truth everywhere:

- keep `FullTypePath` and `FullValueTypePath` populated for all message/enum references
- preserve qualified local type names from descriptors instead of truncating to `lastSegment`
- use per-field metadata (`IsEnum`, `ValueIsEnum`, `SourceFile`, `FullTypePath`) in generator decisions

Short display names can still exist where convenient, but code generation decisions should not depend on them.

### 2. GDScript type emission model

There are two separate cases:

- local types defined in the same wrapper file
- imported types defined in other generated wrapper files

For local nested types, the generator should emit the same qualified type names regardless of entry point. If a field refers to `Outer.Inner`, generated declarations and constructors should use `Outer.Inner`.

For imported types, the generator needs explicit wrapper references instead of bare names. The design is:

- derive the imported wrapper class from the source proto path using the same wrapper naming function used for the current file
- render imported message and enum types as `<ImportedWrapper>.<QualifiedTypeWithinFile>`
- keep using sibling `proto_core_utils.gd` as today

Example:

- `common.proto` containing `message Shared`
- `main.proto` importing `common.proto`

Generates references like `CommonProto.Shared`, not bare `Shared`.

This avoids adding preload path management to the generator and stays consistent with the existing generated nested-type style.

### 3. Entry-point parity

The CLI currently passes only the basename into `generator.Generate`, while the plugin passes the proto-relative path. The design choice here is to standardize on the proto-relative source name when it is known.

Implementation rule:

- CLI passes the original proto input path normalized into repo/proto-relative semantics already used by the generator
- plugin continues to pass the request filename
- wrapper class naming becomes path-sensitive in the same way for both entry points

Because the README already promises identical output between entry points, parity matters more than preserving the current CLI-only basename behavior.

### 4. Enum handling

Enum logic should stop depending on a global `enumTypes` short-name table.

Instead:

- direct field enum logic uses `Field.IsEnum`
- map value enum logic uses `MapField.ValueIsEnum`
- local nested enum references remain correct because descriptor/parser resolution populates the right metadata

This fixes:

- same-name message/enum collisions in different scopes
- descriptor-path nested enum ambiguity
- incorrect map enum serialization/deserialization

### 5. Test strategy

Add explicit regression coverage for each reviewed defect:

- CLI import end-to-end generation using two proto files
- plugin generation for nested local references such as `Outer.Inner`
- same short-name collision between message and enum in different scopes
- `map<string, Enum>` serialization/deserialization generation checks
- CLI vs plugin parity for a proto in a nested directory

Tests should prefer narrow assertions on generated snippets where possible and use end-to-end file generation where cross-file behavior is the thing being validated.

## Files Expected To Change

- `internal/descriptors/converter.go`
- `internal/generator/generator.go`
- `internal/generator/messages.go`
- `internal/generator/accessors.go`
- `internal/generator/serialize.go`
- `internal/generator/deserialize.go`
- `internal/generator/fromtext.go` if imported/local enum rendering requires it
- `internal/generator/totext.go` if imported/local enum rendering requires it
- `internal/cli/root.go`
- tests in `internal/generator`, `internal/cli`, and `cmd/protoc-gen-gdscript`
- `README.md` if documented behavior needs clarification after parity decisions

## Error Handling

This work should not add new user-facing failure modes unless a code path is truly unsupported. Prefer generating correct references over rejecting inputs that already validate today.

If an imported type cannot be rendered consistently from available metadata, that should become a generator error rather than silently emitting invalid GDScript.

## Success Criteria

- imported message and enum references generate valid, qualified GDScript type references
- plugin-generated nested type references match CLI behavior
- same-name enum/message collisions no longer miscompile
- `map<scalar, enum>` uses varint encode/decode paths
- CLI and plugin parity is covered by tests
- `go test ./...` and `go test -race ./...` pass
