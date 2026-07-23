package structure

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"

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
	if a.storage.loadedPackage == packageID &&
		a.storage.loadedArtifact != nil {
		a.storage.cacheHits++
		return a.storage.loadedArtifact, nil
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
	recordProviderProjection(a.storage, packageID, shard, decoded)
	a.storage.loadedPackage = packageID
	a.storage.loadedArtifact = decoded
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
		decoded.factCount != 0 ||
		len(decoded.filePackages) != len(shard.files) ||
		!sameProviderPackageCensus(
			decoded.packageCensus[packageID],
			shard.census,
		) {
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

// CertifiedFactsForPackage returns the manifest-resident facts for one
// package. Detailed structural shards do not duplicate these records.
func (a *ProviderArtifact) CertifiedFactsForPackage(
	packageID identity.PackageID,
) []CertifiedFact {
	if a == nil {
		return nil
	}
	return append(
		[]CertifiedFact(nil),
		a.factsByPackage[packageID]...,
	)
}
