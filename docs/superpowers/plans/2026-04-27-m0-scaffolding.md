# M0 — Scaffolding Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers-extended-cc:subagent-driven-development (recommended) or superpowers-extended-cc:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Stand up the gogdproto Go module with all build/lint/CI/pre-commit infrastructure and a working `gogdproto` cobra binary that prints version and respects `--log-level`. End state: `task ci` is green locally and on GitHub Actions for a hello-world binary.

**Architecture:** Standard Go module layout with `cmd/gogdproto` for the CLI entry point and empty `internal/*` package skeletons that future milestones will fill in. All tooling lives at the repo root and orchestrates through Taskfile so pre-commit and CI both call into the same commands. Component-tagged `slog.Logger` is constructed at the CLI boundary and passed down (no global loggers).

**Tech Stack:** Go 1.26 (toolchain go1.26.2), `github.com/spf13/cobra`, `log/slog` (stdlib), Taskfile 3.x, golangci-lint v2, prek (pre-commit), GitHub Actions.

**Reference design:** [docs/superpowers/specs/2026-04-27-gogdproto-design.md](../specs/2026-04-27-gogdproto-design.md)

**GitHub tracking:** [issue #1](https://github.com/cafecito-games/gogdproto/issues/1), milestone "M0 — Scaffolding".

---

## File Structure

Files created in this milestone:

| Path                               | Responsibility                                                       |
|------------------------------------|----------------------------------------------------------------------|
| `.gitignore`                       | Ignore `bin/`, `coverage.out`, `dist/`, editor cruft.                |
| `.editorconfig`                    | Indent/EOL rules.                                                    |
| `go.mod` / `go.sum`                | Module declaration, Go 1.26, deps.                                   |
| `README.md`                        | One-paragraph placeholder pointing to design doc.                    |
| `Taskfile.yml`                     | Build/test/lint/fmt/tidy/ci/install tasks.                           |
| `.golangci.yml`                    | golangci-lint v2 config.                                             |
| `.pre-commit-config.yaml`          | prek hook config calling Taskfile targets.                           |
| `.github/workflows/pr.yml`         | CI for `pull_request` and `push: [main]`.                            |
| `cmd/gogdproto/main.go`            | Cobra entry, calls into `internal/cli`.                              |
| `internal/cli/root.go`             | Cobra root command + flag wiring.                                    |
| `internal/cli/root_test.go`        | CLI behavior tests (version, log level).                             |
| `internal/cli/version.go`          | Version constant (single source of truth).                           |
| `internal/applog/applog.go`        | Logger construction helpers (component-tagged, level parsing).       |
| `internal/applog/applog_test.go`   | Logger helper tests.                                                 |
| `internal/lexer/doc.go`            | Empty package skeleton (`package lexer`).                            |
| `internal/parser/doc.go`           | Empty package skeleton.                                              |
| `internal/ast/doc.go`              | Empty package skeleton.                                              |
| `internal/validator/doc.go`        | Empty package skeleton.                                              |
| `internal/importer/doc.go`         | Empty package skeleton.                                              |
| `internal/gdast/doc.go`            | Empty package skeleton.                                              |
| `internal/generator/doc.go`        | Empty package skeleton.                                              |
| `internal/prototypes/doc.go`       | Empty package skeleton.                                              |

Each `internal/*/doc.go` is a single line `package <name>` plus a one-line package comment. Empty packages are intentional — future milestones land code into them.

`internal/cli/` and `internal/applog/` are real packages with code; everything else is a skeleton.

---

## Task 0: Initialize git repository and base hygiene files

**Goal:** A git repo on `main` with `.gitignore`, `.editorconfig`, and `README.md` committed.

**Files:**
- Create: `.gitignore`
- Create: `.editorconfig`
- Create: `README.md`
- Existing (already on disk, will be committed in this task): `docs/superpowers/specs/2026-04-27-gogdproto-design.md`, `docs/superpowers/plans/2026-04-27-m0-scaffolding.md`

**Acceptance Criteria:**
- [ ] `git rev-parse --is-inside-work-tree` returns `true`.
- [ ] Default branch is `main`.
- [ ] `.gitignore`, `.editorconfig`, `README.md`, and the two design/plan docs are in the initial commit.
- [ ] `git status` is clean after the commit.

**Verify:** `git log --oneline` shows one commit; `git status` is clean.

**Steps:**

- [ ] **Step 1: Initialize the repo**

```bash
cd /Users/christian/CafecitoGames/gogdproto
git init -b main
git remote add origin git@github.com:cafecito-games/gogdproto.git
```

- [ ] **Step 2: Write `.gitignore`**

```
# Binaries
bin/
dist/
*.test
*.out

# Coverage
coverage.out
coverage.html

# IDE
.idea/
.vscode/
*.swp
.DS_Store

# Go build/test cache only when explicitly placed locally
/.cache/

# Pre-commit
.prek-cache/
```

- [ ] **Step 3: Write `.editorconfig`**

```
root = true

[*]
charset = utf-8
end_of_line = lf
insert_final_newline = true
trim_trailing_whitespace = true
indent_style = space
indent_size = 4

[*.go]
indent_style = tab

[Makefile]
indent_style = tab

[{*.yml,*.yaml,*.json}]
indent_size = 2
```

- [ ] **Step 4: Write `README.md` (placeholder)**

```markdown
# gogdproto

A Protocol Buffers v3 compiler for GDScript (Godot 4.5), written in Go.

Reimplementation of [gdproto](https://github.com/csueiras/gdproto) with byte-identical output for shared fixtures.

See [design doc](docs/superpowers/specs/2026-04-27-gogdproto-design.md) for architecture and roadmap.

## Status

Pre-alpha. See open milestones for current work.
```

- [ ] **Step 5: Initial commit**

```bash
git add .gitignore .editorconfig README.md docs/
git commit -m "chore: initial scaffolding (gitignore, editorconfig, design doc)"
git status   # expect: nothing to commit, working tree clean
```

---

## Task 1: Go module + package skeletons

**Goal:** `go.mod` declares module `github.com/cafecito-games/gogdproto` at Go 1.26, and every `internal/*` package has a stub `doc.go` so `go build ./...` succeeds.

**Files:**
- Create: `go.mod`
- Create: `cmd/gogdproto/main.go` (minimal `package main` stub)
- Create: `internal/lexer/doc.go`
- Create: `internal/parser/doc.go`
- Create: `internal/ast/doc.go`
- Create: `internal/validator/doc.go`
- Create: `internal/importer/doc.go`
- Create: `internal/gdast/doc.go`
- Create: `internal/generator/doc.go`
- Create: `internal/prototypes/doc.go`
- Create: `internal/cli/doc.go`
- Create: `internal/applog/doc.go`

**Acceptance Criteria:**
- [ ] `go.mod` declares `module github.com/cafecito-games/gogdproto` and `go 1.26`.
- [ ] `go build ./...` succeeds with no errors.
- [ ] `go vet ./...` succeeds with no errors.

**Verify:** `go build ./... && go vet ./...` exits 0.

**Steps:**

- [ ] **Step 1: Initialize go.mod**

```bash
go mod init github.com/cafecito-games/gogdproto
# Confirm the file says `go 1.26`. If `go mod init` wrote a patch version (e.g. `go 1.26.2`),
# edit it down to `go 1.26` so the module declares language version, not toolchain.
```

Final `go.mod` should be:

```
module github.com/cafecito-games/gogdproto

go 1.26
```

- [ ] **Step 2: Write `cmd/gogdproto/main.go` (stub)**

```go
package main

func main() {}
```

This is replaced in Task 2. We need a buildable `package main` so `go build ./...` works now.

- [ ] **Step 3: Write each `internal/*/doc.go`**

For each package (`lexer`, `parser`, `ast`, `validator`, `importer`, `gdast`, `generator`, `prototypes`, `cli`, `applog`), create `internal/<name>/doc.go` with this content (substituting the package name):

```go
// Package lexer tokenizes .proto source files.
package lexer
```

Per-package one-liners:

| Package      | doc comment                                                    |
|--------------|----------------------------------------------------------------|
| `lexer`      | tokenizes .proto source files.                                 |
| `parser`     | parses lexer tokens into a proto AST.                          |
| `ast`        | defines proto AST node types.                                  |
| `validator`  | performs semantic validation on a proto AST.                   |
| `importer`   | resolves imported .proto files and marks external types.       |
| `gdast`      | builds GDScript abstract syntax trees and renders them.        |
| `generator`  | translates a proto AST into a GDScript AST.                    |
| `prototypes` | holds proto-to-GDScript type tables and wire-type constants.   |
| `cli`        | implements the gogdproto cobra CLI.                            |
| `applog`     | constructs slog loggers for components.                        |

- [ ] **Step 4: Verify build and vet**

```bash
go build ./...
go vet ./...
```

Expected: no output, exit 0.

- [ ] **Step 5: Commit**

```bash
git add go.mod cmd/ internal/
git commit -m "chore: scaffold Go module and internal package skeletons"
```

---

## Task 2: Cobra CLI skeleton with `--version`

**Goal:** `gogdproto --version` prints `gogdproto 0.1.0`. Tests drive this with TDD red/green.

**Files:**
- Create: `internal/cli/version.go`
- Create: `internal/cli/root.go`
- Create: `internal/cli/root_test.go`
- Modify: `cmd/gogdproto/main.go`

**Acceptance Criteria:**
- [ ] `gogdproto --version` prints `gogdproto 0.1.0\n` on stdout.
- [ ] `gogdproto --help` exits 0 and mentions the program name.
- [ ] `gogdproto` (no args) exits 0 and prints help (matches Cobra default for a root command with no Run).
- [ ] All tests pass via `go test ./internal/cli/...`.

**Verify:** `go test ./internal/cli/... -v -run TestRoot`

**Steps:**

- [ ] **Step 1: Add cobra dependency**

```bash
go get github.com/spf13/cobra@latest
go mod tidy
```

- [ ] **Step 2: Write the failing tests**

Create `internal/cli/root_test.go`:

```go
package cli_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/cafecito-games/gogdproto/internal/cli"
)

func TestRootVersionFlag(t *testing.T) {
	var out, errOut bytes.Buffer
	code := cli.Execute([]string{"--version"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", code, errOut.String())
	}
	got := out.String()
	want := "gogdproto 0.1.0\n"
	if got != want {
		t.Fatalf("version output = %q, want %q", got, want)
	}
}

func TestRootHelpFlag(t *testing.T) {
	var out, errOut bytes.Buffer
	code := cli.Execute([]string{"--help"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", code, errOut.String())
	}
	if !strings.Contains(out.String(), "gogdproto") {
		t.Fatalf("help output missing program name; got: %q", out.String())
	}
}

func TestRootNoArgsPrintsHelp(t *testing.T) {
	var out, errOut bytes.Buffer
	code := cli.Execute(nil, &out, &errOut)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", code, errOut.String())
	}
	if !strings.Contains(out.String(), "Usage:") {
		t.Fatalf("expected help output, got: %q", out.String())
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

```bash
go test ./internal/cli/... -v
```

Expected: compilation error (`undefined: cli.Execute`) — that's our failing red.

- [ ] **Step 4: Write `internal/cli/version.go`**

```go
package cli

const Version = "0.1.0"
```

- [ ] **Step 5: Write `internal/cli/root.go`**

```go
package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

// Execute runs the root command with the given args and IO streams.
// It returns the process exit code. A non-zero return means an error
// was reported on errOut.
func Execute(args []string, out, errOut io.Writer) int {
	cmd := newRootCommand(out, errOut)
	cmd.SetArgs(args)
	if err := cmd.Execute(); err != nil {
		// Cobra prints its own error; we just translate to exit code.
		return 1
	}
	return 0
}

func newRootCommand(out, errOut io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "gogdproto [flags] INPUT",
		Short:         "Protocol Buffers compiler for GDScript (Godot 4.5)",
		Long:          "gogdproto compiles .proto files to GDScript for use in Godot 4.5.",
		SilenceUsage:  true,
		SilenceErrors: false,
		Version:       Version,
	}

	cmd.SetOut(out)
	cmd.SetErr(errOut)

	// Match the gdproto Python CLI: print "gogdproto 0.1.0\n" exactly.
	cmd.SetVersionTemplate(fmt.Sprintf("gogdproto %s\n", Version))

	return cmd
}
```

- [ ] **Step 6: Wire `cmd/gogdproto/main.go`**

Replace the stub with:

```go
package main

