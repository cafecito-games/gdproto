# Per-class file output Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers-extended-cc:subagent-driven-development (recommended) or superpowers-extended-cc:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace gdproto's single-file `<Name>Proto` wrapper output with one `.pb.gd` file per message / top-level enum, prefixed via filename-derived or option-overridden prefix.

**Architecture:** A new prefix+name resolver feeds a rewritten generator that returns `[]GeneratedFile`. The CLI and protoc plugin iterate the slice and write per-class files. A vendored `gdproto/options.proto` extension defines the `class_prefix` file option, surfaced through both the parser path and the descriptor path.

**Tech Stack:** Go 1.26, `google.golang.org/protobuf`, Cobra CLI, GDScript 4.6 (Godot), Vest test framework, Buf, Taskfile.

**Project note:** The user commits manually for this project. All "commit" steps in this plan stage files (`git add ...`) and leave commit creation to the user. Do not run `git commit`.

**Spec:** `docs/superpowers/specs/2026-05-25-per-class-files-design.md`

---

## File Structure

| Path | Status | Responsibility |
| --- | --- | --- |
| `proto/gdproto/options.proto` | new | Extension definition for `gdproto.class_prefix` (proto2, FileOptions field 51000) |
| `internal/gdprotopb/options.pb.go` | new (generated, committed) | Go stubs for the extension; consumed by descriptor path |
| `internal/gdprotopb/embed.go` | new | `//go:embed` of `options.proto` plus a `Bytes()` accessor |
| `internal/generator/names.go` | new | `ResolvePrefix(file)`, `BuildNameResolver(files, prefixes)`, identifier validation |
| `internal/generator/names_test.go` | new | Unit tests for prefix and name resolution |
| `internal/generator/generator.go` | rewrite | Return `[]GeneratedFile`; orchestrate per-class emission |
| `internal/generator/messages.go` | modify | Build a single message into a file; nested messages become siblings |
| `internal/generator/{accessors,deserialize,serialize,fromtext,totext,oneofs,tostring}.go` | modify | Call new resolver for type references |
| `internal/generator/generator_test.go` | rewrite goldens | Snapshot per-file output |
| `internal/descriptors/converter.go` | modify | Surface `(gdproto.class_prefix)` from `FileOptions` |
| `internal/descriptors/converter_test.go` | modify | Cover option propagation |
| `internal/cli/root.go` | modify | `-o` = directory; `--print-options-proto` flag |
| `internal/cli/root_test.go` | modify | New directory semantics |
| `cmd/protoc-gen-gdscript/main.go` | modify | Emit multiple `CodeGeneratorResponse_File`s; `--print-options-proto` |
| `cmd/protoc-gen-gdscript/main_test.go` | modify | Snapshot per-file plugin output |
| `examples/example.proto` | unchanged | Sample input |
| `examples/golden/` | new directory | Replaces `examples/golden.gd` with one file per class |
| `examples/golden.gd` | delete | Superseded by `examples/golden/` |
| `tests/integration/options_proto_test.go` | new | End-to-end test of vendor + CLI + protoc + buf paths |
| `tests/integration/print_options_proto_test.go` | new | Verifies `--print-options-proto` byte-equality |
| `tests/godot/...` (Vest harness) | modify | Load per-class globals; round-trip wire bytes |
| `README.md` | modify | Document new install snippets, import requirement, option |
| `Taskfile.yml` | modify | Add `task gen-options` (regenerate Go stubs from options.proto) |

---

## Task 0: Vendor `gdproto/options.proto` and Go stubs

**Goal:** Ship the extension proto and committed Go stubs so the generator can read the option from descriptors and so `--print-options-proto` can hand users an authoritative copy.

**Files:**
- Create: `proto/gdproto/options.proto`
- Create: `internal/gdprotopb/options.pb.go` (generated)
- Create: `internal/gdprotopb/embed.go`
- Create: `internal/gdprotopb/embed_test.go`
- Modify: `Taskfile.yml`

**Acceptance Criteria:**
- [ ] `proto/gdproto/options.proto` defines `extend google.protobuf.FileOptions { optional string class_prefix = 51000; }` under `package gdproto;` and `syntax = "proto2";`.
- [ ] `internal/gdprotopb/options.pb.go` exists, is committed, exports `E_ClassPrefix`, and compiles with the current `google.golang.org/protobuf` version pinned in `go.mod`.
- [ ] `gdprotopb.Bytes()` returns the raw `.proto` source via `//go:embed`.
- [ ] `task gen-options` regenerates the stubs from `proto/gdproto/options.proto` and is idempotent (re-running produces no diff).

**Verify:**
- `go build ./...` → exits 0
- `go test ./internal/gdprotopb/...` → PASS
- `task gen-options && git diff --quiet internal/gdprotopb/` → exits 0

**Steps:**

- [ ] **Step 1: Author `proto/gdproto/options.proto`**

```proto
syntax = "proto2";

package gdproto;

import "google/protobuf/descriptor.proto";

option go_package = "github.com/cafecito-games/gdproto/internal/gdprotopb";

// Override the auto-derived class_name prefix for generated GDScript files.
// Field number 51000 is inside the documented 50000-99999 range reserved for
// internal/third-party extensions to FileOptions.
extend google.protobuf.FileOptions {
  optional string class_prefix = 51000;
}
```

- [ ] **Step 2: Add `gen-options` to `Taskfile.yml`**

```yaml
  gen-options:
    desc: Regenerate Go stubs for proto/gdproto/options.proto
    cmds:
      - protoc -I proto --go_out=. --go_opt=module=github.com/cafecito-games/gdproto proto/gdproto/options.proto
```

- [ ] **Step 3: Run `task gen-options`** to produce `internal/gdprotopb/options.pb.go`. Commit it to the repo so building gdproto does not require `protoc` at install time. If `protoc-gen-go` is missing, install with `go install google.golang.org/protobuf/cmd/protoc-gen-go@latest`.

- [ ] **Step 4: Add `internal/gdprotopb/embed.go`**

```go
package gdprotopb

import _ "embed"

//go:embed options.proto
var optionsProto []byte

// Bytes returns the embedded gdproto/options.proto source.
// The slice is read-only; callers must not modify it.
func Bytes() []byte {
    out := make([]byte, len(optionsProto))
    copy(out, optionsProto)
    return out
}
```

- [ ] **Step 5: Copy `proto/gdproto/options.proto` to `internal/gdprotopb/options.proto`** so the `//go:embed` directive can find it (Go embed only reads files in the same package directory).

```bash
cp proto/gdproto/options.proto internal/gdprotopb/options.proto
```

