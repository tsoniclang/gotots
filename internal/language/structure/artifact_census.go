package structure

import (
	"sort"

	"github.com/tsoniclang/gotots/internal/identity"
)

// ProviderPackageCensus is the bounded manifest projection needed before a
// certified package's detailed structural shard is loaded.
type ProviderPackageCensus struct {
	pkg               identity.PackageID
	definitions       []identity.DefinitionID
	headerOccurrences int
	boundaryEntries   int
}

// ProviderManifestStats are the bounded logical and physical denominators
// available without decoding a detailed package shard.
type ProviderManifestStats struct {
	PackageContexts       int
	Files                 int
	SyntheticPackages     int
	Definitions           int
	HeaderOccurrences     int
	BoundaryEntries       int
	SelectionFacts        int
	LargestShardBytes     int64
	LargestPackageRecords int
	largestShards         []ProviderPackageSize
}

func (s ProviderManifestStats) LargestShards() []ProviderPackageSize {
	return append([]ProviderPackageSize(nil), s.largestShards...)
}

func newProviderPackageCensus(
	pkg identity.PackageID,
	definitions []identity.DefinitionID,
	headerOccurrences int,
	boundaryEntries int,
) (ProviderPackageCensus, error) {
	if pkg.IsZero() || headerOccurrences < 0 || boundaryEntries < 0 {
		return ProviderPackageCensus{}, providerArtifactError(
			"provider package census has invalid identity or counts",
		)
	}
	canonical := append([]identity.DefinitionID(nil), definitions...)
	sort.Slice(canonical, func(i, j int) bool {
		return canonical[i].String() < canonical[j].String()
	})
	previous := ""
	for _, definition := range canonical {
		if definition.IsZero() || definition.String() <= previous {
			return ProviderPackageCensus{}, providerArtifactError(
				"provider package census has duplicate definition",
			)
		}
		previous = definition.String()
	}
	return ProviderPackageCensus{
		pkg:               pkg,
		definitions:       canonical,
		headerOccurrences: headerOccurrences,
		boundaryEntries:   boundaryEntries,
	}, nil
}

func (c ProviderPackageCensus) Package() identity.PackageID {
	return c.pkg
}

func (c ProviderPackageCensus) Definitions() []identity.DefinitionID {
	return append([]identity.DefinitionID(nil), c.definitions...)
}

func (c ProviderPackageCensus) HeaderOccurrenceCount() int {
	return c.headerOccurrences
}

func (c ProviderPackageCensus) BoundaryEntryCount() int {
	return c.boundaryEntries
}

func (a *ProviderArtifact) PackageCensus(
	pkg identity.PackageID,
) (ProviderPackageCensus, bool) {
	if a == nil {
		return ProviderPackageCensus{}, false
	}
	census, present := a.packageCensus[pkg]
	if !present {
		return ProviderPackageCensus{}, false
	}
	census.definitions = append(
		[]identity.DefinitionID(nil),
		census.definitions...,
	)
	return census, true
}

// ManifestStats reports only manifest-resident evidence. It never opens a
// package shard or inspects detailed structural payload.
func (a *ProviderArtifact) ManifestStats() ProviderManifestStats {
	if a == nil {
		return ProviderManifestStats{}
	}
	stats := ProviderManifestStats{
		PackageContexts:   len(a.packageDigests),
		Files:             len(a.filePackages),
		SyntheticPackages: len(a.syntheticPackages),
		SelectionFacts:    a.factCount,
	}
	for _, census := range a.packageCensus {
		stats.Definitions += len(census.definitions)
		stats.HeaderOccurrences += census.headerOccurrences
		stats.BoundaryEntries += census.boundaryEntries
	}
	if a.storage != nil {
		for packageID, shard := range a.storage.shards {
			if shard.bytes > stats.LargestShardBytes {
				stats.LargestShardBytes = shard.bytes
			}
			records := 1 +
				len(shard.files) +
				len(shard.census.definitions) +
				shard.census.headerOccurrences +
				shard.census.boundaryEntries +
				len(a.factsByPackage[packageID])
			if records > stats.LargestPackageRecords {
				stats.LargestPackageRecords = records
			}
			stats.largestShards = append(
				stats.largestShards,
				ProviderPackageSize{
					Package: packageID,
					Bytes:   shard.bytes,
					Records: records,
				},
			)
		}
		sortProviderPackageSizes(stats.largestShards)
		if len(stats.largestShards) > providerTailLimit {
			stats.largestShards =
				stats.largestShards[:providerTailLimit]
		}
	}
	return stats
}

func sealProviderPackageCensus(artifact *ProviderArtifact) error {
	for packageID := range artifact.packageDigests {
		var definitions []identity.DefinitionID
		headerOccurrences := 0
		boundaryEntries := 0
		for _, file := range artifact.packageFiles[packageID] {
			graph, present := artifact.fileGraphs[file]
			if !present {
				return providerArtifactError(
					"provider census lacks file graph " + file.String(),
				)
			}
			for _, definition := range graph.definitions {
				definitions = append(definitions, definition.id)
			}
			for _, header := range graph.headers {
				headerOccurrences += len(header.members)
			}
			for _, boundary := range graph.boundaries {
				boundaryEntries += len(boundary.entries)
			}
		}
		if artifact.syntheticPackages[packageID] {
			graph, present := artifact.packageGraphs[packageID]
			if !present {
				return providerArtifactError(
					"provider census lacks synthetic graph " +
						packageID.String(),
				)
			}
			for _, definition := range graph.ownedDefinitions {
				definitions = append(definitions, definition.id)
			}
			for _, header := range graph.ownedHeaders {
				headerOccurrences += len(header.members)
			}
			for _, boundary := range graph.ownedBoundaries {
				boundaryEntries += len(boundary.entries)
			}
		}
		census, err := newProviderPackageCensus(
			packageID,
			definitions,
			headerOccurrences,
			boundaryEntries,
		)
		if err != nil {
			return err
		}
		artifact.packageCensus[packageID] = census
	}
	return nil
}

func sameProviderPackageCensus(
	left ProviderPackageCensus,
	right ProviderPackageCensus,
) bool {
	if left.pkg != right.pkg ||
		left.headerOccurrences != right.headerOccurrences ||
		left.boundaryEntries != right.boundaryEntries ||
		len(left.definitions) != len(right.definitions) {
		return false
	}
	for index := range left.definitions {
		if left.definitions[index] != right.definitions[index] {
			return false
		}
	}
	return true
}
