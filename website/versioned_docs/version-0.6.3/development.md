---
title: Development
description: Build, test, and update generated fixtures.
---

# Development

The repository is a Go module with a Taskfile for common commands.

## Common Commands

```bash
task          # full local CI pipeline
task test     # Go tests with -race
task test:cover
task lint     # golangci-lint v2
task fmt      # go fmt and goimports
task build    # writes bin/gdproto and bin/protoc-gen-gdscript
```

`task ci` runs formatting checks, module tidy checks, linting, tests, and
binary builds.

## Pre-commit Hooks

The project uses `prek` for pre-commit hooks:

```bash
prek install
```

Hooks run formatting checks, linting, and tests for Go changes. Whitespace
fixers intentionally skip generated GDScript golden files because those files
lock byte-for-byte generator output.

## Golden Fixtures

`examples/example.proto` pins representative generator output:

```text
examples/example.proto
examples/golden.gd
examples/proto_core_utils_golden.gd
```

When an intentional generator change alters output, regenerate the fixtures:

```bash
go run ./cmd/gdproto examples/example.proto -o examples/golden.gd
cp examples/proto_core_utils_golden.gd internal/generator/proto_core_utils_data.gd
go test ./internal/generator/... -run 'TestGoldenExample|TestProtoCoreUtilsGolden' -v
```

Review the diff before committing because the golden files are the generator
contract.

## Godot Integration Tests

The integration suite builds `protoc-gen-gdscript`, generates fixture wrappers,
imports the Godot project headlessly, and runs Vest tests:

```bash
bash tests/godot/run.sh
```

Required tools on `PATH`:

- `go`
- `protoc`
- `godot` 4.6.x

The test path exercises cross-file references in a real Godot project, which is
important for generated wrapper class resolution.
