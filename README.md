# gdproto

Protocol Buffers v3 to GDScript compiler for Godot 4.6+.

`gdproto` generates Godot-friendly GDScript wrappers that can serialize and
deserialize protobuf binary wire format. It ships as two Go binaries:

- `gdproto`: direct CLI for one-off `.proto` to `.gd` generation.
- `protoc-gen-gdscript`: standard `protoc` plugin for `protoc`, Buf, and CI.

Full documentation: <https://cafecito-games.github.io/gdproto/>

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

Direct CLI:

```bash
gdproto path/to/player.proto -o godot/generated/player.gd
```

This writes `player.gd` and a sibling `proto_core_utils.gd`.

`protoc` plugin:

```bash
protoc \
  --plugin=protoc-gen-gdscript="$(which protoc-gen-gdscript)" \
  --gdscript_out=godot/generated \
  -I proto \
  proto/player.proto
```

Plugin mode writes `.pb.gd` wrappers and `proto_core_utils.gd`.

Buf:

```yaml
version: v2
plugins:
  - local: protoc-gen-gdscript
    out: godot/generated
```

Then run:

```bash
buf generate
```

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
