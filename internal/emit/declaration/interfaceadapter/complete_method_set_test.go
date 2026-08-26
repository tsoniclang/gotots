package interfaceadapter

import (
	"go/token"
	"go/types"
	"testing"
)

func TestCompleteMethodSetIncludesEachConcreteMethodOnce(t *testing.T) {
	sourcePackage := types.NewPackage("example.com/reflected", "reflected")
	typeName := types.NewTypeName(token.NoPos, sourcePackage, "Value", nil)
	named := types.NewNamed(typeName, types.NewStruct(nil, nil), nil)
	method := func(name string, receiver types.Type) *types.Func {
		return types.NewFunc(
			token.NoPos,
			sourcePackage,
			name,
			types.NewSignatureType(
				types.NewVar(token.NoPos, sourcePackage, "value", receiver),
				nil,
				nil,
				nil,
				nil,
				false,
			),
		)
	}
	named.AddMethod(method("First", named))
	named.AddMethod(method("Second", types.NewPointer(named)))
	named.AddMethod(method("Unobserved", named))
	pointer := types.NewPointer(named)
	contractMethod := func(name string) *types.Func {
		return types.NewFunc(
			token.NoPos,
			sourcePackage,
			name,
			types.NewSignatureType(nil, nil, nil, nil, nil, false),
		)
	}
	first := types.NewInterfaceType(
		[]*types.Func{contractMethod("First")},
		nil,
	).Complete()
	second := types.NewInterfaceType(
		[]*types.Func{contractMethod("Second")},
		nil,
	).Complete()
	demanded, err := demandedMethods(pointer, []Contract{
		{sourceType: first, methodSet: first},
		{sourceType: second, methodSet: second},
	}, true)
	if err != nil {
		t.Fatal(err)
	}
	got := make(map[string]int)
	for _, selected := range demanded {
		got[selected.method.Name()]++
	}
	for _, name := range []string{"First", "Second", "Unobserved"} {
		if got[name] != 1 {
			t.Fatalf("complete method-set count for %s = %d, want 1", name, got[name])
		}
	}
	if len(got) != 3 {
		t.Fatalf("complete method-set names = %v, want three exact methods", got)
	}
}