import (
	"os"

	"github.com/cafecito-games/gogdproto/internal/cli"
)

func main() {
	os.Exit(cli.Execute(os.Args[1:], os.Stdout, os.Stderr))
}
```

- [ ] **Step 7: Run tests to verify they pass**

```bash
go test ./internal/cli/... -v
```

Expected: 3 tests PASS.

- [ ] **Step 8: Smoke-test the binary**

```bash
go run ./cmd/gogdproto --version    # → gogdproto 0.1.0
go run ./cmd/gogdproto --help       # → usage text, exit 0
```

- [ ] **Step 9: Commit**

```bash
git add go.mod go.sum cmd/ internal/cli/
git commit -m "feat(cli): cobra root command with --version"
```

---

## Task 3: `--log-level` flag and component-tagged `slog` loggers

**Goal:** CLI accepts `--log-level={debug,info,warn,error}` (default `warn`), constructs a root `*slog.Logger` writing to stderr, and provides `applog.For(name)` for components to derive a tagged child logger. TDD red/green.

**Files:**
- Create: `internal/applog/applog.go`
- Create: `internal/applog/applog_test.go`
- Modify: `internal/cli/root.go`
- Modify: `internal/cli/root_test.go`

**Acceptance Criteria:**
- [ ] `applog.ParseLevel(s)` returns the right `slog.Level` for `debug|info|warn|error` (case-insensitive) and an error otherwise.
- [ ] `applog.New(w, level)` returns a `*slog.Logger` writing JSON-by-default to `w` at `level`.
- [ ] `applog.For(parent, "lexer")` returns a logger that, when invoked, emits an attribute `component=lexer`.
- [ ] `applog.Discard()` returns a no-op logger usable in tests (writes to `io.Discard`).
- [ ] CLI `--log-level invalid` fails with a non-zero exit code and an error on stderr.
- [ ] CLI `--log-level debug` succeeds and the resulting root command stores a logger at `--log-level=debug`.

**Verify:** `go test ./internal/applog/... ./internal/cli/... -v`

**Steps:**

- [ ] **Step 1: Write the failing applog tests**

Create `internal/applog/applog_test.go`:

```go
package applog_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/cafecito-games/gogdproto/internal/applog"
)