Add a `task` step to keep them in sync (extend `gen-options` to `cp` after running protoc).

- [ ] **Step 6: Write `internal/gdprotopb/embed_test.go`**

```go
package gdprotopb

import (
    "bytes"
    "os"
    "testing"
)

func TestBytesMatchesProtoFile(t *testing.T) {
    want, err := os.ReadFile("options.proto")
    if err != nil {
        t.Fatalf("read options.proto: %v", err)
    }
    if got := Bytes(); !bytes.Equal(got, want) {
        t.Fatalf("Bytes() differs from options.proto on disk")
    }
}

func TestExtensionDescriptorAvailable(t *testing.T) {
    if E_ClassPrefix == nil {
        t.Fatal("E_ClassPrefix not generated")
    }
    if got := E_ClassPrefix.TypeDescriptor().Number(); int32(got) != 51000 {
        t.Fatalf("unexpected extension number: got %d want 51000", got)
    }
}
```

- [ ] **Step 7: Verify**

```bash
go test ./internal/gdprotopb/...
```

Expected: PASS.

- [ ] **Step 8: Stage**

```bash
git add proto/gdproto/options.proto internal/gdprotopb/ Taskfile.yml
```

Do NOT commit — leave for the user.

---

## Task 1: Plumb `class_prefix` through both parser and descriptor paths

**Goal:** `ast.ProtoFile.Options["(gdproto.class_prefix)"]` is populated regardless of whether the file was parsed by our lexer or read from a `FileDescriptorProto`.

**Files:**
- Modify: `internal/descriptors/converter.go`
- Modify: `internal/descriptors/converter_test.go`
- Modify: `internal/parser/parser_test.go` (add coverage of the option-name keying)

**Acceptance Criteria:**
- [ ] Direct CLI parsing of `option (gdproto.class_prefix) = "Game";` populates `file.Options["(gdproto.class_prefix)"]` with the string `"Game"`. (Existing parser already does this — test confirms.)
- [ ] Descriptor conversion populates the same map key with the same string value when the descriptor's `FileOptions` carries the `gdproto.class_prefix` extension.
- [ ] If the option is absent, the map key is absent (not empty string).
- [ ] An invalid descriptor where the extension is present but holds an empty string surfaces as `""`; validation is deferred to Task 2.

**Verify:** `go test ./internal/parser/... ./internal/descriptors/... -run "ClassPrefix|Option" -v` → PASS

**Steps:**

- [ ] **Step 1: Write parser test** in `internal/parser/parser_test.go`:

```go
func TestParseClassPrefixOption(t *testing.T) {
    src := `syntax = "proto3";
import "gdproto/options.proto";
option (gdproto.class_prefix) = "Game";
message Hero { string name = 1; }
`
    tokens, err := lexer.Tokenize(src, "hero.proto")
    if err != nil { t.Fatal(err) }
    file, err := Parse(tokens, "hero.proto")
    if err != nil { t.Fatal(err) }
    got, ok := file.Options["(gdproto.class_prefix)"]
    if !ok { t.Fatal("option key not set") }
    if got != "Game" { t.Fatalf("got %v want Game", got) }
}
```

- [ ] **Step 2: Run parser test — expect PASS** (existing parser already handles `(...)` option names per `internal/parser/options.go`).

- [ ] **Step 3: Write descriptor test** in `internal/descriptors/converter_test.go`:

```go
func TestConvertFilePropagatesClassPrefix(t *testing.T) {
    opts := &descriptorpb.FileOptions{}
    proto.SetExtension(opts, gdprotopb.E_ClassPrefix, "Game")
    fd := &descriptorpb.FileDescriptorProto{
        Name:    proto.String("hero.proto"),
        Syntax:  proto.String("proto3"),
        Options: opts,
    }
    file, err := ConvertFile(fd, nil)
    if err != nil { t.Fatal(err) }
    got, ok := file.Options["(gdproto.class_prefix)"]
    if !ok { t.Fatal("option key not set") }
    if got != "Game" { t.Fatalf("got %v want Game", got) }
}

func TestConvertFileMissingClassPrefix(t *testing.T) {
    fd := &descriptorpb.FileDescriptorProto{
        Name:   proto.String("hero.proto"),
        Syntax: proto.String("proto3"),
    }
    file, _ := ConvertFile(fd, nil)
    if _, ok := file.Options["(gdproto.class_prefix)"]; ok {
        t.Fatal("option key should be absent")
    }
}
```

Add imports for `github.com/cafecito-games/gdproto/internal/gdprotopb`, `google.golang.org/protobuf/proto`, `google.golang.org/protobuf/types/descriptorpb`.

- [ ] **Step 4: Run descriptor test — expect FAIL** with the key missing.

- [ ] **Step 5: Implement** in `internal/descriptors/converter.go`. In the function that builds the `ast.ProtoFile` (around line 133 where `Options: map[string]any{}` is initialized for the file), add:

```go
if fdOpts := fd.GetOptions(); fdOpts != nil {
    if proto.HasExtension(fdOpts, gdprotopb.E_ClassPrefix) {
        v := proto.GetExtension(fdOpts, gdprotopb.E_ClassPrefix).(string)
        file.Options["(gdproto.class_prefix)"] = v
    }
}
```

Add the necessary imports to `converter.go`.

- [ ] **Step 6: Run all parser + descriptor tests** → PASS.

- [ ] **Step 7: Stage**

```bash
git add internal/descriptors/ internal/parser/parser_test.go
```

---

## Task 2: Prefix and name resolvers

**Goal:** Pure helpers that (a) compute a `.proto` file's class prefix and (b) map any proto FQN (e.g. `pkg.Player.Position`) to its generated GDScript class name (e.g. `ExamplePlayerPosition`).

**Files:**
- Create: `internal/generator/names.go`
- Create: `internal/generator/names_test.go`

