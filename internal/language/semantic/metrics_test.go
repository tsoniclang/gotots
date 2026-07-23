package semantic

import (
	"fmt"
	"testing"

	"github.com/tsoniclang/gotots/internal/identity"
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
	metrics, err := measureProviderManifest([]providerManifestPackage{{
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