func TestParseLevel(t *testing.T) {
	cases := map[string]slog.Level{
		"debug": slog.LevelDebug,
		"DEBUG": slog.LevelDebug,
		"info":  slog.LevelInfo,
		"warn":  slog.LevelWarn,
		"error": slog.LevelError,
	}
	for input, want := range cases {
		got, err := applog.ParseLevel(input)
		if err != nil {
			t.Errorf("ParseLevel(%q) error: %v", input, err)
			continue
		}
		if got != want {
			t.Errorf("ParseLevel(%q) = %v, want %v", input, got, want)
		}
	}
}

func TestParseLevelInvalid(t *testing.T) {
	if _, err := applog.ParseLevel("loud"); err == nil {
		t.Fatal("expected error for invalid level, got nil")
	}
}

func TestForAddsComponentAttribute(t *testing.T) {
	var buf bytes.Buffer
	parent := applog.New(&buf, slog.LevelDebug)
	logger := applog.For(parent, "lexer")
	logger.Info("hello")

	var record map[string]any
	if err := json.Unmarshal(buf.Bytes(), &record); err != nil {
		t.Fatalf("log line not JSON: %v; raw=%q", err, buf.String())
	}
	if got := record["component"]; got != "lexer" {
		t.Fatalf("component attr = %v, want %q", got, "lexer")
	}
	if got := record["msg"]; got != "hello" {
		t.Fatalf("msg = %v, want %q", got, "hello")
	}
}

