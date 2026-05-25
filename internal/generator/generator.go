package generator

import (
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
func Generate(file *ast.ProtoFile, sourceName string) ([]GeneratedFile, error) {
	prefix, err := ResolvePrefix(file, sourceName)
	if err != nil {
		return nil, err
	}
	resolver, err := NewNameResolver([]FileEntry{{File: file, Filename: sourceName}})
	if err != nil {
		return nil, err
	}
	g := &generator{
		file:       file,
		sourceName: sourceName,
		prefix:     prefix,
		resolver:   resolver,
	}
	g.annotateLocalEnumUsage()
	return g.generate()
}

type generator struct {
	file       *ast.ProtoFile
	sourceName string
	prefix     string
	resolver   *NameResolver
	// currentScope is the dotted proto-FQN-style path (without the package
	// prefix) of the message whose body is currently being rendered. It is
	// set at the entrypoint of generateMessageClass and consulted by
	// renderedType to resolve same-file type references through the
	// NameResolver. Empty when rendering top-level constructs.
	currentScope string
}

func (g *generator) generate() ([]GeneratedFile, error) {
	var files []GeneratedFile

	for _, e := range g.file.Enums {
		files = append(files, g.generateTopLevelEnumFile(e))
	}
	for _, m := range g.file.Messages {
		files = append(files, g.generateMessageFiles(m, "", "")...)
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
	}
}

func (g *generator) renderedFieldType(f *ast.Field) string {
	return g.renderedType(f.FieldType, f.SourceFile)
}

func (g *generator) renderedMapValueType(mf *ast.MapField) string {
	return g.renderedType(mf.ValueType, mf.ValueSourceFile)
}

// renderedType returns the GDScript type to use for a proto type reference,
// resolving same-file message/enum references to their generated prefixed
// class names. Cross-file references derive their prefix from the imported
// file's filename (no access to the imported file's options for now; a
// follow-up could thread the imported file's ProtoFile through Generate).
func (g *generator) renderedType(protoType, sourceFile string) string {
	if t, ok := scalarTypeMap[protoType]; ok {
		return t
	}
	if sourceFile != "" && sourceFile != g.sourceName && sourceFile != filepath.Base(g.sourceName) {
		otherPrefix, err := ResolvePrefix(&ast.ProtoFile{}, sourceFile)
		if err == nil {
			return otherPrefix + concatProtoPath(protoType)
		}
		return strings.TrimPrefix(protoType, ".")
	}
	for _, candidate := range buildLookupCandidates(protoType, g.currentScope, g.file.Package) {
		if name, ok := g.resolver.Lookup(candidate); ok {
			return name
		}
	}
	// Fallback: bare type name (handles nested-enum references which the
	// resolver does not index; they render inside the parent class scope
	// as e.g. "Status.ONLINE").
	return strings.TrimPrefix(protoType, ".")
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

// buildLookupCandidates produces the set of proto FQNs to try when resolving
// a type reference written inside currentScope. It mirrors the scope walk
// used by isLocalEnumReference: the bare name, the bare name under the
// package, and the name appended to every prefix of the current scope (with
// and without the package).
func buildLookupCandidates(typeName, currentScope, pkg string) []string {
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
	add(typeName)
	if pkg != "" {
		add(pkg + "." + typeName)
	}
	if currentScope != "" {
		parts := strings.Split(currentScope, ".")
		for i := len(parts); i > 0; i-- {
			candidate := strings.Join(append(append([]string{}, parts[:i]...), typeName), ".")
			add(candidate)
			if pkg != "" {
				add(pkg + "." + candidate)
			}
		}
	}
	return out
}

func (g *generator) annotateLocalEnumUsage() {
	enumPaths := map[string]bool{}
	prefix := ""
	if g.file.Package != "" {
		prefix = g.file.Package + "."
	}
	for _, e := range g.file.Enums {
		enumPaths[prefix+e.Name] = true
	}
	for _, m := range g.file.Messages {
		g.collectLocalEnumPaths(m, prefix+m.Name, enumPaths)
	}
	for _, m := range g.file.Messages {
		g.annotateLocalEnumMessage(m, m.Name, enumPaths)
	}
}

func (g *generator) collectLocalEnumPaths(m *ast.Message, scope string, enumPaths map[string]bool) {
	for _, e := range m.NestedEnums {
		enumPaths[scope+"."+e.Name] = true
	}
	for _, nested := range m.NestedMessages {
		g.collectLocalEnumPaths(nested, scope+"."+nested.Name, enumPaths)
	}
}

func (g *generator) annotateLocalEnumMessage(m *ast.Message, scope string, enumPaths map[string]bool) {
	for _, f := range m.Fields {
		if f.SourceFile == "" && isLocalEnumReference(f.FieldType, f.FullTypePath, scope, g.file.Package, enumPaths) {
			f.IsEnum = true
		}
	}
	for _, mf := range m.Maps {
		if mf.ValueSourceFile == "" && isLocalEnumReference(mf.ValueType, mf.FullValueTypePath, scope, g.file.Package, enumPaths) {
			mf.ValueIsEnum = true
		}
	}
	for _, oneof := range m.Oneofs {
		for _, f := range oneof.Fields {
			if f.SourceFile == "" && isLocalEnumReference(f.FieldType, f.FullTypePath, scope, g.file.Package, enumPaths) {
				f.IsEnum = true
			}
		}
	}
	for _, nested := range m.NestedMessages {
		g.annotateLocalEnumMessage(nested, scope+"."+nested.Name, enumPaths)
	}
}

func isLocalEnumReference(typeName, fullTypePath, currentScope, pkg string, enumPaths map[string]bool) bool {
	if fullTypePath != "" && enumPaths[strings.TrimPrefix(fullTypePath, ".")] {
		return true
	}

	normalizedType := strings.TrimPrefix(typeName, ".")
	if enumPaths[normalizedType] {
		return true
	}

	if pkg != "" {
		packagePrefix := pkg + "."
		if strings.HasPrefix(normalizedType, packagePrefix) && enumPaths[normalizedType] {
			return true
		}
	}

	typeParts := strings.Split(normalizedType, ".")
	scopeParts := strings.Split(currentScope, ".")
	for i := len(scopeParts); i > 0; i-- {
		candidate := strings.Join(append(append([]string{}, scopeParts[:i]...), typeParts...), ".")
		prefix := ""
		if pkg != "" {
			prefix = pkg + "."
		}
		if enumPaths[candidate] || (prefix != "" && enumPaths[prefix+candidate]) {
			return true
		}
	}

	return false
}
