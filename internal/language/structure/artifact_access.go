package structure

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/tsoniclang/gotots/internal/identity"
)

func (a *ProviderArtifact) packageArtifact(
	packageID identity.PackageID,
) (*ProviderArtifact, error) {
	if a == nil {
		return nil, providerArtifactError("artifact is absent")
	}
	if a.storage == nil {
		if _, present := a.packageDigests[packageID]; !present {
			return nil, providerArtifactError(
				"artifact has no package " + packageID.String(),
			)
		}
		return a, nil
	}
	a.storage.mu.Lock()
	defer a.storage.mu.Unlock()
	if loaded := a.storage.loaded[packageID]; loaded != nil {
		return loaded, nil
	}
	shard, present := a.storage.shards[packageID]
	if !present {
		return nil, providerArtifactError(
			"artifact has no package shard " + packageID.String(),
		)
	}
	file, err := os.Open(a.storage.path)
	if err != nil {
		return nil, fmt.Errorf("provider shard unreadable: %w", err)
	}
	defer file.Close()
	section := io.NewSectionReader(file, shard.offset, shard.bytes)
	hash := sha256.New()
	if _, err := io.Copy(hash, section); err != nil {
		return nil, fmt.Errorf("provider shard digest failed: %w", err)
	}
	if fmt.Sprintf("%x", hash.Sum(nil)) != shard.digest {
		return nil, providerArtifactError(
			"package shard digest mismatch " + packageID.String(),
		)
	}
	section = io.NewSectionReader(file, shard.offset, shard.bytes)
	decoded, err := decodeProviderStream(section)
	if err != nil {
		return nil, err
	}
	if artifactContext(decoded) != artifactContext(a) {
		return nil, providerArtifactError(
			"package shard context mismatch " + packageID.String(),
		)
	}
	if err := validateLoadedShard(packageID, shard, decoded); err != nil {
		return nil, err
	}
	a.storage.loaded[packageID] = decoded
	return decoded, nil
}

func validateLoadedShard(
	packageID identity.PackageID,
	shard providerShard,
	decoded *ProviderArtifact,
) error {
	if len(decoded.packageDigests) != 1 ||
		decoded.packageDigests[packageID] != shard.inputDigest ||
		decoded.syntheticPackages[packageID] != shard.synthetic ||
		len(decoded.factsByID) != shard.factCount ||
		len(decoded.filePackages) != len(shard.files) {
		return providerArtifactError(
			"package shard manifest mismatch " + packageID.String(),
		)
	}
	for _, file := range shard.files {
		if decoded.filePackages[file] != packageID {
			return providerArtifactError(
				"package shard omits manifest file " + file.String(),
			)
		}
	}
	for id := range decoded.factsByID {
		if decoded.packageForDefinition(id.definition) != packageID {
			return providerArtifactError(
				"package shard contains a foreign selection fact " +
					id.definition.String(),
			)
		}
	}
	return nil
}

func (a *ProviderArtifact) packageForDefinition(
	definition identity.DefinitionID,
) identity.PackageID {
	if file := definition.File(); !file.IsZero() {
		return a.filePackages[file]
	}
	return definition.Package()
}

// CertifiedFactsForPackage returns one isolated canonical package projection.
// It never loads an unrelated provider shard.
func (a *ProviderArtifact) CertifiedFactsForPackage(
	packageID identity.PackageID,
) (
	[]CertifiedFact,
	error,
) {
	if a == nil {
		return nil, nil
	}
	shard, err := a.packageArtifact(packageID)
	if err != nil {
		return nil, err
	}
	var out []CertifiedFact
	for id, fact := range shard.factsByID {
		if shard.packageForDefinition(id.definition) == packageID {
			out = append(out, fact)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].definition != out[j].definition {
			return out[i].definition.String() <
				out[j].definition.String()
		}
		return out[i].kind < out[j].kind
	})
	return out, nil
}
