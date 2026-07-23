package structure

import (
	"fmt"
	"sort"

	"github.com/tsoniclang/gotots/internal/identity"
)

// DefinitionCensusRecord is the bounded identity-only projection of one
// logical definition. Definition payload and occurrence topology remain owned
// by the resident local graph or one certified package shard.
type DefinitionCensusRecord struct {
	pkg identity.PackageID
	id  identity.DefinitionID
}

func (r DefinitionCensusRecord) Package() identity.PackageID {
	return r.pkg
}
func (r DefinitionCensusRecord) ID() identity.DefinitionID {
	return r.id
}

func (g *Graph) DefinitionCensus() []DefinitionCensusRecord {
	if g == nil {
		return nil
	}
	return append([]DefinitionCensusRecord(nil), g.definitions...)
}

func (g *Graph) HasDefinition(id identity.DefinitionID) bool {
	if g == nil {
		return false
	}
	_, present := g.definitionByID[id]
	return present
}

func (g *Graph) HeaderOccurrenceCount() int {
	if g == nil {
		return 0
	}
	return g.headerOccurrences
}

func (g *Graph) BoundaryEntryCount() int {
	if g == nil {
		return 0
	}
	return g.boundaryEntries
}

// ProviderManifestStats returns the bounded certified projection admitted into
// this graph. A zero result means the graph has no certified authority.
func (g *Graph) ProviderManifestStats() ProviderManifestStats {
	if g == nil {
		return ProviderManifestStats{}
	}
	return g.provider.ManifestStats()
}

func sealDefinitionCensus(graph *Graph) error {
	var records []DefinitionCensusRecord
	headerOccurrences := 0
	boundaryEntries := 0
	for index, pkg := range graph.packages {
		for _, definition := range pkg.Definitions() {
			records = append(records, DefinitionCensusRecord{
				pkg: pkg.id,
				id:  definition.id,
			})
		}
		for _, header := range pkg.Headers() {
			headerOccurrences += len(header.members)
		}
		for _, boundary := range pkg.Boundaries() {
			boundaryEntries += len(boundary.entries)
		}
		projection := graph.projections[index]
		if len(projection.certifiedFiles) == 0 &&
			!projection.certifiedSynthetic {
			continue
		}
		if graph.provider == nil {
			return fmt.Errorf(
				"definition census lacks provider for %s",
				pkg.id,
			)
		}
		certified, present := graph.provider.PackageCensus(pkg.id)
		if !present {
			return fmt.Errorf(
				"definition census lacks certified package %s",
				pkg.id,
			)
		}
		certifiedFiles := map[identity.FileID]bool{}
		for _, file := range projection.certifiedFiles {
			certifiedFiles[file.id] = true
		}
		for _, definition := range certified.Definitions() {
			belongs := !definition.File().IsZero() &&
				certifiedFiles[definition.File()]
			if definition.SyntheticRole().Valid() {
				belongs = projection.certifiedSynthetic &&
					definition.Package() == pkg.id
			}
			if !belongs {
				return fmt.Errorf(
					"certified definition %s is outside projection %s",
					definition,
					pkg.id,
				)
			}
			records = append(records, DefinitionCensusRecord{
				pkg: pkg.id,
				id:  definition,
			})
		}
		headerOccurrences += certified.HeaderOccurrenceCount()
		boundaryEntries += certified.BoundaryEntryCount()
	}
	sort.Slice(records, func(i, j int) bool {
		return definitionCensusKey(records[i]) <
			definitionCensusKey(records[j])
	})
	byID := make(
		map[identity.DefinitionID]*DefinitionCensusRecord,
		len(records),
	)
	for index := range records {
		record := &records[index]
		if _, duplicate := byID[record.id]; duplicate {
			return fmt.Errorf(
				"definition census duplicates %s", record.id,
			)
		}
		byID[record.id] = record
	}
	graph.definitions = records
	graph.definitionByID = byID
	graph.headerOccurrences = headerOccurrences
	graph.boundaryEntries = boundaryEntries
	return nil
}

func validateDefinitionCensus(graph *Graph) error {
	if graph.definitionByID == nil ||
		len(graph.definitions) != len(graph.definitionByID) ||
		graph.headerOccurrences < 0 ||
		graph.boundaryEntries < 0 {
		return fmt.Errorf("definition census is absent or incoherent")
	}
	previous := ""
	for index := range graph.definitions {
		record := &graph.definitions[index]
		if record.pkg.IsZero() ||
			record.id.IsZero() ||
			previous >= definitionCensusKey(*record) ||
			graph.definitionByID[record.id] != record {
			return fmt.Errorf(
				"definition census has invalid record %s", record.id,
			)
		}
		previous = definitionCensusKey(*record)
	}
	for _, definition := range graph.residentDefinitions() {
		_, present := graph.definitionByID[definition.id]
		if !present {
			return fmt.Errorf(
				"resident definition %s is absent from census",
				definition.id,
			)
		}
	}
	return nil
}

func definitionCensusKey(record DefinitionCensusRecord) string {
	return record.pkg.String() + "\x00" + record.id.String()
}
