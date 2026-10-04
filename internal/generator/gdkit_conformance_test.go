package generator_test

import (
	"os"
	"path/filepath"
	"testing"

	gdkitformat "github.com/cafecito-games/gdkit/format"
	"github.com/cafecito-games/gdkit/lint"
	"github.com/cafecito-games/gdkit/project"
	gdkituid "github.com/cafecito-games/gdkit/uid"

	"github.com/cafecito-games/gdproto/internal/generator"
	"github.com/cafecito-games/gdproto/internal/lexer"
	"github.com/cafecito-games/gdproto/internal/parser"
)

// conformanceFixture is one schema the gdkit conformance gate generates from
// and then checks the output of.
type conformanceFixture struct {
	// name labels the subtest and names the schema in failure messages.
	name string
	// protoPath is the schema's location, relative to this package.
	protoPath string
}

// conformanceFixtures are the schemas the gate attests to. examples covers
// every construct the generator emits that gdkit has an opinion about: nested
// messages, enums, oneofs, maps and repeated fields. collections is the
// counterweight: its map-heavy Bag message generates well over a thousand
// lines, which is the only way a file-length limit is ever exercised.
var conformanceFixtures = []conformanceFixture{
	{name: "example", protoPath: "../../examples/example.proto"},
	{name: "collections", protoPath: "../../tests/godot/fixtures/proto/collections.proto"},
}

// writeCorpusProject compiles a fixture's schema into a throwaway Godot
// project and returns its root. The project holds the generated .pb.gd files
// plus the proto_core_utils.gd runtime that gdproto ships alongside them, which
// is what a consumer's project actually contains.
func writeCorpusProject(t *testing.T, fixture conformanceFixture) string {
	t.Helper()

	source, err := os.ReadFile(fixture.protoPath)
	if err != nil {
		t.Fatalf("read %s: %v", fixture.protoPath, err)
	}
	sourceName := filepath.Base(fixture.protoPath)
	tokens, err := lexer.Tokenize(string(source), sourceName)
	if err != nil {
		t.Fatalf("tokenize %s: %v", sourceName, err)
	}
	file, err := parser.Parse(tokens, sourceName)
	if err != nil {
		t.Fatalf("parse %s: %v", sourceName, err)
	}
	files, err := generator.Generate(file, sourceName, nil)
	if err != nil {
		t.Fatalf("generate %s: %v", sourceName, err)
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
		writeCorpusFile(t, root, generated.Filename, content)
	}
	writeCorpusFile(t, root, "proto_core_utils.gd", generator.GenerateProtoCoreUtilsRaw())
	return root
}

// writeCorpusFile writes one generated GDScript file into the throwaway
// project along with its .uid sidecar, mirroring what both emission paths put
// on disk.
func writeCorpusFile(t *testing.T, root, filename, content string) {
	t.Helper()

	if err := os.WriteFile(filepath.Join(root, filename), []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", filename, err)
	}
	sidecar := generator.SidecarFilename(filename)
	if err := os.WriteFile(filepath.Join(root, sidecar), []byte(generator.SidecarSource(filename)), 0o600); err != nil {
		t.Fatalf("write %s: %v", sidecar, err)
	}
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
// configuration, over the GDScript generated from each conformance fixture and
// requires zero diagnostics.
//
// The gate is scoped to those schemas rather than to every possible schema.
// The formatter cannot break a single long identifier or type annotation across
// lines, so a schema with very long message, field or enum names still exceeds
// gdkit's max-line-length. Suppressing that rule in generated output would hide
// genuine formatting regressions, so the gate attests to the fixtures instead.
func TestGeneratedProjectPassesGdkitLint(t *testing.T) {
	for _, fixture := range conformanceFixtures {
		t.Run(fixture.name, func(t *testing.T) {
			root := writeCorpusProject(t, fixture)

			linter, err := lint.New(lint.DefaultConfig())
			if err != nil {
				t.Fatalf("new linter: %v", err)
			}
			snapshot := loadGdkitSnapshot(t, root)
			for _, diagnostic := range linter.Lint(snapshot).Diagnostics {
				t.Errorf("%s: %s:%d:%d: %s (%s)", fixture.name, diagnostic.Path,
					diagnostic.Line, diagnostic.Column, diagnostic.Message, diagnostic.Rule)
			}
			if len(snapshot.Scripts) == 0 {
				t.Errorf("%s: lint examined no scripts, so it attests to nothing", fixture.name)
			}
		})
	}
}

// TestGeneratedProjectPassesGdkitFormat runs gdkit's formatter, under its
// default configuration, over the GDScript generated from each conformance
// fixture and requires that no file would be rewritten.
//
// This is the in-process equivalent of `gdkit format --check`: the formatter
// performs no I/O, and a file whose canonical form differs from what the
// generator emitted comes back as a result with Changed set. A file that could
// not be formatted at all — it does not parse, or formatting it would not
// preserve its syntax tree, its tokens or the code a lint suppression applies
// to — comes back as a diagnostic instead. Both are failures here.
//
// Like the lint gate, this covers the conformance fixtures rather than every
// schema; see TestGeneratedProjectPassesGdkitLint for why.
func TestGeneratedProjectPassesGdkitFormat(t *testing.T) {
	for _, fixture := range conformanceFixtures {
		t.Run(fixture.name, func(t *testing.T) {
			root := writeCorpusProject(t, fixture)

			formatter, err := gdkitformat.New(gdkitformat.DefaultConfig())
			if err != nil {
				t.Fatalf("new formatter: %v", err)
			}
			report := formatter.Format(loadGdkitSnapshot(t, root))

			for _, diagnostic := range report.Diagnostics {
				t.Errorf("%s: %s:%d:%d: %s (%s)", fixture.name, diagnostic.Path,
					diagnostic.Line, diagnostic.Column, diagnostic.Message, diagnostic.Rule)
			}
			// Report.Results carries an entry for every discovered file, so only
			// the ones flagged as changed are conformance failures.
			for _, result := range report.Changed() {
				t.Errorf("%s: %s: generated source is not in gdkit canonical format",
					fixture.name, result.Path)
			}
			if len(report.Results) == 0 {
				t.Errorf("%s: format check examined no scripts, so it attests to nothing",
					fixture.name)
			}
		})
	}
}

// TestGeneratedProjectPassesGdkitUID runs gdkit's uid check over the project
// generated from each conformance fixture and requires zero diagnostics: every
// generated script must have a sidecar, holding a well-formed identifier no
// other script claims.
//
// This is the in-process equivalent of `gdkit uid check`.
func TestGeneratedProjectPassesGdkitUID(t *testing.T) {
	for _, fixture := range conformanceFixtures {
		t.Run(fixture.name, func(t *testing.T) {
			root := writeCorpusProject(t, fixture)

			report := gdkituid.Check(loadGdkitSnapshot(t, root))
			for _, diagnostic := range report.Diagnostics {
				t.Errorf("%s: %s: %s (%s)", fixture.name, diagnostic.Path,
					diagnostic.Message, diagnostic.Rule)
			}
			if report.Scripts == 0 {
				t.Errorf("%s: uid check examined no scripts, so it attests to nothing",
					fixture.name)
			}
		})
	}
}
