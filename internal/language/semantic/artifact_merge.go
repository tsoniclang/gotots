package semantic

import (
	"encoding/json"
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
	checkerDecoder := json.NewDecoder(checkerInput)
	checkerDecoder.DisallowUnknownFields()
	providerDecoder := json.NewDecoder(providerInput)
	providerDecoder.DisallowUnknownFields()
	var normalized normalizedPackageBuilder
	checkerShard, err := beginNormalizedShard(
		checkerDecoder,
		checkerEntry,
		&normalized,
	)
	if err != nil {
		return Package{}, err
	}
	providerShard, err := beginNormalizedShard(
		providerDecoder,
		providerEntry,
		&normalized,
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
	merge := mixedShardMerge{
		checker:           checkerShard,
		provider:          providerShard,
		checkerAuthority:  checkerAuthority,
		providerAuthority: providerAuthority,
		projection:        projection,
		normalized:        &normalized,
	}
	if err := merge.records(checkerEntry, providerEntry); err != nil {
		return Package{}, err
	}
	if err := finishNormalizedShard(checkerShard); err != nil {
		return Package{}, err
	}
	if err := finishNormalizedShard(providerShard); err != nil {
		return Package{}, err
	}
	return newPackageFromBuilder(
		projection.id,
		projection.provenance,
		&normalized,
	)
}

type mixedShardMerge struct {
	checker           normalizedShardDecoder
	provider          normalizedShardDecoder
	checkerAuthority  Authority
	providerAuthority Authority
	projection        packageProjection
	normalized        *normalizedPackageBuilder
}

func (merge *mixedShardMerge) records(
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
