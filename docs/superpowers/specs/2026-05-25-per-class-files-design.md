# Per-class file output for gdproto

**Status:** Draft
**Date:** 2026-05-25

## Goal

Replace gdproto's current "one big `.gd` file per `.proto` with a top-level wrapper class and many nested classes" output with one file per generated class. Each generated class is named with a common prefix (derived from the proto filename by default, or overridden via a custom file option), so it can be referenced globally via Godot's `class_name` registry without preloads or wrapper navigation.

## Motivation

Today, `example.proto` generates a single `example.gd` like:

```gdscript
class_name ExampleProto extends RefCounted

enum PlayerStatus { ... }

class Player extends RefCounted:
    class Position extends RefCounted:
        ...
    ...

class GameState extends RefCounted:
    ...
```

Consumers must navigate through the wrapper (`ExampleProto.Player.Position`), cannot autoload individual message types, and every `.proto` file produces one class entry that the editor's global class list collapses into. A medium-sized schema (50+ messages) becomes one large file with deeply nested references.

The new output produces individual files like `ExamplePlayer.pb.gd`, `ExamplePlayerPosition.pb.gd`, `ExampleGameState.pb.gd`, each with `class_name ExamplePlayer`/etc. Consumers reference them directly as globals.

This is a breaking change to the generated output. gdproto is on v0.3.x and this is the right time to make it the canonical format.

## Naming and file layout

### Prefix resolution

The prefix used for every class generated from a `.proto` file is resolved in this order:

1. `option (gdproto.class_prefix) = "MyPrefix";` if present at file scope.
2. PascalCase of the `.proto` filename stem: `example.proto` → `Example`, `game_state.proto` → `GameState`, `weird-name.proto` → `WeirdName`.

The prefix must match `^[A-Z][A-Za-z0-9]*$`. If a user supplies an invalid value via the option, generation fails with an error pointing at the option site.

### Generated class names

Class names concatenate the prefix with the proto type's nesting path inside the file. The proto `package` is not part of the class name (consistent with today's behavior).

| Proto symbol in `example.proto` | Generated `class_name` | File |
| --- | --- | --- |
| `message Player` | `ExamplePlayer` | `ExamplePlayer.pb.gd` |
| `message Player.Position` (nested) | `ExamplePlayerPosition` | `ExamplePlayerPosition.pb.gd` |
| `message GameState` | `ExampleGameState` | `ExampleGameState.pb.gd` |
| Top-level `enum Foo` | `ExampleFoo` (wrapper class) | `ExampleFoo.pb.gd` |
| `Player.Status` (nested enum) | `enum Status` inside `ExamplePlayer` | (same file as `ExamplePlayer`) |

Map-entry synthetic messages (`map<string, int32>`) remain an implementation detail and are not emitted as separate files.

### File naming

All generated files use the `.pb.gd` suffix (`ExamplePlayer.pb.gd`) so they are easy to identify, gitignore by glob, and distinguish from hand-written GDScript.

### Cross-file references

Because every message and top-level enum is a `class_name` global, references between them — within the same proto and across imported protos — compile to bare global class identifiers (`ExamplePlayer`, `OtherInventory`). No `preload(...)` or load statements are needed in the generated code.

### `proto_core_utils.gd`

Continues to be written as a sibling of the generated files, unchanged.

## Custom file option: `gdproto.class_prefix`

We ship a small well-known proto file that defines the extension.

### `proto/gdproto/options.proto`

```proto
syntax = "proto2";
package gdproto;
import "google/protobuf/descriptor.proto";

extend google.protobuf.FileOptions {
  // Override the auto-derived class_name prefix for generated GDScript files.
  optional string class_prefix = 51000;
}
```

Field number `51000` is inside protobuf's documented `50000`–`99999` range reserved for internal/third-party extensions ([protobuf.dev](https://protobuf.dev/programming-guides/proto3/#customoptions)). The README documents the choice so users extending `FileOptions` for their own purposes can avoid the same number.

### User usage

```proto
syntax = "proto3";
import "gdproto/options.proto";

option (gdproto.class_prefix) = "Game";

message Player { ... }    // → class_name GamePlayer
```

### Delivery paths

