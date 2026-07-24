package semantic

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
)

func TestSemanticMetricsKeepIndependentBoundedTails(t *testing.T) {
	first := semanticFixture(t).pkg
	module, err := identity.NewModuleID(
		"example.com/semantic", "",
	)
	if err != nil {
		t.Fatal(err)
	}
	owner, err := identity.NewModuleOwner(module)
	if err != nil {
		t.Fatal(err)
	}
	second, err := identity.NewPackageID(
		owner, "example.com/semantic/second",
	)
	if err != nil {
		t.Fatal(err)
	}
	metrics := Metrics{
		packages: 1, definitions: 25, operations: 25, types: 25,
		encodedBytes: 1000, largestRecords: 75,
		packageTail: []PackageSize{{
			Package: first, EncodedBytes: 1000, Records: 75,
		}},
	}
	for index := 0; index < 25; index++ {
		metrics.definitionTail = append(
			metrics.definitionTail, RecordSize{
				Package:      first,
				Identity:     fmt.Sprintf("definition-%02d", index),
				EncodedBytes: int64(index + 1),
			},
		)
		metrics.operationTail = append(
			metrics.operationTail, RecordSize{
				Package:      first,
				Identity:     fmt.Sprintf("operation-%02d", index),
				EncodedBytes: int64(index + 1),
			},
		)
		metrics.typeTail = append(
			metrics.typeTail, RecordSize{
				Package:      first,
				Identity:     fmt.Sprintf("type-%02d", index),
				EncodedBytes: int64(index + 1),
			},
		)
	}
	metrics.trim()
	secondMetrics := Metrics{
		packages: 1, resolutions: 80,
		encodedBytes: 100, largestRecords: 80,
		packageTail: []PackageSize{{
			Package: second, EncodedBytes: 100, Records: 80,
		}},
	}
	metrics.merge(secondMetrics)
	if metrics.Packages() != 2 ||
		metrics.Definitions() != 25 ||
		metrics.Resolutions() != 80 ||
		metrics.Operations() != 25 ||
		metrics.Types() != 25 ||
		metrics.EncodedBytes() != 1100 ||
		metrics.LargestPackageRecords() != 80 {
		t.Fatalf("semantic metrics totals = %+v", metrics)
	}
	if packages := metrics.LargestPackages(); len(packages) != 2 ||
		packages[0].Package != first ||
		packages[1].Package != second {
		t.Fatalf("semantic package tail = %+v", packages)
	}
	for name, records := range map[string][]RecordSize{
		"definitions": metrics.LargestDefinitions(),
		"operations":  metrics.LargestOperations(),
		"types":       metrics.LargestTypes(),
	} {
		if len(records) != semanticTailLimit {
			t.Fatalf(
				"%s tail=%d, want %d",
				name, len(records), semanticTailLimit,
			)
		}
	}
	copyTail := metrics.LargestDefinitions()
	copyTail[0] = RecordSize{}
	if metrics.LargestDefinitions()[0].Identity == "" {
		t.Fatal("semantic metrics exposed mutable tail storage")
	}
}

func TestMeasurePackagesRejectsDuplicateIdentity(t *testing.T) {
	pkg := Package{id: semanticFixture(t).pkg}
	if _, err := MeasurePackages([]Package{pkg, pkg}); err == nil {
		t.Fatal("semantic metrics accepted a duplicate package identity")
	}
}

func TestProviderManifestMetricsDoNotRequireRecordPayload(t *testing.T) {
	pkg := semanticFixture(t).pkg
	metrics, err := measureShardManifest([]packageShardManifest{{
		Package: pkg.String(), ShardBytes: 4096,
		DefinitionCount: 3, ResolutionCount: 17,
		DeclarationCount: 4, BindingCount: 5,
		TypeCount: 6, OperationCount: 7, UnsupportedCount: 2,
	}})
	if err != nil {
		t.Fatal(err)
	}
	if metrics.Packages() != 1 ||
		metrics.Definitions() != 3 ||
		metrics.Resolutions() != 17 ||
		metrics.Declarations() != 4 ||
		metrics.Bindings() != 5 ||
		metrics.Types() != 6 ||
		metrics.Operations() != 7 ||
		metrics.Unsupported() != 2 ||
		metrics.EncodedBytes() != 4096 ||
		metrics.LargestPackageRecords() != 44 {
		t.Fatalf("provider manifest metrics = %+v", metrics)
	}
	tail := metrics.LargestPackages()
	if len(tail) != 1 ||
		tail[0].Package != pkg ||
		tail[0].EncodedBytes != 4096 ||
		tail[0].Records != 44 ||
		len(metrics.LargestDefinitions()) != 0 ||
		len(metrics.LargestOperations()) != 0 ||
		len(metrics.LargestTypes()) != 0 {
		t.Fatalf("provider manifest metric tails = %+v", metrics)
	}
}