func TestDiscardLoggerSwallowsOutput(t *testing.T) {
	logger := applog.Discard()
	logger.Info("nothing")
	// No way to assert silence directly; this test exists to ensure the function returns a usable logger.
	if logger == nil {
		t.Fatal("Discard() returned nil")
	}
}

func TestNewWritesAtLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := applog.New(&buf, slog.LevelWarn)
	logger.Debug("noisy")
	logger.Warn("important")

	out := buf.String()
	if strings.Contains(out, "noisy") {
		t.Errorf("debug message leaked at warn level: %q", out)
	}
	if !strings.Contains(out, "important") {
		t.Errorf("warn message missing: %q", out)
	}
}

// Compile-time check we did not accidentally export the wrong types.
var _ io.Writer = (*bytes.Buffer)(nil)
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/applog/... -v
```

Expected: compilation error (`undefined: applog.ParseLevel` etc).

- [ ] **Step 3: Write `internal/applog/applog.go`**

```go
package applog

import (
	"fmt"
	"io"
	"log/slog"
	"strings"
)

// ParseLevel converts a level name (case-insensitive) into a slog.Level.
// Accepted: debug, info, warn, error.
func ParseLevel(s string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("unknown log level %q (want debug|info|warn|error)", s)
	}
}

// New constructs a JSON-formatted slog.Logger writing to w at the given level.
func New(w io.Writer, level slog.Level) *slog.Logger {
	handler := slog.NewJSONHandler(w, &slog.HandlerOptions{Level: level})
	return slog.New(handler)
}

