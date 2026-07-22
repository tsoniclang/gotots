package catalog

import "fmt"

// TokenKind is the closed catalog of Go lexical tokens for the selected
// toolchain. Values are explicit and permanent; the reconciliation gate joins
// this catalog bijectively against the toolchain's own go/token constant set.
type TokenKind uint16

// TokenClass is the closed lexical class of one token.
type TokenClass uint8

const (
	TokenClassInvalid TokenClass = iota
	TokenClassSpecial
	TokenClassLiteral
	TokenClassOperator
	TokenClassKeyword

	numTokenClasses
)

var tokenClassNames = [numTokenClasses]string{
	TokenClassSpecial: "special", TokenClassLiteral: "literal",
	TokenClassOperator: "operator", TokenClassKeyword: "keyword",
}

// Valid reports whether c names a token class.
func (c TokenClass) Valid() bool { return c > TokenClassInvalid && c < numTokenClasses }

// String renders c for reports.
func (c TokenClass) String() string {
	if c.Valid() {
		return tokenClassNames[c]
	}
	return fmt.Sprintf("catalog.TokenClass(%d)", uint8(c))
}

// Explicit, permanent token identities. Do not renumber; append only.
const (
	TokenInvalid TokenKind = 0

	TokenILLEGAL TokenKind = 1
	TokenEOF     TokenKind = 2
	TokenCOMMENT TokenKind = 3

	TokenIDENT  TokenKind = 4
	TokenINT    TokenKind = 5
	TokenFLOAT  TokenKind = 6
	TokenIMAG   TokenKind = 7
	TokenCHAR   TokenKind = 8
	TokenSTRING TokenKind = 9

	TokenADD            TokenKind = 10
	TokenSUB            TokenKind = 11
	TokenMUL            TokenKind = 12
	TokenQUO            TokenKind = 13
	TokenREM            TokenKind = 14
	TokenAND            TokenKind = 15
	TokenOR             TokenKind = 16
	TokenXOR            TokenKind = 17
	TokenSHL            TokenKind = 18
	TokenSHR            TokenKind = 19
	TokenAND_NOT        TokenKind = 20
	TokenADD_ASSIGN     TokenKind = 21
	TokenSUB_ASSIGN     TokenKind = 22
	TokenMUL_ASSIGN     TokenKind = 23
	TokenQUO_ASSIGN     TokenKind = 24
	TokenREM_ASSIGN     TokenKind = 25
	TokenAND_ASSIGN     TokenKind = 26
	TokenOR_ASSIGN      TokenKind = 27
	TokenXOR_ASSIGN     TokenKind = 28
	TokenSHL_ASSIGN     TokenKind = 29
	TokenSHR_ASSIGN     TokenKind = 30
	TokenAND_NOT_ASSIGN TokenKind = 31
	TokenLAND           TokenKind = 32
	TokenLOR            TokenKind = 33
	TokenARROW          TokenKind = 34
	TokenINC            TokenKind = 35
	TokenDEC            TokenKind = 36
	TokenEQL            TokenKind = 37
	TokenLSS            TokenKind = 38
	TokenGTR            TokenKind = 39
	TokenASSIGN         TokenKind = 40
	TokenNOT            TokenKind = 41
	TokenNEQ            TokenKind = 42
	TokenLEQ            TokenKind = 43
	TokenGEQ            TokenKind = 44
	TokenDEFINE         TokenKind = 45
	TokenELLIPSIS       TokenKind = 46
	TokenLPAREN         TokenKind = 47
	TokenLBRACK         TokenKind = 48
	TokenLBRACE         TokenKind = 49
	TokenCOMMA          TokenKind = 50
	TokenPERIOD         TokenKind = 51
	TokenRPAREN         TokenKind = 52
	TokenRBRACK         TokenKind = 53
	TokenRBRACE         TokenKind = 54
	TokenSEMICOLON      TokenKind = 55
	TokenCOLON          TokenKind = 56

	TokenBREAK       TokenKind = 57
	TokenCASE        TokenKind = 58
	TokenCHAN        TokenKind = 59
	TokenCONST       TokenKind = 60
	TokenCONTINUE    TokenKind = 61
	TokenDEFAULT     TokenKind = 62
	TokenDEFER       TokenKind = 63
	TokenELSE        TokenKind = 64
	TokenFALLTHROUGH TokenKind = 65
	TokenFOR         TokenKind = 66
	TokenFUNC        TokenKind = 67
	TokenGO          TokenKind = 68
	TokenGOTO        TokenKind = 69
	TokenIF          TokenKind = 70
	TokenIMPORT      TokenKind = 71
	TokenINTERFACE   TokenKind = 72
	TokenMAP         TokenKind = 73
	TokenPACKAGE     TokenKind = 74
	TokenRANGE       TokenKind = 75
	TokenRETURN      TokenKind = 76
	TokenSELECT      TokenKind = 77
	TokenSTRUCT      TokenKind = 78
	TokenSWITCH      TokenKind = 79
	TokenTYPE        TokenKind = 80
	TokenVAR         TokenKind = 81

	TokenTILDE TokenKind = 82

	// tokenCount is the highest assigned identity; append-only.
	tokenCount = 82
)