**Acceptance Criteria:**
- [ ] `ResolvePrefix(file *ast.ProtoFile) (string, error)` returns the option value when present, else filename-derived prefix.
- [ ] Filename derivation: `example.proto` → `Example`, `game_state.proto` → `GameState`, `weird-name.proto` → `WeirdName`, `nested/foo_bar.proto` → `FooBar` (basename only).
- [ ] Prefix must match `^[A-Z][A-Za-z0-9]*$`. Option-supplied prefix that fails validation returns an error mentioning the offending value.
- [ ] `NewNameResolver(files []*ast.ProtoFile)` returns a resolver that, given a proto FQN, returns the generated class name.
- [ ] Resolver covers: top-level messages, nested messages (recursive), top-level enums, nested enums (for cross-file references, the nested enum still maps to the parent message's file — i.e. `ExamplePlayer.Status`).
- [ ] Unknown FQN returns `("", false)`; callers surface as an error.

**Verify:** `go test ./internal/generator/ -run "Prefix|NameResolver" -v` → PASS

**Steps:**

- [ ] **Step 1: Tests first** in `internal/generator/names_test.go`:

```go
package generator

import (
    "testing"
    "github.com/cafecito-games/gdproto/internal/ast"
)

func TestResolvePrefixFromFilename(t *testing.T) {
    cases := []struct{ in, want string }{
        {"example.proto", "Example"},
        {"game_state.proto", "GameState"},
        {"weird-name.proto", "WeirdName"},
        {"nested/foo_bar.proto", "FooBar"},
        {"v1/api.proto", "Api"},
    }
    for _, c := range cases {
        f := &ast.ProtoFile{Filename: c.in}
        got, err := ResolvePrefix(f)
        if err != nil { t.Fatalf("%s: %v", c.in, err) }
        if got != c.want { t.Fatalf("%s: got %q want %q", c.in, got, c.want) }
    }
}

func TestResolvePrefixFromOption(t *testing.T) {
    f := &ast.ProtoFile{
        Filename: "example.proto",
        Options:  map[string]any{"(gdproto.class_prefix)": "Game"},
    }
    got, _ := ResolvePrefix(f)
    if got != "Game" { t.Fatalf("got %q want Game", got) }
}

func TestResolvePrefixOptionValidation(t *testing.T) {
    bad := []string{"game", "1Game", "Game-X", ""}
    for _, v := range bad {
        f := &ast.ProtoFile{
            Filename: "example.proto",
            Options:  map[string]any{"(gdproto.class_prefix)": v},
        }
        if _, err := ResolvePrefix(f); err == nil {
            t.Fatalf("%q: expected error", v)
        }
    }
}

func TestNameResolverResolvesNested(t *testing.T) {
    // Build a synthetic ProtoFile with Player { Position }, GameState, top-level enum Foo.
    file := &ast.ProtoFile{
        Filename: "example.proto",
        Package:  "",
        Messages: []*ast.Message{
            {Name: "Player", NestedMessages: []*ast.Message{{Name: "Position"}}},
            {Name: "GameState"},
        },
        Enums: []*ast.Enum{{Name: "Foo"}},
    }
    r, err := NewNameResolver([]*ast.ProtoFile{file})
    if err != nil { t.Fatal(err) }
    cases := map[string]string{
        "Player":          "ExamplePlayer",
        "Player.Position": "ExamplePlayerPosition",
        "GameState":       "ExampleGameState",
        "Foo":             "ExampleFoo",
    }
    for fqn, want := range cases {
        got, ok := r.Lookup(fqn)
        if !ok { t.Fatalf("missing %s", fqn) }
        if got != want { t.Fatalf("%s: got %s want %s", fqn, got, want) }
    }
}

func TestNameResolverWithPackage(t *testing.T) {
    file := &ast.ProtoFile{
        Filename: "example.proto",
        Package:  "game.v1",
        Messages: []*ast.Message{{Name: "Player"}},
    }
    r, _ := NewNameResolver([]*ast.ProtoFile{file})
    got, ok := r.Lookup("game.v1.Player")
    if !ok || got != "ExamplePlayer" {
        t.Fatalf("got %q ok=%v", got, ok)
    }
}
```

- [ ] **Step 2: Run tests — expect compile error** (names.go not yet written).

- [ ] **Step 3: Implement** in `internal/generator/names.go`:

```go
package generator

import (
    "fmt"
    "path/filepath"
    "regexp"
    "strings"

    "github.com/cafecito-games/gdproto/internal/ast"
)

var prefixPattern = regexp.MustCompile(`^[A-Z][A-Za-z0-9]*$`)

// ResolvePrefix returns the GDScript class_name prefix for the given proto file.
// The option (gdproto.class_prefix) wins if present; otherwise the basename of
// the proto path is split on non-alphanumerics and PascalCased.
func ResolvePrefix(file *ast.ProtoFile) (string, error) {
    if raw, ok := file.Options["(gdproto.class_prefix)"]; ok {
        s, isString := raw.(string)
        if !isString {
            return "", fmt.Errorf("option (gdproto.class_prefix) must be a string, got %T", raw)
        }
        if !prefixPattern.MatchString(s) {
            return "", fmt.Errorf("option (gdproto.class_prefix) %q is not a valid GDScript identifier (must match %s)", s, prefixPattern.String())
        }
        return s, nil
    }
    base := strings.TrimSuffix(filepath.Base(file.Filename), ".proto")
    parts := regexp.MustCompile(`[^A-Za-z0-9]+`).Split(base, -1)
    var b strings.Builder
    for _, p := range parts {
        if p == "" { continue }
        // capitalize first rune, lower the rest
        b.WriteString(strings.ToUpper(p[:1]) + strings.ToLower(p[1:]))
    }
    out := b.String()
    if out == "" || !prefixPattern.MatchString(out) {
        return "", fmt.Errorf("cannot derive prefix from filename %q", file.Filename)
    }
    return out, nil
}

// NameResolver maps proto fully-qualified names to generated GDScript class
// names. It indexes every message and top-level enum across the provided files.
// Nested enums are NOT indexed: they are accessed via "<ParentClass>.<EnumName>"
// in generated GDScript, which generator code constructs from the parent name.
type NameResolver struct {
    // key: proto FQN with leading dot stripped (e.g. "pkg.Player.Position").
    classByFQN map[string]string
}

func NewNameResolver(files []*ast.ProtoFile) (*NameResolver, error) {
    r := &NameResolver{classByFQN: map[string]string{}}
    for _, f := range files {
        prefix, err := ResolvePrefix(f)
        if err != nil { return nil, err }
        scope := ""
        if f.Package != "" { scope = f.Package + "." }
        for _, e := range f.Enums {
            r.classByFQN[scope+e.Name] = prefix + e.Name
        }
        for _, m := range f.Messages {
            r.indexMessage(m, scope, prefix, "")
        }
    }
    return r, nil
}

func (r *NameResolver) indexMessage(m *ast.Message, packageScope, prefix, parentChain string) {
    name := parentChain + m.Name
    r.classByFQN[packageScope+name] = prefix + name
    // Nested enums are intentionally not indexed at top level.
    for _, nm := range m.NestedMessages {
        r.indexMessage(nm, packageScope, prefix, name)
    }
}

func (r *NameResolver) Lookup(fqn string) (string, bool) {
    fqn = strings.TrimPrefix(fqn, ".")
    s, ok := r.classByFQN[fqn]
    return s, ok
}
```

- [ ] **Step 4: Run tests** → PASS. If `ast.ProtoFile` lacks a `Filename` field, use `Path` or the equivalent field present in the AST; check `internal/ast/ast.go` first and adapt the test + resolver accordingly. The field that today stores the input path is the one to use.

- [ ] **Step 5: Stage**

```bash
git add internal/generator/names.go internal/generator/names_test.go
```

---

## Task 3: Rewrite `generator.Generate` for multi-file output

**Goal:** `generator.Generate` returns `[]GeneratedFile`; each generated file holds one message-class or one enum-wrapper-class. All cross-message type references resolve through `NameResolver`.

This is the largest task. It touches every generator file. Keep the diff focused: behavior changes only at the boundaries; per-field codegen logic for serialize/deserialize/accessors stays intact, only the `renderedType` output changes.

**Files:**
- Modify: `internal/generator/generator.go`
- Modify: `internal/generator/messages.go`
- Modify: `internal/generator/accessors.go`
- Modify: `internal/generator/serialize.go`
- Modify: `internal/generator/deserialize.go`
- Modify: `internal/generator/oneofs.go`
- Modify: `internal/generator/fromtext.go`
- Modify: `internal/generator/totext.go`
- Modify: `internal/generator/tostring.go`
- Modify: `internal/generator/generator_test.go`
- Delete: `examples/golden.gd`
- Create: `examples/golden/ExamplePlayer.pb.gd`, `ExamplePlayerPosition.pb.gd`, `ExampleGameState.pb.gd`, `ExamplePlayerStatus.pb.gd` (if PlayerStatus is top-level — see `examples/example.proto`; PlayerStatus is top-level so it gets its own wrapper file), `proto_core_utils.gd`

**Acceptance Criteria:**
- [ ] `generator.Generate(file, sourceName)` signature changes to return `([]GeneratedFile, error)`.
- [ ] Generated files for `examples/example.proto` match golden directory byte-for-byte.
- [ ] Cross-message references use prefixed global class names (no `Player.Position`, only `ExamplePlayerPosition`).
- [ ] Nested enums (`Player.Status` if it were nested) remain inline inside the parent message's file as `enum Status { ... }`, referenced internally as `Status.ONLINE` and externally as `ExamplePlayer.Status.ONLINE`.
- [ ] Top-level enums emit their own wrapper file: `class_name <Prefix><Enum> extends RefCounted` with `enum <Enum> { ... }` inside.
- [ ] Map-entry synthetic messages do NOT produce a file.
- [ ] Imported types resolve via `NameResolver` to their owning file's prefix.

**Verify:**
- `go test ./internal/generator/... -v` → PASS
- `diff -r examples/golden/ <(go run ./cmd/gdproto examples/example.proto -o /tmp/gd-out && ls /tmp/gd-out)` after Task 4 — for now, the generator tests carry the snapshot.

**Steps:**

- [ ] **Step 1: Define `GeneratedFile` in `generator.go`**

```go
// GeneratedFile is one rendered .gd source file.
type GeneratedFile struct {
    Filename  string                // e.g. "ExamplePlayer.pb.gd"
    ClassName string                // e.g. "ExamplePlayer"
    Class     *gdast.ClassDefinition
}

// Source renders the class to GDScript with a trailing newline.
func (gf GeneratedFile) Source() string {
    out := gf.Class.ToGDScript(0)
    if !strings.HasSuffix(out, "\n") {
        out += "\n"
    }
    return out
}
```

- [ ] **Step 2: Change `Generate` signature and body in `generator.go`**

The new `Generate` walks the AST once and emits one entry per top-level message (recursively producing siblings for nested messages) and one per top-level enum.

```go
func Generate(file *ast.ProtoFile, sourceName string) ([]GeneratedFile, error) {
    prefix, err := ResolvePrefix(file)
    if err != nil { return nil, err }

    resolver, err := NewNameResolver([]*ast.ProtoFile{file})
    if err != nil { return nil, err }

    g := &generator{
        file:       file,
        sourceName: sourceName,
        prefix:     prefix,
        resolver:   resolver,
    }
    g.annotateLocalEnumUsage()

    var out []GeneratedFile
    for _, e := range file.Enums {
        out = append(out, g.generateTopLevelEnumFile(e))
    }
    for _, m := range file.Messages {
        out = append(out, g.generateMessageFiles(m, "")...)
    }
    return out, nil
}
```

Add `prefix string` and `resolver *NameResolver` fields to the existing `generator` struct.

- [ ] **Step 3: Implement `generateTopLevelEnumFile`** in `generator.go`

```go
func (g *generator) generateTopLevelEnumFile(e *ast.Enum) GeneratedFile {
    className := g.prefix + e.Name
    body := []gdast.Node{generateEnum(e)}
    cls := &gdast.ClassDefinition{
        ClassNameDirective: className,
        Extends:            "RefCounted",
        HeaderComment:      headerCommentText(headerSourceName(g.sourceName)),
        Statements:         body,
        TightStatements:    true,
    }
    return GeneratedFile{
        Filename:  className + ".pb.gd",
        ClassName: className,
        Class:     cls,
    }
}
```

- [ ] **Step 4: Implement `generateMessageFiles`** in `messages.go`

It produces one `GeneratedFile` for the message, then recurses into nested messages producing siblings (not nested classes). Nested enums stay inline in the parent's class body. Map-entry synthetic messages produce no file.

Existing `generateMessage` returns a `gdast.ClassDefinition` containing nested classes. Restructure it so:

1. A new `generateMessageClass(m, qualifiedName)` returns a `gdast.ClassDefinition` for just this message (its fields, accessors, serialize/deserialize, oneofs, nested enums) with NO nested message classes inside.
2. `generateMessageFiles(m, parentChain)` calls `generateMessageClass`, wraps it in `GeneratedFile`, then appends the result of recursing into each `m.NestedMessages` with `parentChain = parentChain + m.Name`.

```go
func (g *generator) generateMessageFiles(m *ast.Message, parentChain string) []GeneratedFile {
    if m.MapEntry {
        return nil  // map-entry synthetic message: implementation detail.
    }
    qualified := parentChain + m.Name
    className := g.prefix + qualified
    cls := g.generateMessageClass(m, className)
    files := []GeneratedFile{{
        Filename:  className + ".pb.gd",
        ClassName: className,
        Class:     cls,
    }}
    for _, nm := range m.NestedMessages {
        if nm.MapEntry { continue }
        files = append(files, g.generateMessageFiles(nm, qualified)...)
    }
    return files
}
```

Note: `m.MapEntry` may not be the exact field name in `ast.Message`. The descriptor converter sets the equivalent today via `nested.GetOptions().GetMapEntry()`. Check `internal/ast/ast.go` for the field name on `ast.Message` and use that. If the AST does not track map-entry status directly, derive it from the synthetic name convention used in the descriptor converter (look at `internal/descriptors/converter.go` line ~113-167).

- [ ] **Step 5: Refactor `generateMessage` into `generateMessageClass`** in `messages.go`

Strip the part that emits nested messages as nested classes. Keep:
- Header comments.
- Fields block.
- Nested enums (loop `m.NestedEnums`, append `generateEnum(e)` to statements).
- Accessors.
- Oneofs.
- `to_bytes`, `from_bytes`, `to_text`, `from_text`, `to_string`.

Pass `qualifiedName` (e.g. `Player.Position`) into nested helpers that need to know the message's proto FQN for resolver lookups.

Set `cls.ClassNameDirective = className` so the file emits `class_name ExamplePlayerPosition` at the top.

- [ ] **Step 6: Rewrite `renderedType` to consult the resolver** in `generator.go`

```go
func (g *generator) renderedType(protoType, sourceFile, currentScope string) string {
    if t, ok := scalarTypeMap[protoType]; ok {
        return t
    }
    // Build candidate FQNs: try resolver against (a) raw protoType, (b) scope-qualified, (c) package-qualified.
    candidates := buildFQNCandidates(protoType, currentScope, g.file.Package)
    for _, c := range candidates {
        if name, ok := g.resolver.Lookup(c); ok {
            return name
        }
    }
    // Nested enum reference inside the same message: leave as-is (e.g. "Status").
    return protoType
}
```

`buildFQNCandidates` mirrors the logic in `isLocalEnumReference` (lines 227-258 of the current `generator.go`): try the proto type as-is, then walk the current scope upward concatenating it with the type, then prefix the package. This logic should be extracted as a shared helper (`internal/generator/names.go`) and reused by both the enum annotation pass and the new resolver shim.

Update all call sites of `renderedFieldType` / `renderedMapValueType` to pass the current message's scope (e.g. `Player.Position`) so the resolver can disambiguate. Easiest path: thread `qualifiedName` through `generateMessageClass` and the field helpers it calls.

- [ ] **Step 7: Update `generator_test.go` to compare against `examples/golden/`**

Replace the existing single-file snapshot test with a directory comparison:

```go
func TestGenerateExampleGoldenDirectory(t *testing.T) {
    src, err := os.ReadFile("../../examples/example.proto")
    if err != nil { t.Fatal(err) }
    tokens, _ := lexer.Tokenize(string(src), "examples/example.proto")
    file, _ := parser.Parse(tokens, "examples/example.proto")
    // ... resolve imports / validate as the CLI does ...
    files, err := Generate(file, "example.proto")
    if err != nil { t.Fatal(err) }

    goldenDir := "../../examples/golden"
    seen := map[string]bool{}
    for _, f := range files {
        seen[f.Filename] = true
        want, err := os.ReadFile(filepath.Join(goldenDir, f.Filename))
        if err != nil { t.Fatalf("missing golden for %s: %v", f.Filename, err) }
        if got := f.Source(); got != string(want) {
            t.Errorf("%s mismatch:\n--- want\n%s\n--- got\n%s", f.Filename, want, got)
        }
    }
    entries, _ := os.ReadDir(goldenDir)
    for _, ent := range entries {
        if ent.Name() == "proto_core_utils.gd" { continue }  // produced separately by CLI/plugin
        if !seen[ent.Name()] {
            t.Errorf("golden file not produced: %s", ent.Name())
        }
    }
}
```

- [ ] **Step 8: Generate the golden files for the first time**

The fast path: write the test, run it once, dump each `f.Source()` to `examples/golden/<f.Filename>`, then re-run the test to confirm equality. Hand-verify each file is structurally correct (top-level `class_name`, no stray nested classes, cross-references use prefixed names). The expected files for `examples/example.proto` are:

- `ExamplePlayerStatus.pb.gd` (top-level enum wrapper)
- `ExamplePlayer.pb.gd` (contains `Player` fields, `position` references `ExamplePlayerPosition`, oneof `contact`, map fields)
- `ExamplePlayerPosition.pb.gd`
- `ExampleGameState.pb.gd` (its `players` repeated field references `ExamplePlayer`)

Delete the old `examples/golden.gd`.

- [ ] **Step 9: Update all other generator tests** that exercise specific code paths (oneofs, maps, fromtext, totext). Each may need to declare the prefix it expects and compare against new fragment snapshots. Where a test asserts the produced GDScript string contains `Outer.Inner`, change it to `<Prefix>OuterInner`. Where it asserts `class Inner extends RefCounted`, change to expecting separate files.

- [ ] **Step 10: Run the whole generator test suite**

```bash
go test ./internal/generator/... -v
```

Expected: PASS. Iterate on mismatches; do not edit goldens to match buggy output — fix the generator.

- [ ] **Step 11: Stage**

```bash
git add internal/generator/ examples/golden/
git rm examples/golden.gd
```

---

## Task 4: CLI `-o` directory semantics and `--print-options-proto`

**Goal:** `gdproto -o <dir> foo.proto` writes per-class files into `<dir>/` plus `proto_core_utils.gd`. `gdproto --print-options-proto` writes the embedded options proto to stdout.

**Files:**
- Modify: `internal/cli/root.go`
- Modify: `internal/cli/root_test.go`

**Acceptance Criteria:**
- [ ] `-o some/dir` (with or without trailing slash) writes every `GeneratedFile` into `some/dir/<Filename>` plus `some/dir/proto_core_utils.gd`. Directory is created if missing.
- [ ] `-o some/file.gd` errors with: `-o must be a directory; per-message files are written inside it. Got: some/file.gd`.
- [ ] If `-o` is omitted, files write to the current working directory.
- [ ] One stderr line per invocation: `wrote N files to <dir>/`.
- [ ] `gdproto --print-options-proto` writes `gdprotopb.Bytes()` to stdout and exits 0; the flag short-circuits the normal pipeline (no input file required).

**Verify:** `go test ./internal/cli/... -v` → PASS

**Steps:**

- [ ] **Step 1: Write CLI tests** in `internal/cli/root_test.go`. Add tests for: directory output, file-path-for-output errors, printing options proto. Example:

```go
func TestCLIWritesPerClassFilesToDirectory(t *testing.T) {
    dir := t.TempDir()
    var out, errOut bytes.Buffer
    code := Execute([]string{"--output", dir, "../../examples/example.proto"}, &out, &errOut)
    if code != 0 { t.Fatalf("exit %d: %s", code, errOut.String()) }
    must := []string{
        "ExamplePlayer.pb.gd", "ExamplePlayerPosition.pb.gd",
        "ExampleGameState.pb.gd", "ExamplePlayerStatus.pb.gd",
        "proto_core_utils.gd",
    }
    for _, name := range must {
        if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
            t.Errorf("missing %s: %v", name, err)
        }
    }
}

func TestCLIRejectsFilePathAsOutput(t *testing.T) {
    var out, errOut bytes.Buffer
    code := Execute([]string{"--output", "some/file.gd", "../../examples/example.proto"}, &out, &errOut)
    if code == 0 { t.Fatal("expected non-zero exit") }
    if !strings.Contains(errOut.String(), "-o must be a directory") {
        t.Fatalf("missing directory error in %q", errOut.String())
    }
}

func TestCLIPrintOptionsProto(t *testing.T) {
    var out, errOut bytes.Buffer
    code := Execute([]string{"--print-options-proto"}, &out, &errOut)
    if code != 0 { t.Fatalf("exit %d: %s", code, errOut.String()) }
    if !bytes.Equal(out.Bytes(), gdprotopb.Bytes()) {
        t.Fatalf("stdout differs from gdprotopb.Bytes()")
    }
}
```

- [ ] **Step 2: Run tests — expect FAIL**.

- [ ] **Step 3: Update `runCompile`** in `internal/cli/root.go`:

Replace the section starting at line 110 (`cls, err := generator.Generate(...)`) through line 134 with:

```go
files, err := generator.Generate(file, sourceNameForCLI(inputPath))
if err != nil {
    return err
}

outDir := outputPath
if outDir == "" { outDir = "." }
if err := validateOutputDir(outDir); err != nil { return err }
if err := os.MkdirAll(outDir, 0o750); err != nil {
    return fmt.Errorf("create output dir: %w", err)
}

written := 0
for _, gf := range files {
    p := filepath.Join(outDir, gf.Filename)
    if err := os.WriteFile(p, []byte(gf.Source()), 0o644); err != nil { //nolint:gosec
        return fmt.Errorf("write %s: %w", p, err)
    }
    written++
}
sibling := filepath.Join(outDir, "proto_core_utils.gd")
if err := os.WriteFile(sibling, []byte(generator.GenerateProtoCoreUtilsRaw()), 0o644); err != nil { //nolint:gosec
    return fmt.Errorf("write %s: %w", sibling, err)
}
written++
_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "wrote %d files to %s/\n", written, outDir)
return nil
```

Add `validateOutputDir`:

```go
func validateOutputDir(p string) error {
    if strings.HasSuffix(p, ".gd") {
        return fmt.Errorf("-o must be a directory; per-message files are written inside it. Got: %s", p)
    }
    info, err := os.Stat(p)
    if err == nil && !info.IsDir() {
        return fmt.Errorf("-o must be a directory; per-message files are written inside it. Got: %s", p)
    }
    return nil
}
```

- [ ] **Step 4: Add `--print-options-proto`**

In `newRootCommand`, add the flag:

```go
var printOptionsProto bool
cmd.Flags().BoolVar(&printOptionsProto, "print-options-proto", false,
    "print the embedded gdproto/options.proto to stdout and exit")
```

At the top of `RunE`, before the `len(args) == 0` check:

```go
if printOptionsProto {
    _, err := cmd.OutOrStdout().Write(gdprotopb.Bytes())
    return err
}
```

Drop the now-redundant required-output check (since omitting `-o` is allowed). Keep the help fallback when no args and no flag are passed.

- [ ] **Step 5: Run CLI tests** → PASS.

- [ ] **Step 6: Stage**

```bash
git add internal/cli/
```

---

## Task 5: Protoc plugin emits multiple files

**Goal:** `protoc-gen-gdscript` emits one `CodeGeneratorResponse_File` per generated class plus `proto_core_utils.gd`. The plugin also supports `--print-options-proto` (handy when only the plugin is installed).

**Files:**
- Modify: `cmd/protoc-gen-gdscript/main.go`
- Modify: `cmd/protoc-gen-gdscript/main_test.go`

**Acceptance Criteria:**
- [ ] For each input file the protoc request asks to generate, the plugin emits one response file per `GeneratedFile` plus exactly one shared `proto_core_utils.gd`.
- [ ] Output filenames are flat (no package-derived subdirectories): `ExamplePlayer.pb.gd`, etc.
- [ ] When `os.Args` includes `--print-options-proto`, the binary writes the embedded proto to stdout and exits 0 without consuming stdin.
- [ ] Existing tests covering message ordering, transitive imports, and import-only files continue to pass under the new shape.

**Verify:** `go test ./cmd/protoc-gen-gdscript/... -v` → PASS

**Steps:**

- [ ] **Step 1: Update plugin test snapshots** in `cmd/protoc-gen-gdscript/main_test.go` to expect a slice of files per input. For the example.proto input, the expected outputs are the same set as Task 3, prefixed by the package's path if the test feeds a path like `pkg/example.proto` (the plugin's responsibility is just to emit; no path prefix is added).