- **Plugin / Buf / `protoc`**: users must vendor `gdproto/options.proto` onto their proto include path. `protoc` and `buf` reject unknown extensions at parse time ([buf#36](https://github.com/bufbuild/buf/issues/36)), so the import is mandatory in this path.
- **Direct CLI (`gdproto foo.proto`)**: our own parser already tolerates unresolved extension names (`parseOptionName` stores `(gdproto.class_prefix)` as a string key). The CLI honors the option with no import, as a convenience for one-off generation. The README documents this asymmetry explicitly so users are not surprised when the plugin path requires the import.

To make vendoring easy, both binaries expose `--print-options-proto` which writes the embedded `gdproto/options.proto` to stdout. Users can `gdproto --print-options-proto > proto/gdproto/options.proto` and commit it once.

Publishing the options proto to the BSR (`buf.build/cafecito-games/gdproto`) is documented as a planned follow-up; not in this scope.

## Generator changes

### New top-level API

```go
// internal/generator/generator.go

func Generate(file *ast.ProtoFile, sourceName string) ([]GeneratedFile, error)

type GeneratedFile struct {
    Filename  string                // "ExamplePlayer.pb.gd"
    ClassName string                // "ExamplePlayer"
    Class     *gdast.ClassDefinition
}
```

Callers in `internal/cli` and `cmd/protoc-gen-gdscript` iterate the slice and write each file.

### Internal walk

1. Resolve the prefix once (option > filename-derived). Validate against the identifier regex.
2. Walk the AST and produce a `GeneratedFile` for:
   - Each top-level enum → a wrapper `class_name <Prefix><Enum> extends RefCounted` whose body declares `enum <Enum> { ... }`. Values are referenced as `<Prefix><Enum>.<Enum>.<VALUE>` (e.g. `ExamplePlayerStatus.PlayerStatus.ONLINE`). A flatter `const <VALUE> = <Enum>.<VALUE>` shim is intentionally not emitted in this design; if usage proves clunky, it can be added in a follow-up without changing the file layout.
   - Each message (top-level or nested) → a flat file whose `class_name` is the joined `<Prefix><ParentChain><Name>`. The class body contains its own fields, accessors, serialize/deserialize, oneofs, and nested enums (nested enums stay inline).
3. Map-entry synthetic messages do not produce files.

### Type reference rewriting

This is the load-bearing change. Today `generator.renderedType` returns a bare proto type name (`Position`) when the type lives in the same file, and falls back to `<WrapperClass>.<TypeName>` for cross-file references. With every message globally classed, every cross-message reference resolves to a prefixed global class name.

New resolver behavior:

- At the start of generation, build a map `proto-fully-qualified-name → generated-class-name`. The map covers every message and enum in the current file plus all transitively imported files, using each file's resolved prefix.
- `renderedType` and any other site that produces a type reference consult this map.
- Sites to update: `internal/generator/messages.go`, `accessors.go`, `serialize.go`, `deserialize.go`, `oneofs.go`, `fromtext.go`, `totext.go`. The existing `annotateLocalEnumUsage` logic is preserved (only enum *detection* is needed; the rendered name comes from the map).

### Option plumbing

- **Parser path:** read `file.Options["(gdproto.class_prefix)"]` (already populated by `internal/parser/options.go`). Coerce to string; error if absent-but-keyed or wrong type.
- **Descriptor path:** extend `internal/descriptors/converter.go` to surface `FileOptions` extensions on `ast.ProtoFile.Options`. Implementation: generate Go stubs for `gdproto/options.proto` and vendor them under `internal/gdprotopb/`. Read the extension via `proto.GetExtension(fd.GetOptions(), gdprotopb.E_ClassPrefix)`. The stubs are committed to the repo so building gdproto itself does not require `protoc`; a `task gen-options` target regenerates them when the proto changes.

## CLI and plugin surface

### `gdproto` CLI

- `-o` now means an **output directory**. If it points to an existing regular file or ends in `.gd`, fail with: `-o must be a directory; per-message files are written inside it. Got: foo.gd`.
- If `-o` is omitted, files are written to the current working directory (matches today's default location semantics; just multiple files instead of one).
- `proto_core_utils.gd` continues to be written into the same directory.
- On success, the CLI logs a single line to stderr: `wrote N files to <dir>/`.
- New flag: `--print-options-proto` writes the embedded `gdproto/options.proto` to stdout and exits.

### `protoc-gen-gdscript` plugin

- Emits one `CodeGeneratorResponse_File` per generated class plus the existing `proto_core_utils.gd`.
- Filenames are flat under the plugin's `out` directory (`ExamplePlayer.pb.gd`), matching CLI behavior. Collisions across protos in different packages are resolved by users via `option (gdproto.class_prefix)`.
- No new plugin parameters — prefix is configured exclusively via the file option or filename derivation.
- The plugin's binary also supports `--print-options-proto` for convenience when users have only the plugin installed.

### Removed surface

- The `wrapperClassName(...) + "Proto"` derivation and its top-level wrapper class go away entirely. No deprecation shim — clean break, per the v0.x replacement decision.

## Testing strategy

The user-visible promise is "follow the README and it works end-to-end." Tests must back that up at every layer, especially the documented install paths for `gdproto/options.proto`.

### Unit tests (`internal/generator`)

- Prefix resolution: filename-derived (`example.proto`, `game_state.proto`, `weird-name.proto`), option-overridden, invalid prefix → error with proper line/column.
- Multi-file emission: `example.proto` produces exactly `ExamplePlayer.pb.gd`, `ExamplePlayerPosition.pb.gd`, `ExampleGameState.pb.gd`, `proto_core_utils.gd` and nothing else.
- Cross-message type rewrite: the `Player.position` field's accessors, serialize, and deserialize sites all reference `ExamplePlayerPosition`, not `Position`.
- Nested enum (`Player.Status`) stays inline; top-level enum gets its own wrapper file.
- Cross-`.proto` import: a type from an imported file resolves to that file's prefix (`OtherInventory`, not `Inventory`), with and without an override on the importing/imported side.
- All existing field-shape coverage — oneof, `map<,>`, `repeated`, `optional`, packed encoding, scalar types — re-run under multi-file output.

### Golden update

- `examples/golden.gd` is replaced by `examples/golden/` containing the four expected `.pb.gd` files plus `proto_core_utils.gd`. `examples/example.proto` is unchanged.
- Snapshot tests in `internal/generator/generator_test.go` and `cmd/protoc-gen-gdscript/main_test.go` are updated to compare directory contents instead of single files.

### Integration tests for the documented install paths

New `tests/integration/options_proto_test.go` (build-tagged `integration`):

1. Sets up a tempdir with a `sample.proto` that does `import "gdproto/options.proto"; option (gdproto.class_prefix) = "Game"; message Hero { ... }`.
2. Runs three scenarios end-to-end and asserts each produces the expected file set with `Game*` prefixes:
   - `gdproto` CLI with `-I` pointing at the repo's `proto/` dir.
   - `protoc --plugin=...` invocation, same `-I`.
   - `buf generate` with a `buf.yaml` + `buf.gen.yaml` that vendors `gdproto/options.proto` exactly as the README documents.
3. Auto-skips with a clear message if `protoc` / `buf` aren't on `PATH`; CI installs both and runs the integration suite as a required job.

A second integration test invokes `gdproto --print-options-proto` and asserts byte-for-byte equality with `proto/gdproto/options.proto`.

### Godot runtime tests (`tests/`)

- The existing Vest suite round-trips wire bytes through generated GDScript. Update its harness to load per-class files (no `class_name X = preload(...)` since globals work) and re-run the suite under Godot 4.6 in CI against `examples/example.proto` plus a multi-import scenario, proving `class_name` globals resolve correctly.

### README verification

- After updating the README to describe the new install snippets and the import requirement, run the integration tests in a `git clean -fdx`-ed worktree against snippets copied verbatim from the README. This catches drift between docs and behavior.

## Out of scope

- Publishing `gdproto/options.proto` to the Buf Schema Registry (documented as a planned follow-up).
- Package-derived subdirectories for plugin output (users handle collisions via `class_prefix`).
- A magic-comment fallback for users who want the option without the import (the extension is the only supported mechanism).
- Migration shim for the old wrapper-class output.
