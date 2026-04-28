package parser

import (
	"strconv"

	"github.com/cafecito-games/gogdproto/internal/ast"
	"github.com/cafecito-games/gogdproto/internal/lexer"
)

func (p *parser) parseMessage() (*ast.Message, error) {
	msgTok := p.current()
	if _, err := p.expect(lexer.TokenMessage); err != nil {
		return nil, err
	}
	nameTok, err := p.expect(lexer.TokenIdentifier)
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(lexer.TokenLBrace); err != nil {
		return nil, err
	}

	m := &ast.Message{
		Position: ast.Position{Line: msgTok.Line, Column: msgTok.Column},
		Name:     nameTok.Value,
	}

	for !p.match(lexer.TokenRBrace) {
		switch {
		case p.match(lexer.TokenMessage):
			child, err := p.parseMessage()
			if err != nil {
				return nil, err
			}
			m.NestedMessages = append(m.NestedMessages, child)
		case p.match(lexer.TokenEnum):
			child, err := p.parseEnum()
			if err != nil {
				return nil, err
			}
			m.NestedEnums = append(m.NestedEnums, child)
		case p.match(lexer.TokenOneof):
			return nil, p.errorf(p.current(), "Oneof parsing not yet implemented")
		case p.match(lexer.TokenMap):
			mp, err := p.parseMapField()
			if err != nil {
				return nil, err
			}
			m.Maps = append(m.Maps, mp)
		case p.match(lexer.TokenReserved):
			return nil, p.errorf(p.current(), "Reserved parsing not yet implemented")
		case p.match(lexer.TokenOption):
			opt, err := p.parseOption()
			if err != nil {
				return nil, err
			}
			if m.Options == nil {
				m.Options = map[string]any{}
			}
			m.Options[opt.Name] = opt.Value
		default:
			f, err := p.parseField("")
			if err != nil {
				return nil, err
			}
			m.Fields = append(m.Fields, f)
		}
	}

	if _, err := p.expect(lexer.TokenRBrace); err != nil {
		return nil, err
	}
	return m, nil
}

// parseField parses a field. oneofParent is "" when not in a oneof.
func (p *parser) parseField(oneofParent string) (*ast.Field, error) {
	startTok := p.current()

	repeated := false
	if p.match(lexer.TokenRepeated) {
		p.advance()
		repeated = true
	}
	optional := false
	if p.match(lexer.TokenOptional) {
		p.advance()
		optional = true
	}

	fieldType, err := p.parseType()
	if err != nil {
		return nil, err
	}
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
	num, err := strconv.ParseInt(numTok.Value, 0, 32)
	if err != nil {
		return nil, p.errorf(numTok, "invalid field number %q: %v", numTok.Value, err)
	}

	if p.match(lexer.TokenLBracket) {
		return nil, p.errorf(p.current(), "Field options parsing not yet implemented")
	}

	if _, err := p.expect(lexer.TokenSemicolon); err != nil {
		return nil, err
	}

	return &ast.Field{
		Position:    ast.Position{Line: startTok.Line, Column: startTok.Column},
		FieldType:   fieldType,
		Name:        nameTok.Value,
		Number:      int(num),
		Repeated:    repeated,
		Optional:    optional,
		OneofParent: oneofParent,
	}, nil
}

func (p *parser) parseMapField() (*ast.MapField, error) {
	mapTok := p.current()
	if _, err := p.expect(lexer.TokenMap); err != nil {
		return nil, err
	}
	if _, err := p.expect(lexer.TokenLT); err != nil {
		return nil, err
	}
	keyType, err := p.parseType()
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(lexer.TokenComma); err != nil {
		return nil, err
	}
	valueType, err := p.parseType()
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(lexer.TokenGT); err != nil {
		return nil, err
	}
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
	num, err := strconv.ParseInt(numTok.Value, 0, 32)
	if err != nil {
		return nil, p.errorf(numTok, "invalid map field number %q: %v", numTok.Value, err)
	}

	if p.match(lexer.TokenLBracket) {
		return nil, p.errorf(p.current(), "Field options parsing not yet implemented")
	}

	if _, err := p.expect(lexer.TokenSemicolon); err != nil {
		return nil, err
	}

	return &ast.MapField{
		Position:  ast.Position{Line: mapTok.Line, Column: mapTok.Column},
		KeyType:   keyType,
		ValueType: valueType,
		Name:      nameTok.Value,
		Number:    int(num),
	}, nil
}
