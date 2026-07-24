package semantic

import (
	"bytes"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/identity"
)

func TestNormalizedSemanticShardRoundTripsWithoutRenderedReferences(
	t *testing.T,
) {
	pkg := semanticWirePackage(t)
	var encoded bytes.Buffer
	if _, err := writeSemanticShard(&encoded, pkg); err != nil {
		t.Fatal(err)
	}
	entry := semanticWireManifestEntry(
		t,
		pkg,
		int64(encoded.Len()),
	)
	authority := semanticFixture(t).authority
	decoded, err := decodeSemanticShard(
		bytes.NewReader(encoded.Bytes()),
		authority,
		entry,
	)
	if err != nil {
		t.Fatal(err)
	}
	assertSemanticPackagesEqual(t, pkg, decoded)
	if err := validateProjectedPackage(decoded, entry); err != nil {
		t.Fatal(err)
	}
	var definition identity.DefinitionID
	if err := pkg.VisitDefinitions(
		func(record DefinitionSemantics) error {
			definition = record.Definition()
			return nil
		},
	); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(
		encoded.Bytes(),
		[]byte(definition.String()),
	) || bytes.Contains(
		encoded.Bytes(),
		[]byte(pkg.ID().String()),
	) {
		t.Fatal(
			"normalized semantic shard repeats a rendered composite identity",
		)
	}
}

func TestMixedSemanticShardDecodeBuildsOneAuthoritySelectedPackage(
	t *testing.T,
) {
	pkg := semanticWirePackage(t)
	var encoded bytes.Buffer
	if _, err := writeSemanticShard(&encoded, pkg); err != nil {
		t.Fatal(err)
	}
	entry := semanticWireManifestEntry(
		t,
		pkg,
		int64(encoded.Len()),
	)
	checkerAuthority := semanticFixture(t).authority
	providerAuthority, err := NewCertifiedProviderAuthority(
		strings.Repeat("cd", 32),
		strings.Repeat("de", 32),
		strings.Repeat("ef", 32),
	)
	if err != nil {
		t.Fatal(err)
	}
	localFiles := map[identity.FileID]bool{}
	if err := pkg.VisitDefinitions(
		func(record DefinitionSemantics) error {
			localFiles[record.Definition().File()] = true
			return nil
		},
	); err != nil {
		t.Fatal(err)
	}
	decoded, err := decodeMixedSemanticShards(
		bytes.NewReader(encoded.Bytes()),
		checkerAuthority,
		entry,
		bytes.NewReader(encoded.Bytes()),
		providerAuthority,
		entry,
		packageProjection{
			id:         pkg.ID(),
			provenance: pkg.Provenance(),
			local:      true,
			localFiles: localFiles,
			certified:  true,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	assertSemanticPackagesEqual(t, pkg, decoded)
	if err := decoded.VisitDefinitions(
		func(record DefinitionSemantics) error {
			if record.Authority().Kind() != AuthorityChecker {
				t.Fatalf(
					"mixed definition authority = %v",
					record.Authority().Kind(),
				)
			}
			return nil
		},
	); err != nil {
		t.Fatal(err)
	}
}

func assertSemanticPackagesEqual(
	t *testing.T,
	want Package,
	got Package,
) {
	t.Helper()
	if want.ID() != got.ID() ||
		want.Provenance() != got.Provenance() ||
		want.DefinitionCount() != got.DefinitionCount() ||
		want.ResolutionCount() != got.ResolutionCount() ||
		want.DeclarationCount() != got.DeclarationCount() ||
		want.BindingCount() != got.BindingCount() ||
		want.TypeCount() != got.TypeCount() ||
		want.OperationCount() != got.OperationCount() ||
		want.UnsupportedCount() != got.UnsupportedCount() {
		t.Fatalf(
			"semantic package census differs: want=%+v got=%+v",
			packageCensus(want),
			packageCensus(got),
		)
	}
	assertDefinitionRecordsEqual(t, want, got)
	assertResolutionRecordsEqual(t, want, got)
	assertDeclarationRecordsEqual(t, want, got)
	assertBindingRecordsEqual(t, want, got)
	assertTypeRecordsEqual(t, want, got)
	assertOperationRecordsEqual(t, want, got)
	assertUnsupportedRecordsEqual(t, want, got)
}

type semanticPackageCensus struct {
	definitions  int
	resolutions  int
	declarations int
	bindings     int
	types        int
	operations   int
	unsupported  int
}

func packageCensus(pkg Package) semanticPackageCensus {
	return semanticPackageCensus{
		definitions:  pkg.DefinitionCount(),
		resolutions:  pkg.ResolutionCount(),
		declarations: pkg.DeclarationCount(),
		bindings:     pkg.BindingCount(),
		types:        pkg.TypeCount(),
		operations:   pkg.OperationCount(),
		unsupported:  pkg.UnsupportedCount(),
	}
}
