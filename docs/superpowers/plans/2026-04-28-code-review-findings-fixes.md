# Code Review Findings Fixes Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix generator and entry-point defects so imported, nested, and enum-typed references generate correct GDScript and CLI/plugin output stays consistent.

**Architecture:** Preserve full type identity from parse/descriptor conversion through generation, then render local and imported references from resolved metadata rather than short-name heuristics. Keep the fix incremental: lock behavior with failing tests first, then change type rendering and enum handling, then finish with parity and documentation updates.

**Tech Stack:** Go 1.26, Cobra CLI, `google.golang.org/protobuf` descriptors/plugin protocol, existing `go test` suite

---

### Task 1: Lock In The Failing Behaviors With Tests

**Files:**
- Create: none
- Modify: `internal/cli/root_test.go`
- Modify: `cmd/protoc-gen-gdscript/main_test.go`
- Modify: `internal/generator/generator_test.go`
- Test: `internal/cli/root_test.go`
- Test: `cmd/protoc-gen-gdscript/main_test.go`
- Test: `internal/generator/generator_test.go`

- [ ] **Step 1: Write failing CLI and plugin parity/import tests**

Add the following tests:

```go
func TestRootUsesPathAwareWrapperClassName(t *testing.T) {
	tempDir := t.TempDir()
	inputDir := filepath.Join(tempDir, "foo")
	if err := os.MkdirAll(inputDir, 0o755); err != nil {
		t.Fatal(err)
	}
	inputPath := filepath.Join(inputDir, "bar.proto")
	if err := os.WriteFile(inputPath, []byte("syntax = \"proto3\"; message A {}"), 0o644); err != nil {
		t.Fatal(err)
	}
	outPath := filepath.Join(tempDir, "out.gd")

	var out, errOut bytes.Buffer
	code := cli.Execute([]string{inputPath, "-o", outPath}, &out, &errOut)
	if code != 0 {
		t.Fatalf("exit code = %d; stderr=%q", code, errOut.String())
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "class_name FooBarProto\n") {
		t.Fatalf("output missing path-aware wrapper class:\n%s", string(data))
	}
}
```

```go
func TestRunPreservesNestedTypeQualification(t *testing.T) {
	req := codeGeneratorRequest(t, map[string]string{
		"nested.proto": `syntax = "proto3";
message Outer { message Inner {} }
message Uses { Outer.Inner inner = 1; }`,
	}, []string{"nested.proto"})

	got := runPlugin(t, req)
	if !strings.Contains(got, "var _inner: Outer.Inner = null") {
		t.Fatalf("missing qualified field type:\n%s", got)
	}
	if !strings.Contains(got, "_inner = Outer.Inner.new()") {
		t.Fatalf("missing qualified constructor:\n%s", got)
	}
}
```

```go
func TestGenerateImportedMessageUsesWrapperQualification(t *testing.T) {
	file := &ast.ProtoFile{
		Syntax: "proto3",
		Messages: []*ast.Message{{
			Name: "Uses",
			Fields: []*ast.Field{{
				FieldType:    "Shared",
				FullTypePath: "Shared",
				SourceFile:   "common.proto",
				Name:         "shared",
				Number:       1,
			}},
		}},
	}

	class, err := generator.Generate(file, "main.proto")
	if err != nil {
		t.Fatal(err)
	}
	got := class.ToGDScript(0)
	if !strings.Contains(got, "var _shared: CommonProto.Shared = null") {
		t.Fatalf("missing imported wrapper qualification:\n%s", got)
	}
}
```

Also add generator tests for:

```go
func TestGenerateMapEnumUsesVarintPaths(t *testing.T)
func TestGenerateMessageEnumNameCollisionDoesNotUseEnumPaths(t *testing.T)
```

- [ ] **Step 2: Run targeted tests to verify they fail**

Run:

```bash
go test ./internal/cli ./cmd/protoc-gen-gdscript ./internal/generator -run 'TestRootUsesPathAwareWrapperClassName|TestRunPreservesNestedTypeQualification|TestGenerateImportedMessageUsesWrapperQualification|TestGenerateMapEnumUsesVarintPaths|TestGenerateMessageEnumNameCollisionDoesNotUseEnumPaths' -v
```

Expected:
- FAIL because the CLI still emits `BarProto`
- FAIL because plugin generation still emits `Inner`
- FAIL because imported references are still bare names
- FAIL because map enum generation still routes through message helpers
- FAIL because enum/message short-name collision still misclassifies the field

- [ ] **Step 3: Commit the failing tests**

```bash
git add internal/cli/root_test.go cmd/protoc-gen-gdscript/main_test.go internal/generator/generator_test.go
git commit -m "test: cover reviewed generator regressions"
```

### Task 2: Preserve Full Type Identity And Generate Correct References

**Files:**
- Modify: `internal/descriptors/converter.go`
- Modify: `internal/generator/generator.go`
- Modify: `internal/generator/messages.go`
- Modify: `internal/generator/accessors.go`
- Modify: `internal/generator/serialize.go`
- Modify: `internal/generator/deserialize.go`
- Test: `cmd/protoc-gen-gdscript/main_test.go`
- Test: `internal/generator/generator_test.go`

- [ ] **Step 1: Update descriptor conversion to preserve qualified local type names**

