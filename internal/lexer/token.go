package lexer

import "fmt"

// TokenType identifies the category of a lexed token.
type TokenType int

// Token type constants. The complete set of categories produced by the lexer.
const (
	// TokenEOF marks the end of input.
	TokenEOF TokenType = iota
	// TokenComment is a // or /* */ comment (currently skipped by the lexer).
	TokenComment

	// TokenSyntax is the "syntax" keyword.
	TokenSyntax
	// TokenMessage is the "message" keyword.
	TokenMessage
	// TokenEnum is the "enum" keyword.
	TokenEnum
	// TokenRepeated is the "repeated" keyword.
	TokenRepeated
	// TokenOptional is the "optional" keyword.
	TokenOptional
	// TokenMap is the "map" keyword.
	TokenMap
	// TokenOneof is the "oneof" keyword.
	TokenOneof
	// TokenImport is the "import" keyword.
	TokenImport
	// TokenPublic is the "public" keyword.
	TokenPublic
	// TokenOption is the "option" keyword.
	TokenOption
	// TokenPacked is the "packed" keyword.
	TokenPacked
	// TokenReserved is the "reserved" keyword.
	TokenReserved
	// TokenPackage is the "package" keyword.
	TokenPackage
	// TokenService is the "service" keyword.
	TokenService
	// TokenRPC is the "rpc" keyword.
	TokenRPC
	// TokenReturns is the "returns" keyword.
	TokenReturns
	// TokenStream is the "stream" keyword.
	TokenStream

	// TokenDouble is the "double" built-in type.
	TokenDouble
	// TokenFloat is the "float" built-in type.
	TokenFloat
	// TokenInt32 is the "int32" built-in type.
	TokenInt32
	// TokenInt64 is the "int64" built-in type.
	TokenInt64
	// TokenUInt32 is the "uint32" built-in type.
	TokenUInt32
	// TokenUInt64 is the "uint64" built-in type.
	TokenUInt64
	// TokenSInt32 is the "sint32" built-in type.
	TokenSInt32
	// TokenSInt64 is the "sint64" built-in type.
	TokenSInt64
	// TokenFixed32 is the "fixed32" built-in type.
	TokenFixed32
	// TokenFixed64 is the "fixed64" built-in type.
	TokenFixed64
	// TokenSFixed32 is the "sfixed32" built-in type.
	TokenSFixed32
	// TokenSFixed64 is the "sfixed64" built-in type.
	TokenSFixed64
	// TokenBool is the "bool" built-in type.
	TokenBool
	// TokenString is the "string" built-in type.
	TokenString
	// TokenBytes is the "bytes" built-in type.
	TokenBytes

	// TokenIdentifier is a user-defined name.
	TokenIdentifier
	// TokenIntLiteral is an integer literal (decimal, hex, or octal).
	TokenIntLiteral
	// TokenFloatLiteral is a floating-point literal.
	TokenFloatLiteral
	// TokenStringLiteral is a quoted string literal.
	TokenStringLiteral
	// TokenTrue is the "true" boolean literal.
	TokenTrue
	// TokenFalse is the "false" boolean literal.
	TokenFalse

	// TokenLBrace is the "{" symbol.
	TokenLBrace
	// TokenRBrace is the "}" symbol.
	TokenRBrace
	// TokenLBracket is the "[" symbol.
	TokenLBracket
	// TokenRBracket is the "]" symbol.
	TokenRBracket
	// TokenLParen is the "(" symbol.
	TokenLParen
	// TokenRParen is the ")" symbol.
	TokenRParen
	// TokenLT is the "<" symbol.
	TokenLT
	// TokenGT is the ">" symbol.
	TokenGT
	// TokenSemicolon is the ";" symbol.
	TokenSemicolon
	// TokenEquals is the "=" symbol.
	TokenEquals
	// TokenComma is the "," symbol.
	TokenComma
	// TokenDot is the "." symbol.
	TokenDot
)

