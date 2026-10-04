package generator

import (
	"fmt"
	"strings"

	"github.com/cafecito-games/gdproto/internal/ast"
)

// declarationKind distinguishes the two kinds of protobuf declaration a type
// reference can bind to. The kind decides a field's GDScript default value,
// accessor shape, wire codec and text-format helpers, so it must come from the
// same answer as the declared type.
type declarationKind int

const (
	// declarationMessage is a `message` declaration.
	declarationMessage declarationKind = iota
	// declarationEnum is an `enum` declaration.
	declarationEnum
)

// declaration is one protobuf message or enum visible to the file being
// generated — declared either in that file or in one of the imports threaded
// into Generate.
type declaration struct {
	kind declarationKind
	// fullName is the package-qualified proto name without a leading dot,
	// e.g. "game.v1.Outer.Mode".
	fullName string
	// name is the declaration's own final name segment, e.g. "Mode".
	name string
	// parentName is the package-qualified proto name of the enclosing
	// message, or "" when the declaration sits at file scope.
	parentName string
	// enum is the AST node of an enum declaration. It lets consumers that
	// need the bound enum's values read them from the binding instead of
	// searching for an enum by name, which cannot distinguish same-named
	// declarations in different scopes.
	enum *ast.Enum
	// declaredLocally is true when the declaration comes from the file being
	// generated rather than from one of its imports.
	declaredLocally bool
}

// declarationIndex maps package-qualified proto names to the declarations they
// name, across the file being generated and every import threaded into
// Generate.
//
// It complements NameResolver rather than replacing it. NameResolver answers
// "what is this type's generated GDScript class name" and deliberately omits
// nested enums, which are addressed through their declaring message's class.
// Resolution additionally needs to know that a nested enum declaration exists,
// so that a scope walk stops there instead of running on to a same-named outer
// declaration, and it needs each declaration's kind.
type declarationIndex struct {
	byFullName map[string]declaration
}

// newDeclarationIndex indexes every message and enum, nested at any depth,
// declared by the given file entries. The first entry is treated as the file
// being generated; where two entries declare the same proto name — which valid
// protobuf cannot do — the earlier entry wins, so a local declaration is never
// displaced by an import.
func newDeclarationIndex(entries []FileEntry) *declarationIndex {
	index := &declarationIndex{byFullName: map[string]declaration{}}
	for i, entry := range entries {
		local := i == 0
		scope := packageScope(entry.File.Package)
		for _, e := range entry.File.Enums {
			index.add(declaration{
				kind:            declarationEnum,
				fullName:        scope + e.Name,
				name:            e.Name,
				enum:            e,
				declaredLocally: local,
			})
		}
		for _, m := range entry.File.Messages {
			index.addMessage(m, scope+m.Name, "", local)
		}
	}
	return index
}

// addMessage records the message at fullName together with its nested enums
// and messages, recursing to arbitrary depth.
func (index *declarationIndex) addMessage(m *ast.Message, fullName, parentName string, local bool) {
	index.add(declaration{
		kind:            declarationMessage,
		fullName:        fullName,
		name:            m.Name,
		parentName:      parentName,
		declaredLocally: local,
	})
	for _, e := range m.NestedEnums {
		index.add(declaration{
			kind:            declarationEnum,
			fullName:        fullName + "." + e.Name,
			name:            e.Name,
			parentName:      fullName,
			enum:            e,
			declaredLocally: local,
		})
	}
	for _, nested := range m.NestedMessages {
		index.addMessage(nested, fullName+"."+nested.Name, fullName, local)
	}
}

func (index *declarationIndex) add(d declaration) {
	if _, exists := index.byFullName[d.fullName]; exists {
		return
	}
	index.byFullName[d.fullName] = d
}

// lookup returns the declaration named by a package-qualified proto name. A
// leading dot is stripped, so descriptor-shaped names can be passed directly.
func (index *declarationIndex) lookup(fullName string) (declaration, bool) {
	d, ok := index.byFullName[strings.TrimPrefix(fullName, ".")]
	return d, ok
}

