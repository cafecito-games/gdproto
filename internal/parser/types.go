package parser

import (
	"github.com/cafecito-games/gogdproto/internal/lexer"
)

// scalarTypeTokens is the set of token types that name a built-in scalar.
var scalarTypeTokens = map[lexer.TokenType]bool{
	lexer.TokenDouble:   true,
	lexer.TokenFloat:    true,
	lexer.TokenInt32:    true,
	lexer.TokenInt64:    true,
	lexer.TokenUInt32:   true,
	lexer.TokenUInt64:   true,
	lexer.TokenSInt32:   true,
	lexer.TokenSInt64:   true,
	lexer.TokenFixed32:  true,
	lexer.TokenFixed64:  true,
	lexer.TokenSFixed32: true,
	lexer.TokenSFixed64: true,
	lexer.TokenBool:     true,
	lexer.TokenString:   true,
	lexer.TokenBytes:    true,
}

// parseType parses a field type. Built-in scalars return their keyword
// string. Identifier paths return "Foo", "Foo.Bar", or ".pkg.Foo" for
// absolute references.
func (p *parser) parseType() (string, error) {
	tok := p.current()
	if scalarTypeTokens[tok.Type] {
		p.advance()
		return tok.Value, nil
	}

	// Absolute (.pkg.Foo) or relative (Foo.Bar) message type.
	if tok.Type == lexer.TokenDot {
		p.advance()
		rest, err := p.parseDottedIdent()
		if err != nil {
			return "", err
		}
		return "." + rest, nil
	}
	if tok.Type == lexer.TokenIdentifier {
		return p.parseDottedIdent()
	}

	return "", p.errorf(tok, "Expected type name, got %s", tok.Type)
}
