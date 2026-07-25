package catalog

import "testing"

func TestLexicalScopeCatalogIsTotalAndPinned(t *testing.T) {
	checkerOwned := map[Kind]bool{
		KindFuncType: true, KindTypeSpec: true, KindBlockStmt: true,
		KindIfStmt: true, KindSwitchStmt: true, KindTypeSwitchStmt: true,
		KindCaseClause: true, KindCommClause: true, KindForStmt: true,
		KindRangeStmt: true,
	}
	for _, kind := range AllKinds() {
		rule := LexicalScope(kind)
		if !rule.Valid() {
			t.Fatalf("%s has invalid lexical-scope rule", kind)
		}
		switch {
		case kind == KindFile && rule != LexicalScopeAlways:
			t.Fatal("file scope is not unconditional")
		case checkerOwned[kind] && rule != LexicalScopeChecker:
			t.Fatalf("%s scope is not checker-evidenced", kind)
		case kind != KindFile && !checkerOwned[kind] &&
			rule != LexicalScopeNone:
			t.Fatalf("%s unexpectedly owns lexical scope", kind)
		}
	}
}
