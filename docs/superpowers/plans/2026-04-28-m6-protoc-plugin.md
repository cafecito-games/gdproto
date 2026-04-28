# M6 — protoc-gen-gdscript Plugin Implementation Plan

> **For agentic workers:** Use superpowers-extended-cc:subagent-driven-development.

**Goal:** Port `gdproto/descriptor_converter.py` (466 lines) to `internal/descriptors` and `gdproto/plugin.py` (300 lines) to `cmd/protoc-gen-gdscript`. End state: `protoc --plugin=protoc-gen-gdscript=./protoc-gen-gdscript --gdscript_out=./gen example.proto` produces byte-identical output to the CLI for the same input.

**Architecture:** `descriptors.FromCodeGeneratorRequest(req *pluginpb.CodeGeneratorRequest) ([]*ast.ProtoFile, error)` converts protoc descriptors to our existing AST. The plugin binary reads `CodeGeneratorRequest` from stdin (length-delimited), runs the conversion, validates, generates, and writes `CodeGeneratorResponse` to stdout. Reuses `internal/generator` and `internal/gdast` unchanged.

**Tech Stack:** Adds `google.golang.org/protobuf` (the only new runtime dep — first in M6).

**Reference:**
- `~/foss/gdproto/src/gdproto/descriptor_converter.py` (466 lines).
- `~/foss/gdproto/src/gdproto/plugin.py` (300 lines).
- `~/foss/gdproto/tests/test_plugin.py` and `test_plugin_integration.py`.