func TestSemanticShardWriterOwnsExactBoundedRecordTails(
	t *testing.T,
) {
	fixture := semanticFixture(t)
	operationID, err := identity.NewOperationID(
		fixture.definition, fixture.body,
	)
	if err != nil {
		t.Fatal(err)
	}
	basic, err := NewType(TypeSpec{
		Kind:  TypeBasic,
		Basic: BasicInt,
	})
	if err != nil {
		t.Fatal(err)
	}
	signature, err := NewType(TypeSpec{
		Kind: TypeSignature,
		Signature: Signature{
			Results: []identity.SemanticTypeID{basic.ID()},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	declarationID, err := identity.NewPackageDeclarationID(
		fixture.pkg, identity.SemanticObjectFunction, "F",
	)
	if err != nil {
		t.Fatal(err)
	}
	declaration, err := NewDeclaration(
		declarationID,
		fixture.pkg,
		identity.SemanticObjectFunction,
		"F",
		signature.ID(),
		true,
		Constant{},
		fixture.authority,
	)
	if err != nil {
		t.Fatal(err)
	}
	definition, err := NewDefinitionSemantics(
		DefinitionSemanticsSpec{
			Definition: fixture.definition,
			Package:    fixture.pkg,
			Form:       DefinitionFormCallable,
			Authority:  fixture.authority,
			Name:       "F",
			Declarations: []identity.SemanticDeclarationID{
				declarationID,
			},
			Signature: signature.ID(),
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	operation, err := NewOperation(OperationSpec{
		ID:         operationID,
		Kind:       OperationLiteral,
		Syntax:     catalog.KindBasicLit,
		Variant:    catalog.VariantNone,
		Role:       catalog.RoleReturnValue,
		Token:      catalog.TokenINT,
		Mode:       ValueModeValue,
		Arity:      ResultArityOne,
		Place:      PlaceNone,
		ResultType: basic.ID(),
		Object:     NoObjectReference(),
	})
	if err != nil {
		t.Fatal(err)
	}
	resolution, err := NewOccurrenceResolution(ResolutionSpec{
		Occurrence: fixture.body,
		Owner:      fixture.definition,
		Syntax:     catalog.KindBasicLit,
		Role:       catalog.RoleReturnValue,
		Variant:    catalog.VariantNone,
		Domain:     catalog.ResolutionDomainExecutable,
		Kind:       ResolutionOperation,
		Operation:  operationID,
	})
	if err != nil {
		t.Fatal(err)
	}
	pkg, err := NewPackage(PackageInput{
		ID:           fixture.pkg,
		Provenance:   ProvenanceWorkspaceModule,
		Definitions:  []DefinitionSemantics{definition},
		Resolutions:  []OccurrenceResolution{resolution},
		Declarations: []Declaration{declaration},
		Types:        []Type{basic, signature},
		TypeWitnesses: []TypeWitness{
			mustTypeWitness(
				t, fixture.pkg, basic.ID(), fixture.authority,
			),
			mustTypeWitness(
				t, fixture.pkg, signature.ID(), fixture.authority,
			),
		},
		Operations: []Operation{operation},
	})
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	measurement, err := writeSemanticShard(&output, pkg)
	if err != nil {
		t.Fatal(err)
	}
	var operationOutput bytes.Buffer
	operationMeasurement := newSemanticShardMeasurement(pkg.ID())
	operationEncoder := newBinaryShardEncoder(&operationOutput)
	writeBinaryOperations(
		operationEncoder, pkg, &operationMeasurement,
	)
	operationSectionBytes, err := operationEncoder.finish()
	if err != nil {
		t.Fatal(err)
	}
	operationTail := materializeRecordSizes(
		pkg.ID(),
		operationMeasurement.operationCandidates,
		func(reference operationRef) string {
			return pkg.identities.operation(reference).String()
		},
	)
	measurementOperationTail := materializeRecordSizes(
		pkg.ID(),
		measurement.operationCandidates,
		func(reference operationRef) string {
			return pkg.identities.operation(reference).String()
		},
	)
	if measurement.encodedBytes != int64(output.Len()) ||
		len(measurementOperationTail) != 1 ||
		measurementOperationTail[0].Identity !=
			operationID.String() ||
		measurementOperationTail[0].EncodedBytes !=
			operationTail[0].EncodedBytes ||
		operationSectionBytes !=
			operationTail[0].EncodedBytes+1 {
		t.Fatalf(
			"semantic shard measurement=%+v output=%d operation=%d",
			measurement,
			output.Len(),
			operationTail[0].EncodedBytes,
		)
	}

	measurement = newSemanticShardMeasurement(fixture.pkg)
	for index := 0; index < 1000; index++ {
		measurement.considerOperation(
			operationRef(index+1),
			int64(index+1),
		)
		if len(measurement.operationCandidates) >
			semanticTailLimit {
			t.Fatalf(
				"record tail retained %d entries",
				len(measurement.operationCandidates),
			)
		}
	}
	rendered := 0
	tail := materializeRecordSizes(
		fixture.pkg,
		measurement.operationCandidates,
		func(reference operationRef) string {
			rendered++
			return fmt.Sprintf("operation-%04d", reference)
		},
	)
	if rendered != semanticTailLimit ||
		tail[0].EncodedBytes != 1000 ||
		tail[semanticTailLimit-1].EncodedBytes != 981 {
		t.Fatalf(
			"bounded record tail=%+v rendered=%d",
			tail,
			rendered,
		)
	}
}