- [ ] **Step 2: Update the plugin main**

Replace the loop body starting at line 91 (`class, err := generator.Generate(...)`) with:

```go
files, err := generator.Generate(file, name)
if err != nil {
    return err
}
for _, gf := range files {
    resp.File = append(resp.File, &pluginpb.CodeGeneratorResponse_File{
        Name:    proto.String(gf.Filename),
        Content: proto.String(gf.Source()),
    })
}
```

Keep the single emission of `proto_core_utils.gd` outside the per-file loop. Ensure the `seen` deduplication that prevents emitting the same generated file twice now keys off the filename (since multiple input files can produce overlapping class names only if users misconfigure prefixes — in that case we should error explicitly).

Add at the top of `main()`:

```go
for _, a := range os.Args[1:] {
    if a == "--print-options-proto" {
        if _, err := os.Stdout.Write(gdprotopb.Bytes()); err != nil { os.Exit(1) }
        return
    }
}
```

- [ ] **Step 3: Add a duplicate-filename guard** in the plugin's emission loop. If two `GeneratedFile`s want to write to the same filename, return an error: `class name collision: <Filename> emitted by both <protoA> and <protoB>; set option (gdproto.class_prefix) to disambiguate`.

- [ ] **Step 4: Run plugin tests** → PASS.

