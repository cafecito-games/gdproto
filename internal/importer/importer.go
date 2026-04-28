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
// and nested types keyed by fully qualified name.
func buildExternalRegistry(file *ast.ProtoFile, fs FS) map[string]importedType {
	out := map[string]importedType{}
	for _, imp := range file.Imports {
		if !fs.Exists(imp.Path) {
			continue
		}
		data, err := fs.Read(imp.Path)
		if err != nil {
			continue
		}
		tokens, err := lexer.Tokenize(string(data), imp.Path)
		if err != nil {
			continue
		}
		impFile, err := parser.Parse(tokens, imp.Path)
		if err != nil {
			continue
		}
		prefix := ""
		if impFile.Package != "" {
			prefix = impFile.Package + "."
		}
		for _, m := range impFile.Messages {
			collectFromMessage(out, m, prefix, imp.Path)
		}
		for _, e := range impFile.Enums {
			fullName := prefix + e.Name
			out[fullName] = importedType{SourceFile: imp.Path, IsEnum: true, FullName: fullName}
		}
	}
	return out
}

// collectFromMessage recursively records a message and all of its
// nested messages and enums.
func collectFromMessage(out map[string]importedType, m *ast.Message, prefix, sourceFile string) {
	fullName := prefix + m.Name
	out[fullName] = importedType{SourceFile: sourceFile, IsEnum: false, FullName: fullName}
	innerPrefix := fullName + "."
	for _, n := range m.NestedMessages {
		collectFromMessage(out, n, innerPrefix, sourceFile)
	}
	for _, e := range m.NestedEnums {
		inner := innerPrefix + e.Name
		out[inner] = importedType{SourceFile: sourceFile, IsEnum: true, FullName: inner}
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
			f.SourceFile = t.SourceFile
			f.FullTypePath = t.FullName
			f.IsEnum = t.IsEnum
		}
	}
	for _, mp := range m.Maps {
		if t, ok := resolveOne(mp.ValueType, mp.FullValueTypePath, lookup); ok {
			mp.ValueSourceFile = t.SourceFile
			mp.FullValueTypePath = t.FullName
			mp.ValueIsEnum = t.IsEnum
		}
	}
	for _, o := range m.Oneofs {
		for _, f := range o.Fields {
			if t, ok := resolveOne(f.FieldType, f.FullTypePath, lookup); ok {
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