// For returns a child logger tagged with the given component name.
// Components should call this once at construction time.
func For(parent *slog.Logger, component string) *slog.Logger {
	if parent == nil {
		return Discard()
	}
	return parent.With("component", component)
}

// Discard returns a logger that writes nothing. Use in tests.
func Discard() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
}
```

- [ ] **Step 4: Run applog tests to verify pass**

```bash
go test ./internal/applog/... -v
```

Expected: all tests PASS.

- [ ] **Step 5: Add `--log-level` to the CLI (failing test first)**

Append to `internal/cli/root_test.go`:

```go
func TestRootInvalidLogLevel(t *testing.T) {
	var out, errOut bytes.Buffer
	code := cli.Execute([]string{"--log-level", "loud"}, &out, &errOut)
	if code == 0 {
		t.Fatalf("expected non-zero exit, got 0; stdout=%q stderr=%q", out.String(), errOut.String())
	}
	if !strings.Contains(errOut.String(), "log level") {
		t.Fatalf("expected error mentioning log level, got: %q", errOut.String())
	}
}

func TestRootValidLogLevel(t *testing.T) {
	var out, errOut bytes.Buffer
	code := cli.Execute([]string{"--log-level", "debug", "--help"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", code, errOut.String())
	}
}
```

- [ ] **Step 6: Run tests to verify the new ones fail**

```bash
go test ./internal/cli/... -v
```

Expected: `TestRootInvalidLogLevel` and `TestRootValidLogLevel` fail (unknown flag).

- [ ] **Step 7: Wire `--log-level` into `internal/cli/root.go`**

Replace `internal/cli/root.go` with:

```go
package cli

import (
	"fmt"
	"io"
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/cafecito-games/gogdproto/internal/applog"
)

// Execute runs the root command with the given args and IO streams.
// It returns the process exit code.
func Execute(args []string, out, errOut io.Writer) int {
	cmd := newRootCommand(out, errOut)
	cmd.SetArgs(args)
	if err := cmd.Execute(); err != nil {
		return 1
	}
	return 0
}

func newRootCommand(out, errOut io.Writer) *cobra.Command {
	var logLevelFlag string
	var rootLogger *slog.Logger

	cmd := &cobra.Command{
		Use:           "gogdproto [flags] INPUT",
		Short:         "Protocol Buffers compiler for GDScript (Godot 4.5)",
		Long:          "gogdproto compiles .proto files to GDScript for use in Godot 4.5.",
		SilenceUsage:  true,
		SilenceErrors: false,
		Version:       Version,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			level, err := applog.ParseLevel(logLevelFlag)
			if err != nil {
				return fmt.Errorf("invalid log level: %w", err)
			}
			rootLogger = applog.New(cmd.ErrOrStderr(), level)
			_ = rootLogger // future subcommands will read this through cmd context
			return nil
		},
	}

	cmd.SetOut(out)
	cmd.SetErr(errOut)
	cmd.SetVersionTemplate(fmt.Sprintf("gogdproto %s\n", Version))

	cmd.PersistentFlags().StringVar(
		&logLevelFlag, "log-level", "warn",
		"log verbosity (debug|info|warn|error)",
	)

	return cmd
}
```

- [ ] **Step 8: Run tests to verify everything passes**

```bash
go test ./internal/cli/... ./internal/applog/... -v
```

Expected: all tests PASS.

- [ ] **Step 9: Smoke-test the binary**

```bash
go run ./cmd/gogdproto --log-level debug --help    # → exit 0
go run ./cmd/gogdproto --log-level loud --help     # → non-zero exit, error on stderr
```

- [ ] **Step 10: Commit**

```bash
git add internal/applog/ internal/cli/ go.mod go.sum
git commit -m "feat(cli): --log-level flag and component-tagged slog logger"
```

---

## Task 4: Taskfile

**Goal:** `Taskfile.yml` with `build`, `test`, `test:cover`, `lint`, `fmt`, `tidy`, `ci`, `install` tasks. Default = `ci`. `task ci` is green.

**Files:**
- Create: `Taskfile.yml`

**Acceptance Criteria:**
- [ ] `task --list` prints all 8 expected tasks.
- [ ] `task build` produces `bin/gogdproto`.
- [ ] `task test` runs unit tests successfully.
- [ ] `task test:cover` produces `coverage.out`.
- [ ] `task fmt` runs `gofmt -w` and `goimports -w` (skip the goimports invocation if `goimports` is missing — see step 1).
- [ ] `task tidy` runs `go mod tidy` and verifies `go.sum` is unchanged in CI (the CI workflow re-checks).
- [ ] `task lint` runs `golangci-lint run`.
- [ ] `task ci` runs fmt-check + lint + test + build and exits 0.

**Verify:** `task ci` exits 0.

**Steps:**

- [ ] **Step 1: Decide on `goimports` policy**

`goimports` is not in the Go distribution. Add it as a `go install` step inside `fmt` so it's bootstrapped on demand:

```yaml
fmt:
  desc: Format Go source.
  cmds:
    - go fmt ./...
    - 'go run golang.org/x/tools/cmd/goimports@latest -w .'
