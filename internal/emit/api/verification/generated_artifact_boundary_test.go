package api_test

import (
	"errors"
	"go/ast"
	"go/token"
	"go/types"
	"strings"
	"testing"

	. "github.com/tsoniclang/gotots/internal/emit/api"
)

func TestGeneratedCapabilityErrorNamesItsSemanticOperation(t *testing.T) {
	signature := types.NewSignatureType(
		nil,
		nil,
		nil,
		types.NewTuple(
			types.NewVar(token.NoPos, nil, "value", types.Typ[types.Int32]),
		),
		types.NewTuple(
			types.NewVar(token.NoPos, nil, "", types.Typ[types.Int32]),
		),
		false,
	)
	selection, err := SelectGenericOperation(GenericOperationCopy)
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := NewCompilationGenericCapabilityArtifact(
		selection,
		signature,
		"copy-int32",
		"$goCapability_copy_int32",
		"support/generics/capabilities/copy-int32.ts",
	)
	if err != nil {
		t.Fatal(err)
	}
	message := WrapGeneratedArtifactError(
		artifact,
		errors.New("failed"),
	).Error()
	if !strings.Contains(message, "operation copy") {
		t.Fatalf("generated capability diagnostic = %q", message)
	}
}

func TestPackageAssemblyArtifactOwnerIsExact(t *testing.T) {
	sourcePackage := types.NewPackage("example.com/exported", "exported")
	owner, err := PackageAssemblyArtifactOwner(sourcePackage)
	if err != nil {
		t.Fatal(err)
	}
	selected, ok := owner.PackageAssembly()
	if !owner.Valid() ||
		!ok ||
		selected != sourcePackage ||
		owner.Package() != sourcePackage ||
		owner.Name() != "example.com/exported.$assembly" {
		t.Fatalf("package assembly owner = %#v", owner)
	}
	if _, err := PackageAssemblyArtifactOwner(
		types.NewPackage("", "invalid"),
	); err == nil {
		t.Fatal("package assembly owner accepted an empty package path")
	}
}

func TestLexicalGenericCapabilityFacetUsesItsReconstructionOwner(
	t *testing.T,
) {
	sourcePackage := types.NewPackage("example.com/lexical", "lexical")
	owner := types.NewFunc(
		token.Pos(10),
		sourcePackage,
		"Use",
		types.NewSignatureType(
			nil,
			nil,
			nil,
			types.NewTuple(),
			types.NewTuple(),
			false,
		),
	)
	if existing := sourcePackage.Scope().Insert(owner); existing != nil {
		t.Fatal("source owner insertion failed")
	}
	scope := types.NewScope(
		sourcePackage.Scope(),
		token.Pos(20),
		token.Pos(100),
		"Use",
	)
	anchor := types.NewTypeName(
		token.Pos(30),
		sourcePackage,
		"Local",
		types.Typ[types.Int32],
	)
	if existing := scope.Insert(anchor); existing != nil {
		t.Fatal("local anchor insertion failed")
	}
	signature := types.NewSignatureType(
		nil,
		nil,
		nil,
		types.NewTuple(
			types.NewVar(token.NoPos, nil, "value", anchor.Type()),
		),
		types.NewTuple(
			types.NewVar(token.NoPos, nil, "", anchor.Type()),
		),
		false,
	)
	selection, err := SelectGenericOperation(GenericOperationCopy)
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := NewLexicalGenericCapabilityArtifact(
		selection,
		signature,
		"copy-local",
		"$goCapability_copy_local",
		MustSourceArtifactOwner(owner),
		anchor,
	)
	if err != nil {
		t.Fatal(err)
	}
	facet, err := NewGenericCapabilityCallableFacet(artifact)
	if err != nil {
		t.Fatal(err)
	}
	selected, ok := facet.GenericCapability()
	if facet.Owner() != artifact.ReconstructionOwner() || !ok || selected != artifact {
		t.Fatalf(
			"lexical capability owner = facet %#v, want reconstruction %#v",
			facet.Owner(),
			artifact.ReconstructionOwner(),
		)
	}
}