- [ ] **Step 5: Stage**

```bash
git add cmd/protoc-gen-gdscript/
```

---

## Task 6: Integration tests for documented install paths

**Goal:** Prove that the three install paths in the README — direct CLI, raw `protoc`, and `buf generate` — actually work when users follow the documented steps.

**Files:**
- Create: `tests/integration/options_proto_test.go`
- Create: `tests/integration/print_options_proto_test.go`
- Create: `tests/integration/fixtures/options/sample.proto`
- Create: `tests/integration/fixtures/options/buf.yaml`
- Create: `tests/integration/fixtures/options/buf.gen.yaml`

**Acceptance Criteria:**
- [ ] `go test -tags=integration ./tests/integration/...` passes locally when `protoc` and `buf` are installed.
- [ ] Each scenario asserts the generated directory contains `GameHero.pb.gd` and `proto_core_utils.gd` and that `GameHero.pb.gd` starts with `class_name GameHero`.
- [ ] If `protoc` / `buf` is absent, the corresponding subtest skips with a descriptive message rather than failing.
- [ ] CI is updated to install both tools and run the integration suite as a required job.

**Verify:**
```bash
go test -tags=integration ./tests/integration/... -v
```

**Steps:**

- [ ] **Step 1: Build the gdproto binaries into a tempdir** in test setup

