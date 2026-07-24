package compiler

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/semantic"
	"github.com/tsoniclang/gotots/internal/scope/contract"
	"github.com/tsoniclang/gotots/internal/source"
)

type mutableSemanticManifest struct {
	Context  json.RawMessage                  `json:"context"`
	Packages []mutableSemanticManifestPackage `json:"packages"`
}

type mutableSemanticManifestPackage struct {
	Package            string   `json:"package"`
	Provenance         uint8    `json:"provenance"`
	PackageInput       string   `json:"packageInputDigest"`
	Structure          string   `json:"structureDigest"`
	Selection          string   `json:"selectionDigest"`
	Definitions        []string `json:"definitions"`
	Declarations       []string `json:"declarations"`
	DefinitionCount    int      `json:"definitionCount"`
	ResolutionCount    int      `json:"resolutionCount"`
	DeclarationCount   int      `json:"declarationCount"`
	MemberTargetCount  int      `json:"memberTargetCount"`
	MemberTargetDigest string   `json:"memberTargetDigest"`
	BindingCount       int      `json:"bindingCount"`
	TypeCount          int      `json:"typeCount"`
	OperationCount     int      `json:"operationCount"`
	UnsupportedCount   int      `json:"unsupportedCount"`
	ShardOffset        int64    `json:"shardOffset"`
	ShardBytes         int64    `json:"shardBytes"`
	ShardDigest        string   `json:"shardDigest"`
}

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

	t.Run("missing-referenced-declaration", func(t *testing.T) {
		path, digest, packageID := rewriteSemanticArtifact(
			t,
			semanticPath,
			removeReferencedSemanticDeclaration,
		)
		request.ProviderSemanticArtifact = path
		request.ProviderSemanticDigest = digest
		inspectErr := projectMutatedSemanticPackage(
			t, request, packageID,
		)
		if inspectErr == nil ||
			!strings.Contains(
				inspectErr.Error(),
				"absent owned declaration",
			) ||
			!strings.Contains(inspectErr.Error(), packageID) {
			t.Fatalf(
				"resealed internal relationship error = %v",
				inspectErr,
			)
		}
	})

	t.Run("target-specific-field", func(t *testing.T) {
		path, digest, packageID := rewriteSemanticArtifact(
			t,
			semanticPath,
			injectTargetSpecificSemanticField,
		)
		request.ProviderSemanticArtifact = path
		request.ProviderSemanticDigest = digest
		inspectErr := projectMutatedSemanticPackage(
			t, request, packageID,
		)
		if inspectErr == nil ||
			!strings.Contains(
				inspectErr.Error(), "typescriptShape",
			) ||
			!strings.Contains(inspectErr.Error(), packageID) {
			t.Fatalf(
				"resealed target-specific field error = %v",
				inspectErr,
			)
		}
	})

	t.Run("manifest-capacity-exceeds-shard", func(t *testing.T) {
		path, digest, _ := rewriteSemanticArtifact(
			t,
			semanticPath,
			exceedSemanticManifestCapacity,
		)
		request.ProviderSemanticArtifact = path
		request.ProviderSemanticDigest = digest
		_, inspectErr := inspectConstructsForTest(t, request)
		if inspectErr == nil ||
			!strings.Contains(
				inspectErr.Error(),
				"semantic provider manifest package is invalid",
			) {
			t.Fatalf(
				"manifest capacity mutation error = %v",
				inspectErr,
			)
		}
	})

	t.Run("record-exceeds-manifest-count", func(t *testing.T) {
		path, digest, packageID := rewriteSemanticArtifact(
			t,
			semanticPath,
			duplicateSemanticResolutionWithoutCount,
		)
		request.ProviderSemanticArtifact = path
		request.ProviderSemanticDigest = digest
		inspectErr := projectMutatedSemanticPackage(
			t, request, packageID,
		)
		if inspectErr == nil ||
			!strings.Contains(
				inspectErr.Error(),
				"resolutions exceeds manifest count",
			) ||
			!strings.Contains(inspectErr.Error(), packageID) {
			t.Fatalf(
				"streamed record-count mutation error = %v",
				inspectErr,
			)
		}
	})

	t.Run("unexported-member-without-package", func(t *testing.T) {
		path, digest, packageID := rewriteSemanticArtifact(
			t,
			semanticPath,
			dropUnexportedMemberPackage,
		)
		request.ProviderSemanticArtifact = path
		request.ProviderSemanticDigest = digest
		inspectErr := projectMutatedSemanticPackage(
			t, request, packageID,
		)
		if inspectErr == nil ||
			!strings.Contains(
				inspectErr.Error(), packageID,
			) ||
			!strings.Contains(
				inspectErr.Error(), "semantic provider type",
			) ||
			!strings.Contains(
				inspectErr.Error(), "is not canonical",
			) {
			t.Fatalf(
				"unexported-member package mutation error = %v",
				inspectErr,
			)
		}
	})

	t.Run("circular-interface-method-identity", func(t *testing.T) {
		path, digest, packageID := rewriteSemanticArtifact(
			t,
			semanticPath,
			introduceCircularInterfaceMethodIdentity,
		)
		request.ProviderSemanticArtifact = path
		request.ProviderSemanticDigest = digest
		inspectErr := projectMutatedSemanticPackage(
			t, request, packageID,
		)
		if inspectErr == nil ||
			!strings.Contains(
				inspectErr.Error(), packageID,
			) ||
			!strings.Contains(
				inspectErr.Error(),
				"semantic wire type payload disagrees with identity",
			) {
			t.Fatalf(
				"circular interface identity mutation error = %v",
				inspectErr,
			)
		}
	})

	t.Run("serialized-member-declaration", func(t *testing.T) {
		path, digest, packageID := rewriteSemanticArtifact(
			t,
			semanticPath,
			serializeSemanticMemberDeclaration,
		)
		request.ProviderSemanticArtifact = path
		request.ProviderSemanticDigest = digest
		_, inspectErr := inspectConstructsForTest(t, request)
		if inspectErr == nil ||
			!strings.Contains(
				inspectErr.Error(),
				"serializes member declaration",
			) ||
			!strings.Contains(inspectErr.Error(), packageID) {
			t.Fatalf(
				"serialized member declaration error = %v",
				inspectErr,
			)
		}
	})

	for name, mutate := range map[string]semanticShardMutation{
		"member-target-count":  alterSemanticMemberTargetCount,
		"member-target-digest": alterSemanticMemberTargetDigest,
	} {
		t.Run(name, func(t *testing.T) {
			path, digest, packageID := rewriteSemanticArtifact(
				t, semanticPath, mutate,
			)
			request.ProviderSemanticArtifact = path
			request.ProviderSemanticDigest = digest
			inspectErr := projectMutatedSemanticPackage(
				t, request, packageID,
			)
			if inspectErr == nil ||
				!strings.Contains(
					inspectErr.Error(),
					"member targets disagree with manifest",
				) ||
				!strings.Contains(inspectErr.Error(), packageID) {
				t.Fatalf(
					"%s mutation error = %v",
					name, inspectErr,
				)
			}
		})
	}
}

func projectMutatedSemanticPackage(
	t *testing.T,
	request source.Request,
	encodedPackage string,
) error {
	t.Helper()
	inspection, err := inspectConstructsForTest(t, request)
	if err != nil {
		t.Fatalf(
			"trusted manifest admission opened provider detail: %v",
			err,
		)
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

type semanticShardMutation func(
	map[string]json.RawMessage,
	*mutableSemanticManifestPackage,
) bool
