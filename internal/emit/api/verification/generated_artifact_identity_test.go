package api_test

import (
	"go/ast"
	"go/token"
	"go/types"
	"strings"
	"testing"

	. "github.com/tsoniclang/gotots/internal/emit/api"
)

func TestMissingCooperativeFacetDiagnosticNamesOwnerAndRole(t *testing.T) {
	sourcePackage := types.NewPackage("example.com/diagnostic", "diagnostic")
	owner := types.NewFunc(
		token.Pos(1),
		sourcePackage,
		"Run",
		types.NewSignatureType(nil, nil, nil, nil, nil, false),
	)
	context := (Context{}).
		WithArtifactOwner(MustSourceArtifactOwner(owner)).
		WithRole(RoleFunctionBody)
	_, err := context.CooperativeRequest()
	if err == nil ||
		!strings.Contains(err.Error(), "Run") ||
		!strings.Contains(err.Error(), string(RoleFunctionBody)) {
		t.Fatalf("missing-facet diagnostic = %v", err)
	}
}

func TestAnonymousStructRequestCarriesExactGeneratedArtifact(t *testing.T) {
	sourceType := types.NewStruct(
		[]*types.Var{types.NewField(0, nil, "Value", types.Typ[types.Int32], false)},
		nil,
	)
	artifact, err := NewCompilationGeneratedArtifact(
		GeneratedArtifactAnonymousStruct,
		sourceType,
		"artifact",
		"$goStruct$artifact",
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
		"$goStruct$artifact",
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
		"$goMap$artifact",
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
		"$goStruct$artifact",
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
		CallableFacetPackageInitializer != 6 ||
		CallableFacetKind(7).Valid() ||
		CallableFacetInterfaceMethod != 8 ||
		CallableFacetInvalid.Valid() ||
		CallableFacetKind(9).Valid() {
		t.Fatal("callable-facet kind IDs drifted")
	}
}
