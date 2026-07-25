package semantic

import (
	"fmt"
	"io"
)

func decodeMixedSemanticShards(
	checkerInput io.Reader,
	checkerAuthority Authority,
	checkerEntry packageShardManifest,
	providerInput io.Reader,
	providerAuthority Authority,
	providerEntry packageShardManifest,
	projection packageProjection,
) (Package, error) {
	if checkerInput == nil || providerInput == nil ||
		!checkerAuthority.Valid() || !providerAuthority.Valid() {
		return Package{}, fmt.Errorf(
			"mixed semantic shard decoder requires both authorities",
		)
	}
	checkerShard, err := beginBinarySemanticShard(
		checkerInput, checkerEntry,
	)
	if err != nil {
		return Package{}, err
	}
	providerShard, err := beginBinarySemanticShard(
		providerInput, providerEntry,
	)
	if err != nil {
		return Package{}, err
	}
	if checkerShard.pkg != projection.id ||
		providerShard.pkg != projection.id ||
		checkerShard.provenance != projection.provenance ||
		providerShard.provenance != projection.provenance {
		return Package{}, fmt.Errorf(
			"mixed semantic shard context disagrees with projection %s",
			projection.id,
		)
	}
	var normalized normalizedPackageBuilder
	merge := mixedBinaryShardMerge{
		checker: checkerShard, provider: providerShard,
		checkerAuthority:  checkerAuthority,
		providerAuthority: providerAuthority,
		projection:        projection, normalized: &normalized,
	}
	if err := merge.records(checkerEntry, providerEntry); err != nil {
		return Package{}, err
	}
	if err := checkerShard.decoder.identityUses.complete(); err != nil {
		return Package{}, err
	}
	if err := providerShard.decoder.identityUses.complete(); err != nil {
		return Package{}, err
	}
	if err := checkerShard.decoder.finish(); err != nil {
		return Package{}, err
	}
	if err := providerShard.decoder.finish(); err != nil {
		return Package{}, err
	}
	return newPackageFromBuilder(
		projection.id,
		projection.provenance,
		&normalized,
	)
}

type mixedBinaryShardMerge struct {
	checker           binarySemanticShard
	provider          binarySemanticShard
	checkerAuthority  Authority
	providerAuthority Authority
	projection        packageProjection
	normalized        *normalizedPackageBuilder
}

func (merge *mixedBinaryShardMerge) records(
	checkerEntry packageShardManifest,
	providerEntry packageShardManifest,
) error {
	if err := merge.definitions(
		checkerEntry.DefinitionCount,
		providerEntry.DefinitionCount,
	); err != nil {
		return err
	}
	if err := merge.resolutions(
		checkerEntry.ResolutionCount,
		providerEntry.ResolutionCount,
	); err != nil {
		return err
	}
	if err := merge.declarations(
		checkerEntry.DeclarationCount,
		providerEntry.DeclarationCount,
	); err != nil {
		return err
	}
	if err := merge.bindings(
		checkerEntry.BindingCount,
		providerEntry.BindingCount,
	); err != nil {
		return err
	}
	if err := merge.types(
		checkerEntry.TypeCount,
		providerEntry.TypeCount,
	); err != nil {
		return err
	}
	if err := merge.operations(
		checkerEntry.OperationCount,
		providerEntry.OperationCount,
	); err != nil {
		return err
	}
	return merge.unsupported(
		checkerEntry.UnsupportedCount,
		providerEntry.UnsupportedCount,
	)
}