Change descriptor conversion so message/enum references keep both:

```go
fullPath := strings.TrimPrefix(f.GetTypeName(), ".")
field.FullTypePath = fullPath
field.FieldType = typeNameForFile(fullPath, fd.GetPackage())
field.SourceFile = c.typeRegistry[fullPath]
field.IsEnum = f.GetType() == descriptorpb.FieldDescriptorProto_TYPE_ENUM
```

and for map values:

```go
fullPath := strings.TrimPrefix(valueDescriptor.GetTypeName(), ".")
mf.FullValueTypePath = fullPath
mf.ValueType = typeNameForFile(fullPath, fd.GetPackage())
mf.ValueSourceFile = c.typeRegistry[fullPath]
mf.ValueIsEnum = valueDescriptor.GetType() == descriptorpb.FieldDescriptorProto_TYPE_ENUM
```

Add helper behavior equivalent to:

```go
func typeNameForFile(fullPath, pkg string) string {
	if pkg == "" {
		return fullPath
	}
	prefix := pkg + "."
	if strings.HasPrefix(fullPath, prefix) {
		return strings.TrimPrefix(fullPath, prefix)
	}
	return fullPath
}
```

- [ ] **Step 2: Replace short-name enum inference with per-field rendering helpers**

In `internal/generator/generator.go`, remove the `enumTypes` registry and add explicit helpers that resolve:

```go
func (g *generator) renderedFieldType(f *ast.Field) string
func (g *generator) renderedMapValueType(mf *ast.MapField) string
func (g *generator) renderedType(protoType, fullPath, sourceFile string) string
```

Rules:
- scalars still map through `scalarTypeMap`
- same-file types render as `FieldType` / `ValueType`
- imported types render as `<WrapperClass>.<TypeWithinImportedFile>`
- imported wrapper class name comes from `wrapperClassName(sourceFile)`

Update declarations and accessors to use these helpers:

```go
gdType := g.renderedFieldType(f)
valueType := g.renderedMapValueType(mf)
```

- [ ] **Step 3: Fix serialization and deserialization enum paths**

Update regular field logic to rely on metadata:

```go
func isEnumField(f *ast.Field) bool { return f.IsEnum }
```

and map logic to use:

```go
func isEnumMapValue(mf *ast.MapField) bool { return mf.ValueIsEnum }
```

Add explicit enum branches in map helpers:

```go
if mf.ValueIsEnum {
	return []gdast.Statement{rawf("entry.append_array(ProtoCoreUtils.encode_varint(%s))", varName)}
}
```

and:

```go
if valueIsEnum {
	return mapVarintAssign(target, false)
}
```

Thread `ValueIsEnum` into any helper signatures that need it.

- [ ] **Step 4: Run targeted tests to verify they pass**

Run:

```bash
go test ./internal/cli ./cmd/protoc-gen-gdscript ./internal/generator -run 'TestRootUsesPathAwareWrapperClassName|TestRunPreservesNestedTypeQualification|TestGenerateImportedMessageUsesWrapperQualification|TestGenerateMapEnumUsesVarintPaths|TestGenerateMessageEnumNameCollisionDoesNotUseEnumPaths' -v
```

Expected:
- PASS for all five targeted regressions

- [ ] **Step 5: Commit the implementation**

```bash
git add internal/descriptors/converter.go internal/generator/generator.go internal/generator/messages.go internal/generator/accessors.go internal/generator/serialize.go internal/generator/deserialize.go internal/cli/root_test.go cmd/protoc-gen-gdscript/main_test.go internal/generator/generator_test.go
git commit -m "fix: preserve type identity in code generation"
```

### Task 3: Restore Entry-Point Parity And Update Docs

**Files:**
- Modify: `internal/cli/root.go`
- Modify: `README.md`
- Test: `internal/cli/root_test.go`
- Test: `cmd/protoc-gen-gdscript/main_test.go`
- Test: `internal/generator/generator_test.go`

- [ ] **Step 1: Make the CLI pass the same source-name semantics as the plugin**

Update the CLI call site from:

```go
cls, err := generator.Generate(file, filepath.Base(inputPath))
```

to:

```go
cls, err := generator.Generate(file, filepath.ToSlash(inputPath))
```

Keep the generated header comment behavior unchanged by continuing to derive `# Source:` from the basename inside the generator.

- [ ] **Step 2: Add or adjust parity assertions and docs**

Update tests so nested-path generation expectations are explicit, and update README wording where needed. The doc change should keep the promise of identical output while reflecting the path-aware class naming behavior.

Add/update assertions like:

```go
if !strings.Contains(got, "class_name FooBarProto\n") {
	t.Fatalf("missing expected class name:\n%s", got)
}
```

and if README examples mention entry-point identity, ensure the wording still matches the fixed implementation.

- [ ] **Step 3: Run the full verification suite**

Run:

```bash
go test ./...
go test -race ./...
golangci-lint run
```

Expected:
- all packages PASS in normal and race test runs
- `golangci-lint run` exits with `0 issues.`

- [ ] **Step 4: Commit final cleanup**

```bash
git add internal/cli/root.go README.md internal/cli/root_test.go cmd/protoc-gen-gdscript/main_test.go internal/generator/generator_test.go
git commit -m "fix: align cli and plugin output behavior"
```
