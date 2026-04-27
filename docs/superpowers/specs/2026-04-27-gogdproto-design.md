# gogdproto — Design

**Date:** 2026-04-27
**Status:** Approved (awaiting written-spec review)
**Repo:** github.com/cafecito-games/gogdproto

## Goal

A Go 1.26 reimplementation of [gdproto](https://github.com/csueiras/gdproto) (Python) that generates GDScript serialization/deserialization code from `.proto` files. Distributed as a single static binary `gogdproto` (CLI) plus `protoc-gen-gdscript` (protoc plugin). Output must be byte-identical to the Python tool for shared fixtures, except where the Python tool has bugs we intentionally fix (tracked in `PARITY.md`).

## Non-Goals

- Proto2 (the Python tool is proto3-only; we match).
- Well-known types, services/RPCs, JSON mapping, custom options, extensions — unsupported in the source tool, unsupported here.
- Publishing `gdast` as its own Go module in v1 (lives in `internal/gdast`; promotable later).

## Architecture

Pipeline mirrors the Python tool's pipeline so parity work is mechanical:

```
.proto file ──► lexer ──► parser ──► AST ──► importer ──► validator ──► generator ──► gdast ──► .gd file
                                                                              │
                                                       protoc CodeGeneratorRequest ──► descriptors ─┘
```

Only `cmd/` and `importer` perform filesystem I/O. All other packages are pure functions over values. Errors propagate as values; no `log.Fatal` / `os.Exit` outside `cmd/`.

## Repo Layout

```
gogdproto/
├── cmd/
│   ├── gogdproto/              # cobra CLI
│   └── protoc-gen-gdscript/    # protoc plugin (M6)
├── internal/
│   ├── lexer/                  # .proto tokenizer
│   ├── parser/                 # tokens → AST
│   ├── ast/                    # proto AST node types
│   ├── validator/              # semantic validation
│   ├── importer/               # import resolution + external type marking
│   ├── descriptors/            # protoc descriptor → AST (M6)
│   ├── gdast/                  # GDScript AST builder (Go port of gdast)
│   ├── generator/              # AST → gdast tree
│   └── prototypes/             # type/wire-type tables
├── examples/                   # .proto + golden .gd fixtures
├── testdata/                   # test fixtures
├── docs/superpowers/specs/     # design docs
├── .github/workflows/pr.yml
├── .pre-commit-config.yaml
├── .golangci.yml
├── Taskfile.yml
├── go.mod                      # module github.com/cafecito-games/gogdproto, go 1.26
├── PARITY.md                   # intentional deviations from Python gdproto
└── README.md
```

`internal/` keeps everything unimportable from outside the module.

## Components

Each component is a Go package with a single purpose, plain inputs/outputs, no shared global state. Each accepts a `*slog.Logger` at construction; default is `slog.New(slog.DiscardHandler)` for tests. Loggers are tagged at construction: `logger.With("component", "<name>")`.

| Package                | Responsibility                                                | I/O   |
|------------------------|---------------------------------------------------------------|-------|
| `internal/lexer`       | `Tokenize(source, filename) ([]Token, error)`                 | none  |
| `internal/ast`         | Plain structs for proto AST. Carries `Line`/`Column`.         | none  |
| `internal/parser`      | `Parse(tokens, filename) (*ast.ProtoFile, error)`             | none  |
| `internal/importer`    | `ResolveExternal(file, inputPath, fs FS) error`               | FS    |
| `internal/validator`   | `Validate(file, filename) []ValidationError` (multi-error)    | none  |
| `internal/gdast`       | GDScript AST nodes + `ToGDScript() string`                    | none  |
| `internal/generator`   | `Generate(file, sourceName) (*gdast.ClassDefinition, error)`  | none  |
| `internal/prototypes`  | Static maps: proto→GDScript types, defaults, wire types       | none  |
| `internal/descriptors` | `FromCodeGeneratorRequest(...) ([]*ast.ProtoFile, error)` (M6)| none  |
| `cmd/gogdproto`        | Cobra CLI; only place that reads input/writes output          | files |
| `cmd/protoc-gen-gdscript` | Reads `CodeGeneratorRequest` from stdin, writes response  | stdio |

### CLI Surface

Mirrors the Python `gdproto` CLI:

```
gogdproto [flags] INPUT
  -o, --output string      Output .gd file (required)
      --log-level string   debug|info|warn|error (default "warn")
      --version            Print version and exit
  -h, --help               Help for gogdproto
```

Exit codes match the Python tool: `0` success, `1` error, `130` interrupt.

### Plugin Surface (M6)

`protoc-gen-gdscript` reads a `CodeGeneratorRequest` from stdin and writes a `CodeGeneratorResponse` to stdout. Invoked via:

```
protoc --plugin=protoc-gen-gdscript=./protoc-gen-gdscript --gdscript_out=./gen foo.proto
```

Shares `internal/generator` and `internal/gdast` with the CLI; bypasses lexer/parser by feeding `internal/descriptors` output directly into the generator.

## Data Flow Notes

- The lexer never reads files; the CLI reads source text and passes it as a string.
- The importer is the *only* component besides `cmd/` that touches the filesystem. It accepts an `FS` interface (something close to `fs.FS` plus directory walk) so unit tests use an in-memory implementation.
- The validator returns a *slice* of errors (matching the Python tool), not a single error. The CLI prints them all before exiting non-zero.
- The generator produces a `gdast` tree; rendering to text happens at the CLI/plugin boundary via `tree.ToGDScript()`. Tests can assert on tree structure or rendered output.

## Testing Strategy

**Discipline: TDD red/green.** Every feature is a failing test first, then minimum code to pass. Commits look like `lexer: red — empty source` then `lexer: green — empty source`. Squash-on-merge is allowed.

- **Unit tests** in `_test.go` files alongside source. Standard library `testing` only. `github.com/google/go-cmp` permitted for deep diffs if needed.
- **Golden-file tests for the generator.** `testdata/golden/*.proto` ↔ `testdata/golden/*.gd`. Goldens are seeded by running the Python `gdproto` against fixtures and copying the output. `go test -update` regenerates goldens for legitimate changes.
- **Differential parity test.** A test harness invokes the Python `gdproto` against each fixture (skipped if not on `PATH`) and asserts byte-equality with our output. Gates parity without making CI depend on Python.
- **gdast tests separately.** Construct small trees, assert exact GDScript strings.
- **CLI tests** use Cobra's in-memory I/O; filesystem effects use `t.TempDir()`. No subprocess spawning of our own binary.
- **Coverage gate** in CI: combined coverage ≥ 85% (raised as milestones land; target ≥ 95% by M5).

### Parity Bug Policy

If the Python tool emits something we believe is buggy, we deviate intentionally:

1. Document the deviation in `PARITY.md` (input snippet, Python output, our output, reasoning).
2. Add a Go-only golden file for the fixture.
3. The differential test skips fixtures listed in `PARITY.md`.

We do not import bugs.

## Tooling

- **Go 1.26** (`go 1.26` in `go.mod`).
- **Runtime deps:** `github.com/spf13/cobra`. M6 adds `google.golang.org/protobuf`. Nothing else for v1.
- **golangci-lint** (`.golangci.yml`) with: `errcheck`, `govet`, `staticcheck`, `revive`, `gofmt`, `goimports`, `gocritic`, `misspell`, `unused`, `ineffassign`, `unconvert`, `gosec`, `nilerr`, `bodyclose`, `errorlint`. Pinned version in CI.
- **Taskfile.yml** tasks: `build`, `test`, `test:cover`, `lint`, `fmt`, `tidy`, `ci` (= fmt-check + lint + test + build), `install`. Default = `ci`. Hooks call into Taskfile so logic lives in one place.
- **`.pre-commit-config.yaml`** with: `go fmt`, `goimports`, `golangci-lint run`, `go test ./...` (fast unit tests), trailing-whitespace, end-of-file-fixer, large-file check. Installed via `prek install` after the file is committed.
- **GitHub Actions** at `.github/workflows/pr.yml`, triggered on `pull_request` and `push: [main]`:
  1. Checkout
  2. Setup Go 1.26 with module + build cache
  3. `task tidy` — fail if `go.sum` changes
  4. `task lint`
  5. `task test:cover`
  6. `task build`
  7. Upload coverage artifact
- **`.gitignore`** for Go (`bin/`, `coverage.out`, `dist/`), `.editorconfig`.
- No `LICENSE`, no `CONTRIBUTING.md`, no `CODEOWNERS` — private project.

## Milestones

Each milestone gets its own implementation plan (writing-plans skill) and is shipped/merged before the next starts. Each milestone has a tracking GitHub issue with sub-issues created during plan-writing.

### M0 — Scaffolding
`go.mod`, repo layout, Taskfile, `.golangci.yml`, GH Actions workflow, `.pre-commit-config.yaml`, `prek install`, cobra skeleton with `--version` and `--log-level`, `git init`, initial commit.
**Acceptance:** `task ci` passes locally and in CI on a hello-world binary.

### M1 — Lexer
Port `lexer.py`. All token types, error positions, comment handling. Tests ported from `tests/test_lexer.py`.

### M2 — AST + Parser
Port `ast_nodes.py` and `parser.py`. Tests from `tests/test_parser.py`. AST output validated structurally.

### M3 — Validator + Importer
Port `validator.py` plus `resolve_external_enum_types` from `cli.py`. FS interface for testability. Tests from `tests/test_validator.py` and `tests/test_external_enums.py`.

### M4 — gdast (Go)
Port `gdast/nodes.py` + `helpers.py`. Standalone package. Tests mirror `gdast/tests`. **Acceptance:** byte-identical output for the gdast README example.

### M5 — Generator + CLI parity
Port `generator/gdscript.py`, `templates.py`, `gdproto_helpers.py`. Wire up CLI. Golden tests against `~/foss/gdproto/examples/`. Differential parity test gates the milestone. **Acceptance:** `gogdproto example.proto -o out.gd` produces byte-identical output to Python `gdproto` for all fixtures (modulo `PARITY.md` deviations).

### M6 — `protoc-gen-gdscript` plugin
Add `google.golang.org/protobuf`. Port `descriptor_converter.py` to `internal/descriptors/`. Port `plugin.py` to `cmd/protoc-gen-gdscript/`. Tests: round-trip a synthetic `CodeGeneratorRequest`; integration test that shells out to `protoc` (skipped if not on PATH). **Acceptance:** `protoc --gdscript_out=./gen foo.proto` produces byte-identical output to the CLI for the same fixture.

## GitHub Project Setup

- **Milestones:** `M0 — Scaffolding`, `M1 — Lexer`, `M2 — AST + Parser`, `M3 — Validator + Importer`, `M4 — gdast`, `M5 — Generator + CLI parity`, `M6 — protoc plugin`.
- **Labels:** `area:lexer`, `area:parser`, `area:validator`, `area:generator`, `area:gdast`, `area:cli`, `area:plugin`, `area:tooling`, `area:importer`, `area:descriptors`; `type:test`, `type:bug`, `type:parity-deviation`, `type:docs`.
- **Tracking issues:** one per milestone with a checklist of sub-issues. Sub-issues created when each milestone's implementation plan is written.
