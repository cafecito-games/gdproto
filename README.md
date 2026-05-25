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
  ExamplePlayerPosition.pb.gd
  ExampleGameState.pb.gd
  ExamplePlayerStatus.pb.gd
  proto_core_utils.gd
```

The class prefix (`Example`) is derived from the proto filename. Each file
declares a top-level `class_name` so the classes are globally available in
Godot without `preload`.

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

## Custom prefix

By default the class prefix for generated files is derived from the input
`.proto` filename (`example.proto` -> `Example`). To override it, set the
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
