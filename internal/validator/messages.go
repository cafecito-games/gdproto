package validator

import (
	"fmt"
	"strings"

	"github.com/cafecito-games/gogdproto/internal/ast"
)

// validateMessage validates the fields and map fields of a single message,
// along with any conflicts against its reserved declarations. Recursion into
// nested messages and orchestration of oneofs/reserved-range validation is
// handled by a later task; this function intentionally does not descend into
// nested types.
func (v *validator) validateMessage(message *ast.Message, scope string) {
	fieldNumbers := make(map[int]string)
	fieldNames := make(map[string]bool)

	for _, field := range message.Fields {
		v.validateField(field, message, fieldNumbers, fieldNames, scope)
	}
	for _, mapField := range message.Maps {
		v.validateMapField(mapField, message, fieldNumbers, fieldNames, scope)
	}
}

// validateField enforces field-number bounds, reserved-range usage, duplicate
// number/name detection, reserved-keyword names, and conflicts with the
// parent message's reserved declarations.
func (v *validator) validateField(
	field *ast.Field,
	message *ast.Message,
	fieldNumbers map[int]string,
	fieldNames map[string]bool,
	_ string,
) {
	if field.Number < minFieldNumber || field.Number > maxFieldNumber {
		v.addError(
			fmt.Sprintf("Field number %d is out of valid range (%d-%d)",
				field.Number, minFieldNumber, maxFieldNumber),
			field.Line,
			field.Column,
		)
	}

	if field.Number >= reservedStart && field.Number <= reservedEnd {
		v.addError(
			fmt.Sprintf("Field number %d is in reserved range (%d-%d)",
				field.Number, reservedStart, reservedEnd),
			field.Line,
			field.Column,
		)
	}

	if existing, seen := fieldNumbers[field.Number]; seen {
		v.addError(
			fmt.Sprintf("Duplicate field number %d in message %q (also used by %q)",
				field.Number, message.Name, existing),
			field.Line,
			field.Column,
		)
	}
	fieldNumbers[field.Number] = field.Name

	if fieldNames[field.Name] {
		v.addError(
			fmt.Sprintf("Duplicate field name %q in message %q", field.Name, message.Name),
			field.Line,
			field.Column,
		)
	}
	fieldNames[field.Name] = true

	if reservedKeywords[strings.ToLower(field.Name)] {
		v.addError(
			fmt.Sprintf("Field name %q is a reserved keyword", field.Name),
			field.Line,
			field.Column,
		)
	}

	v.checkReservedConflicts(field.Name, field.Number, field.Line, field.Column, message)
}

// validateMapField applies field-level checks to a map field and additionally
// enforces that the key type is one of the integral or string types permitted
// by proto3.
func (v *validator) validateMapField(
	mapField *ast.MapField,
	message *ast.Message,
	fieldNumbers map[int]string,
	fieldNames map[string]bool,
	_ string,
) {
	if mapField.Number < minFieldNumber || mapField.Number > maxFieldNumber {
		v.addError(
			fmt.Sprintf("Field number %d is out of valid range (%d-%d)",
				mapField.Number, minFieldNumber, maxFieldNumber),
			mapField.Line,
			mapField.Column,
		)
	}

	if mapField.Number >= reservedStart && mapField.Number <= reservedEnd {
		v.addError(
			fmt.Sprintf("Field number %d is in reserved range (%d-%d)",
				mapField.Number, reservedStart, reservedEnd),
			mapField.Line,
			mapField.Column,
		)
	}

	if existing, seen := fieldNumbers[mapField.Number]; seen {
		v.addError(
			fmt.Sprintf("Duplicate field number %d in message %q (also used by %q)",
				mapField.Number, message.Name, existing),
			mapField.Line,
			mapField.Column,
		)
	}
	fieldNumbers[mapField.Number] = mapField.Name

	if fieldNames[mapField.Name] {
		v.addError(
			fmt.Sprintf("Duplicate field name %q in message %q", mapField.Name, message.Name),
			mapField.Line,
			mapField.Column,
		)
	}
	fieldNames[mapField.Name] = true

	if !validMapKeyTypes[mapField.KeyType] {
		v.addError(
			fmt.Sprintf("Invalid map key type: %s (must be integral or string type)",
				mapField.KeyType),
			mapField.Line,
			mapField.Column,
		)
	}

	v.checkReservedConflicts(mapField.Name, mapField.Number, mapField.Line, mapField.Column, message)
}

// checkReservedConflicts reports conflicts between a field's number/name and
// the parent message's reserved declarations. Singleton reservations (where
// Start == End) produce a "is reserved" error; proper ranges produce a
// "conflicts with reserved range" error.
func (v *validator) checkReservedConflicts(name string, number, line, column int, message *ast.Message) {
	for _, reserved := range message.Reserved {
		for _, rng := range reserved.Numbers {
			if rng.Start == rng.End {
				if number == rng.Start {
					v.addError(
						fmt.Sprintf("Field number %d is reserved", number),
						line,
						column,
					)
				}
				continue
			}
			if number >= rng.Start && number <= rng.End {
				v.addError(
					fmt.Sprintf("Field number %d conflicts with reserved range %d to %d",
						number, rng.Start, rng.End),
					line,
					column,
				)
			}
		}

		for _, reservedName := range reserved.Names {
			if name == reservedName {
				v.addError(
					fmt.Sprintf("Field name %q is reserved", name),
					line,
					column,
				)
			}
		}
	}
}