```

This avoids requiring engineers to pre-install goimports. The first run is slower; subsequent runs use the build cache.

- [ ] **Step 2: Write `Taskfile.yml`**

```yaml
version: '3'

vars:
  BIN: bin/gogdproto
  PKG: ./...

tasks:
  default:
    desc: Run the full CI pipeline locally.
    cmds:
      - task: ci

  build:
    desc: Build the gogdproto binary into ./bin.
    cmds:
      - mkdir -p bin
      - go build -o {{.BIN}} ./cmd/gogdproto

  install:
    desc: Install gogdproto into $GOPATH/bin.
    cmds:
      - go install ./cmd/gogdproto

  test:
    desc: Run unit tests.
    cmds:
      - go test -race -count=1 {{.PKG}}

  test:cover:
    desc: Run tests with coverage; output coverage.out.
    cmds:
      - go test -race -count=1 -coverprofile=coverage.out -covermode=atomic {{.PKG}}
      - go tool cover -func=coverage.out | tail -1

  lint:
    desc: Run golangci-lint.
    cmds:
      - golangci-lint run

  fmt:
    desc: Format Go source.
    cmds:
      - go fmt {{.PKG}}
      - go run golang.org/x/tools/cmd/goimports@latest -w .

  fmt:check:
    desc: Verify all Go files are formatted (CI use).
    cmds:
      - |
        unformatted=$(gofmt -l . | grep -v '^vendor/' || true)
        if [ -n "$unformatted" ]; then
          echo "Unformatted files:"
          echo "$unformatted"
          exit 1
        fi
    silent: true

  tidy:
    desc: Run go mod tidy.
    cmds:
      - go mod tidy

  tidy:check:
    desc: Verify go.mod/go.sum are tidy (CI use).
    cmds:
      - go mod tidy
      - |
        if ! git diff --quiet -- go.mod go.sum; then
          echo "go.mod/go.sum are not tidy. Run 'task tidy' and commit the result."
          git --no-pager diff -- go.mod go.sum
          exit 1
        fi
    silent: true

  ci:
    desc: Full CI pipeline (fmt:check + tidy:check + lint + test + build).
    cmds:
      - task: fmt:check
      - task: tidy:check
      - task: lint
      - task: test
      - task: build
```

- [ ] **Step 3: Smoke-test each task**

```bash
task --list
task build           # produces bin/gogdproto
./bin/gogdproto --version   # → gogdproto 0.1.0
task test
task test:cover
task fmt
task fmt:check
task tidy
```

`lint` will be tested in Task 5; `ci` in Task 5 too. Skip `task lint` and `task ci` here — they will fail until Task 5 lands `.golangci.yml`.

- [ ] **Step 4: Commit**

```bash
git add Taskfile.yml
git commit -m "build: add Taskfile with build/test/lint/ci targets"
```

---

## Task 5: golangci-lint v2 configuration

**Goal:** `.golangci.yml` configured for golangci-lint v2 with the agreed linter set; `task lint` and `task ci` are green.

**Files:**
- Create: `.golangci.yml`

**Acceptance Criteria:**
- [ ] `golangci-lint run` exits 0 against the current code.
- [ ] `task lint` exits 0.
- [ ] `task ci` exits 0.
- [ ] Config uses golangci-lint v2 schema (`version: "2"`, `linters.enable`, etc.).

**Verify:** `task ci` exits 0.

**Steps:**

- [ ] **Step 1: Write `.golangci.yml` (v2 schema)**

```yaml
version: "2"

