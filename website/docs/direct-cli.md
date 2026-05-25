---
title: Direct CLI
description: Generate GDScript wrappers with the gdproto command.
---

# Direct CLI

The `gdproto` command reads one `.proto` file and writes one `.pb.gd` file
per top-level message or enum into an output directory. It is useful for
small projects, quick experiments, and golden fixture updates.

## Command

```bash
gdproto examples/example.proto -o godot/generated/
```

For `examples/example.proto` this writes:

```text
godot/generated/
  ExamplePlayer.pb.gd
  ExamplePlayerPosition.pb.gd
  ExampleGameState.pb.gd
  ExamplePlayerStatus.pb.gd
  proto_core_utils.gd
```

The class prefix (`Example`) is derived from the input filename. Nested
messages are flattened into sibling files using the same prefix; top-level
enums get a thin wrapper class so the values stay globally addressable in
Godot. See [Generated GDScript](./generated-code.md) for the addressing
rules and the `(gdproto.class_prefix)` option that overrides the prefix.

If `-o` is omitted, files are written into the current working directory.

## Flags

| Flag | Default | Notes |
| --- | --- | --- |
| `-o, --output` | Current directory | Output **directory** for generated `.pb.gd` files and `proto_core_utils.gd`. Must not end in `.gd` and must not point at an existing file. |
| `-I, --proto_path` | | Directory searched for imported `.proto` files. Repeatable; matches `protoc`'s convention. Each path is checked in order before falling back to the input file's directory. |
| `--print-options-proto` | | Prints the embedded `gdproto/options.proto` descriptor to stdout and exits. Useful for vendoring without cloning the repo. |
| `--log-level` | `warn` | One of `debug`, `info`, `warn`, or `error`. Logs are JSON on stderr. |
| `--version` | | Prints the binary version. |

## Vendoring `gdproto/options.proto`

The direct CLI tolerates a missing `import "gdproto/options.proto";` when
parsing schemas that use `(gdproto.class_prefix)`, but `protoc` and `buf`
reject unknown extensions. Vendor the descriptor to keep the same schema
portable across all three paths:

```bash
mkdir -p proto-include/gdproto
gdproto --print-options-proto > proto-include/gdproto/options.proto
```

Then tell `gdproto` where to find the vendored descriptor with `-I`:

```bash
gdproto -I proto-include -o godot/generated proto/example.proto
```

## Import Resolution

The direct CLI resolves each `import "path";` by checking, in order:

1. Each `-I/--proto_path` directory, joined with the import path.
2. The input file's own directory, joined with the import path.
3. Parent directories of the input file (a backwards-compatible fallback
   for projects that rely on the original walk-up behavior).

For example, given:

```text
proto-include/
  gdproto/options.proto
proto/
  player.proto
  shared/team.proto
```

if `player.proto` writes `import "shared/team.proto";` and
`import "gdproto/options.proto";`, run:

```bash
gdproto -I proto-include -o godot/generated/ proto/player.proto
```

`-I` is repeatable, so multiple include roots may be stacked
(`-I proto-include -I third-party`). This mirrors `protoc`'s
convention.

## Exit Codes

| Code | Meaning |
| --- | --- |
| `0` | Success |
| `1` | Error |
| `130` | Interrupted |