```go
//go:build integration

package integration

import (
    "os"
    "os/exec"
    "path/filepath"
    "strings"
    "testing"
)

func buildBinaries(t *testing.T) (gdproto, plugin string) {
    t.Helper()
    dir := t.TempDir()
    gdproto = filepath.Join(dir, "gdproto")
    plugin = filepath.Join(dir, "protoc-gen-gdscript")
    for path, pkg := range map[string]string{
        gdproto: "./cmd/gdproto",
        plugin:  "./cmd/protoc-gen-gdscript",
    } {
        cmd := exec.Command("go", "build", "-o", path, pkg)
        cmd.Dir = repoRoot(t)
        out, err := cmd.CombinedOutput()
        if err != nil { t.Fatalf("build %s: %v\n%s", pkg, err, out) }
    }
    return gdproto, plugin
}

func repoRoot(t *testing.T) string {
    out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
    if err != nil { t.Fatalf("git rev-parse: %v", err) }
    return strings.TrimSpace(string(out))
}
```

- [ ] **Step 2: Write the `sample.proto` fixture**

```proto
syntax = "proto3";
import "gdproto/options.proto";

option (gdproto.class_prefix) = "Game";

message Hero {
  string name = 1;
  int32 hp = 2;
}
```

- [ ] **Step 3: Write the test for direct CLI**