type tokenDescriptor struct {
	name     string // toolchain constant name, e.g. "ADD"
	spelling string // token.Token.String() of the toolchain constant
	class    TokenClass
}

var tokenTable = [tokenCount + 1]tokenDescriptor{
	TokenILLEGAL: {"ILLEGAL", "ILLEGAL", TokenClassSpecial},
	TokenEOF:     {"EOF", "EOF", TokenClassSpecial},
	TokenCOMMENT: {"COMMENT", "COMMENT", TokenClassSpecial},
	TokenIDENT:   {"IDENT", "IDENT", TokenClassLiteral},
	TokenINT:     {"INT", "INT", TokenClassLiteral},
	TokenFLOAT:   {"FLOAT", "FLOAT", TokenClassLiteral},
	TokenIMAG:    {"IMAG", "IMAG", TokenClassLiteral},
	TokenCHAR:    {"CHAR", "CHAR", TokenClassLiteral},
	TokenSTRING:  {"STRING", "STRING", TokenClassLiteral},

	TokenADD: {"ADD", "+", TokenClassOperator}, TokenSUB: {"SUB", "-", TokenClassOperator},
	TokenMUL: {"MUL", "*", TokenClassOperator}, TokenQUO: {"QUO", "/", TokenClassOperator},
	TokenREM: {"REM", "%", TokenClassOperator}, TokenAND: {"AND", "&", TokenClassOperator},
	TokenOR: {"OR", "|", TokenClassOperator}, TokenXOR: {"XOR", "^", TokenClassOperator},
	TokenSHL: {"SHL", "<<", TokenClassOperator}, TokenSHR: {"SHR", ">>", TokenClassOperator},
	TokenAND_NOT:    {"AND_NOT", "&^", TokenClassOperator},
	TokenADD_ASSIGN: {"ADD_ASSIGN", "+=", TokenClassOperator}, TokenSUB_ASSIGN: {"SUB_ASSIGN", "-=", TokenClassOperator},
	TokenMUL_ASSIGN: {"MUL_ASSIGN", "*=", TokenClassOperator}, TokenQUO_ASSIGN: {"QUO_ASSIGN", "/=", TokenClassOperator},
	TokenREM_ASSIGN: {"REM_ASSIGN", "%=", TokenClassOperator}, TokenAND_ASSIGN: {"AND_ASSIGN", "&=", TokenClassOperator},
	TokenOR_ASSIGN: {"OR_ASSIGN", "|=", TokenClassOperator}, TokenXOR_ASSIGN: {"XOR_ASSIGN", "^=", TokenClassOperator},
	TokenSHL_ASSIGN: {"SHL_ASSIGN", "<<=", TokenClassOperator}, TokenSHR_ASSIGN: {"SHR_ASSIGN", ">>=", TokenClassOperator},
	TokenAND_NOT_ASSIGN: {"AND_NOT_ASSIGN", "&^=", TokenClassOperator},
	TokenLAND:           {"LAND", "&&", TokenClassOperator}, TokenLOR: {"LOR", "||", TokenClassOperator},
	TokenARROW: {"ARROW", "<-", TokenClassOperator}, TokenINC: {"INC", "++", TokenClassOperator},
	TokenDEC: {"DEC", "--", TokenClassOperator}, TokenEQL: {"EQL", "==", TokenClassOperator},
	TokenLSS: {"LSS", "<", TokenClassOperator}, TokenGTR: {"GTR", ">", TokenClassOperator},
	TokenASSIGN: {"ASSIGN", "=", TokenClassOperator}, TokenNOT: {"NOT", "!", TokenClassOperator},
	TokenNEQ: {"NEQ", "!=", TokenClassOperator}, TokenLEQ: {"LEQ", "<=", TokenClassOperator},
	TokenGEQ: {"GEQ", ">=", TokenClassOperator}, TokenDEFINE: {"DEFINE", ":=", TokenClassOperator},
	TokenELLIPSIS: {"ELLIPSIS", "...", TokenClassOperator},
	TokenLPAREN:   {"LPAREN", "(", TokenClassOperator}, TokenLBRACK: {"LBRACK", "[", TokenClassOperator},
	TokenLBRACE: {"LBRACE", "{", TokenClassOperator}, TokenCOMMA: {"COMMA", ",", TokenClassOperator},
	TokenPERIOD: {"PERIOD", ".", TokenClassOperator}, TokenRPAREN: {"RPAREN", ")", TokenClassOperator},
	TokenRBRACK: {"RBRACK", "]", TokenClassOperator}, TokenRBRACE: {"RBRACE", "}", TokenClassOperator},
	TokenSEMICOLON: {"SEMICOLON", ";", TokenClassOperator}, TokenCOLON: {"COLON", ":", TokenClassOperator},

	TokenBREAK: {"BREAK", "break", TokenClassKeyword}, TokenCASE: {"CASE", "case", TokenClassKeyword},
	TokenCHAN: {"CHAN", "chan", TokenClassKeyword}, TokenCONST: {"CONST", "const", TokenClassKeyword},
	TokenCONTINUE: {"CONTINUE", "continue", TokenClassKeyword}, TokenDEFAULT: {"DEFAULT", "default", TokenClassKeyword},
	TokenDEFER: {"DEFER", "defer", TokenClassKeyword}, TokenELSE: {"ELSE", "else", TokenClassKeyword},
	TokenFALLTHROUGH: {"FALLTHROUGH", "fallthrough", TokenClassKeyword}, TokenFOR: {"FOR", "for", TokenClassKeyword},
	TokenFUNC: {"FUNC", "func", TokenClassKeyword}, TokenGO: {"GO", "go", TokenClassKeyword},
	TokenGOTO: {"GOTO", "goto", TokenClassKeyword}, TokenIF: {"IF", "if", TokenClassKeyword},
	TokenIMPORT: {"IMPORT", "import", TokenClassKeyword}, TokenINTERFACE: {"INTERFACE", "interface", TokenClassKeyword},
	TokenMAP: {"MAP", "map", TokenClassKeyword}, TokenPACKAGE: {"PACKAGE", "package", TokenClassKeyword},
	TokenRANGE: {"RANGE", "range", TokenClassKeyword}, TokenRETURN: {"RETURN", "return", TokenClassKeyword},
	TokenSELECT: {"SELECT", "select", TokenClassKeyword}, TokenSTRUCT: {"STRUCT", "struct", TokenClassKeyword},
	TokenSWITCH: {"SWITCH", "switch", TokenClassKeyword}, TokenTYPE: {"TYPE", "type", TokenClassKeyword},
	TokenVAR: {"VAR", "var", TokenClassKeyword},

	TokenTILDE: {"TILDE", "~", TokenClassOperator},
}

