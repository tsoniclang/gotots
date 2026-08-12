package naming

import (
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/generic/semanticname"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestGenericConcretizationInterningUsesSemanticIdentity(t *testing.T) {
	constraint := types.NewInterfaceType(nil, nil).Complete()
	typeParameter := types.NewTypeParam(
		types.NewTypeName(token.NoPos, nil, "T", nil),
		constraint,
	)
	parameter := types.NewVar(token.NoPos, nil, "value", typeParameter)
	signature := types.NewSignatureType(
		nil,
		nil,
		[]*types.TypeParam{typeParameter},
		types.NewTuple(parameter),
		types.NewTuple(types.NewVar(token.NoPos, nil, "", typeParameter)),
		false,
	)
	owner := types.NewFunc(token.NoPos, types.NewPackage("example.test/p", "p"), "Identity", signature)
	key := strings.Repeat("a", 64)

	first := newGenericConcretizationForTest(
		t, owner, types.Typ[types.Int], key,
		api.GenericConcretizationEffectCanonical,
	)
	equivalent := newGenericConcretizationForTest(
		t, owner, types.Typ[types.Int], key,
		api.GenericConcretizationEffectCanonical,
	)
	conflicting := newGenericConcretizationForTest(
		t, owner, types.Typ[types.String], key,
		api.GenericConcretizationEffectCanonical,
	)
	synchronous := newGenericConcretizationForTest(
		t, owner, types.Typ[types.Int], key,
		api.GenericConcretizationEffectSynchronous,
	)
	if first == equivalent {
		t.Fatal("test requires independently allocated concretizations")
	}

	registry := NewRegistry()
	canonical, err := registry.internGenericConcretization(first)
	if err != nil {
		t.Fatal(err)
	}
	rejoined, err := registry.internGenericConcretization(equivalent)
	if err != nil {
		t.Fatal(err)
	}
	if rejoined.owner != canonical.owner {
		t.Fatal("equivalent concretization created a sibling canonical artifact")
	}
	if _, err := registry.internGenericConcretization(conflicting); err == nil {
		t.Fatal("same-key concretizations with different arguments were joined")
	}
	if _, err := registry.internGenericConcretization(synchronous); err == nil {
		t.Fatal("same-key concretizations with different effects were joined")
	}
}

func TestConcretizationImportsCannotMaskSameSemanticName(t *testing.T) {
	firstOwner := genericIdentityOwner("example.com/first", "first")
	secondOwner := genericIdentityOwner("example.com/second", "second")
	first := newGenericConcretizationForTest(
		t,
		firstOwner,
		types.Typ[types.Int],
		strings.Repeat("a", 64),
		api.GenericConcretizationEffectCanonical,
	)
	second := newGenericConcretizationForTest(
		t,
		secondOwner,
		types.Typ[types.Int],
		strings.Repeat("b", 64),
		api.GenericConcretizationEffectCanonical,
	)
	registry := NewRegistry()
	names := testFileNames(
		t,
		NewOwner(nil, nil, registry),
		nil,
		nil,
		tsgo.NewFactory(),
		"modules/consumer/source.ts",
		nil,
	)
	firstReference, err := names.GenericConcretization(first)
	if err != nil {
		t.Fatal(err)
	}
	secondReference, err := names.GenericConcretization(second)
	if err != nil {
		t.Fatal(err)
	}
	if firstReference.Name() != "Identity$int" ||
		secondReference.Name() != "Identity$int__from_second" {
		t.Fatalf(
			"colliding concretization imports = %q / %q",
			firstReference.Name(),
			secondReference.Name(),
		)
	}
}

