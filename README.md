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

Generated output passes [gdkit](https://github.com/cafecito-games/gdkit)'s
`gdkit format check`, `gdkit lint check`, and `gdkit uid check` with gdkit's
default configuration. A project that gates CI on gdkit can therefore keep its
generated protocol directory inside the checked set. Excluding that directory
also excludes it from the checks that would catch a real problem in it.

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
# gdkit:disable = max-returns, max-public-methods
```

Those two rules are gdkit *design limits* that generated protobuf code cannot
satisfy by construction: a wire-format parser is an early-return function with
one return per field, and every proto field contributes a set of public
accessors, so method count grows with the schema. Restructuring generated code
to fit the limits would make it worse, not better. No other rule is
suppressed. The directive has to be line 1 because
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
gdproto derives them from the filename instead, because a generator re-runs on
every schema sync, and reissuing an identifier changes what every existing
`uid://` reference to that script resolves to, with nothing rewriting those
references. Deriving the identifier makes regeneration byte identical, makes
two developers' output agree, and is the only scheme that works in the
`protoc` plugin path, which never learns its own output directory.

### Limits

- **`max-line-length` is corpus dependent, not guaranteed.** A schema with very
  long message, field, or enum names still produces lines past gdkit's
  100-column limit, because a single long identifier or type annotation has no
  legal wrap point. Formatting stays canonical and idempotent for such a
  schema; it is only the line-length lint rule that reports.
- **The agreement is with gdkit's default configuration.** A project with its
  own `.gdkit/format.json` — a different `line_width`, spaces instead of tabs —
  still needs its own `gdkit format write` pass over the generated directory.
  gdproto cannot read a downstream project's configuration.
- **The first regeneration replaces Godot-assigned sidecars.** A project that
  already holds Godot-written `.uid` files for generated scripts will see them
  replaced, since the derived identifier differs from the random one Godot
  assigned. The exposure is small in practice — every generated class declares
  `class_name` and is referenced by that global identifier rather than by a
  `uid://` path — but it is a real one-time event. If a scene or resource does
  reference a generated script by uid, grep for the old identifiers and update
  them after the first regeneration.

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
