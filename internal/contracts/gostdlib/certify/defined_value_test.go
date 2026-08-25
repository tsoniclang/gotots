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
	)
	if err != nil {
		t.Fatal(err)
	}
	bindings := selected[0].Bindings
	if bindings[0].DefinedValue != gostdlib.DefinedValueRepresentationOperations ||
		bindings[1].DefinedValue != gostdlib.DefinedValueRepresentationCanonical {
		t.Fatalf("defined value assignments = %#v", bindings)
	}

	for _, testCase := range []struct {
		name   string
		facets []facetSeed
		want   string
	}{
		{
			name: "missing",
			want: basicIdentity,
		},
		{
			name:   "duplicate class",
			facets: append(append([]facetSeed{}, operations...), operations...),
			want:   "operation representation is duplicated",
		},
		{
			name: "orphan",
			facets: append(append([]facetSeed{}, operations...), facetSeed{
				Kind:           gostdlib.FacetDefinedValueOperations,
				SourceIdentity: "absent",
			}),
			want: "operation representation has no selected type",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := applyDefinedValueRepresentations(
				source,
				modules,
				testCase.facets,
			)
			if err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("error = %v, want %q", err, testCase.want)
			}
		})
	}

	methodIdentity := "example|kind=2|receiver=|name=MethodAction"
	methodSource := goSurface{objects: map[string]goObject{
		methodIdentity: {object: testDefinedCallableWithMethod("MethodAction")},
	}}
	methodModules := []gostdlib.ModuleDocument{{Bindings: []gostdlib.BindingDocument{{
		Identity: methodIdentity,
		Kind:     gostdlib.BindingType,
		Access:   gostdlib.AccessExport,
	}}}}
	_, err = applyDefinedValueRepresentations(methodSource, methodModules, nil)
	if err == nil || !strings.Contains(err.Error(), "no exact representation owner") {
		t.Fatalf("method-bearing callable error = %v", err)
	}
}

func TestDefinedCallableEffectsExactJoinRepresentations(t *testing.T) {
	canonicalIdentity := "example|kind=2|receiver=|name=CanonicalAction"
	operationCallableIdentity := "example|kind=2|receiver=|name=OperationAction"
	operationValueIdentity := "example|kind=2|receiver=|name=OperationCount"
	source := goSurface{objects: map[string]goObject{
		canonicalIdentity: {
			object: testDefinedType("CanonicalAction", testSignature()),
		},
		operationCallableIdentity: {
			object: testDefinedType("OperationAction", testSignature()),
		},
		operationValueIdentity: {
			object: testDefinedType("OperationCount", types.Typ[types.Int]),
		},
	}}
	modules := []gostdlib.ModuleDocument{{
		Bindings: []gostdlib.BindingDocument{
			{
				Identity:     canonicalIdentity,
				Kind:         gostdlib.BindingType,
				Access:       gostdlib.AccessExport,
				DefinedValue: gostdlib.DefinedValueRepresentationCanonical,
				Effect:       gostdlib.EffectSynchronous,
			},
			{
				Identity:     operationCallableIdentity,
				Kind:         gostdlib.BindingType,
				Access:       gostdlib.AccessExport,
				DefinedValue: gostdlib.DefinedValueRepresentationOperations,
			},
			{
				Identity:     operationValueIdentity,
				Kind:         gostdlib.BindingType,
				Access:       gostdlib.AccessExport,
				DefinedValue: gostdlib.DefinedValueRepresentationOperations,
			},
		},
	}}
	facets := []gostdlib.FacetModuleDocument{{
		Facets: []gostdlib.FacetDocument{
			{
				Kind:           gostdlib.FacetDefinedValueOperations,
				SourceIdentity: operationCallableIdentity,
				Effect:         gostdlib.EffectSynchronous,
			},
			{
				Kind:           gostdlib.FacetDefinedValueOperations,
				SourceIdentity: operationValueIdentity,
			},
		},
	}}
	if err := verifyDefinedCallableEffects(source, modules, facets); err != nil {
		t.Fatal(err)
	}

	missingCanonical := append([]gostdlib.ModuleDocument(nil), modules...)
	missingCanonical[0].Bindings = append(
		[]gostdlib.BindingDocument(nil),
		modules[0].Bindings...,
	)
	missingCanonical[0].Bindings[0].Effect = gostdlib.EffectInvalid
	if err := verifyDefinedCallableEffects(
		source,
		missingCanonical,
		facets,
	); err == nil || !strings.Contains(err.Error(), "canonical callable effect") {
		t.Fatalf("missing canonical effect error = %v", err)
	}

	extraOperation := append([]gostdlib.FacetModuleDocument(nil), facets...)
	extraOperation[0].Facets = append(
		[]gostdlib.FacetDocument(nil),
		facets[0].Facets...,
	)
	extraOperation[0].Facets[1].Effect = gostdlib.EffectSynchronous
	if err := verifyDefinedCallableEffects(
		source,
		modules,
		extraOperation,
	); err == nil || !strings.Contains(err.Error(), "operation callable effect") {
		t.Fatalf("extra operation effect error = %v", err)
	}

	missingOperation := append([]gostdlib.FacetModuleDocument(nil), facets...)
	missingOperation[0].Facets = append(
		[]gostdlib.FacetDocument(nil),
		facets[0].Facets[1:]...,
	)
	if err := verifyDefinedCallableEffects(
		source,
		modules,
		missingOperation,
	); err == nil || !strings.Contains(err.Error(), "operation callable effect") {
		t.Fatalf("missing operation effect error = %v", err)
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

func testDefinedCallableWithMethod(name string) *types.TypeName {
	object := testDefinedType(name, testSignature())
	named := object.Type().(*types.Named)
	receiver := types.NewVar(token.NoPos, object.Pkg(), "receiver", named)
	method := types.NewFunc(
		token.NoPos,
		object.Pkg(),
		"Method",
		types.NewSignatureType(receiver, nil, nil, nil, nil, false),
	)
	named.AddMethod(method)
	return object
}
