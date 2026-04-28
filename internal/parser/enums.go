package parser

import (
	"strconv"

	"github.com/cafecito-games/gogdproto/internal/ast"
	"github.com/cafecito-games/gogdproto/internal/lexer"
)

func (p *parser) parseEnum() (*ast.Enum, error) {
	enumTok := p.current()
	if _, err := p.expect(lexer.TokenEnum); err != nil {
		return nil, err
	}
	nameTok, err := p.expect(lexer.TokenIdentifier)
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(lexer.TokenLBrace); err != nil {
		return nil, err
	}

	e := &ast.Enum{
		Position: ast.Position{Line: enumTok.Line, Column: enumTok.Column},
		Name:     nameTok.Value,
	}

	for !p.match(lexer.TokenRBrace) {
		if p.match(lexer.TokenOption) {
			opt, err := p.parseOption()
			if err != nil {
				return nil, err
			}
			if e.Options == nil {
				e.Options = map[string]any{}
			}
			e.Options[opt.Name] = opt.Value
			continue
		}
		v, err := p.parseEnumValue()
		if err != nil {
			return nil, err
		}
		e.Values = append(e.Values, v)
	}

	if _, err := p.expect(lexer.TokenRBrace); err != nil {
		return nil, err
	}
	return e, nil
}

func (p *parser) parseEnumValue() (*ast.EnumValue, error) {
	nameTok, err := p.expect(lexer.TokenIdentifier)
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(lexer.TokenEquals); err != nil {
		return nil, err
	}
	numTok, err := p.expect(lexer.TokenIntLiteral)
	if err != nil {
		return nil, err
	}
	number, err := strconv.ParseInt(numTok.Value, 0, 32)
	if err != nil {
		return nil, p.errorf(numTok, "invalid enum value number %q: %v", numTok.Value, err)
	}
	if _, err := p.expect(lexer.TokenSemicolon); err != nil {
		return nil, err
	}
	return &ast.EnumValue{
		Position: ast.Position{Line: nameTok.Line, Column: nameTok.Column},
		Name:     nameTok.Value,
		Number:   int(number),
	}, nil
}
