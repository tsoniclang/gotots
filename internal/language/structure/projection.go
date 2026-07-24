package structure

import (
	"fmt"
	"sort"

	"github.com/tsoniclang/gotots/internal/identity"
)

// PackageCount is the exact number of logical package graphs.
func (g *Graph) PackageCount() int {
	if g == nil {
		return 0
	}
	return len(g.projections)
}

// VisitPackages presents the complete logical graph one package at a time.
// Certified payloads never become an application-wide resident collection.
func (g *Graph) VisitPackages(
	visit func(PackageGraph) error,
) error {
	if g == nil || visit == nil {
		return fmt.Errorf("package projection requires graph and visitor")
	}
	if len(g.packages) != len(g.projections) {
		return fmt.Errorf(
			"structural package projections are not cardinality-aligned",
		)
	}
	for index, projection := range g.projections {
		resident := g.packages[index]
		if resident.id != projection.id {
			return fmt.Errorf(
				"structural package projection order disagrees at %s",
				projection.id,
			)
		}
		pkg, err := g.projectPackage(resident, projection)
		if err != nil {
			return err
		}
		if err := visit(pkg); err != nil {
			return err
		}
	}
	return nil
}

// VisitResidentPackages presents only locally extracted records. It exists for
// stages whose contract is explicitly limited to full-semantic local evidence.
func (g *Graph) VisitResidentPackages(
	visit func(PackageGraph) error,
) error {
	if g == nil || visit == nil {
		return fmt.Errorf(
			"resident package projection requires graph and visitor",
		)
	}
	for _, pkg := range g.packages {
		if err := visit(clonePackageGraph(pkg)); err != nil {
			return err
		}
	}
	return nil
}

func (g *Graph) projectPackage(
	resident PackageGraph,
	projection packageProjection,
) (PackageGraph, error) {
	pkg := clonePackageGraph(resident)
	if len(projection.certifiedFiles) == 0 &&
		!projection.certifiedSynthetic {
		return pkg, nil
	}
	if g.provider == nil {
		return PackageGraph{}, fmt.Errorf(
			"package %s requires absent certified evidence",
			projection.id,
		)
	}
	shard, err := g.provider.packageArtifact(projection.id)
	if err != nil {
		return PackageGraph{}, err
	}
	for _, expected := range projection.certifiedFiles {
		file, present := shard.fileGraphs[expected.id]
		if !present {
			return PackageGraph{}, fmt.Errorf(
				"provider graph omits %s", expected.id,
			)
		}
		if shard.fileDigests[expected.id] != expected.byteDigest {
			return PackageGraph{}, fmt.Errorf(
				"provider graph byte digest drift for %s", expected.id,
			)
		}
		pkg.files = append(pkg.files, file)
	}
	if projection.certifiedSynthetic {
		synthetic, present := shard.packageGraphs[projection.id]
		if !present {
			return PackageGraph{}, fmt.Errorf(
				"provider graph omits synthetic owner %s",
				projection.id,
			)
		}
		pkg.synthetic = append(pkg.synthetic, synthetic.synthetic...)
		pkg.ownedDefinitions = append(
			pkg.ownedDefinitions, synthetic.ownedDefinitions...,
		)
		pkg.ownedSites = append(
			pkg.ownedSites, synthetic.ownedSites...,
		)
		pkg.ownedHeaders = append(
			pkg.ownedHeaders, synthetic.ownedHeaders...,
		)
		pkg.ownedBoundaries = append(
			pkg.ownedBoundaries, synthetic.ownedBoundaries...,
		)
	}
	sort.Slice(pkg.files, func(i, j int) bool {
		return pkg.files[i].owner.id.file.Compare(
			pkg.files[j].owner.id.file,
		) < 0
	})
	return pkg, nil
}

func clonePackageGraph(source PackageGraph) PackageGraph {
	return PackageGraph{
		id:        source.id,
		files:     append([]FileGraph(nil), source.files...),
		synthetic: append([]OwnerRegion(nil), source.synthetic...),
		ownedDefinitions: append(
			[]ImplementationDefinition(nil),
			source.ownedDefinitions...,
		),
		ownedSites: append(
			[]DefinitionSite(nil), source.ownedSites...,
		),
		ownedHeaders: append(
			[]HeaderRegion(nil), source.ownedHeaders...,
		),
		ownedBoundaries: append(
			[]ExecutionBoundary(nil), source.ownedBoundaries...,
		),
	}
}

// ResidentDefinitions are the definitions whose syntax was selected locally.
// Provider definitions are available only through VisitPackages.
func (g *Graph) ResidentDefinitions() []ImplementationDefinition {
	if g == nil {
		return nil
	}
	return g.residentDefinitions()
}

func (g *Graph) ResidentDefinition(
	id identity.DefinitionID,
) (ImplementationDefinition, bool) {
	if g == nil {
		return ImplementationDefinition{}, false
	}
	return g.residentDefinition(id)
}

func (g *Graph) ResidentBoundary(
	id identity.DefinitionID,
) (ExecutionBoundary, bool) {
	if g == nil {
		return ExecutionBoundary{}, false
	}
	return g.residentBoundary(id)
}

func (g *Graph) ResidentOccurrence(
	id identity.OccurrenceID,
) (Occurrence, bool) {
	if g == nil {
		return Occurrence{}, false
	}
	return g.residentOccurrence(id)
}

func (g *Graph) ResidentOccurrences() []Occurrence {
	if g == nil {
		return nil
	}
	return g.residentOccurrences()
}