run:
  timeout: 5m
  tests: true

linters:
  default: none
  enable:
    - errcheck
    - govet
    - staticcheck
    - revive
    - gocritic
    - misspell
    - unused
    - ineffassign
    - unconvert
    - gosec
    - nilerr
    - bodyclose
    - errorlint
  settings:
    revive:
      rules:
        - name: blank-imports
        - name: context-as-argument
        - name: context-keys-type
        - name: dot-imports
        - name: empty-block
        - name: error-naming
        - name: error-return
        - name: error-strings
        - name: errorf
        - name: exported
          arguments:
            - "checkPrivateReceivers"
        - name: increment-decrement
        - name: indent-error-flow
        - name: package-comments
        - name: range
        - name: receiver-naming
        - name: redefines-builtin-id
        - name: superfluous-else
        - name: time-naming
        - name: unexported-return
        - name: unreachable-code
        - name: unused-parameter
        - name: var-declaration
        - name: var-naming
    gosec:
      excludes:
        - G104   # already covered by errcheck
    gocritic:
      enabled-tags:
        - diagnostic
        - performance
        - style
      disabled-checks:
        - ifElseChain
        - hugeParam   # noisy on AST node value receivers
  exclusions:
    rules:
      - path: _test\.go
        linters:
          - gosec
          - errcheck
          - revive

formatters:
  enable:
    - gofmt
    - goimports
```

- [ ] **Step 2: Run lint**

```bash
golangci-lint run
```

Expected: no findings, exit 0. If revive flags `package-comments` on any `doc.go`, the package comment we wrote in Task 1 should satisfy it. If anything fails, fix the source — *do not* loosen the linter config without recording why.

- [ ] **Step 3: Run full CI**

```bash
task ci
```

Expected: all stages pass, exit 0.

- [ ] **Step 4: Commit**

```bash
git add .golangci.yml
git commit -m "build: add golangci-lint v2 configuration"
```

---

## Task 6: Pre-commit (`prek`) configuration

**Goal:** `.pre-commit-config.yaml` runs trailing-whitespace, end-of-file-fixer, large-files-check, `task fmt:check`, `task lint`, and `task test` on every commit. Hooks installed via `prek install`.

**Files:**
- Create: `.pre-commit-config.yaml`

**Acceptance Criteria:**
- [ ] `prek install` succeeds and writes `.git/hooks/pre-commit`.
- [ ] `prek run --all-files` exits 0 against the current tree.
- [ ] An attempted commit on a deliberately-broken file (test step only) is blocked by the hook (test by formatting a file badly, then revert).

**Verify:** `prek run --all-files` exits 0.

**Steps:**

- [ ] **Step 1: Write `.pre-commit-config.yaml`**

```yaml
default_install_hook_types: [pre-commit]

repos:
  - repo: https://github.com/pre-commit/pre-commit-hooks
    rev: v5.0.0
    hooks:
      - id: trailing-whitespace
      - id: end-of-file-fixer
      - id: check-yaml
      - id: check-added-large-files
        args: [--maxkb=512]
      - id: check-merge-conflict

  - repo: local
    hooks:
      - id: task-fmt-check
        name: task fmt:check
        entry: task fmt:check
        language: system
        types: [go]
        pass_filenames: false

      - id: task-lint
        name: task lint
        entry: task lint
        language: system
        types: [go]
        pass_filenames: false

      - id: task-test
        name: task test (fast)
        entry: task test
        language: system
        types: [go]
        pass_filenames: false
        stages: [pre-commit]
