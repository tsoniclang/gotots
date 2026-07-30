package api_test

import (
	"go/ast"
	"go/token"
	"go/types"
	"testing"

	. "github.com/tsoniclang/gotots/internal/emit/api"
)

func TestGeneratedCallableABIBoundaryOwnsNestedCooperativeDemand(
	t *testing.T,
) {
	sourcePackage := types.NewPackage("example.com/callback", "callback")
	sourceOwner := types.NewFunc(
		token.Pos(1),
		sourcePackage,
		"Consume",
		types.NewSignatureType(nil, nil, nil, nil, nil, false),
	)
	signature := types.NewSignatureType(
		nil,
		nil,
		nil,
		types.NewTuple(
			types.NewVar(token.NoPos, nil, "value", types.Typ[types.Int32]),
		),
		types.NewTuple(
			types.NewVar(token.NoPos, nil, "", types.Typ[types.Bool]),
		),
		false,
	)
	artifact, err := NewContractGeneratedArtifact(
		GeneratedArtifactCallableABI,
		signature,
		"callback-boundary",
		"$goCallable_callback_boundary",
	)
	if err != nil {
		t.Fatal(err)
	}
	facet, err := NewCallableABIFacet(artifact)
	if err != nil {
		t.Fatal(err)
	}
	context := (Context{}).
		WithArtifactOwner(MustSourceArtifactOwner(sourceOwner)).
		WithCooperativeCallableABI(facet, true)
	if !context.IsCooperative() {
		t.Fatal("generated callable ABI boundary lost cooperative state")
	}
	request, err := context.CooperativeRequest()
	if err != nil {
		t.Fatal(err)
	}
	requirement, ok := request.DeclarationRequirement()
	selected, cooperative := requirement.CooperativeCallable()
	if !ok ||
		!cooperative ||
		selected != facet ||
		requirement.Owner() != MustGeneratedArtifactOwner(artifact) {
		t.Fatalf("nested cooperative request = %#v", request)
	}
	sourceFacet, err := NewSourceCallableFacet(sourceOwner)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if recover() == nil {
			t.Fatal("generated callable ABI boundary accepted a source facet")
		}
	}()
	_ = context.WithCooperativeCallableABI(sourceFacet, false)
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
	requirement, err := NewCooperativeCallableRequirement(facet)
	if err != nil {
		t.Fatal(err)
	}
	if facet.Owner() != artifact.ReconstructionOwner() ||
		requirement.Owner() != artifact.ReconstructionOwner() {
		t.Fatalf(
			"lexical capability owners = facet %#v, requirement %#v, want reconstruction %#v",
			facet.Owner(),
			requirement.Owner(),
			artifact.ReconstructionOwner(),
		)
	}
	selected, ok := requirement.LexicalGeneratedArtifact()
	if !ok || selected != artifact {
		t.Fatalf("lexical cooperative requirement = %#v", requirement)
	}
}

func TestInterfaceDynamicTypeRequestCarriesExactGoType(t *testing.T) {
	sourceType := types.Typ[types.Int32]
	artifact, err := NewCompilationGeneratedArtifact(
		GeneratedArtifactInterfaceDynamicTypeToken,
		sourceType,
		"artifact",
		"$goDynamicType_artifact",
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
		GeneratedArtifactInvalid.Valid() ||
		GeneratedArtifactKind(10).Valid() {
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
		"$goCallable_signature",
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
		"$goCallable_signature",
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
		"$goInterfaceCallable_method",
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
		"$goInterfaceCallable_open",
	); err != nil {
		t.Fatalf("open callable family rejected: %v", err)
	}
	if _, err := NewCompilationInterfaceMethodTokenArtifact(
		openSignature,
		"open-interface-method",
		"$goInterfaceMethod_open",
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
	request, err := NewCooperativeCallableRequest(facet)
	if err != nil {
		t.Fatal(err)
	}
	requirement, ok := request.DeclarationRequirement()
	selectedFacet, cooperative := requirement.CooperativeCallable()
	if !ok || !cooperative || selectedFacet != facet {
		t.Fatalf("package initializer cooperative request = %#v", request)
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
