package emit

import (
	"fmt"
	"go/types"
	"testing"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestNamedStructAssemblyMaterializesOnlyExactValueDemand(t *testing.T) {
	tests := []struct {
		name      string
		operation api.NamedStructOperation
		want      []string
	}{
		{name: "definition"},
		{name: "zero", operation: api.NamedStructOperationZero, want: []string{"$zero"}},
		{name: "copy", operation: api.NamedStructOperationCopy, want: []string{"$copy"}},
		{name: "equal", operation: api.NamedStructOperationEqual, want: []string{"$equal"}},
		{name: "hash", operation: api.NamedStructOperationHash, want: []string{"$hash"}},
		{
			name:      "convert",
			operation: api.NamedStructOperationConvert,
			want:      []string{"$convert"},
		},
		{
			name:      "storage",
			operation: api.NamedStructOperationStorage,
			want:      []string{"$storageOf", "$fromStorage"},
		},
		{
			name:      "assign",
			operation: api.NamedStructOperationAssign,
			want:      []string{"$assign"},
		},
	}
	if got, want := len(tests)-1, int(api.NamedStructOperationAssign); got != want {
		t.Fatalf("operation cases = %d, want complete enum denominator %d", got, want)
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			program := loadDeclarationAssemblyFixture(t)
			record := program.Roots()[0].Types().Scope().
				Lookup("Item").(*types.TypeName)
			session, err := newProgramSession(program, DefaultOptions())
			if err != nil {
				t.Fatal(err)
			}
			if err := session.RequireUse(
				record,
				rootUseDemand(record),
				gostdlib.NoUseSelection(),
			); err != nil {
				t.Fatal(err)
			}
			drainProgramSession(t, session)
			if test.operation.Valid() {
				requirement, err := api.NewNamedStructOperationRequirement(
					record,
					test.operation,
				)
				if err != nil {
					t.Fatal(err)
				}
				if err := session.scheduleDeclarationRequirement(requirement); err != nil {
					t.Fatal(err)
				}
				drainProgramSession(t, session)
			}
			got := namedStructStaticMembers(
				t,
				declarationForObject(t, session, record),
			)
			if fmt.Sprint(got) != fmt.Sprint(test.want) {
				t.Fatalf("Item static surface = %v, want %v", got, test.want)
			}
		})
	}
}

func namedStructStaticMembers(
	t *testing.T,
	declaration *targetDeclaration,
) []string {
	t.Helper()
	var class tsgo.ClassDeclaration
	for _, statement := range declaration.statements {
		candidate, ok := statement.(tsgo.ClassDeclaration)
		if !ok {
			continue
		}
		if class != nil {
			t.Fatal("named-struct assembly emitted multiple classes")
		}
		class = candidate
	}
	if class == nil {
		t.Fatal("named-struct assembly emitted no class")
	}
	var result []string
	for _, member := range class.Members() {
		method, ok := member.(tsgo.MethodDeclaration)
		if !ok || !hasSyntaxModifier(method.Modifiers(), tsgo.SyntaxKindStaticKeyword) {
			continue
		}
		name, ok := method.Name().(tsgo.Identifier)
		if !ok {
			t.Fatalf("static member name = %T, want identifier", method.Name())
		}
		result = append(result, name.Text())
	}
	return result
}

func hasSyntaxModifier(
	modifiers []tsgo.ModifierLike,
	kind tsgo.SyntaxKind,
) bool {
	for _, modifier := range modifiers {
		if modifier.Kind() == kind {
			return true
		}
	}
	return false
}