func TestConcreteInstancesShareTheirSemanticOwnerModule(t *testing.T) {
	owner := genericIdentityOwner("example.com/model", "model")
	integer := newGenericConcretizationForTest(
		t,
		owner,
		types.Typ[types.Int],
		strings.Repeat("a", 64),
		api.GenericConcretizationEffectCanonical,
	)
	text := newGenericConcretizationForTest(
		t,
		owner,
		types.Typ[types.String],
		strings.Repeat("b", 64),
		api.GenericConcretizationEffectCanonical,
	)
	registry := NewRegistry()
	integerBinding, err := registry.internGenericConcretization(integer)
	if err != nil {
		t.Fatal(err)
	}
	textBinding, err := registry.internGenericConcretization(text)
	if err != nil {
		t.Fatal(err)
	}
	if integerBinding.owner.OutputPath() != textBinding.owner.OutputPath() ||
		integerBinding.name != "Identity$int" ||
		textBinding.name != "Identity$string" {
		t.Fatalf(
			"concrete instance module/names = %q:%q / %q:%q",
			integerBinding.owner.OutputPath(),
			integerBinding.name,
			textBinding.owner.OutputPath(),
			textBinding.name,
		)
	}
}

func TestConcretizationModulesRejectDistinctOwnerCollisions(t *testing.T) {
	assertGenericModuleCollision(t, false)
}

func TestGenericModuleReservationsAreOrderIndependent(t *testing.T) {
	assertGenericModuleCollision(t, false)
	assertGenericModuleCollision(t, true)
}

func TestGenericCapabilityModuleAllowsDistinctSemanticExports(t *testing.T) {
	names := make(map[genericGeneratedNameScope]string)
	first := genericGeneratedNameScope{
		placement: api.GeneratedArtifactPlacementCompilation,
		module:    "comparison",
		name:      "$go$comparison$int_int_to_boolean",
	}
	second := first
	second.name = "$go$comparison$string_string_to_boolean"
	if err := reserveGenericGeneratedName(
		names,
		first,
		strings.Repeat("a", 64),
		"generic capability",
	); err != nil {
		t.Fatal(err)
	}
	if err := reserveGenericGeneratedName(
		names,
		second,
		strings.Repeat("b", 64),
		"generic capability",
	); err != nil {
		t.Fatalf("shared capability module rejected a distinct export: %v", err)
	}
}

func assertGenericModuleCollision(t *testing.T, reverse bool) {
	t.Helper()
	first := genericIdentityOwner("example.com/first", "first")
	second := genericIdentityOwner("example.com/second", "second")
	if reverse {
		first, second = second, first
	}
	modules := make(map[genericGeneratedModuleScope]*types.Func)
	scope := genericGeneratedModuleScope{
		placement: api.GeneratedArtifactPlacementCompilation,
		module:    "example/model/Identity",
	}
	if err := reserveGenericConcretizationModule(
		modules,
		scope,
		first,
	); err != nil {
		t.Fatal(err)
	}
	if err := reserveGenericConcretizationModule(
		modules,
		scope,
		second,
	); err == nil {
		t.Fatal("distinct owners with one semantic module were joined")
	}
}

func genericIdentityOwner(path string, packageName string) *types.Func {
	sourcePackage := types.NewPackage(path, packageName)
	parameter := types.NewTypeParam(
		types.NewTypeName(token.NoPos, sourcePackage, "T", nil),
		types.NewInterfaceType(nil, nil).Complete(),
	)
	value := types.NewVar(token.NoPos, sourcePackage, "value", parameter)
	return types.NewFunc(
		token.NoPos,
		sourcePackage,
		"Identity",
		types.NewSignatureType(
			nil,
			nil,
			[]*types.TypeParam{parameter},
			types.NewTuple(value),
			types.NewTuple(types.NewVar(token.NoPos, sourcePackage, "", parameter)),
			false,
		),
	)
}

func newGenericConcretizationForTest(
	t *testing.T,
	owner *types.Func,
	argument types.Type,
	key string,
	effect api.GenericConcretizationEffect,
) *api.GenericConcretization {
	t.Helper()
	instantiated, err := types.Instantiate(
		nil,
		owner.Type(),
		[]types.Type{argument},
		true,
	)
	if err != nil {
		t.Fatal(err)
	}
	suffix, err := semanticname.ConcretizationSuffix(
		[]types.Type{argument},
		effect.Synchronous(),
	)
	if err != nil {
		t.Fatal(err)
	}
	concretization, err := api.NewGenericConcretization(
		owner,
		[]types.Type{argument},
		instantiated.(*types.Signature),
		effect,
		key,
		suffix,
		api.GeneratedArtifactPlacementCompilation,
		api.ArtifactOwner{},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	return concretization
}
