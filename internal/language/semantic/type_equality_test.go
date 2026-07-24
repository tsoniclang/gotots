package semantic

import (
	"reflect"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/identity"
)

func TestTypeRetainsNoRenderedCanonicalForm(t *testing.T) {
	typ := reflect.TypeFor[Type]()
	if typ.NumField() != 2 ||
		typ.Field(0).Name != "id" ||
		typ.Field(1).Name != "spec" {
		t.Fatalf("semantic type retains a parallel representation: %v", typ)
	}
	for index := 0; index < typ.NumField(); index++ {
		if typ.Field(index).Type.Kind() == reflect.String {
			t.Fatalf(
				"semantic type retains rendered field %s",
				typ.Field(index).Name,
			)
		}
	}
}

func TestTypeEqualityCoversEveryDescriptorComponent(t *testing.T) {
	first := testSemanticTypeID(t, "ab")
	second := testSemanticTypeID(t, "cd")
	fixture := semanticFixture(t)
	declaration, err := identity.NewPackageDeclarationID(
		fixture.pkg,
		identity.SemanticObjectType,
		"Element",
	)
	if err != nil {
		t.Fatal(err)
	}
	otherDeclaration, err := identity.NewPackageDeclarationID(
		fixture.pkg,
		identity.SemanticObjectType,
		"Other",
	)
	if err != nil {
		t.Fatal(err)
	}
	parameter, err := NewTypeParameterOwner(
		declaration,
		identity.DefinitionID{},
		TypeParameterDeclared,
		0,
	)
	if err != nil {
		t.Fatal(err)
	}
	otherParameter, err := NewTypeParameterOwner(
		otherDeclaration,
		identity.DefinitionID{},
		TypeParameterDeclared,
		0,
	)
	if err != nil {
		t.Fatal(err)
	}
	base := TypeSpec{
		Kind: TypeInterface, Basic: BasicInt,
		Declaration: declaration, Parameter: parameter,
		Arguments:  []identity.SemanticTypeID{first},
		Underlying: first, Target: first, Constraint: first,
		Element: first, Key: first, Length: 1,
		Direction: ChannelSendReceive,
		Signature: Signature{
			Receiver:               first,
			ReceiverTypeParameters: []identity.SemanticTypeID{first},
			TypeParameters:         []identity.SemanticTypeID{first},
			Parameters:             []identity.SemanticTypeID{first},
			Results:                []identity.SemanticTypeID{first},
		},
		Fields: []TypeField{{
			Name: "Value", Type: first, Ordinal: 0,
		}},
		Methods: []TypeMethod{{
			Name: "Run", Signature: first, Ordinal: 0,
		}},
		Embeddeds:  []identity.SemanticTypeID{first},
		Terms:      []TypeTerm{{Type: first}},
		TypeSet:    TypeSetFinite,
		Comparable: true,
		Elements:   []identity.SemanticTypeID{first},
	}
	mutations := map[string]func(*TypeSpec){
		"kind":        func(spec *TypeSpec) { spec.Kind = TypeUnion },
		"basic":       func(spec *TypeSpec) { spec.Basic = BasicString },
		"declaration": func(spec *TypeSpec) { spec.Declaration = otherDeclaration },
		"parameter":   func(spec *TypeSpec) { spec.Parameter = otherParameter },
		"arguments":   func(spec *TypeSpec) { spec.Arguments[0] = second },
		"underlying":  func(spec *TypeSpec) { spec.Underlying = second },
		"target":      func(spec *TypeSpec) { spec.Target = second },
		"constraint":  func(spec *TypeSpec) { spec.Constraint = second },
		"element":     func(spec *TypeSpec) { spec.Element = second },
		"key":         func(spec *TypeSpec) { spec.Key = second },
		"length":      func(spec *TypeSpec) { spec.Length++ },
		"direction": func(spec *TypeSpec) {
			spec.Direction = ChannelReceiveOnly
		},
		"receiver": func(spec *TypeSpec) {
			spec.Signature.Receiver = second
		},
		"receiver-parameters": func(spec *TypeSpec) {
			spec.Signature.ReceiverTypeParameters[0] = second
		},
		"type-parameters": func(spec *TypeSpec) {
			spec.Signature.TypeParameters[0] = second
		},
		"parameters": func(spec *TypeSpec) {
			spec.Signature.Parameters[0] = second
		},
		"results": func(spec *TypeSpec) {
			spec.Signature.Results[0] = second
		},
		"variadic": func(spec *TypeSpec) {
			spec.Signature.Variadic = true
		},
		"fields": func(spec *TypeSpec) {
			spec.Fields[0].Type = second
		},
		"methods": func(spec *TypeSpec) {
			spec.Methods[0].Signature = second
		},
		"embeddeds": func(spec *TypeSpec) {
			spec.Embeddeds[0] = second
		},
		"terms": func(spec *TypeSpec) {
			spec.Terms[0].Type = second
		},
		"type-set": func(spec *TypeSpec) {
			spec.TypeSet = TypeSetUniverse
		},
		"comparable": func(spec *TypeSpec) {
			spec.Comparable = false
		},
		"elements": func(spec *TypeSpec) {
			spec.Elements[0] = second
		},
	}
	left := Type{id: first, spec: cloneTypeSpec(base)}
	if !left.Equal(Type{id: first, spec: cloneTypeSpec(base)}) {
		t.Fatal("identical semantic type descriptors compare unequal")
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			changed := cloneTypeSpec(base)
			mutate(&changed)
			if left.Equal(Type{id: first, spec: changed}) {
				t.Fatal("descriptor mutation compared equal")
			}
		})
	}
	if left.Equal(Type{id: second, spec: cloneTypeSpec(base)}) {
		t.Fatal("type identity mutation compared equal")
	}
	if allocations := testing.AllocsPerRun(1_000, func() {
		_ = left.Equal(left)
	}); allocations != 0 {
		t.Fatalf("semantic type equality allocates %.2f times", allocations)
	}
}

func testSemanticTypeID(
	t *testing.T,
	pair string,
) identity.SemanticTypeID {
	t.Helper()
	id, err := identity.NewSemanticTypeID(strings.Repeat(pair, 32))
	if err != nil {
		t.Fatal(err)
	}
	return id
}