```go
func TestCLIWithOptionsProto(t *testing.T) {
    gdproto, _ := buildBinaries(t)
    root := repoRoot(t)
    work := t.TempDir()
    // Stage sample.proto and gdproto/options.proto on a single include path.
    must(t, copyFile(filepath.Join(root, "proto/gdproto/options.proto"),
        filepath.Join(work, "gdproto/options.proto")))
    must(t, copyFile(filepath.Join(root, "tests/integration/fixtures/options/sample.proto"),
        filepath.Join(work, "sample.proto")))
    outDir := filepath.Join(work, "out")
    cmd := exec.Command(gdproto, "-o", outDir, filepath.Join(work, "sample.proto"))
    if out, err := cmd.CombinedOutput(); err != nil {
        t.Fatalf("gdproto: %v\n%s", err, out)
    }
    assertGameHeroGenerated(t, outDir)
}
```

Helper `assertGameHeroGenerated`:

```go
func assertGameHeroGenerated(t *testing.T, dir string) {
    t.Helper()
    data, err := os.ReadFile(filepath.Join(dir, "GameHero.pb.gd"))
    if err != nil { t.Fatalf("read GameHero.pb.gd: %v", err) }
    if !strings.Contains(string(data), "class_name GameHero") {
        t.Fatalf("missing class_name directive in GameHero.pb.gd")
    }
    if _, err := os.Stat(filepath.Join(dir, "proto_core_utils.gd")); err != nil {
        t.Fatalf("missing proto_core_utils.gd: %v", err)
    }
}
```

- [ ] **Step 4: Write the protoc test**

```go
func TestProtocWithOptionsProto(t *testing.T) {
    if _, err := exec.LookPath("protoc"); err != nil {
        t.Skip("protoc not installed; skipping")
    }
    _, plugin := buildBinaries(t)
    root := repoRoot(t)
    work := t.TempDir()
    must(t, copyFile(filepath.Join(root, "proto/gdproto/options.proto"),
        filepath.Join(work, "gdproto/options.proto")))
    must(t, copyFile(filepath.Join(root, "tests/integration/fixtures/options/sample.proto"),
        filepath.Join(work, "sample.proto")))
    outDir := filepath.Join(work, "out")
    os.MkdirAll(outDir, 0o755)
    cmd := exec.Command("protoc",
        "--plugin=protoc-gen-gdscript="+plugin,
        "-I", work,
        "--gdscript_out", outDir,
        "sample.proto",
    )
    if out, err := cmd.CombinedOutput(); err != nil {
        t.Fatalf("protoc: %v\n%s", err, out)
    }
    assertGameHeroGenerated(t, outDir)
}
```

- [ ] **Step 5: Write the buf test**

`tests/integration/fixtures/options/buf.yaml`:

```yaml
version: v2
modules:
  - path: .
```

`tests/integration/fixtures/options/buf.gen.yaml`:

```yaml
version: v2
plugins:
  - local: protoc-gen-gdscript
    out: out
```

```go
func TestBufWithOptionsProto(t *testing.T) {
    if _, err := exec.LookPath("buf"); err != nil {
        t.Skip("buf not installed; skipping")
    }
    _, plugin := buildBinaries(t)
    root := repoRoot(t)
    work := t.TempDir()
    must(t, copyFile(filepath.Join(root, "proto/gdproto/options.proto"),
        filepath.Join(work, "gdproto/options.proto")))
    must(t, copyFile(filepath.Join(root, "tests/integration/fixtures/options/sample.proto"),
        filepath.Join(work, "sample.proto")))
    must(t, copyFile(filepath.Join(root, "tests/integration/fixtures/options/buf.yaml"),
        filepath.Join(work, "buf.yaml")))
    must(t, copyFile(filepath.Join(root, "tests/integration/fixtures/options/buf.gen.yaml"),
        filepath.Join(work, "buf.gen.yaml")))
    cmd := exec.Command("buf", "generate")
    cmd.Dir = work
    cmd.Env = append(os.Environ(), "PATH="+filepath.Dir(plugin)+string(os.PathListSeparator)+os.Getenv("PATH"))
    if out, err := cmd.CombinedOutput(); err != nil {
        t.Fatalf("buf: %v\n%s", err, out)
    }
    assertGameHeroGenerated(t, filepath.Join(work, "out"))
}
```