```

- [ ] **Step 2: Install hooks**

```bash
prek install
```

Expected: writes `.git/hooks/pre-commit`.

- [ ] **Step 3: Run hooks across the repo**

```bash
prek run --all-files
```

Expected: every hook reports `Passed`.

- [ ] **Step 4: Commit**

```bash
git add .pre-commit-config.yaml
git commit -m "build: add pre-commit hooks via prek"
```

The commit itself exercises the hooks. If the commit is blocked, fix the issue and recommit (do not use `--no-verify`).

---

## Task 7: GitHub Actions workflow + push + green CI

**Goal:** `.github/workflows/pr.yml` runs `task ci` on `pull_request` and on `push` to `main`. Pushed branch shows a green run on GitHub.

**Files:**
- Create: `.github/workflows/pr.yml`

**Acceptance Criteria:**
- [ ] Workflow triggers on `pull_request` and `push: [main]`.
- [ ] Workflow installs Go 1.26, Task, golangci-lint at pinned versions.
- [ ] Workflow runs `task ci` and uploads `coverage.out` as an artifact.
- [ ] After pushing `main`, the workflow run succeeds in GitHub Actions.

**Verify:** `gh run list --branch main --limit 1` shows status `success`.

**Steps:**

- [ ] **Step 1: Write `.github/workflows/pr.yml`**

```yaml
name: CI

on:
  pull_request:
  push:
    branches: [main]

permissions:
  contents: read

jobs:
  ci:
    name: CI (${{ matrix.os }})
    runs-on: ${{ matrix.os }}
    strategy:
      fail-fast: false
      matrix:
        os: [ubuntu-latest]
    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Set up Go 1.26
        uses: actions/setup-go@v5
        with:
          go-version: '1.26.x'
          cache: true

      - name: Install Task
        uses: arduino/setup-task@v2
        with:
          version: 3.x
          repo-token: ${{ secrets.GITHUB_TOKEN }}

      - name: Install golangci-lint
        uses: golangci/golangci-lint-action@v6
        with:
          version: v2.11.4
          install-mode: binary
          args: --version

      - name: Run CI
        run: task ci

      - name: Run coverage
        run: task test:cover

      - name: Upload coverage
        if: always()
        uses: actions/upload-artifact@v4
        with:
          name: coverage-${{ matrix.os }}
          path: coverage.out
          if-no-files-found: error
```

- [ ] **Step 2: Push to GitHub**

```bash
git add .github/
git commit -m "ci: add GitHub Actions workflow for PR checks"
git push -u origin main
```

The `git push` will fire the pre-commit hook again; it should pass.

- [ ] **Step 3: Watch the run**

```bash
gh run watch --exit-status
```

Expected: exits 0 when the run finishes successfully. If it fails, fix the underlying issue and push again — do not skip steps in CI.

- [ ] **Step 4: Close out the M0 milestone**

```bash
gh issue close 1 --reason completed --comment "M0 complete: scaffolding green on CI."
```

---

## Self-Review

**Spec coverage:**

| Spec section          | Implemented in                                                       |
|-----------------------|----------------------------------------------------------------------|
| Repo layout           | Tasks 0, 1                                                           |
| `go 1.26` go.mod      | Task 1                                                               |
| Cobra CLI + flags     | Tasks 2, 3                                                           |
| `slog` component tags | Task 3 (`internal/applog`)                                           |
| Taskfile              | Task 4                                                               |
| golangci-lint v2      | Task 5                                                               |
| pre-commit + prek     | Task 6                                                               |
| GitHub Actions        | Task 7                                                               |
| `git init` + commit   | Task 0 + each subsequent task commits                                |
| `.gitignore`, `.editorconfig` | Task 0                                                       |
| README placeholder    | Task 0                                                               |
| No license/CONTRIBUTING/CODEOWNERS | (intentionally not created — private project)           |

**Placeholder scan:** No "TBD"/"TODO"/"implement later" left. Each step shows the actual file content or command. Linter exclusions are enumerated, not "appropriate exclusions".

**Type/identifier consistency:** `applog.Execute` was almost a typo — it's `cli.Execute` and `applog.{ParseLevel,New,For,Discard}`. `Version` constant referenced consistently. Cobra command builder named `newRootCommand` everywhere. Logger flag named `--log-level` everywhere.

**Risks worth flagging during execution:**

1. golangci-lint v2 schema can shift between minor versions; if `revive` rules names differ on the engineer's installed version, drop the unrecognized rule rather than pinning a different lint version.
2. The pinned `golangci-lint v2.11.4` in CI must match local. If a developer upgrades locally, also bump CI in the same PR.
3. `actions/setup-go` may resolve `1.26.x` to a patch newer than what we've used locally — that's fine; `go 1.26` in `go.mod` is a language version, not a toolchain pin.
