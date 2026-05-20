package parser

import (
	"strconv"

	"github.com/cafecito-games/gdproto/internal/ast"
	"github.com/cafecito-games/gdproto/internal/lexer"
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
	var options map[string]any
	if p.match(lexer.TokenLBracket) {
		options, err = p.parseFieldOptions()
		if err != nil {
			return nil, err
		}
	}
	if _, err := p.expect(lexer.TokenSemicolon); err != nil {
		return nil, err
	}
	return &ast.EnumValue{
		Position: ast.Position{Line: nameTok.Line, Column: nameTok.Column},
		Name:     nameTok.Value,
		Number:   int(number),
		Options:  options,
	}, nil
}

func (p *parser) parseReserved() (*ast.Reserved, error) {
	resTok := p.current()
	if _, err := p.expect(lexer.TokenReserved); err != nil {
		return nil, err
	}

	r := &ast.Reserved{
		Position: ast.Position{Line: resTok.Line, Column: resTok.Column},
	}

	if p.match(lexer.TokenStringLiteral) {
		for {
			nameTok, err := p.expect(lexer.TokenStringLiteral)
			if err != nil {
				return nil, err
			}
			r.Names = append(r.Names, nameTok.Value)
			if !p.match(lexer.TokenComma) {
				break
			}
			p.advance()
		}
	} else {
		for {
			startTok, err := p.expect(lexer.TokenIntLiteral)
			if err != nil {
				return nil, err
			}
			start, err := strconv.ParseInt(startTok.Value, 0, 32)
			if err != nil {
				return nil, p.errorf(startTok, "invalid reserved number %q: %v", startTok.Value, err)
			}

			rng := ast.ReservedRange{Start: int(start), End: int(start)}
			if p.match(lexer.TokenIdentifier) && p.current().Value == "to" {
				p.advance()
				endTok, err := p.expect(lexer.TokenIntLiteral)
				if err != nil {
					return nil, err
				}
				end, err := strconv.ParseInt(endTok.Value, 0, 32)
				if err != nil {
					return nil, p.errorf(endTok, "invalid reserved range end %q: %v", endTok.Value, err)
				}
				rng.End = int(end)
			}
			r.Numbers = append(r.Numbers, rng)

			if !p.match(lexer.TokenComma) {
				break
			}
			p.advance()
		}
	}

	if _, err := p.expect(lexer.TokenSemicolon); err != nil {
		return nil, err
	}
	return r, nil
}
