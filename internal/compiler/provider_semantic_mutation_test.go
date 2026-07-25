package compiler

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/semantic"
	"github.com/tsoniclang/gotots/internal/scope/contract"
	"github.com/tsoniclang/gotots/internal/source"
)

func TestProviderSemanticAdmissionRejectsResealedCorruption(
	t *testing.T,
) {
	project := writeProviderFixture(
		t,
		"example.com/provider-semantic-mutation",
		"provider-semantic-mutation",
	)
	request := source.Request{
		Dir: project, Patterns: []string{"."},
		ProviderContract: contract.DefaultID,
	}
	output := t.TempDir()
	structurePath := filepath.Join(
		output, "provider.structure.gotots",
	)
	semanticPath := filepath.Join(
		output, "provider.semantic.gotots",
	)
	result, err := AuditCatalog(
		request, structurePath, semanticPath,
	)
	if err != nil {
		t.Fatal(err)
	}
	request.ProviderStructureArtifact = structurePath
	request.ProviderStructureDigest = result.Structure.Digest
	tests := []struct {
		name   string
		mutate semanticShardMutation
		want   string
	}{
		{
			name: "shard-magic",
			mutate: func(
				shard []byte,
				_ *mutableSemanticManifestPackage,
			) ([]byte, bool) {
				shard[0] ^= 0xff
				return shard, true
			},
			want: "shard magic",
		},
		{
			name: "shard-version",
			mutate: func(
				shard []byte,
				_ *mutableSemanticManifestPackage,
			) ([]byte, bool) {
				shard[8]++
				return shard, true
			},
			want: "shard version",
		},
		{
			name: "shard-truncation",
			mutate: func(
				shard []byte,
				_ *mutableSemanticManifestPackage,
			) ([]byte, bool) {
				return shard[:len(shard)-1], true
			},
		},
		{
			name: "shard-trailing-byte",
			mutate: func(
				shard []byte,
				_ *mutableSemanticManifestPackage,
			) ([]byte, bool) {
				return append(shard, 0), true
			},
			want: "trailing bytes",
		},
		{
			name: "manifest-record-count",
			mutate: func(
				shard []byte,
				entry *mutableSemanticManifestPackage,
			) ([]byte, bool) {
				entry.ResolutionCount++
				return shard, true
			},
			want: "resolutions count",
		},
		{
			name: "manifest-capacity",
			mutate: func(
				shard []byte,
				entry *mutableSemanticManifestPackage,
			) ([]byte, bool) {
				entry.ResolutionCount = len(shard) + 1
				return shard, true
			},
			want: "manifest package is invalid",
		},
		{
			name: "member-target-count",
			mutate: func(
				shard []byte,
				entry *mutableSemanticManifestPackage,
			) ([]byte, bool) {
				entry.MemberTargetCount++
				return shard, true
			},
			want: "member targets disagree with manifest",
		},
		{
			name: "member-target-digest",
			mutate: func(
				shard []byte,
				entry *mutableSemanticManifestPackage,
			) ([]byte, bool) {
				entry.MemberTargetDigest = strings.Repeat("00", 32)
				return shard, true
			},
			want: "member targets disagree with manifest",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path, digest, packageID := rewriteSemanticArtifact(
				t, semanticPath, test.mutate,
			)
			mutated := request
			mutated.ProviderSemanticArtifact = path
			mutated.ProviderSemanticDigest = digest
			inspectErr := admitOrProjectMutatedSemanticPackage(
				t, mutated, packageID,
			)
			if inspectErr == nil {
				t.Fatal("resealed semantic mutation was accepted")
			}
			if test.want != "" &&
				!strings.Contains(inspectErr.Error(), test.want) {
				t.Fatalf(
					"resealed mutation error = %v, want %q",
					inspectErr,
					test.want,
				)
			}
		})
	}
}

func admitOrProjectMutatedSemanticPackage(
	t *testing.T,
	request source.Request,
	encodedPackage string,
) error {
	t.Helper()
	inspection, err := inspectConstructsForTest(t, request)
	if err != nil {
		return err
	}
	before := inspection.Semantic().ProviderReadStats()
	if before.ShardLoads != 0 ||
		before.MaxProviderPackagesResident != 0 {
		t.Fatalf(
			"ordinary inspection opened mutated provider detail: %+v",
			before,
		)
	}
	packageID, err := identity.ParsePackageID(encodedPackage)
	if err != nil {
		t.Fatal(err)
	}
	return inspection.Semantic().VisitPackage(
		packageID,
		func(semantic.Package) error {
			return nil
		},
	)
}