// Valid reports whether k names a token in the catalog.
func (k TokenKind) Valid() bool { return k >= 1 && k <= tokenCount }

// ConstName is the toolchain constant name, e.g. "ADD".
func (k TokenKind) ConstName() string {
	if !k.Valid() {
		return ""
	}
	return tokenTable[k].name
}

// Spelling is the toolchain String() rendering, e.g. "+".
func (k TokenKind) Spelling() string {
	if !k.Valid() {
		return ""
	}
	return tokenTable[k].spelling
}

// Class is the lexical class.
func (k TokenKind) Class() TokenClass {
	if !k.Valid() {
		return TokenClassInvalid
	}
	return tokenTable[k].class
}

// String renders k for reports.
func (k TokenKind) String() string {
	if name := k.ConstName(); name != "" {
		return name
	}
	return fmt.Sprintf("catalog.TokenKind(%d)", uint16(k))
}

// AllTokens returns every token in ascending identity order.
func AllTokens() []TokenKind {
	out := make([]TokenKind, 0, tokenCount)
	for id := 1; id <= tokenCount; id++ {
		out = append(out, TokenKind(id))
	}
	return out
}

// tokenBySpelling indexes tokens by their toolchain spelling once; spellings
// are unique across the token set.
var tokenBySpelling = func() map[string]TokenKind {
	m := make(map[string]TokenKind, tokenCount)
	for id := TokenKind(1); id <= tokenCount; id++ {
		m[tokenTable[id].spelling] = id
	}
	return m
}()

// TokenBySpelling resolves a toolchain token spelling (token.Token.String())
// to the catalog token, or TokenInvalid when unknown.
func TokenBySpelling(spelling string) TokenKind { return tokenBySpelling[spelling] }