func TestInterfaceDynamicTypeRequestCarriesExactGoType(t *testing.T) {
	sourceType := types.Typ[types.Int32]
	artifact, err := NewCompilationGeneratedArtifact(
		GeneratedArtifactInterfaceDynamicTypeToken,
		sourceType,
		"artifact",
		"$goDynamicType$artifact",
		"support/interface-types.ts",
	)
	if err != nil {
		t.Fatal(err)
	}
	request, err := NewInterfaceDynamicTypeTokenRequest(artifact)
	if err != nil {
		t.Fatal(err)
	}
	requirement, ok := request.DeclarationRequirement()
	selected, selectedOK := requirement.InterfaceDynamicTypeToken()
	dynamicType, typeOK := selected.InterfaceDynamicType()
	if !ok ||
		!selectedOK ||
		!typeOK ||
		selected != artifact ||
		dynamicType != sourceType ||
		request.LegalScope() != ScopeCompilationSupport ||
		request.Execution() != ExecutionStatic {
		t.Fatalf("dynamic-type request = %#v", request)
	}
}

func TestGeneratedArtifactDomainsArePinned(t *testing.T) {
	if GeneratedArtifactAnonymousStruct != 1 ||
		GeneratedArtifactMapSpecialization != 2 ||
		GeneratedArtifactInterfaceAdapter != 3 ||
		GeneratedArtifactAnonymousInterface != 4 ||
		GeneratedArtifactInterfaceMethodToken != 5 ||
		GeneratedArtifactInterfaceDynamicTypeToken != 6 ||
		GeneratedArtifactGenericCapability != 7 ||
		GeneratedArtifactCallableABI != 8 ||
		GeneratedArtifactInterfaceMethodCallable != 9 ||
		GeneratedArtifactProviderInterfaceBridge != 11 ||
		GeneratedArtifactProviderStatefulRepresentation != 12 ||
		GeneratedArtifactDeferredCallableRegistry != 13 ||
		GeneratedArtifactGenericConcretization != 14 ||
		GeneratedArtifactReflectionType != 15 ||
		GeneratedArtifactInvalid.Valid() ||
		GeneratedArtifactKind(10).Valid() ||
		GeneratedArtifactKind(16).Valid() ||
		GeneratedArtifactKind(17).Valid() {
		t.Fatal("generated-artifact kind IDs drifted")
	}
	if GeneratedArtifactPlacementCompilation != 1 ||
		GeneratedArtifactPlacementLexical != 2 ||
		GeneratedArtifactPlacementContract != 3 ||
		GeneratedArtifactPlacementInvalid.Valid() ||
		GeneratedArtifactPlacement(4).Valid() {
		t.Fatal("generated-artifact placement IDs drifted")
	}
	if AnonymousStructDemandDefinition != 1 ||
		AnonymousStructDemandZero != 2 ||
		AnonymousStructDemandCopy != 3 ||
		AnonymousStructDemandEqual != 4 ||
		AnonymousStructDemandHash != 5 ||
		AnonymousStructDemandConvert != 6 ||
		AnonymousStructDemandStorage != 7 ||
		AnonymousStructDemandInvalid.Valid() ||
		AnonymousStructDemand(8).Valid() {
		t.Fatal("anonymous-struct demand IDs drifted")
	}
	if MapSpecializationDemandDefinition != 1 ||
		MapSpecializationDemandStatic != 2 ||
		MapSpecializationDemandInvalid.Valid() ||
		MapSpecializationDemand(3).Valid() {
		t.Fatal("map-specialization demand IDs drifted")
	}
}

func TestCallableABIIsContractOnlyCompilationSupport(t *testing.T) {
	signature := types.NewSignatureType(
		nil,
		nil,
		nil,
		types.NewTuple(types.NewVar(
			token.NoPos,
			nil,
			"value",
			types.Typ[types.Int],
		)),
		types.NewTuple(),
		false,
	)
	artifact, err := NewContractGeneratedArtifact(
		GeneratedArtifactCallableABI,
		signature,
		"callable-signature",
		"$goCallable$signature",
	)
	if err != nil {
		t.Fatal(err)
	}
	request, err := NewCallableABIRequest(artifact)
	if err != nil {
		t.Fatal(err)
	}
	if !artifact.Valid() ||
		artifact.Placement() != GeneratedArtifactPlacementContract ||
		artifact.OutputPath() != "" ||
		request.LegalScope() != ScopeCompilationSupport {
		t.Fatalf("contract-only callable ABI = %#v / %#v", artifact, request)
	}
	if _, err := NewCompilationGeneratedArtifact(
		GeneratedArtifactCallableABI,
		signature,
		"callable-signature",
		"$goCallable$signature",
		"support/callable.ts",
	); err == nil {
		t.Fatal("callable ABI accepted a materialized support file")
	}
}

