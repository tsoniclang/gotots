package catalog

// LexicalScopeRule is the closed relation between syntax and Go lexical
// scopes. Conditional owners require a matching scope in the selected
// toolchain's type evidence; an AST kind alone cannot assert one.
type LexicalScopeRule uint8

const (
	LexicalScopeInvalid LexicalScopeRule = iota
	LexicalScopeNone
	LexicalScopeAlways
	LexicalScopeChecker
)

func (rule LexicalScopeRule) Valid() bool {
	return rule >= LexicalScopeNone && rule <= LexicalScopeChecker
}

var lexicalScopeByKind = [kindCount + 1]LexicalScopeRule{
	KindFile:           LexicalScopeAlways,
	KindFuncType:       LexicalScopeChecker,
	KindTypeSpec:       LexicalScopeChecker,
	KindBlockStmt:      LexicalScopeChecker,
	KindIfStmt:         LexicalScopeChecker,
	KindSwitchStmt:     LexicalScopeChecker,
	KindTypeSwitchStmt: LexicalScopeChecker,
	KindCaseClause:     LexicalScopeChecker,
	KindCommClause:     LexicalScopeChecker,
	KindForStmt:        LexicalScopeChecker,
	KindRangeStmt:      LexicalScopeChecker,
}

// LexicalScope returns the sole cataloged lexical-scope rule for a syntax
// kind. Every active kind without an entry explicitly owns no lexical scope.
func LexicalScope(kind Kind) LexicalScopeRule {
	if !kind.Valid() {
		return LexicalScopeInvalid
	}
	rule := lexicalScopeByKind[kind]
	if rule == LexicalScopeInvalid {
		return LexicalScopeNone
	}
	return rule
}
