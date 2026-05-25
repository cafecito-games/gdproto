package generator

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
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
			return "", fmt.Errorf("%soption %s must be a string, got %T",
				optionPositionPrefix(file, classPrefixOptionKey), classPrefixOptionKey, raw)
		}
		if !prefixPattern.MatchString(s) {
			return "", fmt.Errorf("%soption %s value %q is not a valid GDScript identifier (must match %s)",
				optionPositionPrefix(file, classPrefixOptionKey), classPrefixOptionKey, s, prefixPattern.String())
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

// optionPositionPrefix returns "at line X:Y " when the file's parser stored
// a non-zero position for the named option, or "" otherwise. The trailing
// space is part of the prefix so call sites can concatenate cleanly.
func optionPositionPrefix(file *ast.ProtoFile, name string) string {
	if file.OptionPositions == nil {
		return ""
	}
	pos, ok := file.OptionPositions[name]
	if !ok || (pos.Line == 0 && pos.Column == 0) {
		return ""
	}
	return fmt.Sprintf("at line %d:%d ", pos.Line, pos.Column)
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
//
// For top-level enums, the resolved class is the wrapper class that holds
// the enum declaration; the enum's inner name is also tracked so callers can
// produce qualified references like "<Wrapper>.<EnumName>".
type NameResolver struct {
	classByFQN map[string]string
	// enumInnerByFQN records the inner enum name for FQNs that point at a
	// top-level enum wrapper class. Absence means the FQN refers to a
	// message (or is unknown).
	enumInnerByFQN map[string]string
}

// indexEntry records one indexed proto type along with the file it came
// from, so cross-file class-name collisions can report both sites.
type indexEntry struct {
	fqn      string
	class    string
	filename string
	// innerEnum is set for top-level enums; it carries the enum's inner
	// name so callers can produce qualified `<Wrapper>.<EnumName>`
	// references.
	innerEnum string
}

// NewNameResolver builds a NameResolver from the given file entries. It
// returns an error if two distinct proto FQNs across the input files would
// generate the same GDScript class name; the user can disambiguate by
// setting (gdproto.class_prefix) on one of the offending files.
func NewNameResolver(entries []FileEntry) (*NameResolver, error) {
	var indexed []indexEntry
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
			indexed = append(indexed, indexEntry{
				fqn:       scope + en.Name,
				class:     prefix + en.Name,
				filename:  e.Filename,
				innerEnum: en.Name,
			})
		}
		for _, m := range e.File.Messages {
			indexed = appendMessageIndex(indexed, m, scope, prefix, "", e.Filename)
		}
	}

	byClass := make(map[string][]indexEntry, len(indexed))
	for _, entry := range indexed {
		byClass[entry.class] = append(byClass[entry.class], entry)
	}
	for class, entries := range byClass {
		if len(entries) < 2 {
			continue
		}
		// Same FQN registered twice from a single file (shouldn't happen
		// for valid inputs, but be defensive) is not a cross-file
		// collision — only report when the FQNs differ.
		distinctFQNs := map[string]indexEntry{}
		for _, entry := range entries {
			if _, seen := distinctFQNs[entry.fqn]; !seen {
				distinctFQNs[entry.fqn] = entry
			}
		}
		if len(distinctFQNs) < 2 {
			continue
		}
		ordered := make([]indexEntry, 0, len(distinctFQNs))
		for _, entry := range distinctFQNs {
			ordered = append(ordered, entry)
		}
		sort.Slice(ordered, func(i, j int) bool {
			if ordered[i].filename != ordered[j].filename {
				return ordered[i].filename < ordered[j].filename
			}
			return ordered[i].fqn < ordered[j].fqn
		})
		return nil, fmt.Errorf(
			"class name collision across files: class %q produced by both %q (from %s) and %q (from %s); set option (gdproto.class_prefix) on one of them",
			class,
			ordered[0].fqn, ordered[0].filename,
			ordered[1].fqn, ordered[1].filename,
		)
	}

	r := &NameResolver{
		classByFQN:     make(map[string]string, len(indexed)),
		enumInnerByFQN: make(map[string]string),
	}
	for _, entry := range indexed {
		r.classByFQN[entry.fqn] = entry.class
		if entry.innerEnum != "" {
			r.enumInnerByFQN[entry.fqn] = entry.innerEnum
		}
	}
	return r, nil
}

func appendMessageIndex(out []indexEntry, m *ast.Message, packageScope, prefix, parentChain, filename string) []indexEntry {
	name := parentChain + m.Name
	out = append(out, indexEntry{
		fqn:      packageScope + name,
		class:    prefix + strings.ReplaceAll(name, ".", ""),
		filename: filename,
	})
	for _, nm := range m.NestedMessages {
		out = appendMessageIndex(out, nm, packageScope, prefix, name+".", filename)
	}
	return out
}

// Lookup returns the generated GDScript class name for a proto FQN. The FQN
// may include a leading dot (as in descriptor source); it is stripped. For
// top-level enums, the returned name is the wrapper class; use LookupEnum to
// also obtain the inner enum name for qualified references.
func (r *NameResolver) Lookup(fqn string) (string, bool) {
	fqn = strings.TrimPrefix(fqn, ".")
	s, ok := r.classByFQN[fqn]
	return s, ok
}

// LookupEnum returns the wrapper class and inner enum name for an FQN that
// refers to a top-level enum. The third return is false when the FQN is
// unknown or refers to a message.
func (r *NameResolver) LookupEnum(fqn string) (wrapperClass, enumName string, ok bool) {
	fqn = strings.TrimPrefix(fqn, ".")
	inner, hasEnum := r.enumInnerByFQN[fqn]
	if !hasEnum {
		return "", "", false
	}
	return r.classByFQN[fqn], inner, true
}

// IsEnum reports whether the given FQN refers to a top-level enum wrapper
// class indexed by this resolver. Nested enums are not indexed.
func (r *NameResolver) IsEnum(fqn string) bool {
	fqn = strings.TrimPrefix(fqn, ".")
	_, ok := r.enumInnerByFQN[fqn]
	return ok
}
