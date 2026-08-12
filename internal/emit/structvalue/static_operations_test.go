package structvalue_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestNamedStructValueOperationsAreStaticClassMembers(t *testing.T) {
	source := structTargetSource(t, compileStructFixture(t))
	expected := map[string][]string{
		"Point": {"$zero", "$copy", "$equal"},
		"Box":   {"$zero", "$copy", "$equal"},
		"Empty": {"$zero", "$equal"},
	}
	for owner, want := range expected {
		class := targetClass(t, source, owner)
		var actual []string
		for _, member := range class.Members() {
			method, ok := member.(tsgo.MethodDeclaration)
			if !ok || !strings.HasPrefix(targetName(method.Name()), "$") {
				continue
			}
			name := targetName(method.Name())
			if len(method.Modifiers()) != 1 ||
				method.Modifiers()[0].Kind() != tsgo.SyntaxKindStaticKeyword {
				t.Fatalf("%s.%s is not exactly static", owner, name)
			}
			actual = append(actual, name)
		}
		if !slices.Equal(actual, want) {
			t.Fatalf("%s static value operations = %v, want %v", owner, actual, want)
		}
	}
	for _, owner := range []string{"Mirror", "Reserved", "Grouped"} {
		for _, member := range targetClass(t, source, owner).Members() {
			method, ok := member.(tsgo.MethodDeclaration)
			if ok && strings.HasPrefix(targetName(method.Name()), "$") {
				t.Fatalf("undemanded static operation %s.%s was emitted", owner, targetName(method.Name()))
			}
		}
	}
	for _, statement := range source.Statements() {
		function, ok := statement.(tsgo.FunctionDeclaration)
		if !ok {
			continue
		}
		for _, suffix := range []string{"$zero", "$copy", "$equal"} {
			if strings.HasSuffix(function.Name().Text(), suffix) {
				t.Fatalf(
					"legacy top-level operation %s was emitted",
					function.Name().Text(),
				)
			}
		}
	}
}
