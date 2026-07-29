package api_test

import (
	"go/ast"
	"go/token"
	"go/types"
	"testing"

	. "github.com/tsoniclang/gotots/internal/emit/api"
)

func TestAnonymousStructRequestCarriesExactGeneratedArtifact(t *testing.T) {
	sourceType := types.NewStruct(
		[]*types.Var{types.NewField(0, nil, "Value", types.Typ[types.Int32], false)},
		nil,
	)
	artifact, err := NewCompilationGeneratedArtifact(
		GeneratedArtifactAnonymousStruct,
		sourceType,
		"artifact",
		"$goStruct_artifact",
		"support/anonymous-structs.ts",
	)
	if err != nil {
		t.Fatal(err)
	}
	request, err := NewAnonymousStructRequest(
		artifact,
		AnonymousStructDemandCopy,
	)
	if err != nil {
		t.Fatal(err)
	}
	requirement, ok := request.DeclarationRequirement()
	requirementArtifact, demand, anonymous := requirement.AnonymousStruct()
	ownerArtifact, generatedOwned := requirement.Owner().Generated()
	if !ok ||
		!anonymous ||
		!generatedOwned ||
		requirementArtifact != artifact ||
		ownerArtifact != artifact ||
		requirementArtifact.SourceType() != sourceType ||
		demand != AnonymousStructDemandCopy {
		t.Fatalf("anonymous-struct requirement = %#v, %t", requirement, ok)
	}
	if request.Kind() != RootRequestDeclarationRequirement ||
		request.LegalScope() != ScopeCompilationSupport ||
		request.PreferredScope() != ScopeCompilationSupport ||
		request.Execution() != ExecutionStatic {
		t.Fatalf(
			"anonymous-struct request contract = kind %d, scope %d/%d, execution %d",
			request.Kind(),
			request.LegalScope(),
			request.PreferredScope(),
			request.Execution(),
		)
	}
	if _, fabricated := requirement.Owner().Source(); fabricated {
		t.Fatal("generated artifact fabricated a source go/types object")
	}
}

