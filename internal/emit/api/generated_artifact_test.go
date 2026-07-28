package api

import (
	"go/token"
	"go/types"
	"testing"
)

func TestAnonymousStructRequestCarriesExactGeneratedArtifact(t *testing.T) {
	sourceType := types.NewStruct(
		[]*types.Var{types.NewField(0, nil, "Value", types.Typ[types.Int32], false)},
		nil,
	)
	artifact, err := NewCompilationGeneratedArtifact(
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

func TestGeneratedArtifactRejectsStringOnlyIdentity(t *testing.T) {
	if _, err := NewCompilationGeneratedArtifact(
		nil,
		"artifact",
		"$goStruct_artifact",
		"support/anonymous-structs.ts",
	); err == nil {
		t.Fatal("generated artifact accepted a key without an exact Go type")
	}
}
