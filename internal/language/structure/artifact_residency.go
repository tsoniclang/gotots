package structure

import (
	"encoding/json"
	"sort"

	"github.com/tsoniclang/gotots/internal/identity"
)

const providerTailLimit = 20

// ProviderPackageSize is bounded provider-package cost evidence. It carries no
// structural payload and cannot be used as semantic authority.
type ProviderPackageSize struct {
	Package identity.PackageID
	Bytes   int64
	Records int
}

// HeaderArtifactSize is bounded tail evidence for one canonical header
// artifact. It carries identity and physical encoding cost, never structure.
type HeaderArtifactSize struct {
	Header       identity.HeaderRegionID
	EncodedBytes int
	Occurrences  int
}

// ProviderProjectionStats reports physical package-shard projection without
// exposing or retaining decoded package graphs.
type ProviderProjectionStats struct {
	ShardLoads            int
	CacheHits             int
	MaxResidentPackages   int
	ProjectedPackages     int
	LargestPackageBytes   int64
	LargestPackageRecords int
	largestPackages       []ProviderPackageSize
	largestHeaders        []HeaderArtifactSize
}

func (s ProviderProjectionStats) LargestPackages() []ProviderPackageSize {
	return append([]ProviderPackageSize(nil), s.largestPackages...)
}

func (s ProviderProjectionStats) LargestHeaders() []HeaderArtifactSize {
	return append([]HeaderArtifactSize(nil), s.largestHeaders...)
}

func (a *ProviderArtifact) ProjectionStats() ProviderProjectionStats {
	if a == nil || a.storage == nil {
		return ProviderProjectionStats{}
	}
	a.storage.mu.Lock()
	defer a.storage.mu.Unlock()
	stats := ProviderProjectionStats{
		ShardLoads:          a.storage.shardLoads,
		CacheHits:           a.storage.cacheHits,
		MaxResidentPackages: a.storage.maxResident,
		ProjectedPackages:   len(a.storage.projected),
	}
	for _, record := range a.storage.projected {
		if record.Bytes > stats.LargestPackageBytes {
			stats.LargestPackageBytes = record.Bytes
		}
		if record.Records > stats.LargestPackageRecords {
			stats.LargestPackageRecords = record.Records
		}
		stats.largestPackages = append(stats.largestPackages, record)
	}
	sortProviderPackageSizes(stats.largestPackages)
	if len(stats.largestPackages) > providerTailLimit {
		stats.largestPackages = stats.largestPackages[:providerTailLimit]
	}
	stats.largestHeaders = append(
		stats.largestHeaders,
		a.storage.largestHeaders...,
	)
	return stats
}

func (g *Graph) ProviderProjectionStats() ProviderProjectionStats {
	if g == nil {
		return ProviderProjectionStats{}
	}
	return g.provider.ProjectionStats()
}

func recordProviderProjection(
	storage *providerStorage,
	packageID identity.PackageID,
	shard providerShard,
	artifact *ProviderArtifact,
) {
	storage.shardLoads++
	storage.maxResident = 1
	if storage.projected == nil {
		storage.projected = map[identity.PackageID]ProviderPackageSize{}
	}
	storage.projected[packageID] = ProviderPackageSize{
		Package: packageID,
		Bytes:   shard.bytes,
		Records: providerDetailedRecordCount(artifact),
	}
	storage.largestHeaders = mergeHeaderArtifactSizes(
		storage.largestHeaders,
		providerHeaderArtifactSizes(artifact),
	)
}

