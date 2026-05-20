package parser

import (
	"strconv"

	"github.com/cafecito-games/gdproto/internal/ast"
	"github.com/cafecito-games/gdproto/internal/lexer"
)

func (p *parser) parseOption() (*ast.Option, error) {
	optTok := p.current()
	if _, err := p.expect(lexer.TokenOption); err != nil {
		return nil, err
	}
	name, err := p.parseOptionName()
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(lexer.TokenEquals); err != nil {
		return nil, err
	}
	value, err := p.parseOptionValue()
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(lexer.TokenSemicolon); err != nil {
		return nil, err
	}
	return &ast.Option{
		Position: ast.Position{Line: optTok.Line, Column: optTok.Column},
		Name:     name,
		Value:    value,
	}, nil
}

// parseOptionName handles "foo", "foo.bar", and "(foo.bar)" (parenthesized,
// which becomes "(foo.bar)" with surrounding parens included).
func (p *parser) parseOptionName() (string, error) {
	if p.match(lexer.TokenLParen) {
		p.advance()
		name, err := p.parseDottedIdent()
		if err != nil {
			return "", err
		}
		if _, err := p.expect(lexer.TokenRParen); err != nil {
			return "", err
		}
		return "(" + name + ")", nil
	}
	return p.parseDottedIdent()
}

func (p *parser) parseOneof() (*ast.Oneof, error) {
	oneofTok := p.current()
	if _, err := p.expect(lexer.TokenOneof); err != nil {
		return nil, err
	}
	nameTok, err := p.expect(lexer.TokenIdentifier)
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(lexer.TokenLBrace); err != nil {
		return nil, err
	}

	o := &ast.Oneof{
		Position: ast.Position{Line: oneofTok.Line, Column: oneofTok.Column},
		Name:     nameTok.Value,
	}

	for !p.match(lexer.TokenRBrace) {
		if p.match(lexer.TokenOption) {
			if _, err := p.parseOption(); err != nil {
				return nil, err
			}
			continue
		}
		f, err := p.parseField(nameTok.Value)
		if err != nil {
			return nil, err
		}
		if f.Repeated {
			return nil, p.errorf(oneofTok, "Oneof fields cannot be repeated")
		}
		o.Fields = append(o.Fields, f)
	}

	if _, err := p.expect(lexer.TokenRBrace); err != nil {
		return nil, err
	}
	return o, nil
}

// parseOptionValue accepts string, int, float, bool, or identifier.
func (p *parser) parseOptionValue() (any, error) {
	tok := p.current()
	switch tok.Type {
	case lexer.TokenStringLiteral:
		p.advance()
		return tok.Value, nil
	case lexer.TokenIntLiteral:
		p.advance()
		v, err := strconv.ParseInt(tok.Value, 0, 64)
		if err != nil {
			return nil, p.errorf(tok, "invalid integer literal %q: %v", tok.Value, err)
		}
		return v, nil
	case lexer.TokenFloatLiteral:
		p.advance()
		v, err := strconv.ParseFloat(tok.Value, 64)
		if err != nil {
			return nil, p.errorf(tok, "invalid float literal %q: %v", tok.Value, err)
		}
		return v, nil
	case lexer.TokenTrue:
		p.advance()
		return true, nil
	case lexer.TokenFalse:
		p.advance()
		return false, nil
	case lexer.TokenIdentifier:
		p.advance()
		return tok.Value, nil
	default:
		return nil, p.errorf(tok, "Expected option value, got %s", tok.Type)
	}
}
