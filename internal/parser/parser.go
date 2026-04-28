package parser

import (
	"fmt"
	"strings"

	"github.com/cafecito-games/gogdproto/internal/ast"
	"github.com/cafecito-games/gogdproto/internal/lexer"
)

// Parse consumes a token stream from the lexer and produces a *ast.ProtoFile.
// The filename is used only for error messages; pass "" for "<input>".
func Parse(tokens []lexer.Token, filename string) (*ast.ProtoFile, error) {
	p := &parser{tokens: tokens, filename: filename}
	return p.parseFile()
}

type parser struct {
	tokens   []lexer.Token
	filename string
	pos      int
}

// current returns the token at the cursor, or the last token (EOF) if past
// the end. The lexer always emits a trailing TokenEOF, so this is safe.
func (p *parser) current() lexer.Token {
	if p.pos >= len(p.tokens) {
		return p.tokens[len(p.tokens)-1]
	}
	return p.tokens[p.pos]
}

//nolint:unused // used by parsing tasks landing in subsequent commits.
func (p *parser) peek(offset int) lexer.Token {
	pos := p.pos + offset
	if pos >= len(p.tokens) {
		return p.tokens[len(p.tokens)-1]
	}
	return p.tokens[pos]
}

func (p *parser) advance() lexer.Token {
	tok := p.current()
	if p.pos < len(p.tokens)-1 {
		p.pos++
	}
	return tok
}

func (p *parser) match(types ...lexer.TokenType) bool {
	cur := p.current().Type
	for _, t := range types {
		if cur == t {
			return true
		}
	}
	return false
}

func (p *parser) expect(t lexer.TokenType) (lexer.Token, error) {
	tok := p.current()
	if tok.Type != t {
		return lexer.Token{}, p.errorf(tok, "Expected %s, got %s", t, tok.Type)
	}
	return p.advance(), nil
}

func (p *parser) errorf(tok lexer.Token, format string, args ...any) *ParserError {
	return &ParserError{
		File:    p.filename,
		Token:   tok,
		Message: fmt.Sprintf(format, args...),
	}
}

func (p *parser) parseFile() (*ast.ProtoFile, error) {
	first := p.current()

	syntax, err := p.parseSyntax()
	if err != nil {
		return nil, err
	}

	file := &ast.ProtoFile{
		Position: ast.Position{Line: first.Line, Column: first.Column},
		Syntax:   syntax,
	}

	for !p.match(lexer.TokenEOF) {
		switch {
		case p.match(lexer.TokenImport):
			imp, err := p.parseImport()
			if err != nil {
				return nil, err
			}
			file.Imports = append(file.Imports, imp)
		case p.match(lexer.TokenPackage):
			pkg, err := p.parsePackage()
			if err != nil {
				return nil, err
			}
			file.Package = pkg
		case p.match(lexer.TokenOption):
			opt, err := p.parseOption()
			if err != nil {
				return nil, err
			}
			if file.Options == nil {
				file.Options = map[string]any{}
			}
			file.Options[opt.Name] = opt.Value
		case p.match(lexer.TokenMessage):
			return nil, p.errorf(p.current(), "Message parsing not yet implemented")
		case p.match(lexer.TokenEnum):
			return nil, p.errorf(p.current(), "Enum parsing not yet implemented")
		default:
			tok := p.current()
			return nil, p.errorf(tok, "Unexpected token: %s", tok.Type)
		}
	}

	return file, nil
}

func (p *parser) parseImport() (*ast.Import, error) {
	impTok := p.current()
	if _, err := p.expect(lexer.TokenImport); err != nil {
		return nil, err
	}
	public := false
	if p.match(lexer.TokenPublic) {
		public = true
		p.advance()
	}
	pathTok, err := p.expect(lexer.TokenStringLiteral)
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(lexer.TokenSemicolon); err != nil {
		return nil, err
	}
	return &ast.Import{
		Position: ast.Position{Line: impTok.Line, Column: impTok.Column},
		Path:     pathTok.Value,
		Public:   public,
	}, nil
}

func (p *parser) parsePackage() (string, error) {
	if _, err := p.expect(lexer.TokenPackage); err != nil {
		return "", err
	}
	name, err := p.parseDottedIdent()
	if err != nil {
		return "", err
	}
	if _, err := p.expect(lexer.TokenSemicolon); err != nil {
		return "", err
	}
	return name, nil
}

// parseDottedIdent parses Foo, Foo.Bar, Foo.Bar.Baz (identifiers separated
// by dots). At least one identifier required.
func (p *parser) parseDottedIdent() (string, error) {
	head, err := p.expect(lexer.TokenIdentifier)
	if err != nil {
		return "", err
	}
	parts := []string{head.Value}
	for p.match(lexer.TokenDot) {
		p.advance()
		next, err := p.expect(lexer.TokenIdentifier)
		if err != nil {
			return "", err
		}
		parts = append(parts, next.Value)
	}
	return strings.Join(parts, "."), nil
}

func (p *parser) parseSyntax() (string, error) {
	if _, err := p.expect(lexer.TokenSyntax); err != nil {
		return "", err
	}
	if _, err := p.expect(lexer.TokenEquals); err != nil {
		return "", err
	}
	tok, err := p.expect(lexer.TokenStringLiteral)
	if err != nil {
		return "", err
	}
	if _, err := p.expect(lexer.TokenSemicolon); err != nil {
		return "", err
	}
	return tok.Value, nil
}
