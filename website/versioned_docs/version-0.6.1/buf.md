---
title: Using buf
description: Configure buf generate to run protoc-gen-gdscript.
---

# Using buf

Buf can run `protoc-gen-gdscript` as a local plugin. The plugin binary must be
installed on `PATH` before you run `buf generate`.

## Example Layout

```text
game/
  buf.yaml
  buf.gen.yaml
  proto/
    example.proto
    gdproto/
      options.proto
  godot/
    generated/
```

## buf.yaml

For a single local proto module:

```yaml
version: v2
modules:
  - path: proto
```

## buf.gen.yaml

Use a local plugin entry:

```yaml
version: v2
plugins:
  - local: protoc-gen-gdscript
    out: godot/generated
```

Buf resolves `local: protoc-gen-gdscript` the same way your shell does. Verify
the binary before generation:

```bash
which protoc-gen-gdscript
```

Then run:

```bash
buf generate
```

Buf reads `buf.gen.yaml`, builds a CodeGeneratorRequest for your input module,
and invokes the plugin.

## Output Paths

The plugin writes one `.pb.gd` file per top-level message or enum, plus the
runtime, under the configured output directory.

With this input:

```text
proto/example.proto
```

(the `examples/example.proto` schema with `Player`, nested `Position`,
`GameState`, and top-level `PlayerStatus`)

and this config:

```yaml
version: v2
plugins:
  - local: protoc-gen-gdscript
    out: godot/generated
```

Buf invokes the plugin and the generated tree is:

```text
godot/generated/
  ExamplePlayer.pb.gd
  ExamplePlayerPosition.pb.gd
  ExampleGameState.pb.gd
  ExamplePlayerStatus.pb.gd
  proto_core_utils.gd
```

Keep `proto_core_utils.gd` with the generated wrappers in your Godot project.

## Files Generated

The plugin only emits files for the protos Buf includes in
`file_to_generate`. Imported `.proto` files are parsed for type resolution
but do not automatically produce `.pb.gd` output. If you want wrappers for
an imported schema, make sure it is part of your buf module (or add a
second module).

Cross-file type references render with the imported file's own
`(gdproto.class_prefix)` (or its filename-derived default), so mixed
projects where some files set an explicit prefix and others rely on the
default still resolve correctly. See
[Generated GDScript](./generated-code.md#cross-file-references-honor-imported-prefixes).

## Custom Class Prefix

To override the auto-derived class prefix, add the `(gdproto.class_prefix)`
file option:

```protobuf
syntax = "proto3";
import "gdproto/options.proto";

option (gdproto.class_prefix) = "Game";

message Hero {
  string name = 1;
  int32 hp = 2;
}
```

Buf needs the `gdproto/options.proto` descriptor available inside the module
to resolve the `(gdproto.class_prefix)` extension. Vendor it into the buf
module path:

```bash
mkdir -p proto/gdproto
gdproto --print-options-proto > proto/gdproto/options.proto
```

With `modules: - path: proto`, Buf will pick up `gdproto/options.proto`
alongside your schemas and the plugin can read the prefix. The integration
fixture under `tests/integration/fixtures/options/` shows the same layout.

See [Generated GDScript](./generated-code.md#class-prefix) for the full
explanation of how the prefix affects generated class names.

## CI Example

```yaml
name: Generate protos

on:
  push:
    branches: [main]

jobs:
  generate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: "1.26.x"

      - uses: bufbuild/buf-action@v1

      - name: Install gdproto plugin
        run: go install github.com/cafecito-games/gdproto/cmd/protoc-gen-gdscript@latest

      - name: Generate GDScript
        run: buf generate
```

Pin the plugin version in CI when reproducible generation matters.