func TestLexicalAnonymousStructRequirementReconstructsSourceOwner(t *testing.T) {
	sourcePackage := types.NewPackage("example.com/local", "local")
	owner := types.NewVar(
		token.Pos(10),
		sourcePackage,
		"Value",
		types.Typ[types.Int32],
	)
	scope := types.NewScope(
		sourcePackage.Scope(),
		token.Pos(20),
		token.Pos(100),
		"initializer",
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
	sourceType := types.NewStruct(
		[]*types.Var{types.NewField(0, sourcePackage, "Value", anchor.Type(), false)},
		nil,
	)
	artifact, err := NewLexicalGeneratedArtifact(
		GeneratedArtifactAnonymousStruct,
		sourceType,
		"artifact",
		"$goStruct_artifact",
		MustSourceArtifactOwner(owner),
		anchor,
	)
	if err != nil {
		t.Fatal(err)
	}
	requirement, err := NewAnonymousStructRequirement(
		artifact,
		AnonymousStructDemandDefinition,
	)
	if err != nil {
		t.Fatal(err)
	}
	sourceOwner, sourceOwned := requirement.Owner().Source()
	if !sourceOwned ||
		sourceOwner != owner ||
		requirement.Owner() != artifact.ReconstructionOwner() {
		t.Fatalf("lexical reconstruction owner = %#v", requirement.Owner())
	}
}

func TestLexicalGeneratedArtifactReconstructsExactPackageInitializer(
	t *testing.T,
) {
	sourcePackage := types.NewPackage("example.com/local", "local")
	target := types.NewVar(
		token.Pos(10),
		sourcePackage,
		"Value",
		types.Typ[types.Int32],
	)
	if existing := sourcePackage.Scope().Insert(target); existing != nil {
		t.Fatal("package target insertion failed")
	}
	initializer := &types.Initializer{
		Lhs: []*types.Var{target},
		Rhs: &ast.BasicLit{
			ValuePos: token.Pos(20),
			Kind:     token.INT,
			Value:    "1",
		},
	}
	scope := types.NewScope(
		sourcePackage.Scope(),
		token.Pos(20),
		token.Pos(100),
		"initializer",
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
	sourceType := types.NewMap(anchor.Type(), types.Typ[types.Int32])
	initializerOwner := MustPackageInitializerArtifactOwner(
		sourcePackage,
		initializer,
	)
	artifact, err := NewLexicalGeneratedArtifact(
		GeneratedArtifactMapSpecialization,
		sourceType,
		"artifact",
		"$goMap_artifact",
		initializerOwner,
		anchor,
	)
	if err != nil {
		t.Fatal(err)
	}
	requirement, err := NewMapSpecializationRequirement(
		artifact,
		MapSpecializationDemandDefinition,
	)
	if err != nil {
		t.Fatal(err)
	}
	selectedPackage, selectedInitializer, initializerOwned :=
		requirement.Owner().PackageInitializer()
	if !initializerOwned ||
		selectedPackage != sourcePackage ||
		selectedInitializer != initializer ||
		requirement.Owner() != artifact.ReconstructionOwner() {
		t.Fatalf("lexical reconstruction owner = %#v", requirement.Owner())
	}
	if _, sourceOwned := requirement.Owner().Source(); sourceOwned {
		t.Fatal("package initializer fabricated a variable source owner")
	}
}

func TestGeneratedArtifactRejectsStringOnlyIdentity(t *testing.T) {
	if _, err := NewCompilationGeneratedArtifact(
		GeneratedArtifactAnonymousStruct,
		nil,
		"artifact",
		"$goStruct_artifact",
		"support/anonymous-structs.ts",
	); err == nil {
		t.Fatal("generated artifact accepted a key without an exact Go type")
	}
}

func TestGenericCapabilityUsesItsExactValidatingConstructor(t *testing.T) {
	signature := types.NewSignatureType(
		nil,
		nil,
		nil,
		types.NewTuple(
			types.NewVar(token.NoPos, nil, "left", types.Typ[types.Int32]),
			types.NewVar(token.NoPos, nil, "right", types.Typ[types.Int32]),
		),
		types.NewTuple(
			types.NewVar(token.NoPos, nil, "", types.Typ[types.Int32]),
		),
		false,
	)
	if _, err := NewCompilationGeneratedArtifact(
		GeneratedArtifactGenericCapability,
		signature,
		"artifact",
		"$goCapability_artifact",
		"support/generics/capabilities/artifact.ts",
	); err == nil {
		t.Fatal("generic constructor accepted a capability without its operation")
	}
	selection, err := SelectGenericOperation(GenericOperationBinaryAdd)
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := NewCompilationGenericCapabilityArtifact(
		selection,
		signature,
		"artifact",
		"$goCapability_artifact",
		"support/generics/capabilities/artifact.ts",
	)
	if err != nil {
		t.Fatal(err)
	}
	selected, operation, ok := artifact.GenericCapability()
	if !artifact.Valid() ||
		!ok ||
		operation != selection ||
		!types.Identical(selected, signature) {
		t.Fatalf("generic-capability artifact = %#v", artifact)
	}
	request, err := NewGenericCapabilityRequest(artifact)
	if err != nil {
		t.Fatal(err)
	}
	requirement, ok := request.DeclarationRequirement()
	selectedArtifact, capability := requirement.GenericCapability()
	if !ok || !capability || selectedArtifact != artifact {
		t.Fatalf("generic-capability requirement = %#v", requirement)
	}
}

func TestGenericCooperativeFacetsCarryExactCallableIdentity(t *testing.T) {
	valueSignature := types.NewSignatureType(
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
		valueSignature,
		"copy-int32",
		"$goCapability_copy_int32",
		"support/generics/capabilities/copy-int32.ts",
	)
	if err != nil {
		t.Fatal(err)
	}
	capabilityFacet, err := NewGenericCapabilityCallableFacet(artifact)
	if err != nil {
		t.Fatal(err)
	}
	selectedArtifact, capability := capabilityFacet.GenericCapability()
	if !capabilityFacet.Valid() ||
		capabilityFacet.Kind() != CallableFacetGenericCapability ||
		!capability ||
		selectedArtifact != artifact ||
		capabilityFacet.Owner() != MustGeneratedArtifactOwner(artifact) {
		t.Fatalf("generic-capability facet = %#v", capabilityFacet)
	}
	reference, err := NewGenericCapabilityReference(
		artifact,
		artifact.TargetName(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if reference.Artifact() != artifact ||
		reference.Name() != artifact.TargetName() {
		t.Fatalf("generic-capability reference = %#v", reference)
	}
	if _, err := NewGenericCapabilityReference(
		artifact,
		"$foreign",
	); err == nil {
		t.Fatal("generic-capability reference accepted a foreign target name")
	}

	constraint := types.NewInterfaceType(nil, nil)
	constraint.Complete()
	parameter := types.NewTypeParam(
		types.NewTypeName(token.NoPos, nil, "T", nil),
		constraint,
	)
	operationSignature := types.NewSignatureType(
		nil,
		nil,
		nil,
		types.NewTuple(
			types.NewVar(token.NoPos, nil, "value", parameter),
		),
		types.NewTuple(
			types.NewVar(token.NoPos, nil, "", parameter),
		),
		false,
	)
	owner := types.NewFunc(
		token.NoPos,
		nil,
		"Copy",
		types.NewSignatureType(
			nil,
			nil,
			[]*types.TypeParam{parameter},
			operationSignature.Params(),
			operationSignature.Results(),
			false,
		),
	)
	operation, err := NewGenericOperationContract(
		owner,
		"function|copy|(T)->T",
		"$go$copy_operation",
		GenericFunctionOperationConsumer(),
		selection,
		operationSignature,
	)
	if err != nil {
		t.Fatal(err)
	}
	operationFacet, err := NewGenericOperationCallableFacet(operation)
	if err != nil {
		t.Fatal(err)
	}
	selectedOperation, selected := operationFacet.GenericOperation()
	if !operationFacet.Valid() ||
		operationFacet.Kind() != CallableFacetGenericOperation ||
		!selected ||
		selectedOperation != operation ||
		operationFacet.Owner() != MustSourceArtifactOwner(owner) {
		t.Fatalf("generic-operation facet = %#v", operationFacet)
	}
	operationReference, err := NewGenericOperationReference(
		operation,
		operation.TargetName(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if operationReference.Contract() != operation ||
		operationReference.Name() != operation.TargetName() {
		t.Fatalf("generic-operation reference = %#v", operationReference)
	}
	for _, facet := range []CallableFacet{
		capabilityFacet,
		operationFacet,
	} {
		request, requestErr := NewCooperativeCallableRequest(facet)
		if requestErr != nil {
			t.Fatal(requestErr)
		}
		requirement, ok := request.DeclarationRequirement()
		selectedFacet, cooperative := requirement.CooperativeCallable()
		if !ok || !cooperative || selectedFacet != facet {
			t.Fatalf("cooperative facet request = %#v", request)
		}
	}
	if CallableFacetSource != 1 ||
		CallableFacetFunctionLiteral != 2 ||
		CallableFacetABI != 3 ||
		CallableFacetGenericCapability != 4 ||
		CallableFacetGenericOperation != 5 ||
		CallableFacetInvalid.Valid() ||
		CallableFacetKind(6).Valid() {
		t.Fatal("callable-facet kind IDs drifted")
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
		GeneratedArtifactInvalid.Valid() ||
		GeneratedArtifactKind(9).Valid() {
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
