# gdproto

Protocol Buffers v3 to GDScript compiler for Godot 4.6+.

`gdproto` generates Godot-friendly GDScript wrappers that can serialize and
deserialize protobuf binary wire format. It ships as two Go binaries:

- `gdproto`: direct CLI for one-off `.proto` to `.gd` generation.
- `protoc-gen-gdscript`: standard `protoc` plugin for `protoc`, Buf, and CI.

Full documentation: <https://cafecito-games.github.io/gdproto/>

> **Breaking changes in the next release.** `gdproto` now emits one
> `.pb.gd` file per top-level proto message or enum instead of a single
> `<Name>Proto` wrapper containing nested classes. The `-o` flag on the
> direct CLI now takes an **output directory** rather than a `.gd` file
> path, and `protoc-gen-gdscript` only generates files explicitly listed
> in `file_to_generate` (matching `protoc-gen-go`) — imported `.proto`
> files no longer trigger transitive generation. References change from
> e.g. `ExampleProto.Player.Position` to `ExamplePlayerPosition`. See
> [Custom prefix](#custom-prefix) for the new `(gdproto.class_prefix)`
> option that lets you override the auto-derived class prefix.

## Install

Homebrew:

```bash
brew install --cask cafecito-games/tap/gdproto
```

Go:

```bash
go install github.com/cafecito-games/gdproto/cmd/gdproto@latest
go install github.com/cafecito-games/gdproto/cmd/protoc-gen-gdscript@latest
```

When installing with Go, make sure `$GOPATH/bin` is on `PATH` before running Buf
or `protoc`.

To build from a checkout into `./bin` instead:

```bash
task build
```

Requires Go 1.26+.

## Quick Usage

Direct CLI — `-o` is an output **directory**:

```bash
gdproto examples/example.proto -o godot/generated/
```

For the schema in `examples/example.proto` (a `Player` message with a nested
`Position` message, a `GameState` message, and a top-level `PlayerStatus`
enum) this produces:

```text
godot/generated/
  ExamplePlayer.pb.gd
  ExamplePlayer.pb.gd.uid
  ExamplePlayerPosition.pb.gd
  ExamplePlayerPosition.pb.gd.uid
  ExampleGameState.pb.gd
  ExampleGameState.pb.gd.uid
  ExamplePlayerStatus.pb.gd
  ExamplePlayerStatus.pb.gd.uid
  proto_core_utils.gd
  proto_core_utils.gd.uid
```

The class prefix (`Example`) is derived from the proto filename. Each file
declares a top-level `class_name` so the classes are globally available in
Godot without `preload`. The `.uid` files are Godot resource identifier
sidecars; see [gdkit conformance](#gdkit-conformance).

`protoc` plugin — `--gdscript_out` is the output directory:

```bash
protoc \
  --plugin=protoc-gen-gdscript="$(which protoc-gen-gdscript)" \
  --gdscript_out=godot/generated \
  -I proto \
  proto/example.proto
```

The plugin emits the same per-class files plus `proto_core_utils.gd`. Only
the files passed on the command line (i.e. listed in `file_to_generate`) are
generated; imports are walked for type resolution but do not produce output.
If you need wrappers for an imported `.proto`, add it to the input set.

Buf:

```yaml
# buf.gen.yaml
version: v2
plugins:
  - local: protoc-gen-gdscript
    out: godot/generated
```

Then run:

```bash
buf generate
```

## gdkit conformance

For a schema that follows [protobuf's own style
guide](https://protobuf.dev/programming-guides/style/), generated output passes
[gdkit](https://github.com/cafecito-games/gdkit)'s `gdkit format check`,
`gdkit lint check`, and `gdkit uid check` with gdkit's default configuration. A
project that gates CI on gdkit can therefore keep its generated protocol
directory inside the checked set. Excluding that directory also excludes it
from the checks that would catch a real problem in it. The style-guide
condition and the other edges of the guarantee are in [Limits](#limits).

Formatting agrees with gdkit by construction rather than by imitation: the
generator parses its own output with
[gdparser](https://github.com/cafecito-games/gdparser) and re-emits it through
`gdparser/format` using the Godot style options. `gdkit format` delegates to
that same engine, and gdkit's default format configuration is field for field
gdparser's `GodotStyle()`, so neither side reimplements the other's wrapping
rules. The generator also verifies that a second formatting pass is a no-op,
because `gdkit format check` reports any file another pass would change.

Every generated file opens with a suppression directive on line 1:

```gdscript
# gdkit:disable = max-returns, max-public-methods, max-file-lines
```

All three rules are gdkit *design limits* whose value scales with the schema
rather than with the quality of the code, so generated protobuf code cannot
satisfy them by construction: a wire-format parser is an early-return function
with one return per field, every proto field contributes a set of public
accessors, and a message's file is as long as its field count demands once the
serializer, deserializer, text-format reader and writer, accessors and enum
helpers are emitted per field. The repository's own `collections.proto` fixture
generates a 1,121-line file from a single map-heavy message. Restructuring
generated code to fit the limits would make it worse, not better. No other rule
is suppressed. The directive has to be line 1 because
`max-public-methods` is reported against the class global scope there, and a
`gdkit:disable` reaches only from its own line to the end of the file.

### `uid://` sidecars

Each generated script is written with a sibling `.uid` file holding its Godot
resource identifier:

```text
$ cat godot/generated/ExamplePlayer.pb.gd.uid
uid://ix8k3hu6vdsf
```

Godot assigns these identifiers at random when it first imports a script.
gdproto derives them from the filename instead, which makes regeneration
produce byte-identical sidecars, makes two developers' output agree, and is the
only scheme that works in the `protoc` plugin path, which never learns its own
output directory.

That is sound because of the contract generated code is used under: **a
generated script is addressed by its `class_name`, not by its `uid://` path.**
Every generated file declares a `class_name`, and generated messages are
instantiated through it — `ExamplePlayer.new()`. Nothing should reference a
generated script by `uid://`: not a scene, not a resource, not `project.godot`.
So the identifier only has to be *stable*, which deriving it from the filename
guarantees; it never has to be preserved across a change of scheme.

A project that happens to hold Godot-assigned sidecars for generated scripts
will see them replaced the first time it regenerates. Under the contract above
that has no effect, because nothing was resolving those identifiers. In a
production Godot client with 688 generated scripts, none of the 689 generated
sidecar identifiers appeared in any scene, resource, or project file.

### Limits

- **Identifier spelling has to follow protobuf's style guide.** Proto
  identifiers are carried through into GDScript names verbatim, so a schema
  that departs from the protobuf style guide — a `camelCase` field, a
  `snake_case` message or enum name, a `camelCase` enum value — produces
  GDScript names that gdkit's naming rules reject. A single
  `string userName = 1;` field yields `var _userName` plus `get_userName()`
  and `set_userName()`, which report `class-variable-name` and
  `function-name`; `snake_case` type names and
  `camelCase` enum values add `class-name`, `enum-name`, and
  `enum-element-name`. The style guide already asks for `lower_snake_case`
  fields, `PascalCase` message and enum names, and `SCREAMING_SNAKE_CASE` enum
  values, so a conforming schema is the normal case. These rules are *not*
  suppressed: they are real naming rules, and silencing them would hide genuine
  naming problems in generated output, while renaming identifiers would change
  the generated API — a decision this project has not taken. The remedy is to
  rename the offending fields in the schema, or to scope gdkit's naming rules
  away from the generated directory in the consuming project's own
  `.gdkit/lint.json`.
- **`max-line-length` reports for deep package paths, and that is the common
  case.** Class prefixes are derived from the full `.proto` path (see
  [Custom prefix](#custom-prefix)), so a layout like
  `uzir/assetpack/client/v1/characters.proto` yields class names near 50
  characters. A declaration that names its type twice then cannot fit in 100
  columns and has no legal wrap point:

  ```gdscript
  var msg_instance: UzirAssetpackClientV1CharactersAvatarBodyDirection = UzirAssetpackClientV1CharactersAvatarBodyDirection.new()
  ```

  Regenerating a 687-file production schema produces 868 of these, and nothing
  else — `format check` and `uid check` pass that same schema completely. So
  treat this as expected for a monorepo layout rather than as an edge case.
  Shortening the prefix with `(gdproto.class_prefix)` reduces it; inferring the
  type instead would fix it but breaks under a strict `untyped_declaration`
  warning policy, so the generator keeps the annotation.

  The rule is left reporting on purpose, unlike `max-file-lines`: wrapping is
  genuinely the formatter's job, and suppressing the rule would hide real
  formatting problems. A project that wants the generated directory silent on
  this disables or allowlists the one rule for that directory.
- **An opt-in logging rule reports.** Generated code calls `push_error` to
  report a decode failure. A project that enables gdkit's `no-engine-logging`
  rule — inert by default — and routes diagnostics through its own logger will
  see that rule report across generated files, since the generator cannot know
  the project's logger. Scope the rule away from the generated directory.
- **The agreement is with gdkit's default configuration.** A project with its
  own `.gdkit/format.json` — a different `line_width`, spaces instead of tabs —
  still needs its own `gdkit format write` pass over the generated directory.
  gdproto cannot read a downstream project's configuration.

## Custom prefix

By default the class prefix for generated files is derived from the input
`.proto` path: each path segment is split on non-alphanumerics, PascalCased,
and concatenated. So `example.proto` -> `Example`, `game_state.proto` ->
`GameState`, and `uzir/common/v1/common.proto` -> `UzirCommonV1Common`.
Using the full path keeps prefixes unique in monorepo layouts that
segregate otherwise-identical filenames (e.g. multiple `common.proto`)
into different directories. To override it, set the
`(gdproto.class_prefix)` file option:

```protobuf
syntax = "proto3";
import "gdproto/options.proto";

option (gdproto.class_prefix) = "Game";

message Hero {
  string name = 1;
  int32 hp = 2;
}
```

With the prefix above, the generator emits `GameHero.pb.gd` (class
`GameHero`) instead of the filename-derived default.

The `import "gdproto/options.proto";` line is **required** when generating
through `protoc` or `buf` — both tools reject unknown extensions and need
the `.proto` descriptor for `gdproto.class_prefix` (field number `51000`)
on disk. The direct `gdproto` CLI tolerates a missing import, but
importing it everywhere keeps a single schema portable across all three
paths.

### Installing `gdproto/options.proto`

There are three supported ways to make the options proto available to your
toolchain:

**(a) Print from the binary.** The simplest path; no clone or download
needed:

```bash
mkdir -p proto-include/gdproto
gdproto --print-options-proto > proto-include/gdproto/options.proto
```

`protoc-gen-gdscript --print-options-proto` works the same way.

Then point the direct CLI at the vendored descriptor with `-I` (alias
`--proto_path`), matching `protoc`'s convention:

```bash
gdproto -I proto-include -o godot/generated proto/example.proto
```

`-I` is repeatable: each directory is searched in order before falling
back to the input file's directory.

**(b) Raw `protoc`.** Vendor the file anywhere on disk and add the
directory as an import root:

```bash
protoc \
  --plugin=protoc-gen-gdscript="$(which protoc-gen-gdscript)" \
  --gdscript_out=godot/generated \
  -I proto \
  -I path/to/vendored/gdproto \
  proto/example.proto
```

The `.proto` that uses the option then says `import "gdproto/options.proto";`
and `protoc` resolves it through the second `-I` root.

**(c) Buf.** Place `gdproto/options.proto` inside your buf module path so
that it is visible to the importer:

```text
proto/
  buf.yaml
  buf.gen.yaml
  gdproto/
    options.proto
  example.proto
```

```yaml
# buf.yaml
version: v2
modules:
  - path: .
```

```yaml
# buf.gen.yaml
version: v2
plugins:
  - local: protoc-gen-gdscript
    out: out
```

Then `buf generate` picks up `gdproto.options` from the module and the
plugin can read the `class_prefix` extension.

The integration tests under `tests/integration/` exercise all three of
these paths end-to-end against `tests/integration/fixtures/options/`.

> **Why field number `51000`?** Protobuf reserves the range `50000`-`99999`
> for internal third-party extensions, which is what `gdproto.class_prefix`
> uses. See <https://protobuf.dev/programming-guides/proto3/#customoptions>.

## Development

```bash
task          # full local CI pipeline
task test     # Go tests with -race
task test:cover
task lint     # golangci-lint v2
task fmt      # go fmt and goimports
task build    # writes bin/gdproto and bin/protoc-gen-gdscript
```

Golden generator fixtures live in `examples/`. See the documentation site for
fixture update instructions, Godot integration tests, feature support, and
release docs.
