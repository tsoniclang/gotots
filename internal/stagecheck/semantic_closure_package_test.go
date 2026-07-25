package stagecheck

import (
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
	"github.com/tsoniclang/gotots/internal/language/semantic"
)

func TestSemanticClosureRejectsMissingExternalDeclarationOwner(
	t *testing.T,
) {
	pkg, externalPackage, externalDeclaration :=
		semanticExternalDeclarationFixture(t)
	memberTargets, err := pkg.MemberTargetCensus()
	if err != nil {
		t.Fatal(err)
	}
	owners := &semanticOwnerCensus{
		definitions:  map[identity.DefinitionID]identity.PackageID{},
		declarations: map[identity.SemanticDeclarationID]identity.PackageID{},
		memberCounts: map[identity.PackageID]int{
			pkg.ID(): memberTargets.Count(),
		},
		memberDigests: map[identity.PackageID]string{
			pkg.ID(): memberTargets.Digest(),
		},
	}
	if err := pkg.VisitDefinitions(func(
		record semantic.DefinitionSemantics,
	) error {
		owners.definitions[record.Definition()] = pkg.ID()
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := pkg.VisitDeclarations(func(
		record semantic.Declaration,
	) error {
		owners.declarations[record.ID()] = pkg.ID()
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	err = verifySemanticPackageClosure(pkg, owners)
	if err == nil ||
		!strings.Contains(
			err.Error(),
			"references absent semantic target "+
				externalDeclaration.String(),
		) {
		t.Fatalf("missing external declaration owner error=%v", err)
	}
	owners.declarations[externalDeclaration] = externalPackage
	if err := verifySemanticPackageClosure(pkg, owners); err != nil {
		t.Fatalf("complete declaration owner census rejected: %v", err)
	}
}

func semanticExternalDeclarationFixture(
	t *testing.T,
) (
	semantic.Package,
	identity.PackageID,
	identity.SemanticDeclarationID,
) {
	t.Helper()
	module, err := identity.NewModuleID("example.com/closure", "")
	if err != nil {
		t.Fatal(err)
	}
	owner, err := identity.NewModuleOwner(module)
	if err != nil {
		t.Fatal(err)
	}
	localPackage, err := identity.NewPackageID(
		owner, "example.com/closure/local",
	)
	if err != nil {
		t.Fatal(err)
	}
	externalPackage, err := identity.NewPackageID(
		owner, "example.com/closure/external",
	)
	if err != nil {
		t.Fatal(err)
	}
	file, err := identity.NewFileID(owner, "local/local.go")
	if err != nil {
		t.Fatal(err)
	}
	rootSpan, _ := identity.NewSpanID(file, 0, 40)
	root, _ := identity.NewOccurrenceID(
		rootSpan, uint16(catalog.KindFuncDecl),
	)
	operationSpan, _ := identity.NewSpanID(file, 20, 28)
	occurrence, _ := identity.NewOccurrenceID(
		operationSpan, uint16(catalog.KindIdent),
	)
	definition, _ := identity.NewSourceDefinitionID(
		root, identity.DefinitionFuncDecl,
	)
	operationID, _ := identity.NewOperationID(
		definition, occurrence,
	)
	authority, err := semantic.NewCheckerAuthority(
		strings.Repeat("ab", 32),
		strings.Repeat("bc", 32),
		strings.Repeat("cd", 32),
		strings.Repeat("de", 32),
		strings.Repeat("ef", 32),
	)
	if err != nil {
		t.Fatal(err)
	}
	basic, err := semantic.NewType(semantic.TypeSpec{
		Kind:  semantic.TypeBasic,
		Basic: semantic.BasicInt,
	})
	if err != nil {
		t.Fatal(err)
	}
	signature, err := semantic.NewType(semantic.TypeSpec{
		Kind: semantic.TypeSignature,
		Signature: semantic.Signature{
			Results: []identity.SemanticTypeID{basic.ID()},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	localDeclaration, err := identity.NewPackageDeclarationID(
		localPackage,
		identity.SemanticObjectFunction,
		"UseExternal",
	)
	if err != nil {
		t.Fatal(err)
	}
	externalDeclaration, err := identity.NewPackageDeclarationID(
		externalPackage,
		identity.SemanticObjectVariable,
		"Value",
	)
	if err != nil {
		t.Fatal(err)
	}
	declaration, err := semantic.NewDeclaration(
		localDeclaration,
		localPackage,
		identity.SemanticObjectFunction,
		"UseExternal",
		signature.ID(),
		true,
		semantic.Constant{},
		authority,
	)
	if err != nil {
		t.Fatal(err)
	}
	definitionRecord, err := semantic.NewDefinitionSemantics(
		semantic.DefinitionSemanticsSpec{
			Definition:   definition,
			Package:      localPackage,
			Form:         semantic.DefinitionFormCallable,
			Authority:    authority,
			Name:         "UseExternal",
			Declarations: []identity.SemanticDeclarationID{localDeclaration},
			Signature:    signature.ID(),
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	object, err := semantic.DeclarationReference(externalDeclaration)
	if err != nil {
		t.Fatal(err)
	}
	operation, err := semantic.NewOperation(semantic.OperationSpec{
		ID:         operationID,
		Kind:       semantic.OperationLoad,
		Syntax:     catalog.KindIdent,
		Variant:    catalog.VariantNone,
		Role:       catalog.RoleReturnValue,
		Token:      catalog.TokenIDENT,
		Mode:       semantic.ValueModeValue,
		Arity:      semantic.ResultArityOne,
		Place:      semantic.PlaceNone,
		ResultType: basic.ID(),
		Object:     object,
	})
	if err != nil {
		t.Fatal(err)
	}
	resolution, err := semantic.NewOccurrenceResolution(
		semantic.ResolutionSpec{
			Occurrence: occurrence,
			Owner:      definition,
			Syntax:     catalog.KindIdent,
			Role:       catalog.RoleReturnValue,
			Variant:    catalog.VariantNone,
			Domain:     catalog.ResolutionDomainExecutable,
			Kind:       semantic.ResolutionOperation,
			Operation:  operationID,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	witness := func(typeID identity.SemanticTypeID) semantic.TypeWitness {
		record, witnessErr := semantic.NewTypeWitness(
			localPackage, typeID, authority,
		)
		if witnessErr != nil {
			t.Fatal(witnessErr)
		}
		return record
	}
	pkg, err := semantic.NewPackage(semantic.PackageInput{
		ID:           localPackage,
		Provenance:   semantic.ProvenanceWorkspaceModule,
		Definitions:  []semantic.DefinitionSemantics{definitionRecord},
		Resolutions:  []semantic.OccurrenceResolution{resolution},
		Declarations: []semantic.Declaration{declaration},
		Types:        []semantic.Type{basic, signature},
		TypeWitnesses: []semantic.TypeWitness{
			witness(basic.ID()),
			witness(signature.ID()),
		},
		Operations: []semantic.Operation{operation},
	})
	if err != nil {
		t.Fatal(err)
	}
	return pkg, externalPackage, externalDeclaration
}
