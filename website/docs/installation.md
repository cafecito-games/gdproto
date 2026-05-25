---
title: Installation
description: Install gdproto and protoc-gen-gdscript.
---

# Installation

Install `gdproto` with Homebrew on macOS or with `go install` anywhere Go
1.26+ is available.

## Homebrew

```bash
brew install --cask cafecito-games/tap/gdproto
```

The cask installs both binaries:

- `gdproto`
- `protoc-gen-gdscript`

Verify the install:

```bash
gdproto --version
which protoc-gen-gdscript
```

## Go Install

```bash
go install github.com/cafecito-games/gdproto/cmd/gdproto@latest
go install github.com/cafecito-games/gdproto/cmd/protoc-gen-gdscript@latest
```

Go installs the binaries into `$GOPATH/bin`. Add that directory to `PATH`
before using Buf or `protoc` if it is not already there:

```bash
export PATH="$(go env GOPATH)/bin:$PATH"
which gdproto
which protoc-gen-gdscript
```

## Build From A Checkout

For repository development or local testing:

```bash
git clone git@github.com:cafecito-games/gdproto.git
cd gdproto
task build
```

This writes:

```text
bin/
  gdproto
  protoc-gen-gdscript
```

For local plugin generation, either put `bin/` on `PATH` or pass an explicit
plugin path to `protoc`.

## Required Tools By Workflow

| Workflow | Required tools |
| --- | --- |
| Direct CLI | `gdproto` |
| `protoc` plugin | `protoc`, `protoc-gen-gdscript` |
| Buf generation | `buf`, `protoc-gen-gdscript` |
| Repository development | Go 1.26+, Task, golangci-lint |

## Optional: Vendoring `gdproto/options.proto`

If your schemas use the `(gdproto.class_prefix)` file option, you need the
extension descriptor on disk so that `protoc` and `buf` can resolve it.
Either binary can print the embedded descriptor:

```bash
mkdir -p proto/gdproto
gdproto --print-options-proto > proto/gdproto/options.proto
# or
protoc-gen-gdscript --print-options-proto > proto/gdproto/options.proto
```

See [Generated GDScript](./generated-code.md#class-prefix) for the full
explanation of how the option is consumed.