// typeResolution is the single authoritative answer to "which declaration does
// this type reference bind to, and how is it spelled in GDScript". Every part
// of a field's generated code — declared type, default value, accessors, wire
// codec and text-format enum helpers — derives from one of these, so the type
// cannot name one declaration while the codec names another.
type typeResolution struct {
	// declaration is the bound declaration, carrying its kind, its
	// package-qualified proto name, the name of the message declaring it and,
	// for an enum, its AST node.
	declaration
	// gdType is the GDScript type expression naming the binding from inside
	// the class being generated.
	gdType string
}

// isEnum reports whether the reference bound to an enum declaration.
func (r typeResolution) isEnum() bool { return r.kind == declarationEnum }

// candidateTypeNames returns, in protobuf's own resolution order, the
// package-qualified proto names a type reference written inside scope may bind
// to. scope is the dotted chain of enclosing message names without the package
// prefix; pkg is the proto package of the file being generated.
//
// Protobuf resolves a relative reference by walking outward from the innermost
// enclosing scope and taking the first match, so a nested declaration shadows a
// same-named outer one. A leading dot makes the reference absolute: it names a
// fully-qualified type outright and is never resolved against an enclosing
// scope.
//
// Scope-relative candidates are package-qualified in a packaged file, because
// an unqualified name is not something protobuf would resolve against an
// enclosing scope and because the indexes these candidates are looked up in
// also hold imported files' declarations — emitting the unqualified form would
// let an import capture a local reference. The bare reference comes last, which
// is how a reference to another package, or to the root package, is spelled.
func candidateTypeNames(reference, scope, pkg string) []string {
	absolute := strings.HasPrefix(reference, ".")
	reference = strings.TrimPrefix(reference, ".")
	if reference == "" {
		return nil
	}
	var out []string
	seen := map[string]bool{}
	add := func(candidate string) {
		if seen[candidate] {
			return
		}
		seen[candidate] = true
		out = append(out, candidate)
	}
	if absolute {
		add(reference)
		return out
	}
	qualifier := packageScope(pkg)
	if scope != "" {
		parts := strings.Split(scope, ".")
		for i := len(parts); i > 0; i-- {
			add(qualifier + strings.Join(parts[:i], ".") + "." + reference)
		}
	}
	if qualifier != "" {
		add(qualifier + reference)
	}
	add(reference)
	return out
}

// packageScope returns the dotted prefix that qualifies declarations in the
// given proto package, or "" for a file without a package.
func packageScope(pkg string) string {
	if pkg == "" {
		return ""
	}
	return pkg + "."
}

// resolveTypeReference binds a non-scalar type reference to a declaration.
//
// reference is the name as the generator sees it, fullTypePath the binding a
// descriptor or the importer already resolved (authoritative when present, and
// therefore not scope-walked), and scope the dotted chain of enclosing message
// names, without the package prefix, that the reference was written in.
//
// The second return is false when no declaration carries the name, which is a
// fail-closed condition: callers must report an error rather than emit a
// plausible-looking type that Godot would reject at load time.
func (g *generator) resolveTypeReference(reference, fullTypePath, scope string) (typeResolution, bool) {
	if fullTypePath != "" {
		d, ok := g.declarations.lookup(fullTypePath)
		if !ok {
			return typeResolution{}, false
		}
		return g.resolutionFor(d, scope)
	}
	for _, candidate := range candidateTypeNames(reference, scope, g.file.Package) {
		if d, ok := g.declarations.lookup(candidate); ok {
			return g.resolutionFor(d, scope)
		}
	}
	return typeResolution{}, false
}

// resolutionFor completes a binding by rendering the GDScript type expression
// that names d from inside the class generated for scope.
func (g *generator) resolutionFor(d declaration, scope string) (typeResolution, bool) {
	gdType, ok := g.renderDeclaration(d, scope)
	if !ok {
		return typeResolution{}, false
	}
	return typeResolution{declaration: d, gdType: gdType}, true
}

