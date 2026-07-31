package certify

import (
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
)

func TestDefinedValueRepresentationsAreTotalAndExclusive(t *testing.T) {
	basicIdentity := "example|kind=2|receiver=|name=Count"
	callableIdentity := "example|kind=2|receiver=|name=Action"
	source := goSurface{objects: map[string]goObject{
		basicIdentity:    {object: testDefinedType("Count", types.Typ[types.Int])},
		callableIdentity: {object: testDefinedType("Action", testSignature())},
	}}
	modules := []gostdlib.ModuleDocument{{Bindings: []gostdlib.BindingDocument{
		{
			Identity: basicIdentity,
			Kind:     gostdlib.BindingType,
			Access:   gostdlib.AccessExport,
		},
		{
			Identity: callableIdentity,
			Kind:     gostdlib.BindingType,
			Access:   gostdlib.AccessExport,
		},
	}}}
	operations := []facetSeed{{
		Kind:           gostdlib.FacetDefinedValueOperations,
		SourceIdentity: basicIdentity,
	}}

	selected, err := applyDefinedValueRepresentations(
		source,
		modules,
		operations,
		map[string]struct{}{callableIdentity: {}},
	)
	if err != nil {
		t.Fatal(err)
	}
	bindings := selected[0].Bindings
	if bindings[0].DefinedValue != gostdlib.DefinedValueRepresentationOperations ||
		bindings[1].DefinedValue != gostdlib.DefinedValueRepresentationIdentity {
		t.Fatalf("defined value assignments = %#v", bindings)
	}

	for _, testCase := range []struct {
		name       string
		facets     []facetSeed
		identities map[string]struct{}
		want       string
	}{
		{
			name:       "missing",
			identities: map[string]struct{}{callableIdentity: {}},
			want:       basicIdentity,
		},
		{
			name:   "duplicate class",
			facets: append(append([]facetSeed{}, operations...), operations...),
			identities: map[string]struct{}{
				callableIdentity: {},
			},
			want: "operation representation is duplicated",
		},
		{
			name:   "identity and operations",
			facets: operations,
			identities: map[string]struct{}{
				basicIdentity:    {},
				callableIdentity: {},
			},
			want: "missing or duplicated",
		},
		{
			name: "non-callable identity",
			identities: map[string]struct{}{
				basicIdentity:    {},
				callableIdentity: {},
			},
			want: "identity representation is not callable",
		},
		{
			name:   "orphan",
			facets: operations,
			identities: map[string]struct{}{
				callableIdentity: {},
				"absent":         {},
			},
			want: "identity representation has no selected type",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := applyDefinedValueRepresentations(
				source,
				modules,
				testCase.facets,
				testCase.identities,
			)
			if err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("error = %v, want %q", err, testCase.want)
			}
		})
	}
}

func testDefinedType(name string, underlying types.Type) *types.TypeName {
	object := types.NewTypeName(
		token.NoPos,
		types.NewPackage("example", "example"),
		name,
		nil,
	)
	types.NewNamed(object, underlying, nil)
	return object
}

func testSignature() *types.Signature {
	return types.NewSignatureType(nil, nil, nil, nil, nil, false)
}
