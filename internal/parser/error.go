package parser

import (
	"fmt"

	"github.com/cafecito-games/gogdproto/internal/lexer"
)

// ParserError describes a parse failure.
//
//nolint:revive // stuttering name retained for clarity at API boundaries.
type ParserError struct {
	File    string
	Token   lexer.Token
	Message string
}

// Error formats as "<file>:<line>:<col>: error: <message>". When File is
// empty, "<input>" is substituted.
func (e *ParserError) Error() string {
	file := e.File
	if file == "" {
		file = "<input>"
	}
	return fmt.Sprintf("%s:%d:%d: error: %s", file, e.Token.Line, e.Token.Column, e.Message)
}
