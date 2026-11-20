package generator

import (
	"fmt"
	"strings"
)

// OptionEmitTextFormat is the canonical name of the boolean option that
// controls whether generated messages include protobuf text-format methods.
const OptionEmitTextFormat = "emit_text_format"

// Options controls optional code-generation behavior across both the direct
// CLI and the protoc plugin.
type Options struct {
	// EmitTextFormat controls whether generated messages include protobuf
	// text-format serialization methods (to_text and from_text) and their
	// text-only enum helpers.
	EmitTextFormat bool
}

// DefaultOptions returns the default generation options, preserving full
// text-format code generation.
func DefaultOptions() Options {
	return Options{
		EmitTextFormat: true,
	}
}

// ParseBoolOption parses a canonical boolean option value ("true" or "false").
// Other spellings are rejected so both the CLI and protoc plugin enforce the
// same fail-closed contract.
func ParseBoolOption(name, raw string) (bool, error) {
	switch raw {
	case "true":
		return true, nil
	case "false":
		return false, nil
	default:
		return false, fmt.Errorf("invalid value %q for %s: expected \"true\" or \"false\"", raw, name)
	}
}

// ParsePluginParameters parses a protoc CodeGeneratorRequest.parameter string
// (comma-separated key=value pairs) into generation Options. Unknown keys,
// malformed entries, duplicate keys, and non-canonical boolean values are
// rejected with actionable errors.
func ParsePluginParameters(parameter string) (Options, error) {
	options := DefaultOptions()
	if parameter == "" {
		return options, nil
	}

	seen := map[string]bool{}
	for _, entry := range strings.Split(parameter, ",") {
		if entry == "" {
			return Options{}, fmt.Errorf("invalid plugin parameter %q: empty parameter entry", parameter)
		}
		key, rawValue, hasEquals := strings.Cut(entry, "=")
		if !hasEquals || key == "" {
			return Options{}, fmt.Errorf(
				"invalid plugin parameter %q: expected key=value (supported: %s=true|false)",
				entry, OptionEmitTextFormat,
			)
		}
		switch key {
		case OptionEmitTextFormat:
			if seen[key] {
				return Options{}, fmt.Errorf("duplicate plugin parameter %q", key)
			}
			parsed, err := ParseBoolOption(key, rawValue)
			if err != nil {
				return Options{}, err
			}
			options.EmitTextFormat = parsed
			seen[key] = true
		default:
			return Options{}, fmt.Errorf(
				"unknown plugin parameter %q (supported: %s)",
				key, OptionEmitTextFormat,
			)
		}
	}

	return options, nil
}
