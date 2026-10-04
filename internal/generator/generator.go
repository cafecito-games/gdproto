package generator

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/cafecito-games/gdproto/internal/ast"
	"github.com/cafecito-games/gdproto/internal/gdast"
)

// GeneratedFile is one rendered .gd source file produced by Generate. Each
// top-level proto message yields one file; nested messages become sibling
// files with concatenated parent-chain class names; top-level enums get
// their own wrapper class file.
type GeneratedFile struct {
	Filename  string
	ClassName string
	Class     *gdast.ClassDefinition
	// protoFQN is the dotted proto type path (without the package prefix)
	// that this file was generated from — e.g. "FooBar" for a top-level
	// message or "Foo.Bar" for a nested one. Used internally to produce
	// actionable error messages when two protos collapse to the same
	// generated filename.
	protoFQN string
}

// Source renders the class to canonical GDScript — the form gdkit's
// formatter produces — with a trailing newline. It returns an error if the
// rendered source does not parse, which means the generator produced
// malformed GDScript.
func (gf GeneratedFile) Source() (string, error) {
	return Canonicalize(gf.Filename, gf.Class.ToGDScript(0))
}

// Generate produces one GeneratedFile per top-level enum and per message
// (including nested messages, flattened as siblings) for the given proto file.
// sourceName is the .proto path or filename; it is used for both the header
// comment and prefix derivation when the file does not set
// (gdproto.class_prefix).
//
// imports is the set of additional proto files whose types may be referenced
// from file. Each entry contributes its messages/top-level enums to the
// NameResolver so cross-file references render with the imported file's
// (gdproto.class_prefix) — or filename-derived prefix — instead of falling
// back to the importer's filename. Pass nil for single-file inputs.
func Generate(file *ast.ProtoFile, sourceName string, imports []FileEntry) ([]GeneratedFile, error) {
	prefix, err := ResolvePrefix(file, sourceName)
	if err != nil {
		return nil, err
	}
	entries := make([]FileEntry, 0, 1+len(imports))
	entries = append(entries, FileEntry{File: file, Filename: sourceName})
	entries = append(entries, imports...)
	resolver, err := NewNameResolver(entries)
	if err != nil {
		return nil, err
	}
	g := &generator{
		file:                file,
		sourceName:          sourceName,
		prefix:              prefix,
		resolver:            resolver,
		declarations:        newDeclarationIndex(entries),
		fieldResolutions:    map[*ast.Field]typeResolution{},
		mapValueResolutions: map[*ast.MapField]typeResolution{},
	}
	g.resolveFieldTypes()
	if g.firstErr != nil {
		return nil, g.firstErr
	}
	return g.generate()
}

type generator struct {
	file       *ast.ProtoFile
	sourceName string
	prefix     string
	resolver   *NameResolver
	// declarations indexes every message and enum declared by the file being
	// generated and by its imports, which is what resolveTypeReference walks.
	declarations *declarationIndex
	// fieldResolutions and mapValueResolutions hold the one binding chosen
	// for each non-scalar reference, recorded by resolveFieldTypes before any
	// rendering happens. Rendering reads them instead of resolving again, so
	// a field's declared type, default, accessors, codec and text helpers
	// cannot end up describing different declarations.
	fieldResolutions    map[*ast.Field]typeResolution
	mapValueResolutions map[*ast.MapField]typeResolution
	// firstErr captures the first error recorded during the generation walk
	// from contexts (like renderedType) that cannot return an error through
	// their signature. Generate consults it before returning files.
	firstErr error
}

// recordError stores err as the generator's first error, if none is set yet.
// Subsequent errors are discarded — the first failure is the most actionable
// and later ones are usually cascading.
func (g *generator) recordError(err error) {
	if g.firstErr == nil {
		g.firstErr = err
	}
}

func (g *generator) generate() ([]GeneratedFile, error) {
	var files []GeneratedFile

	for _, e := range g.file.Enums {
		files = append(files, g.generateTopLevelEnumFile(e))
	}
	for _, m := range g.file.Messages {
		files = append(files, g.generateMessageFiles(m, "", "")...)
	}

	seen := make(map[string]string, len(files))
	for _, gf := range files {
		if first, dup := seen[gf.Filename]; dup {
			return nil, fmt.Errorf(
				"class name collision in %s: %q and %q both generate %s; rename one of them",
				g.sourceName, first, gf.protoFQN, gf.Filename,
			)
		}
		seen[gf.Filename] = gf.protoFQN
	}
	if g.firstErr != nil {
		return nil, g.firstErr
	}
	return files, nil
}

// generateTopLevelEnumFile wraps a top-level enum in a RefCounted class so it
// can be addressed globally via its class_name directive.
func (g *generator) generateTopLevelEnumFile(e *ast.Enum) GeneratedFile {
	className := g.prefix + e.Name
	class := &gdast.ClassDefinition{
		ClassNameDirective: className,
		Extends:            "RefCounted",
		LeadingComment:     gdkitSuppressions,
		HeaderComment:      headerCommentText(filepath.Base(g.sourceName)),
		Statements:         []gdast.Node{generateEnum(e)},
		TightStatements:    true,
	}
	return GeneratedFile{
		Filename:  className + ".pb.gd",
		ClassName: className,
		Class:     class,
		protoFQN:  e.Name,
	}
}

// renderedFieldType returns the GDScript type expression for a field's
// declared type. Scalars map directly; every other reference is answered from
// the binding resolveFieldTypes recorded for it.
func (g *generator) renderedFieldType(f *ast.Field) string {
	if t, ok := scalarTypeMap[f.FieldType]; ok {
		return t
	}
	if resolution, ok := g.fieldResolution(f); ok {
		return resolution.gdType
	}
	// resolveFieldTypes records an error for any reference it cannot bind, so
	// Generate has already failed by the time this is reached and the rendered
	// text is discarded. Returning the reference as written keeps the walk
	// able to finish instead of panicking on an empty type.
	return strings.TrimPrefix(f.FieldType, ".")
}

// renderedMapValueType returns the GDScript type expression for a map field's
// value type, from the binding resolveFieldTypes recorded for it.
func (g *generator) renderedMapValueType(mf *ast.MapField) string {
	if t, ok := scalarTypeMap[mf.ValueType]; ok {
		return t
	}
	if resolution, ok := g.mapValueResolution(mf); ok {
		return resolution.gdType
	}
	return strings.TrimPrefix(mf.ValueType, ".")
}
