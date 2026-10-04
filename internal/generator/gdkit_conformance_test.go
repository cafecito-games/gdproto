package generator_test

import (
	"os"
	"path/filepath"
	"testing"

	gdkitformat "github.com/cafecito-games/gdkit/format"
	"github.com/cafecito-games/gdkit/lint"
	"github.com/cafecito-games/gdkit/project"

	"github.com/cafecito-games/gdproto/internal/generator"
	"github.com/cafecito-games/gdproto/internal/lexer"
	"github.com/cafecito-games/gdproto/internal/parser"
)

// exampleCorpusPath is the schema the gdkit conformance gate generates from.
// It exercises every construct the generator emits that gdkit has an opinion
// about: nested messages, enums, oneofs, maps and repeated fields.
const exampleCorpusPath = "../../examples/example.proto"

// writeExampleCorpusProject compiles exampleCorpusPath into a throwaway Godot
// project and returns its root. The project holds the generated .pb.gd files
// plus the proto_core_utils.gd runtime that gdproto ships alongside them, which
// is what a consumer's project actually contains.
func writeExampleCorpusProject(t *testing.T) string {
	t.Helper()

	source, err := os.ReadFile(exampleCorpusPath)
	if err != nil {
		t.Fatalf("read %s: %v", exampleCorpusPath, err)
	}
	tokens, err := lexer.Tokenize(string(source), "example.proto")
	if err != nil {
		t.Fatalf("tokenize: %v", err)
	}
	file, err := parser.Parse(tokens, "example.proto")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	files, err := generator.Generate(file, "example.proto", nil)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	root := t.TempDir()
	// gdkit identifies a Godot project by its project.godot; its contents are
	// irrelevant to the linter and the formatter, which only read .gd files.
	if err := os.WriteFile(filepath.Join(root, "project.godot"), []byte(""), 0o600); err != nil {
		t.Fatalf("write project.godot: %v", err)
	}
	for _, generated := range files {
		content, err := generated.Source()
		if err != nil {
			t.Fatalf("source for %s: %v", generated.Filename, err)
		}
		path := filepath.Join(root, generated.Filename)
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatalf("write %s: %v", generated.Filename, err)
		}
	}
	utilsPath := filepath.Join(root, "proto_core_utils.gd")
	if err := os.WriteFile(utilsPath, []byte(generator.GenerateProtoCoreUtilsRaw()), 0o600); err != nil {
		t.Fatalf("write proto_core_utils.gd: %v", err)
	}
	return root
}

// loadGdkitSnapshot discovers every GDScript file under root the way the gdkit
// command-line tools do.
func loadGdkitSnapshot(t *testing.T, root string) *project.Snapshot {
	t.Helper()

	snapshot, err := project.Load(project.Config{Root: root, SourceRoots: []string{"."}})
	if err != nil {
		t.Fatalf("load project: %v", err)
	}
	return snapshot
}

// TestGeneratedProjectPassesGdkitLint runs gdkit's linter, under its default
// configuration, over the GDScript generated from the example schema and
// requires zero diagnostics.
//
// The gate is scoped to that corpus rather than to every possible schema. The
// formatter cannot break a single long identifier or type annotation across
// lines, so a schema with very long message, field or enum names still exceeds
// gdkit's max-line-length. Suppressing that rule in generated output would hide
// genuine formatting regressions, so the gate attests to the corpus instead.
func TestGeneratedProjectPassesGdkitLint(t *testing.T) {
	root := writeExampleCorpusProject(t)

	linter, err := lint.New(lint.DefaultConfig())
	if err != nil {
		t.Fatalf("new linter: %v", err)
	}
	for _, diagnostic := range linter.Lint(loadGdkitSnapshot(t, root)).Diagnostics {
		t.Errorf("%s:%d:%d: %s (%s)", diagnostic.Path, diagnostic.Line, diagnostic.Column,
			diagnostic.Message, diagnostic.Rule)
	}
}

// TestGeneratedProjectPassesGdkitFormat runs gdkit's formatter, under its
// default configuration, over the GDScript generated from the example schema
// and requires that no file would be rewritten.
//
// This is the in-process equivalent of `gdkit format --check`: the formatter
// performs no I/O, and a file whose canonical form differs from what the
// generator emitted comes back as a result with Changed set. A file that could
// not be formatted at all — it does not parse, or formatting it would not
// preserve its syntax tree, its tokens or the code a lint suppression applies
// to — comes back as a diagnostic instead. Both are failures here.
//
// Like the lint gate, this covers the example corpus rather than every schema;
// see TestGeneratedProjectPassesGdkitLint for why.
func TestGeneratedProjectPassesGdkitFormat(t *testing.T) {
	root := writeExampleCorpusProject(t)

	formatter, err := gdkitformat.New(gdkitformat.DefaultConfig())
	if err != nil {
		t.Fatalf("new formatter: %v", err)
	}
	report := formatter.Format(loadGdkitSnapshot(t, root))

	for _, diagnostic := range report.Diagnostics {
		t.Errorf("%s:%d:%d: %s (%s)", diagnostic.Path, diagnostic.Line, diagnostic.Column,
			diagnostic.Message, diagnostic.Rule)
	}
	// Report.Results carries an entry for every discovered file, so only the
	// ones flagged as changed are conformance failures.
	for _, result := range report.Changed() {
		t.Errorf("%s: generated source is not in gdkit canonical format", result.Path)
	}
}
