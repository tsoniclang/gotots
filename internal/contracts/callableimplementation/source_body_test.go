package callableimplementation

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func TestSourceBodyDigestIsCanonicalAndBehaviorSensitive(t *testing.T) {
	formatted := digestFixtureBody(t, "package fixture\nfunc value(input int) int { return input + 1 }\n")
	reformatted := digestFixtureBody(t, "package fixture\nfunc value(input int) int {\nreturn input+1\n}\n")
	changed := digestFixtureBody(t, "package fixture\nfunc value(input int) int { return input + 2 }\n")
	if formatted != reformatted {
		t.Fatal("format-only source change altered the canonical body digest")
	}
	if formatted == changed {
		t.Fatal("behavioral source change preserved the canonical body digest")
	}
}

func digestFixtureBody(t *testing.T, source string) string {
	t.Helper()
	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, "fixture.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	declaration := file.Decls[0].(*ast.FuncDecl)
	digest, err := SourceBodyDigest(fileSet, declaration.Body)
	if err != nil {
		t.Fatal(err)
	}
	return digest
}