func providerDetailedRecordCount(artifact *ProviderArtifact) int {
	if artifact == nil {
		return 0
	}
	records := 0
	for _, file := range artifact.fileGraphs {
		records++
		records += file.OccurrenceCount()
		records += len(file.owner.members)
		records += len(file.owner.directives)
		records += len(file.containment.anchors)
		records += len(file.definitions)
		records += len(file.sites)
		records += len(file.headers)
		records += len(file.boundaries)
		records += len(file.mappings)
		for _, header := range file.headers {
			records += len(header.members)
		}
		for _, boundary := range file.boundaries {
			records += len(boundary.entries)
		}
	}
	for _, pkg := range artifact.packageGraphs {
		records += len(pkg.synthetic)
		records += len(pkg.ownedDefinitions)
		records += len(pkg.ownedSites)
		records += len(pkg.ownedHeaders)
		records += len(pkg.ownedBoundaries)
		for _, header := range pkg.ownedHeaders {
			records += len(header.members)
		}
		for _, boundary := range pkg.ownedBoundaries {
			records += len(boundary.entries)
		}
	}
	return records
}

func sortProviderPackageSizes(records []ProviderPackageSize) {
	sort.Slice(records, func(i, j int) bool {
		if records[i].Bytes != records[j].Bytes {
			return records[i].Bytes > records[j].Bytes
		}
		return records[i].Package.Compare(records[j].Package) < 0
	})
}

// LargestHeaderArtifacts reports the bounded whole-graph header tail after
// all selected provider packages have been projected.
func (g *Graph) LargestHeaderArtifacts() []HeaderArtifactSize {
	if g == nil {
		return nil
	}
	var headers []HeaderArtifactSize
	for _, pkg := range g.packages {
		headers = append(
			headers,
			headerArtifactSizes(pkg.Headers())...,
		)
	}
	if g.provider != nil {
		headers = append(
			headers,
			g.provider.ProjectionStats().LargestHeaders()...,
		)
	}
	return mergeHeaderArtifactSizes(nil, headers)
}

func providerHeaderArtifactSizes(
	artifact *ProviderArtifact,
) []HeaderArtifactSize {
	if artifact == nil {
		return nil
	}
	var headers []HeaderArtifactSize
	for _, file := range artifact.fileGraphs {
		headers = append(
			headers,
			headerArtifactSizes(file.headers)...,
		)
	}
	for _, pkg := range artifact.packageGraphs {
		headers = append(
			headers,
			headerArtifactSizes(pkg.ownedHeaders)...,
		)
	}
	return headers
}

func headerArtifactSizes(
	headers []HeaderRegion,
) []HeaderArtifactSize {
	out := make([]HeaderArtifactSize, 0, len(headers))
	for _, header := range headers {
		raw, err := json.Marshal(artifactHeader{
			ID:      header.id.String(),
			Digest:  header.digest,
			Members: occurrenceStrings(header.members),
		})
		if err != nil {
			panic("canonical header encoding failed: " + err.Error())
		}
		out = append(out, HeaderArtifactSize{
			Header:       header.id,
			EncodedBytes: len(raw),
			Occurrences:  len(header.members),
		})
	}
	return out
}

func mergeHeaderArtifactSizes(
	existing []HeaderArtifactSize,
	added []HeaderArtifactSize,
) []HeaderArtifactSize {
	byID := map[identity.HeaderRegionID]HeaderArtifactSize{}
	for _, record := range append(
		append([]HeaderArtifactSize(nil), existing...),
		added...,
	) {
		current, present := byID[record.Header]
		if !present ||
			record.EncodedBytes > current.EncodedBytes ||
			(record.EncodedBytes == current.EncodedBytes &&
				record.Occurrences > current.Occurrences) {
			byID[record.Header] = record
		}
	}
	out := make([]HeaderArtifactSize, 0, len(byID))
	for _, record := range byID {
		out = append(out, record)
	}
	sort.Slice(out, func(left, right int) bool {
		if out[left].EncodedBytes != out[right].EncodedBytes {
			return out[left].EncodedBytes > out[right].EncodedBytes
		}
		return out[left].Header.Compare(out[right].Header) < 0
	})
	if len(out) > providerTailLimit {
		out = out[:providerTailLimit]
	}
	return out
}
