package compiler

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"

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
	Package          string   `json:"package"`
	Provenance       uint8    `json:"provenance"`
	PackageInput     string   `json:"packageInputDigest"`
	Structure        string   `json:"structureDigest"`
	Selection        string   `json:"selectionDigest"`
	Definitions      []string `json:"definitions"`
	Declarations     []string `json:"declarations"`
	DefinitionCount  int      `json:"definitionCount"`
	ResolutionCount  int      `json:"resolutionCount"`
	DeclarationCount int      `json:"declarationCount"`
	BindingCount     int      `json:"bindingCount"`
	TypeCount        int      `json:"typeCount"`
	OperationCount   int      `json:"operationCount"`
	UnsupportedCount int      `json:"unsupportedCount"`
	ShardOffset      int64    `json:"shardOffset"`
	ShardBytes       int64    `json:"shardBytes"`
	ShardDigest      string   `json:"shardDigest"`
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
				"references absent semantic declaration",
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
				"does not match canonical",
			) {
			t.Fatalf(
				"circular interface identity mutation error = %v",
				inspectErr,
			)
		}
	})
}

func projectMutatedSemanticPackage(
	t *testing.T,
	request source.Request,
	encodedPackage string,
) error {
	t.Helper()
	inspection, err := InspectConstructs(request)
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

func removeReferencedSemanticDeclaration(
	shard map[string]json.RawMessage,
	entry *mutableSemanticManifestPackage,
) bool {
	var declarations []map[string]json.RawMessage
	if err := json.Unmarshal(
		shard["declarations"], &declarations,
	); err != nil || len(declarations) == 0 {
		return false
	}
	var definitions []struct {
		Declarations []string `json:"declarations"`
	}
	if err := json.Unmarshal(
		shard["definitions"], &definitions,
	); err != nil {
		return false
	}
	referenced := map[string]bool{}
	for _, definition := range definitions {
		for _, declaration := range definition.Declarations {
			referenced[declaration] = true
		}
	}
	for index, declaration := range declarations {
		var id string
		if err := json.Unmarshal(declaration["id"], &id); err != nil ||
			!referenced[id] {
			continue
		}
		declarations = append(
			declarations[:index], declarations[index+1:]...,
		)
		encoded, err := json.Marshal(declarations)
		if err != nil {
			return false
		}
		shard["declarations"] = encoded
		entry.DeclarationCount--
		for manifestIndex, manifestID := range entry.Declarations {
			if manifestID != id {
				continue
			}
			entry.Declarations = append(
				entry.Declarations[:manifestIndex],
				entry.Declarations[manifestIndex+1:]...,
			)
			return true
		}
		return false
	}
	return false
}

func injectTargetSpecificSemanticField(
	shard map[string]json.RawMessage,
	_ *mutableSemanticManifestPackage,
) bool {
	if _, exists := shard["typescriptShape"]; exists {
		return false
	}
	shard["typescriptShape"] = json.RawMessage(`"class"`)
	return true
}

func dropUnexportedMemberPackage(
	shard map[string]json.RawMessage,
	_ *mutableSemanticManifestPackage,
) bool {
	var records []map[string]json.RawMessage
	if err := json.Unmarshal(shard["types"], &records); err != nil {
		return false
	}
	for _, record := range records {
		for _, fieldName := range []string{"methods", "fields"} {
			var members []map[string]json.RawMessage
			if err := json.Unmarshal(
				record[fieldName], &members,
			); err != nil {
				continue
			}
			for _, member := range members {
				var name, packageID string
				if err := json.Unmarshal(
					member["name"], &name,
				); err != nil ||
					json.Unmarshal(
						member["package"], &packageID,
					) != nil ||
					packageID == "" ||
					semanticNameIsExported(name) {
					continue
				}
				member["package"] = json.RawMessage(`""`)
				encodedMembers, err := json.Marshal(members)
				if err != nil {
					return false
				}
				record[fieldName] = encodedMembers
				encodedTypes, err := json.Marshal(records)
				if err != nil {
					return false
				}
				shard["types"] = encodedTypes
				return true
			}
		}
	}
	return false
}

func introduceCircularInterfaceMethodIdentity(
	shard map[string]json.RawMessage,
	_ *mutableSemanticManifestPackage,
) bool {
	var records []map[string]json.RawMessage
	if err := json.Unmarshal(shard["types"], &records); err != nil {
		return false
	}
	for _, record := range records {
		var kind uint8
		var id string
		var methods []map[string]json.RawMessage
		if json.Unmarshal(record["kind"], &kind) != nil ||
			semantic.TypeKind(kind) != semantic.TypeInterface ||
			json.Unmarshal(record["id"], &id) != nil ||
			json.Unmarshal(record["methods"], &methods) != nil ||
			len(methods) == 0 {
			continue
		}
		encodedID, err := json.Marshal(id)
		if err != nil {
			return false
		}
		methods[0]["signature"] = encodedID
		encodedMethods, err := json.Marshal(methods)
		if err != nil {
			return false
		}
		record["methods"] = encodedMethods
		encodedTypes, err := json.Marshal(records)
		if err != nil {
			return false
		}
		shard["types"] = encodedTypes
		return true
	}
	return false
}

func semanticNameIsExported(name string) bool {
	first, _ := utf8.DecodeRuneInString(name)
	return first != utf8.RuneError && unicode.IsUpper(first)
}

func rewriteSemanticArtifact(
	t *testing.T,
	path string,
	mutate semanticShardMutation,
) (string, string, string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) < 16 {
		t.Fatal("semantic artifact is shorter than its header")
	}
	manifestBytes := int(binary.BigEndian.Uint64(raw[8:16]))
	shardBase := 16 + manifestBytes
	if manifestBytes <= 0 || shardBase > len(raw) {
		t.Fatal("semantic artifact has an invalid manifest length")
	}
	var manifest mutableSemanticManifest
	if err := json.Unmarshal(raw[16:shardBase], &manifest); err != nil {
		t.Fatal(err)
	}
	shards := make([][]byte, len(manifest.Packages))
	mutated := false
	mutatedPackage := ""
	var nextOffset int64
	for index := range manifest.Packages {
		entry := &manifest.Packages[index]
		start := shardBase + int(entry.ShardOffset)
		end := start + int(entry.ShardBytes)
		if start < shardBase || end > len(raw) || start >= end {
			t.Fatal("semantic shard extent is invalid")
		}
		encoded := append([]byte(nil), raw[start:end]...)
		if !mutated {
			var shard map[string]json.RawMessage
			if err := json.Unmarshal(encoded, &shard); err != nil {
				t.Fatal(err)
			}
			if mutate(shard, entry) {
				encoded, err = json.Marshal(shard)
				if err != nil {
					t.Fatal(err)
				}
				mutated = true
				mutatedPackage = entry.Package
			}
		}
		digest := sha256.Sum256(encoded)
		entry.ShardOffset = nextOffset
		entry.ShardBytes = int64(len(encoded))
		entry.ShardDigest = hex.EncodeToString(digest[:])
		nextOffset += entry.ShardBytes
		shards[index] = encoded
	}
	if !mutated {
		t.Fatal("semantic artifact had no shard accepted by the mutation")
	}
	encodedManifest, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	rewritten := make(
		[]byte, 0, 16+len(encodedManifest)+int(nextOffset),
	)
	rewritten = append(rewritten, raw[:8]...)
	var length [8]byte
	binary.BigEndian.PutUint64(
		length[:], uint64(len(encodedManifest)),
	)
	rewritten = append(rewritten, length[:]...)
	rewritten = append(rewritten, encodedManifest...)
	for _, shard := range shards {
		rewritten = append(rewritten, shard...)
	}
	rewrittenPath := filepath.Join(
		t.TempDir(), "resealed-provider.semantic.gotots",
	)
	if err := os.WriteFile(rewrittenPath, rewritten, 0o600); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(rewritten)
	return rewrittenPath, hex.EncodeToString(digest[:]), mutatedPackage
}