**GitHub tracking:** [issue #7](https://github.com/cafecito-games/gogdproto/issues/7). One PR per milestone (`feat/m6-plugin`). **Auto-merge enabled only on the final task.**

---

## File Structure

| Path                                          | Responsibility                                           |
|-----------------------------------------------|----------------------------------------------------------|
| `internal/descriptors/doc.go`                 | New (package comment).                                   |
| `internal/descriptors/converter.go`           | `FromCodeGeneratorRequest`, message/enum/field conversions. |
| `internal/descriptors/converter_test.go`      | Unit tests with synthetic FileDescriptorProto.           |
| `cmd/protoc-gen-gdscript/main.go`             | Plugin binary: stdin/stdout protocol.                    |
| `cmd/protoc-gen-gdscript/main_test.go`        | Integration test (synthetic CodeGeneratorRequest).       |
| `Taskfile.yml` (modify)                       | Add a build target for the plugin binary.                |

The exported API:

```go
// internal/descriptors
func FromCodeGeneratorRequest(req *pluginpb.CodeGeneratorRequest) ([]*ast.ProtoFile, error)
func FromFileDescriptorProto(fdp *descriptorpb.FileDescriptorProto) (*ast.ProtoFile, error)
```

---

## Task 0: Add protobuf dependency + DescriptorConverter

**Goal:** Add `google.golang.org/protobuf` to `go.mod`. Create `internal/descriptors/{doc.go,converter.go,converter_test.go}` porting `descriptor_converter.py`.

**Files:**
- Create: `internal/descriptors/doc.go`
- Create: `internal/descriptors/converter.go`
- Create: `internal/descriptors/converter_test.go`
- Modify: `go.mod`, `go.sum` (add dependency)

**Reference Python:** `descriptor_converter.py` end-to-end. Key methods: `convert()`, `_convert_message()`, `_convert_field()`, `_convert_enum()`, `_convert_oneof()`, `_make_position()`.

**Acceptance:**
- [ ] `go.mod` has `google.golang.org/protobuf` in require block.
- [ ] `FromCodeGeneratorRequest` converts a synthetic request with one FileDescriptorProto into a `[]*ast.ProtoFile`.
- [ ] `FromFileDescriptorProto` converts a single file descriptor (fields, messages, enums, oneofs, maps).
- [ ] Tests exercise: scalar fields, message fields, repeated, oneof, map, nested messages, nested enums, package, imports.
- [ ] `task ci` green.
- [ ] **Auto-merge NOT enabled** on PR (TBD number).

## Strategy

Build synthetic `FileDescriptorProto` instances in tests (using `descriptorpb` types directly — no need to invoke `protoc`). For each test case, construct the descriptor in code, run conversion, and assert AST shape.

Map descriptor types → AST types:
- `FieldDescriptorProto.Type` enum → proto type names. The enum has TYPE_DOUBLE, TYPE_FLOAT, TYPE_INT64, TYPE_UINT64, TYPE_INT32, TYPE_FIXED64, TYPE_FIXED32, TYPE_BOOL, TYPE_STRING, TYPE_GROUP, TYPE_MESSAGE, TYPE_BYTES, TYPE_UINT32, TYPE_ENUM, TYPE_SFIXED32, TYPE_SFIXED64, TYPE_SINT32, TYPE_SINT64.
- For TYPE_MESSAGE/TYPE_ENUM: use `field.TypeName` (which is fully-qualified like `.pkg.Foo`).
- Maps: in proto descriptors, map fields are represented as repeated messages with a synthesized nested type that has `MAP_ENTRY` option set. Detect via `nested_type.options.map_entry == true`. Convert to `MapField` in AST.
- Oneofs: `field.oneof_index` points into `message.oneof_decl[]`.
- Position info isn't in descriptors (no source code locations by default unless source_code_info is requested). Set `Position{Line:0, Column:0}` for descriptor-derived nodes.

## Steps

1. Add dependency: `go get google.golang.org/protobuf@latest`.
2. Read `descriptor_converter.py` end-to-end.
3. Write `converter.go` with the conversion functions.
4. Write tests with synthetic descriptors.
5. `task ci` green.
6. Commit + push:
   ```bash
   git add go.mod go.sum internal/descriptors/
   git commit -m "feat(descriptors): convert protoc descriptors to AST"
   git push origin feat/m6-plugin
   ```

---

## Task 1: protoc-gen-gdscript plugin binary

**Goal:** `cmd/protoc-gen-gdscript/main.go` reads `CodeGeneratorRequest` from stdin, runs the pipeline, writes `CodeGeneratorResponse` to stdout.

**Files:**
- Create: `cmd/protoc-gen-gdscript/main.go`
- Create: `cmd/protoc-gen-gdscript/main_test.go`
- Modify: `Taskfile.yml` (build the plugin binary too).

**Reference Python:** `plugin.py` end-to-end.

**Acceptance:**
- [ ] Plugin reads bytes from stdin, parses as `CodeGeneratorRequest`, processes each `proto_file`, generates GDScript, writes a `CodeGeneratorResponse` to stdout.
- [ ] Wrapper filename: `proto/foo.proto` → `proto/foo.pb.gd` (snake_case, see `_convert_to_wrapper_filename` in plugin.py).
- [ ] On error: writes `CodeGeneratorResponse` with `error` field set (does NOT exit non-zero — protoc plugin convention).
- [ ] Test: feed a synthetic request, parse response, verify file content matches `examples/golden.gd` for the example.proto fixture.
- [ ] `task ci` green; `task build` produces both `bin/gogdproto` and `bin/protoc-gen-gdscript`.
- [ ] **Auto-merge NOT enabled.**

## Strategy

```go
package main

import (
    "io"
    "os"
    "google.golang.org/protobuf/proto"
    "google.golang.org/protobuf/types/pluginpb"
    "github.com/cafecito-games/gogdproto/internal/descriptors"
    "github.com/cafecito-games/gogdproto/internal/validator"
    "github.com/cafecito-games/gogdproto/internal/generator"
)

func main() {
    if err := run(os.Stdin, os.Stdout); err != nil {
        os.Exit(1)
    }
}

func run(in io.Reader, out io.Writer) error {
    data, err := io.ReadAll(in)
    if err != nil { return err }
    req := &pluginpb.CodeGeneratorRequest{}
    if err := proto.Unmarshal(data, req); err != nil { return err }
    resp := &pluginpb.CodeGeneratorResponse{}

    files, err := descriptors.FromCodeGeneratorRequest(req)
    if err != nil {
        s := err.Error()
        resp.Error = &s
    } else {
        for i, file := range files {
            // validate
            if errs := validator.Validate(file, file.Source); len(errs) != 0 {
                msg := "validation errors..."
                resp.Error = &msg
                break
            }
            // generate
            cls, gerr := generator.Generate(file, file.Source)
            if gerr != nil { continue }
            outName := convertToWrapperFilename(req.FileToGenerate[i])
            content := cls.ToGDScript(0)
            resp.File = append(resp.File, &pluginpb.CodeGeneratorResponse_File{
                Name: &outName,
                Content: &content,
            })
        }
    }
    out_bytes, err := proto.Marshal(resp)
    if err != nil { return err }
    _, err = out.Write(out_bytes)
    return err
}

func convertToWrapperFilename(path string) string {
    // ... port from plugin.py
}
```

## Steps

1. Read `plugin.py`.
2. Write `main.go`.
3. Write integration test: build a `CodeGeneratorRequest` with `example.proto` content, run `run`, parse response, compare to golden.
4. Update Taskfile to also build `protoc-gen-gdscript`.
5. `task ci` green.
6. **Enable auto-merge on the M6 PR** — this is the final task.
7. Watch CI; verify merge.
8. Close issue #7.
9. Sync local main.

## Acceptance

- [ ] Integration test produces byte-identical output to `examples/golden.gd`.
- [ ] PR auto-merges.
- [ ] Issue #7 closed.

---

## Self-Review

**Risks:**
1. Protobuf descriptor field types may have subtle differences from our parser-based AST. Watch for: nullability of `Number` (descriptors use `*int32` — defaults to 0 if unset), oneof `*int32` index, optional vs proto3-default semantics.
2. Map detection requires examining nested_types, not just the field. The Python converter handles this.
3. The plugin binary must NOT print to stderr unless it's a real error — protoc plugin protocol expects clean stdout.
