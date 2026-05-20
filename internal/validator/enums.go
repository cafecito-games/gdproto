package validator

import (
	"fmt"
	"strings"

	"github.com/cafecito-games/gdproto/internal/ast"
)

// validateEnum enforces proto3 enum rules: duplicate value numbers (unless
// allow_alias is set), duplicate value names, the requirement that the first
// value be zero, and reserved-keyword checks for the enum name and each value
// name.
func (v *validator) validateEnum(enum *ast.Enum) {
	allowAlias := false
	if value, ok := enum.Options["allow_alias"]; ok {
		if b, isBool := value.(bool); isBool && b {
			allowAlias = true
		}
	}

	valueNumbers := make(map[int]string)
	for _, value := range enum.Values {
		if existing, seen := valueNumbers[value.Number]; seen && !allowAlias {
			v.addError(
				fmt.Sprintf("Duplicate enum value number %d in enum %q (also used by %q)",
					value.Number, enum.Name, existing),
				value.Line,
				value.Column,
			)
		}
		valueNumbers[value.Number] = value.Name
	}

	valueNames := make(map[string]struct{})
	for _, value := range enum.Values {
		if _, seen := valueNames[value.Name]; seen {
			v.addError(
				fmt.Sprintf("Duplicate enum value name %q in enum %q", value.Name, enum.Name),
				value.Line,
				value.Column,
			)
		}
		valueNames[value.Name] = struct{}{}
	}

	if len(enum.Values) > 0 && enum.Values[0].Number != 0 {
		v.addError(
			fmt.Sprintf("First enum value in proto3 must be zero (got %d)", enum.Values[0].Number),
			enum.Values[0].Line,
			enum.Values[0].Column,
		)
	}

	if reservedKeywords[strings.ToLower(enum.Name)] {
		v.addError(
			fmt.Sprintf("Enum name %q is a reserved keyword", enum.Name),
			enum.Line,
			enum.Column,
		)
	}

	for _, value := range enum.Values {
		if reservedKeywords[strings.ToLower(value.Name)] {
			v.addError(
				fmt.Sprintf("Enum value name %q is a reserved keyword", value.Name),
				value.Line,
				value.Column,
			)
		}
	}
}

// validateReserved checks that each reserved range is well-formed and lies
// within the protobuf field-number bounds. ReservedRange uses Start == End to
// represent a single reserved number; proper ranges additionally require
// Start <= End.
func (v *validator) validateReserved(reserved *ast.Reserved) {
	for _, rng := range reserved.Numbers {
		if rng.Start != rng.End {
			if rng.Start > rng.End {
				v.addError(
					fmt.Sprintf("Invalid reserved range: %d to %d (start > end)", rng.Start, rng.End),
					reserved.Line,
					reserved.Column,
				)
				continue
			}
			if rng.Start < minFieldNumber || rng.End > maxFieldNumber {
				v.addError(
					fmt.Sprintf("Reserved range %d to %d is out of valid field number range (%d-%d)",
						rng.Start, rng.End, minFieldNumber, maxFieldNumber),
					reserved.Line,
					reserved.Column,
				)
			}
			continue
		}
		if rng.Start < minFieldNumber || rng.Start > maxFieldNumber {
			v.addError(
				fmt.Sprintf("Reserved field number %d is out of valid range (%d-%d)",
					rng.Start, minFieldNumber, maxFieldNumber),
				reserved.Line,
				reserved.Column,
			)
		}
	}
}
