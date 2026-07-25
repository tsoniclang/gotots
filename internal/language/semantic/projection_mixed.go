package semantic

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
)

func visitMixedProjection(
	checker *CheckerStore,
	provider *ProviderArtifact,
	projection packageProjection,
	visit func(Package) error,
) error {
	if checker == nil || provider == nil || visit == nil ||
		!projection.local || !projection.certified {
		return fmt.Errorf(
			"mixed semantic projection requires both authorities and a visitor",
		)
	}
	checkerIndex, checkerPresent := checker.byPackage[projection.id]
	providerIndex, providerPresent := provider.byPackage[projection.id]
	if !checkerPresent || !providerPresent {
		return fmt.Errorf(
			"mixed semantic projection %s lacks a selected shard",
			projection.id,
		)
	}
	checker.projection.Lock()
	defer checker.projection.Unlock()
	provider.projection.Lock()
	defer provider.projection.Unlock()
	if err := checker.beginProjection(); err != nil {
		return err
	}
	defer checker.endProjection()
	provider.beginProjection()
	defer provider.endProjection()

	checkerEntry := checker.manifest[checkerIndex]
	providerEntry := provider.manifest[providerIndex]
	checkerAuthority, err := NewCheckerAuthority(
		checker.toolchain,
		checker.config,
		checkerEntry.PackageInput,
		checkerEntry.Structure,
		checkerEntry.Selection,
	)
	if err != nil {
		return err
	}
	providerAuthority, err := NewCertifiedProviderAuthority(
		provider.digest,
		providerEntry.ShardDigest,
		provider.context.StructuralArtifactDigest,
	)
	if err != nil {
		return err
	}
	providerFile, err := os.Open(provider.path)
	if err != nil {
		return err
	}
	defer providerFile.Close()

	checkerHash := sha256.New()
	providerHash := sha256.New()
	checkerReader := io.TeeReader(
		io.NewSectionReader(
			checker.file,
			checkerEntry.ShardOffset,
			checkerEntry.ShardBytes,
		),
		checkerHash,
	)
	providerReader := io.TeeReader(
		io.NewSectionReader(
			providerFile,
			provider.shardBase+providerEntry.ShardOffset,
			providerEntry.ShardBytes,
		),
		providerHash,
	)
	pkg, err := decodeMixedSemanticShards(
		checkerReader,
		checkerAuthority,
		checkerEntry,
		providerReader,
		providerAuthority,
		providerEntry,
		projection,
	)
	if err != nil {
		return err
	}
	if fmt.Sprintf("%x", checkerHash.Sum(nil)) !=
		checkerEntry.ShardDigest {
		return fmt.Errorf("checker semantic shard digest mismatch")
	}
	if fmt.Sprintf("%x", providerHash.Sum(nil)) !=
		providerEntry.ShardDigest {
		return fmt.Errorf("provider semantic shard digest mismatch")
	}
	if err := validateProjectedPackage(pkg, providerEntry); err != nil {
		return err
	}
	checkerMetrics, err := measureShardManifest(
		[]packageShardManifest{checkerEntry},
	)
	if err != nil {
		return err
	}
	checker.recordPackageMetrics(projection.id, checkerMetrics)
	providerMetrics, err := measureShardManifest(
		[]packageShardManifest{providerEntry},
	)
	if err != nil {
		return err
	}
	provider.recordPackageMetrics(projection.id, providerMetrics)
	return visit(pkg)
}
