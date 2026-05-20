---
title: Direct CLI
description: Generate one GDScript wrapper with the gdproto command.
---

# Direct CLI

The `gdproto` command reads one `.proto` file and writes one requested `.gd`
wrapper path. It is useful for small projects, quick experiments, and golden
fixture updates.

## Command

```bash
gdproto path/to/player.proto -o godot/generated/player.gd
```

This writes:

```text
godot/generated/player.gd
godot/generated/proto_core_utils.gd
```

The output filename is exactly the path passed with `--output`. Direct CLI mode
does not append `.pb.gd` for you.

## Flags

| Flag | Default | Notes |
| --- | --- | --- |
| `-o, --output` | Required | Output `.gd` file path. |
| `--log-level` | `warn` | One of `debug`, `info`, `warn`, or `error`. Logs are JSON on stderr. |
| `--version` | | Prints the binary version. |

## Import Resolution

Direct CLI import resolution starts from the input file's directory:

```text
proto/
  player.proto
  shared/team.proto
```

If `player.proto` imports `shared/team.proto`, run:

```bash
gdproto proto/player.proto -o godot/generated/player.gd
```

For projects with multiple import roots, prefer Buf or the `protoc` plugin
because they expose import roots explicitly.

## Exit Codes

| Code | Meaning |
| --- | --- |
| `0` | Success |
| `1` | Error |
| `130` | Interrupted |
