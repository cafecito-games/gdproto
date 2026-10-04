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

// Source renders the class to GDScript, ensuring a trailing newline.
func (gf GeneratedFile) Source() string {
	out := gf.Class.ToGDScript(0)
	if !strings.HasSuffix(out, "\n") {
		out += "\n"
	}
	return out
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
		file:       file,
		sourceName: sourceName,
		prefix:     prefix,
		resolver:   resolver,
	}
	g.declarations = collectLocalDeclarations(file)
	g.annotateLocalEnumUsage()
	return g.generate()
}

type generator struct {
	file       *ast.ProtoFile
	sourceName string
	prefix     string
	resolver   *NameResolver
	// declarations indexes the proto FQNs of every enum and message declared
	// in g.file. It complements the NameResolver, which deliberately omits
	// nested enums, and it is what lets the candidate walk in renderedType
	// and isLocalEnumReference stop at the nearest enclosing declaration of
	// either kind instead of running on to an outer, same-named one.
	declarations localDeclarations
	// currentScope is the dotted proto-FQN-style path (without the package
	// prefix) of the message whose body is currently being rendered. It is
	// set at the entrypoint of generateMessageClass and consulted by
	// renderedType to resolve same-file type references through the
	// NameResolver. Empty when rendering top-level constructs.
	currentScope string
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

func (g *generator) renderedFieldType(f *ast.Field) string {
	return g.renderedType(f.FieldType, f.FullTypePath, f.SourceFile, f.IsEnum)
}

func (g *generator) renderedMapValueType(mf *ast.MapField) string {
	return g.renderedType(mf.ValueType, mf.FullValueTypePath, mf.ValueSourceFile, mf.ValueIsEnum)
}

// renderedType returns the GDScript type to use for a proto type reference,
// resolving same-file message/enum references to their generated prefixed
// class names. Cross-file references are looked up in the resolver, which
// indexes both the input file and every import threaded through Generate.
//
// Reaching the cross-file fallback indicates an internal inconsistency: the
// AST recorded a SourceFile for the reference, but the resolver has no entry
// for any of the candidate FQNs in that file. This should not happen for
// inputs that pass through the plugin or CLI paths (both thread the full
// import closure). When it does, an error is recorded on the generator so
// Generate surfaces the failure instead of emitting a malformed .pb.gd; a
// best-effort placeholder is returned to allow the walk to finish.
//
// isEnumHint biases the placeholder toward a qualified `<Wrapper>.<EnumName>`
// shape when the reference is known to be an enum.
func (g *generator) renderedType(protoType, fullTypePath, sourceFile string, isEnumHint bool) string {
	if t, ok := scalarTypeMap[protoType]; ok {
		return t
	}
	if sourceFile != "" && sourceFile != g.sourceName && sourceFile != filepath.Base(g.sourceName) {
		candidates := buildLookupCandidates(protoType, "", g.file.Package)
		// The definitive name goes first even when it equals the name as
		// written, which is how an absolute reference to a root-package type
		// arrives: both emission paths reduce `.Type` to `Type` in both
		// fields. Without it the package-qualified candidate is tried first,
		// and a same-named local declaration captures the imported type.
		if fullTypePath != "" {
			candidates = append([]string{strings.TrimPrefix(fullTypePath, ".")}, candidates...)
		}
		for _, candidate := range candidates {
			if wrapper, inner, ok := g.resolver.LookupEnum(candidate); ok {
				return wrapper + "." + inner
			}
			if name, ok := g.resolver.Lookup(candidate); ok {
				return name
			}
		}
		g.recordError(fmt.Errorf(
			"internal: no resolver entry for cross-file type %q from %q (import not threaded into Generate?)",
			protoType, sourceFile,
		))
		otherPrefix, err := ResolvePrefix(&ast.ProtoFile{}, sourceFile)
		if err != nil {
			return strings.TrimPrefix(protoType, ".")
		}
		wrapper := otherPrefix + concatProtoPath(protoType)
		if isEnumHint {
			return wrapper + "." + lastProtoSegment(protoType)
		}
		return wrapper
	}
	// A known FullTypePath is the definitive name the descriptor resolved the
	// reference to, so it is tried before any scope candidate. The descriptor
	// converter reduces a same-file type to its bare name in FieldType, which
	// discards the leading dot of an absolute reference; scope-walking that
	// bare name would let a nested declaration capture a reference the
	// descriptor had already bound elsewhere.
	candidates := buildLookupCandidates(protoType, g.currentScope, g.file.Package)
	if fullTypePath != "" {
		if definitive := strings.TrimPrefix(fullTypePath, "."); definitive != "" {
			candidates = append([]string{definitive}, candidates...)
		}
	}
	for _, candidate := range candidates {
		if wrapper, inner, ok := g.resolver.LookupEnum(candidate); ok {
			return wrapper + "." + inner
		}
		if name, ok := g.resolver.Lookup(candidate); ok {
			return name
		}
		// Nested enums are absent from the NameResolver (they are addressed
		// through their declaring message's class), so without this check the
		// walk would run past a nested enum and let an outer, same-named
		// message or top-level enum capture the reference.
		if g.declarations.enums[candidate] {
			return g.renderedNestedEnumType(candidate)
		}
	}
	// Fallback: bare type name. Reached for references the resolver cannot
	// place, where the unqualified name is the best guess available.
	return strings.TrimPrefix(protoType, ".")
}

// renderedNestedEnumType renders a reference to the nested enum at the given
// fully qualified proto path. When the enum is declared by the message
// currently being rendered its bare name is directly in scope; otherwise it
// has to be qualified with the declaring message's generated class, because
// nested messages are flattened into sibling classes rather than inner ones.
func (g *generator) renderedNestedEnumType(enumFQN string) string {
	enumName := lastProtoSegment(enumFQN)
	declaringMessage := strings.TrimSuffix(strings.TrimSuffix(enumFQN, enumName), ".")
	if declaringMessage == packageScope(g.file.Package)+g.currentScope {
		return enumName
	}
	if class, ok := g.resolver.Lookup(declaringMessage); ok {
		return class + "." + enumName
	}
	return enumName
}

// packageScope returns the dotted prefix that qualifies declarations in the
// given proto package, or "" for a file without a package.
func packageScope(pkg string) string {
	if pkg == "" {
		return ""
	}
	return pkg + "."
}

// lastProtoSegment returns the last dotted segment of a proto type path,
// stripping any leading dot. For "shared.Color" it returns "Color".
func lastProtoSegment(typePath string) string {
	s := strings.TrimPrefix(typePath, ".")
	if i := strings.LastIndex(s, "."); i >= 0 {
		return s[i+1:]
	}
	return s
}

// concatProtoPath turns a dotted proto type path like "pkg.Outer.Inner" into a
// concatenated class-name fragment like "OuterInner". When the path has more
// than two segments the leading segment(s) are assumed to be a package
// prefix and dropped. For one- or two-segment paths every segment is kept.
func concatProtoPath(typePath string) string {
	s := strings.TrimPrefix(typePath, ".")
	parts := strings.Split(s, ".")
	if len(parts) > 2 {
		parts = parts[len(parts)-2:]
	}
	return strings.Join(parts, "")
}

// buildLookupCandidates produces, in order, the proto FQNs to try when
// resolving a type reference written inside currentScope. The order is the one
// protobuf itself uses: the name is appended to the innermost enclosing
// message scope first, then to each progressively outer scope, and only last
// is it resolved at file scope (bare, and under the file's package). A nested
// declaration therefore shadows a same-named outer one.
//
// Each scope yields the unqualified candidate before the package-qualified
// one so that both an index keyed by package-qualified FQNs and one keyed by
// package-relative paths can be consulted with a single walk. Duplicates are
// dropped, preserving first occurrence.
func buildLookupCandidates(typeName, currentScope, pkg string) []string {
	// A leading dot makes the reference absolute. Protobuf resolves such a
	// reference against the fully qualified name alone and never against an
	// enclosing scope, so scope-walking one would let a nested declaration
	// capture a reference that names a different type outright.
	absolute := strings.HasPrefix(typeName, ".")
	typeName = strings.TrimPrefix(typeName, ".")
	var out []string
	seen := map[string]bool{}
	add := func(s string) {
		if s == "" || seen[s] {
			return
		}
		seen[s] = true
		out = append(out, s)
	}
	if absolute {
		add(typeName)
		return out
	}
	// A scope-relative candidate names a declaration of the file being
	// generated, so in a packaged file it is only ever the package-qualified
	// form. The unqualified form is not a name protobuf would resolve there,
	// and the index it is looked up in holds other files' declarations too, so
	// emitting it would let an import capture a local reference.
	if currentScope != "" {
		parts := strings.Split(currentScope, ".")
		for i := len(parts); i > 0; i-- {
			candidate := strings.Join(append(append([]string{}, parts[:i]...), typeName), ".")
			if pkg == "" {
				add(candidate)
				continue
			}
			add(pkg + "." + candidate)
		}
	}
	if pkg != "" {
		add(pkg + "." + typeName)
	}
	// Last, the name as written. For a file with no package this is its own
	// file-scope name; for a packaged one it is how a reference to another
	// package, or to the root package, is spelled.
	add(typeName)
	return out
}

// localDeclarations indexes the fully qualified proto paths of the enums and
// messages declared in a single proto file, including nested ones at every
// depth. Paths are qualified with the file's package when it has one, matching
// the keys the NameResolver uses.
//
// Both kinds are indexed because protobuf name resolution binds a reference to
// the nearest enclosing declaration regardless of its kind: a nested message
// shadows a same-named outer enum, and a nested enum shadows a same-named outer
// message. Knowing only the enums cannot distinguish the two.
type localDeclarations struct {
	enums    map[string]bool
	messages map[string]bool
}

// collectLocalDeclarations indexes every enum and message declared in file.
func collectLocalDeclarations(file *ast.ProtoFile) localDeclarations {
	declarations := localDeclarations{
		enums:    map[string]bool{},
		messages: map[string]bool{},
	}
	scope := packageScope(file.Package)
	for _, e := range file.Enums {
		declarations.enums[scope+e.Name] = true
	}
	for _, m := range file.Messages {
		declarations.collectMessage(m, scope+m.Name)
	}
	return declarations
}

// collectMessage records the message at path along with its nested enums and
// messages, recursing to arbitrary depth.
func (d localDeclarations) collectMessage(m *ast.Message, path string) {
	d.messages[path] = true
	for _, e := range m.NestedEnums {
		d.enums[path+"."+e.Name] = true
	}
	for _, nested := range m.NestedMessages {
		d.collectMessage(nested, path+"."+nested.Name)
	}
}

// annotateLocalEnumUsage marks every same-file enum-typed field, map value and
// oneof field in the AST as an enum, so the serialization and accessor
// templates emit varint codecs instead of message handling. References to
// imported types carry a SourceFile and are left to the NameResolver.
func (g *generator) annotateLocalEnumUsage() {
	for _, m := range g.file.Messages {
		g.annotateLocalEnumMessage(m, m.Name)
	}
}

func (g *generator) annotateLocalEnumMessage(m *ast.Message, scope string) {
	for _, f := range m.Fields {
		if f.SourceFile == "" && isLocalEnumReference(f.FieldType, f.FullTypePath, scope, g.file.Package, g.declarations) {
			f.IsEnum = true
		}
	}
	for _, mf := range m.Maps {
		if mf.ValueSourceFile == "" && isLocalEnumReference(mf.ValueType, mf.FullValueTypePath, scope, g.file.Package, g.declarations) {
			mf.ValueIsEnum = true
		}
	}
	for _, oneof := range m.Oneofs {
		for _, f := range oneof.Fields {
			if f.SourceFile == "" && isLocalEnumReference(f.FieldType, f.FullTypePath, scope, g.file.Package, g.declarations) {
				f.IsEnum = true
			}
		}
	}
	for _, nested := range m.NestedMessages {
		g.annotateLocalEnumMessage(nested, scope+"."+nested.Name)
	}
}

// isLocalEnumReference reports whether a type reference written inside
// currentScope names an enum declared in the file being generated.
//
// It follows protobuf name resolution by walking buildLookupCandidates — the
// same candidate order renderedType uses, so the two cannot disagree about
// which declaration a reference binds to — and answering from the first
// candidate that names a declaration of either kind. Stopping at a message is
// as important as matching an enum: a nested message shadows a same-named
// outer enum, and continuing the walk past it would report that message as an
// enum.
//
// fullTypePath is the parser's already-resolved path for the reference, when
// it has one; it short-circuits the walk, and is likewise checked against both
// kinds.
func isLocalEnumReference(typeName, fullTypePath, currentScope, pkg string, declarations localDeclarations) bool {
	if fullTypePath != "" {
		resolved := strings.TrimPrefix(fullTypePath, ".")
		if declarations.enums[resolved] {
			return true
		}
		if declarations.messages[resolved] {
			return false
		}
	}
	for _, candidate := range buildLookupCandidates(typeName, currentScope, pkg) {
		if declarations.enums[candidate] {
			return true
		}
		if declarations.messages[candidate] {
			return false
		}
	}
	return false
}