// renderDeclaration returns the GDScript type expression for d as written
// inside the class generated for scope.
//
// Messages and top-level enums are named through NameResolver, which knows the
// generated class (and, for a top-level enum, the inner enum name inside its
// wrapper class). A nested enum has no class of its own: nested messages are
// flattened into sibling classes, so the enum is reachable as
// "<DeclaringClass>.<Enum>" — or by its bare name when the class being
// generated is the one that declares it.
func (g *generator) renderDeclaration(d declaration, scope string) (string, bool) {
	if d.kind == declarationEnum {
		if wrapper, inner, ok := g.resolver.LookupEnum(d.fullName); ok {
			return wrapper + "." + inner, true
		}
		if d.declaredLocally && d.parentName == packageScope(g.file.Package)+scope {
			return d.name, true
		}
		class, ok := g.resolver.Lookup(d.parentName)
		if !ok {
			return "", false
		}
		return class + "." + d.name, true
	}
	return g.resolver.Lookup(d.fullName)
}

// resolveFieldTypes binds every non-scalar field, map value and oneof field in
// the file being generated, recording one resolution per reference. Rendering
// reads those recorded answers instead of resolving again, which is what keeps
// a field's declared type, codec and text helpers describing the same
// declaration.
//
// A reference that names no declaration is recorded as an error, so Generate
// fails rather than emitting GDScript that names a type Godot cannot find.
func (g *generator) resolveFieldTypes() {
	for _, m := range g.file.Messages {
		g.resolveMessageFieldTypes(m, m.Name)
	}
}

func (g *generator) resolveMessageFieldTypes(m *ast.Message, scope string) {
	for _, f := range m.Fields {
		g.resolveField(f, scope)
	}
	for _, mf := range m.Maps {
		g.resolveMapField(mf, scope)
	}
	for _, oneof := range m.Oneofs {
		for _, f := range oneof.Fields {
			g.resolveField(f, scope)
		}
	}
	for _, nested := range m.NestedMessages {
		g.resolveMessageFieldTypes(nested, scope+"."+nested.Name)
	}
}

func (g *generator) resolveField(f *ast.Field, scope string) {
	if _, scalar := scalarTypeMap[f.FieldType]; scalar {
		return
	}
	resolution, ok := g.resolveTypeReference(f.FieldType, f.FullTypePath, scope)
	if !ok {
		g.recordError(g.unresolvedReferenceError(f.FieldType, f.SourceFile, scope))
		return
	}
	g.fieldResolutions[f] = resolution
	f.IsEnum = resolution.isEnum()
}

func (g *generator) resolveMapField(mf *ast.MapField, scope string) {
	if _, scalar := scalarTypeMap[mf.ValueType]; scalar {
		return
	}
	resolution, ok := g.resolveTypeReference(mf.ValueType, mf.FullValueTypePath, scope)
	if !ok {
		g.recordError(g.unresolvedReferenceError(mf.ValueType, mf.ValueSourceFile, scope))
		return
	}
	g.mapValueResolutions[mf] = resolution
	mf.ValueIsEnum = resolution.isEnum()
}

// unresolvedReferenceError describes a reference that named no declaration.
//
// A reference carrying a source file was already bound by the descriptor set
// or the importer, so failing to find it means that file's declarations were
// never threaded into Generate — an internal inconsistency rather than a
// problem with the schema, and reported as such.
func (g *generator) unresolvedReferenceError(reference, sourceFile, scope string) error {
	if sourceFile != "" {
		return fmt.Errorf(
			"internal: no resolver entry for cross-file type %q from %q (import not threaded into Generate?)",
			reference, sourceFile,
		)
	}
	return fmt.Errorf(
		"%s: undefined type %q referenced in message %q: no message or enum of that name is declared in the file or its imports",
		g.sourceName, reference, scope,
	)
}

// fieldResolution returns the recorded binding for a field reference.
func (g *generator) fieldResolution(f *ast.Field) (typeResolution, bool) {
	resolution, ok := g.fieldResolutions[f]
	return resolution, ok
}

// mapValueResolution returns the recorded binding for a map field's value
// reference.
func (g *generator) mapValueResolution(mf *ast.MapField) (typeResolution, bool) {
	resolution, ok := g.mapValueResolutions[mf]
	return resolution, ok
}
