package compiler

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
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

type semanticShardMutation func(
	[]byte,
	*mutableSemanticManifestPackage,
) ([]byte, bool)

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
			encoded, mutated = mutate(encoded, entry)
			if mutated {
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
		t.Fatal("semantic artifact had no shard accepted by mutation")
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
