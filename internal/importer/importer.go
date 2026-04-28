package importer

import (
	"strings"

	"github.com/cafecito-games/gogdproto/internal/ast"
	"github.com/cafecito-games/gogdproto/internal/lexer"
	"github.com/cafecito-games/gogdproto/internal/parser"
)

// importedType holds resolution metadata for a type sourced from an
// imported file.
type importedType struct {
	SourceFile string
	IsEnum     bool
	FullName   string
	ShortName  string
}

// ResolveExternal walks file's imports, parses them via fs, builds a
// registry of imported types, and annotates Field/MapField/Oneof
// references in file with SourceFile/FullTypePath/IsEnum metadata.
//
// Missing or malformed imports are silently skipped — the validator is
// responsible for reporting unresolved type references. The inputPath
// argument is retained for parity with the Python CLI; resolution
// itself is delegated entirely to fs.
func ResolveExternal(file *ast.ProtoFile, inputPath string, fs FS) error {
	_ = inputPath
	registry := buildExternalRegistry(file, fs)
	if len(registry) == 0 {
		return nil
	}
	lookup := buildLookup(registry, file.Package)
	for _, m := range file.Messages {
		annotateMessage(m, lookup)
	}
	return nil
}

// buildExternalRegistry parses every import and collects its top-level
// and nested types keyed by fully qualified name. Public re-exports are
// followed transitively, with cycle detection to guard against import
// loops.
func buildExternalRegistry(file *ast.ProtoFile, fs FS) map[string]importedType {
	out := map[string]importedType{}
	visited := map[string]bool{}
	for _, imp := range file.Imports {
		collectImportedTypes(out, visited, imp.Path, fs)
	}
	return out
}

// collectImportedTypes parses path through fs, records its types into
// out, then recurses into its public re-exports. Already-visited paths
// short-circuit, preventing infinite loops on circular imports.
func collectImportedTypes(out map[string]importedType, visited map[string]bool, path string, fs FS) {
	if visited[path] {
		return
	}
	visited[path] = true
	if !fs.Exists(path) {
		return
	}
	data, err := fs.Read(path)
	if err != nil {
		return
	}
	tokens, err := lexer.Tokenize(string(data), path)
	if err != nil {
		return
	}
	impFile, err := parser.Parse(tokens, path)
	if err != nil {
		return
	}
	prefix := ""
	if impFile.Package != "" {
		prefix = impFile.Package + "."
	}
	for _, m := range impFile.Messages {
		collectFromMessage(out, m, prefix, "", path)
	}
	for _, e := range impFile.Enums {
		fullName := prefix + e.Name
		out[fullName] = importedType{
			SourceFile: path,
			IsEnum:     true,
			FullName:   fullName,
			ShortName:  e.Name,
		}
	}
	for _, nested := range impFile.Imports {
		if nested.Public {
			collectImportedTypes(out, visited, nested.Path, fs)
		}
	}
}

// collectFromMessage recursively records a message and all of its
// nested messages and enums.
func collectFromMessage(out map[string]importedType, m *ast.Message, prefix, relativePrefix, sourceFile string) {
	fullName := prefix + m.Name
	shortName := m.Name
	if relativePrefix != "" {
		shortName = relativePrefix + "." + m.Name
	}
	out[fullName] = importedType{
		SourceFile: sourceFile,
		IsEnum:     false,
		FullName:   fullName,
		ShortName:  shortName,
	}
	innerPrefix := fullName + "."
	innerRelativePrefix := shortName + "."
	for _, n := range m.NestedMessages {
		collectFromMessage(out, n, innerPrefix, innerRelativePrefix[:len(innerRelativePrefix)-1], sourceFile)
	}
	for _, e := range m.NestedEnums {
		inner := innerPrefix + e.Name
		out[inner] = importedType{
			SourceFile: sourceFile,
			IsEnum:     true,
			FullName:   inner,
			ShortName:  innerRelativePrefix + e.Name,
		}
	}
}

// buildLookup adds package-relative aliases when the current file
// shares the imported file's package, so unqualified references
// resolve.
func buildLookup(registry map[string]importedType, currentPackage string) map[string]importedType {
	lookup := map[string]importedType{}
	prefix := ""
	if currentPackage != "" {
		prefix = currentPackage + "."
	}
	for k, v := range registry {
		lookup[k] = v
		if prefix != "" {
			if rest, ok := strings.CutPrefix(k, prefix); ok {
				lookup[rest] = v
			}
		}
	}
	return lookup
}

// resolveOne looks up a type by its raw reference and, failing that,
// by its already-resolved full type path. Leading dots on absolute
// references are stripped.
func resolveOne(typeName, fullPath string, lookup map[string]importedType) (importedType, bool) {
	cleaned := strings.TrimPrefix(typeName, ".")
	if v, ok := lookup[cleaned]; ok {
		return v, true
	}
	if fullPath != "" {
		if v, ok := lookup[strings.TrimPrefix(fullPath, ".")]; ok {
			return v, true
		}
	}
	return importedType{}, false
}

// annotateMessage walks a message's fields, maps, oneofs, and nested
// messages, marking any references that resolve to an imported type.
func annotateMessage(m *ast.Message, lookup map[string]importedType) {
	for _, f := range m.Fields {
		if t, ok := resolveOne(f.FieldType, f.FullTypePath, lookup); ok {
			f.FieldType = t.ShortName
			f.SourceFile = t.SourceFile
			f.FullTypePath = t.FullName
			f.IsEnum = t.IsEnum
		}
	}
	for _, mp := range m.Maps {
		if t, ok := resolveOne(mp.ValueType, mp.FullValueTypePath, lookup); ok {
			mp.ValueType = t.ShortName
			mp.ValueSourceFile = t.SourceFile
			mp.FullValueTypePath = t.FullName
			mp.ValueIsEnum = t.IsEnum
		}
	}
	for _, o := range m.Oneofs {
		for _, f := range o.Fields {
			if t, ok := resolveOne(f.FieldType, f.FullTypePath, lookup); ok {
				f.FieldType = t.ShortName
				f.SourceFile = t.SourceFile
				f.FullTypePath = t.FullName
				f.IsEnum = t.IsEnum
			}
		}
	}
	for _, n := range m.NestedMessages {
		annotateMessage(n, lookup)
	}
}
