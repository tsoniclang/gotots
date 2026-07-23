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

	"github.com/tsoniclang/gotots/internal/scope/contract"
	"github.com/tsoniclang/gotots/internal/source"
)

func TestProviderRequestedFactOmissionFailsConsumption(t *testing.T) {
	project := writeProviderFixture(
		t,
		"example.com/provider-fact",
		"provider-fact",
	)
	request := source.Request{
		Dir: project, Patterns: []string{"."},
		ProviderContract: contract.DefaultID,
	}
	originalPath := filepath.Join(t.TempDir(), "provider.gotots")
	result, err := AuditCatalog(request, originalPath)
	if err != nil {
		t.Fatal(err)
	}
	mutatedPath, digest := omitOneProviderFact(
		t,
		originalPath,
	)
	request.AuditArtifact = mutatedPath
	request.AuditArtifactDigest = digest
	_, err = InspectConstructs(request)
	if err == nil ||
		(!strings.Contains(err.Error(), "requested fact") &&
			!strings.Contains(err.Error(), "fact-set join failed")) {
		t.Fatalf("provider fact omission error = %v", err)
	}
	if digest == result.Digest {
		t.Fatal("provider mutation retained the original container digest")
	}
}

type mutableProviderManifest struct {
	Version  int                              `json:"version"`
	Context  json.RawMessage                  `json:"context"`
	Packages []mutableProviderManifestPackage `json:"packages"`
}

type mutableProviderManifestPackage struct {
	Package           string            `json:"package"`
	InputDigest       string            `json:"inputDigest"`
	Files             []string          `json:"files"`
	Synthetic         bool              `json:"synthetic"`
	Definitions       []string          `json:"definitions,omitempty"`
	HeaderOccurrences int               `json:"headerOccurrences"`
	BoundaryEntries   int               `json:"boundaryEntries"`
	Facts             []json.RawMessage `json:"selectionFacts,omitempty"`
	ShardBytes        int64             `json:"shardBytes"`
	ShardDigest       string            `json:"shardDigest"`
}

func omitOneProviderFact(
	t *testing.T,
	path string,
) (string, string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) < 16 {
		t.Fatal("provider artifact is shorter than its container header")
	}
	manifestBytes := int(binary.BigEndian.Uint64(raw[8:16]))
	if manifestBytes <= 0 || 16+manifestBytes > len(raw) {
		t.Fatal("provider artifact has an invalid manifest length")
	}
	var manifest mutableProviderManifest
	if err := json.Unmarshal(
		raw[16:16+manifestBytes],
		&manifest,
	); err != nil {
		t.Fatal(err)
	}
	mutated := false
	for index := range manifest.Packages {
		if len(manifest.Packages[index].Facts) == 0 {
			continue
		}
		manifest.Packages[index].Facts =
			manifest.Packages[index].Facts[1:]
		mutated = true
		break
	}
	if !mutated {
		t.Fatal("provider artifact has no requested fact to omit")
	}
	encoded, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	output := make(
		[]byte,
		0,
		16+len(encoded)+len(raw)-(16+manifestBytes),
	)
	output = append(output, raw[:8]...)
	var length [8]byte
	binary.BigEndian.PutUint64(length[:], uint64(len(encoded)))
	output = append(output, length[:]...)
	output = append(output, encoded...)
	output = append(output, raw[16+manifestBytes:]...)
	mutatedPath := filepath.Join(
		t.TempDir(),
		"provider-with-missing-fact.gotots",
	)
	if err := os.WriteFile(mutatedPath, output, 0o600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(output)
	return mutatedPath, hex.EncodeToString(sum[:])
}
