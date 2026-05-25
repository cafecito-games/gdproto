package generator

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/cafecito-games/gdproto/internal/ast"
)

var (
	classPrefixOptionKey = "(gdproto.class_prefix)"
	prefixPattern        = regexp.MustCompile(`^[A-Z][A-Za-z0-9]*$`)
	nonAlnumSplit        = regexp.MustCompile(`[^A-Za-z0-9]+`)
)

// ResolvePrefix returns the GDScript class_name prefix for the given proto
// file. The (gdproto.class_prefix) option wins; otherwise the basename of
// filename is split on non-alphanumerics and PascalCased.
func ResolvePrefix(file *ast.ProtoFile, filename string) (string, error) {
	if raw, ok := file.Options[classPrefixOptionKey]; ok {
		s, isString := raw.(string)
		if !isString {
			return "", fmt.Errorf("option %s must be a string, got %T", classPrefixOptionKey, raw)
		}
		if !prefixPattern.MatchString(s) {
			return "", fmt.Errorf("option %s value %q is not a valid GDScript identifier (must match %s)", classPrefixOptionKey, s, prefixPattern.String())
		}
		return s, nil
	}
	base := strings.TrimSuffix(filepath.Base(filename), ".proto")
	var b strings.Builder
	for _, p := range nonAlnumSplit.Split(base, -1) {
		if p == "" {
			continue
		}
		b.WriteString(strings.ToUpper(p[:1]))
		if len(p) > 1 {
			b.WriteString(strings.ToLower(p[1:]))
		}
	}
	out := b.String()
	if !prefixPattern.MatchString(out) {
		return "", fmt.Errorf("cannot derive class_name prefix from filename %q", filename)
	}
	return out, nil
}

// FileEntry pairs an ast.ProtoFile with the source filename it was parsed
// from. The filename drives prefix derivation when the file does not set
// (gdproto.class_prefix).
type FileEntry struct {
	File     *ast.ProtoFile
	Filename string
}

// NameResolver maps proto fully-qualified names to generated GDScript class
// names. It indexes every message and top-level enum across the provided
// files. Nested enums are intentionally NOT indexed: generator code addresses
// them as "<ParentClass>.<EnumName>" via the parent's lookup.
type NameResolver struct {
	classByFQN map[string]string
}

// NewNameResolver builds a NameResolver from the given file entries.
func NewNameResolver(entries []FileEntry) (*NameResolver, error) {
	r := &NameResolver{classByFQN: map[string]string{}}
	for _, e := range entries {
		prefix, err := ResolvePrefix(e.File, e.Filename)
		if err != nil {
			return nil, err
		}
		scope := ""
		if e.File.Package != "" {
			scope = e.File.Package + "."
		}
		for _, en := range e.File.Enums {
			r.classByFQN[scope+en.Name] = prefix + en.Name
		}
		for _, m := range e.File.Messages {
			r.indexMessage(m, scope, prefix, "")
		}
	}
	return r, nil
}

func (r *NameResolver) indexMessage(m *ast.Message, packageScope, prefix, parentChain string) {
	// TODO: when ast.Message tracks map-entry status, skip synthetic
	// map-entry messages here. For now they will be filtered at emission
	// time in Task 3.
	name := parentChain + m.Name
	r.classByFQN[packageScope+name] = prefix + strings.ReplaceAll(name, ".", "")
	for _, nm := range m.NestedMessages {
		r.indexMessage(nm, packageScope, prefix, name+".")
	}
}

// Lookup returns the generated GDScript class name for a proto FQN. The FQN
// may include a leading dot (as in descriptor source); it is stripped.
func (r *NameResolver) Lookup(fqn string) (string, bool) {
	fqn = strings.TrimPrefix(fqn, ".")
	s, ok := r.classByFQN[fqn]
	return s, ok
}