Add `copyFile`, `must` helpers as small utilities at the top of the test file.

- [ ] **Step 6: Write `print_options_proto_test.go`**

```go
//go:build integration

package integration

import (
    "bytes"
    "os"
    "os/exec"
    "path/filepath"
    "testing"
)

func TestPrintOptionsProtoMatchesRepo(t *testing.T) {
    gdproto, _ := buildBinaries(t)
    out, err := exec.Command(gdproto, "--print-options-proto").Output()
    if err != nil { t.Fatalf("gdproto --print-options-proto: %v", err) }
    want, err := os.ReadFile(filepath.Join(repoRoot(t), "proto/gdproto/options.proto"))
    if err != nil { t.Fatal(err) }
    if !bytes.Equal(out, want) {
        t.Fatalf("--print-options-proto output differs from proto/gdproto/options.proto on disk")
    }
}
```

- [ ] **Step 7: Wire CI** in `.github/workflows/<existing-CI>.yml` (find the existing workflow first; if there are multiple jobs, add this to the test job): install `protoc` and `buf`, then run `go test -tags=integration ./tests/integration/...`. Exact snippet depends on the existing workflow file structure; check it before editing.

- [ ] **Step 8: Run locally**

```bash
go test -tags=integration ./tests/integration/... -v
```

PASS.

- [ ] **Step 9: Stage**

```bash
git add tests/integration/ .github/workflows/
```

---

## Task 7: Update Vest/Godot runtime harness

**Goal:** The existing Vest suite continues to round-trip wire bytes through the generated GDScript, now loaded via `class_name` globals.

**Files:**
- Modify: the Vest test scripts under `tests/` that today load `example.gd`. Find them with `rg "preload.*example|ExampleProto\\." tests/` first.

**Acceptance Criteria:**
- [ ] All existing Vest tests pass against the new per-class output for `examples/example.proto`.
- [ ] At least one new Vest test exercises a two-`.proto` scenario (one file imports another) to prove `class_name` globals resolve across generated files in Godot.

**Verify:** the existing `task` target that runs Vest (check `Taskfile.yml`; likely `task test:godot` or similar) → PASS

**Steps:**

- [ ] **Step 1: Find Vest test files** that reference the old wrapper class

```bash
rg -l "ExampleProto|preload.*example" tests/
```

- [ ] **Step 2: Update each test** to use the new global names:
  - `ExampleProto.Player.new()` → `ExamplePlayer.new()`
  - `ExampleProto.Player.Position.new()` → `ExamplePlayerPosition.new()`
  - `ExampleProto.PlayerStatus.ONLINE` → `ExamplePlayerStatus.PlayerStatus.ONLINE`
  - Remove any `preload("res://.../example.gd")` lines — class_name globals don't need preloads.

- [ ] **Step 3: Add a multi-file fixture** `tests/godot/fixtures/multi/{a,b}.proto` where `b.proto` imports `a.proto` and uses a message from it. Run `gdproto` against both, then write a Vest test that constructs the cross-file message, serializes, deserializes, and asserts equality.

- [ ] **Step 4: Run** the Vest suite (whichever task target is canonical — `task` lists them with `task --list`):

```bash
task test:godot   # or the actual target name found in Taskfile.yml
```

PASS.

- [ ] **Step 5: Stage**

```bash
git add tests/
```

---

## Task 8: README and docs

**Goal:** README explicitly documents the import requirement for `gdproto/options.proto`, the three install paths, and the new CLI directory semantics.

**Files:**
- Modify: `README.md`
- Modify: relevant pages under `website/` (Docusaurus)

**Acceptance Criteria:**
- [ ] README's "Quick Usage" section shows the new CLI invocation with `-o godot/generated/` and lists the files produced.
- [ ] A dedicated "Custom prefix" section explains the `gdproto.class_prefix` option, shows the required `import "gdproto/options.proto";`, and documents three install paths: (a) vendor via `gdproto --print-options-proto > proto/gdproto/options.proto`, (b) raw `protoc -I path/to/gdproto/proto ...`, (c) `buf` with a vendored copy.
- [ ] The README states explicitly that the import is required for `protoc` and `buf`; the direct CLI honors the option without it as a convenience.
- [ ] A breaking-change callout near the top of the README + a `CHANGELOG` (or release notes) entry covering removal of `<Name>Proto` wrapper output.

**Verify:** README integration scenarios (Task 6) executed against snippets copied verbatim from the README pass.

**Steps:**

- [ ] **Step 1: Edit `README.md`.** Replace the existing Quick Usage block and add a "Custom prefix" section. Exact prose can mirror this design doc's "Custom file option" section. Include the field-number footnote (51000 from the 50000-99999 third-party range, with a link to https://protobuf.dev/programming-guides/proto3/#customoptions).

- [ ] **Step 2: Mirror the changes in the Docusaurus site** under `website/docs/`. Find affected pages with `rg -l "ExampleProto|class_name|preload" website/docs/`.

- [ ] **Step 3: Manual verification.** Copy each install snippet verbatim into a temp dir and run it by hand once. Adjust prose if anything is awkward. The Task 6 integration tests cover this automatically going forward.

- [ ] **Step 4: Stage**

```bash
git add README.md website/
```

---

## Self-review notes (post-write)

- Spec coverage: every section of `2026-05-25-per-class-files-design.md` has at least one task. Naming/layout → Tasks 2, 3. Custom option → Tasks 0, 1. Generator changes → Tasks 2, 3. CLI/plugin surface → Tasks 4, 5. Testing strategy → Tasks 3 (golden), 6 (integration), 7 (Godot), 8 (README verify).
- Placeholder scan: each step shows concrete code or a concrete command. The few "find with rg" steps in Tasks 7-8 are deliberate (the exact files vary with the project state, and the search command is exact).
- Type consistency: `GeneratedFile{Filename, ClassName, Class}` and `NameResolver.Lookup(fqn) (string, bool)` are used identically across Tasks 2-5.
- Removed/changed APIs are called out at their first use (Task 3 changes `Generate` signature; Tasks 4 and 5 are the consumers).
- One known-unknown surfaced as an instruction to the implementer: the exact field name on `ast.Message` that flags map-entry synthetic messages — Step 4 of Task 3 says to check `ast.go` and `descriptors/converter.go` and use whatever's there. This is preferable to hard-coding a guessed field name.