func TestInterfaceMethodCallableIsContractOnlyCompilationSupport(t *testing.T) {
	signature := types.NewSignatureType(
		nil,
		nil,
		nil,
		types.NewTuple(),
		types.NewTuple(types.NewVar(
			token.NoPos,
			nil,
			"",
			types.Typ[types.Int32],
		)),
		false,
	)
	artifact, err := NewContractGeneratedArtifact(
		GeneratedArtifactInterfaceMethodCallable,
		signature,
		"interface-method",
		"$goInterfaceCallable$method",
	)
	if err != nil {
		t.Fatal(err)
	}
	request, err := NewInterfaceMethodCallableRequest(artifact)
	if err != nil {
		t.Fatal(err)
	}
	requirement, ok := request.DeclarationRequirement()
	selected, selectedOK := requirement.InterfaceMethodCallable()
	if !artifact.Valid() ||
		artifact.Placement() != GeneratedArtifactPlacementContract ||
		artifact.OutputPath() != "" ||
		request.LegalScope() != ScopeCompilationSupport ||
		!ok ||
		!selectedOK ||
		selected != artifact {
		t.Fatalf(
			"contract-only interface callable = %#v / %#v",
			artifact,
			request,
		)
	}
	parameter := types.NewTypeParam(
		types.NewTypeName(token.NoPos, nil, "T", nil),
		types.NewInterfaceType(nil, nil).Complete(),
	)
	openSignature := types.NewSignatureType(
		nil,
		nil,
		nil,
		types.NewTuple(),
		types.NewTuple(types.NewVar(token.NoPos, nil, "", parameter)),
		false,
	)
	if _, err := NewContractGeneratedArtifact(
		GeneratedArtifactInterfaceMethodCallable,
		openSignature,
		"open-interface-method",
		"$goInterfaceCallable$open",
	); err != nil {
		t.Fatalf("open callable family rejected: %v", err)
	}
	if _, err := NewCompilationInterfaceMethodTokenArtifact(
		openSignature,
		"open-interface-method",
		"$goInterfaceMethod$open",
		"support/interface-methods.ts",
		RuntimeInvalid,
	); err == nil {
		t.Fatal("open interface method received a runtime token")
	}
}

func TestPackageInitializerOwnerAcceptsCheckerBlankTarget(t *testing.T) {
	sourcePackage := types.NewPackage("example.com/source", "source")
	blank := types.NewVar(
		token.Pos(10),
		sourcePackage,
		"_",
		types.Typ[types.Int32],
	)
	initializer := &types.Initializer{
		Lhs: []*types.Var{blank},
		Rhs: &ast.BasicLit{
			ValuePos: token.Pos(20),
			Kind:     token.INT,
			Value:    "1",
		},
	}
	owner, err := PackageInitializerArtifactOwner(
		sourcePackage,
		initializer,
	)
	if err != nil {
		t.Fatal(err)
	}
	selectedPackage, selectedInitializer, ok := owner.PackageInitializer()
	if !ok ||
		selectedPackage != sourcePackage ||
		selectedInitializer != initializer {
		t.Fatalf("package initializer owner = %#v", owner)
	}
	facet, err := NewPackageInitializerCallableFacet(owner)
	if err != nil {
		t.Fatal(err)
	}
	selectedOwner, selected := facet.PackageInitializer()
	if !selected ||
		selectedOwner != owner ||
		facet.Kind() != CallableFacetPackageInitializer ||
		facet.Owner() != owner {
		t.Fatalf("package initializer callable facet = %#v", facet)
	}
}

func TestPackageInitializerOwnerRejectsForeignNonblankTarget(t *testing.T) {
	sourcePackage := types.NewPackage("example.com/source", "source")
	foreignPackage := types.NewPackage("example.com/foreign", "foreign")
	foreign := types.NewVar(
		token.Pos(10),
		foreignPackage,
		"Value",
		types.Typ[types.Int32],
	)
	if existing := foreignPackage.Scope().Insert(foreign); existing != nil {
		t.Fatal("foreign target insertion failed")
	}
	initializer := &types.Initializer{
		Lhs: []*types.Var{foreign},
		Rhs: &ast.BasicLit{
			ValuePos: token.Pos(20),
			Kind:     token.INT,
			Value:    "1",
		},
	}
	if _, err := PackageInitializerArtifactOwner(
		sourcePackage,
		initializer,
	); err == nil {
		t.Fatal("package initializer accepted a foreign nonblank target")
	}
}
