package validator

import "fmt"

// ValidationError describes a semantic validation failure.
type ValidationError struct {
	File    string
	Line    int
	Column  int
	Message string
}

// Error formats the error as "<file>:<line>:<col>: error: <message>".
// If File is empty, "<input>" is used.
func (e *ValidationError) Error() string {
	file := e.File
	if file == "" {
		file = "<input>"
	}
	return fmt.Sprintf("%s:%d:%d: error: %s", file, e.Line, e.Column, e.Message)
}