//nolint:gosec // G101 false positive: these are token-type display names, not credentials.
var tokenTypeNames = [...]string{
	TokenEOF:           "TokenEOF",
	TokenComment:       "TokenComment",
	TokenSyntax:        "TokenSyntax",
	TokenMessage:       "TokenMessage",
	TokenEnum:          "TokenEnum",
	TokenRepeated:      "TokenRepeated",
	TokenOptional:      "TokenOptional",
	TokenMap:           "TokenMap",
	TokenOneof:         "TokenOneof",
	TokenImport:        "TokenImport",
	TokenPublic:        "TokenPublic",
	TokenOption:        "TokenOption",
	TokenPacked:        "TokenPacked",
	TokenReserved:      "TokenReserved",
	TokenPackage:       "TokenPackage",
	TokenService:       "TokenService",
	TokenRPC:           "TokenRPC",
	TokenReturns:       "TokenReturns",
	TokenStream:        "TokenStream",
	TokenDouble:        "TokenDouble",
	TokenFloat:         "TokenFloat",
	TokenInt32:         "TokenInt32",
	TokenInt64:         "TokenInt64",
	TokenUInt32:        "TokenUInt32",
	TokenUInt64:        "TokenUInt64",
	TokenSInt32:        "TokenSInt32",
	TokenSInt64:        "TokenSInt64",
	TokenFixed32:       "TokenFixed32",
	TokenFixed64:       "TokenFixed64",
	TokenSFixed32:      "TokenSFixed32",
	TokenSFixed64:      "TokenSFixed64",
	TokenBool:          "TokenBool",
	TokenString:        "TokenString",
	TokenBytes:         "TokenBytes",
	TokenIdentifier:    "TokenIdentifier",
	TokenIntLiteral:    "TokenIntLiteral",
	TokenFloatLiteral:  "TokenFloatLiteral",
	TokenStringLiteral: "TokenStringLiteral",
	TokenTrue:          "TokenTrue",
	TokenFalse:         "TokenFalse",
	TokenLBrace:        "TokenLBrace",
	TokenRBrace:        "TokenRBrace",
	TokenLBracket:      "TokenLBracket",
	TokenRBracket:      "TokenRBracket",
	TokenLParen:        "TokenLParen",
	TokenRParen:        "TokenRParen",
	TokenLT:            "TokenLT",
	TokenGT:            "TokenGT",
	TokenSemicolon:     "TokenSemicolon",
	TokenEquals:        "TokenEquals",
	TokenComma:         "TokenComma",
	TokenDot:           "TokenDot",
}

// String returns the constant name for the token type
// (e.g. TokenSyntax.String() == "TokenSyntax").
func (t TokenType) String() string {
	if int(t) < 0 || int(t) >= len(tokenTypeNames) {
		return fmt.Sprintf("TokenType(%d)", int(t))
	}
	return tokenTypeNames[t]
}

// Token represents a single lexed token.
type Token struct {
	Type   TokenType
	Value  string
	Line   int
	Column int
}

// keywords maps proto keyword text to its token type. Used by the lexer to
// classify identifiers as keywords when they match a known reserved word.
var keywords = map[string]TokenType{
	"syntax":   TokenSyntax,
	"message":  TokenMessage,
	"enum":     TokenEnum,
	"repeated": TokenRepeated,
	"optional": TokenOptional,
	"map":      TokenMap,
	"oneof":    TokenOneof,
	"import":   TokenImport,
	"public":   TokenPublic,
	"option":   TokenOption,
	"packed":   TokenPacked,
	"reserved": TokenReserved,
	"package":  TokenPackage,
	"service":  TokenService,
	"rpc":      TokenRPC,
	"returns":  TokenReturns,
	"stream":   TokenStream,
	"double":   TokenDouble,
	"float":    TokenFloat,
	"int32":    TokenInt32,
	"int64":    TokenInt64,
	"uint32":   TokenUInt32,
	"uint64":   TokenUInt64,
	"sint32":   TokenSInt32,
	"sint64":   TokenSInt64,
	"fixed32":  TokenFixed32,
	"fixed64":  TokenFixed64,
	"sfixed32": TokenSFixed32,
	"sfixed64": TokenSFixed64,
	"bool":     TokenBool,
	"string":   TokenString,
	"bytes":    TokenBytes,
	"true":     TokenTrue,
	"false":    TokenFalse,
}
