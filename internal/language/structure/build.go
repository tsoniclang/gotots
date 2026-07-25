package structure

import (
	"fmt"
	"sort"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/scope/sourceplan"
	"github.com/tsoniclang/gotots/internal/source"
)

// Build derives the complete depth-independent graph from local selected
// syntax. It never chooses provider or evidence depth.
func Build(universe *source.Universe) (*Graph, *TransientIndex, error) {
	return build(universe, nil, nil, nil)
}

// BuildPackages derives an all-local graph for exactly the named package set.
// It is the bounded provider-audit route; ordinary compilation uses
// BuildPlanned.
func BuildPackages(
	universe *source.Universe,
	packageIDs []identity.PackageID,
) (*Graph, *TransientIndex, error) {
	selected := map[identity.PackageID]bool{}
	for _, packageID := range packageIDs {
		if packageID.IsZero() || selected[packageID] {
			return nil, nil, fmt.Errorf(
				"package graph selection is invalid or duplicated",
			)
		}
		selected[packageID] = true
	}
	if len(selected) == 0 {
		return nil, nil, fmt.Errorf(
			"package graph selection is empty",
		)
	}
	return build(universe, nil, nil, selected)
}

// BuildPlanned derives local file graphs and admits certified provider file
// graphs according to the complete source plan. A certified file is never
// traversed by this producer.
func BuildPlanned(
	universe *source.Universe,
	plan *sourceplan.Plan,
	artifact *ProviderArtifact,
) (*Graph, *TransientIndex, error) {
	if plan == nil {
		return nil, nil, fmt.Errorf(
			"planned structure build requires a source plan",
		)
	}
	if plan.Purpose() != sourceplan.PurposeCompilation {
		return nil, nil, fmt.Errorf(
			"planned structure build requires a compilation source plan",
		)
	}
	return build(universe, plan, artifact, nil)
}

func build(
	universe *source.Universe,
	plan *sourceplan.Plan,
	artifact *ProviderArtifact,
	selectedPackages map[identity.PackageID]bool,
) (*Graph, *TransientIndex, error) {
	if universe == nil || !universe.Hydrated() {
		return nil, nil, fmt.Errorf(
			"structure build requires the selectively hydrated universe",
		)
	}
	graph := &Graph{
		version:  ArtifactVersion,
		provider: artifact,
	}
	index, err := newTransientIndex(universe)
	if err != nil {
		return nil, nil, err
	}
	for _, loadedPackage := range universe.Packages() {
		if selectedPackages != nil &&
			!selectedPackages[loadedPackage.ID()] {
			continue
		}
		if loadedPackage.Disposition() != source.DispositionOrdinarySource &&
			loadedPackage.Disposition() != source.DispositionUnsafeIntrinsic {
			continue
		}
		pkg := PackageGraph{id: loadedPackage.ID()}
		projection := packageProjection{id: loadedPackage.ID()}
		localCgo := false
		for _, file := range loadedPackage.Files() {
			decision := sourceplan.KindLocalSyntax
			if plan != nil {
				planned, present := plan.For(file.ID())
				if !present {
					return nil, nil, fmt.Errorf(
						"source plan omits %s", file.ID(),
					)
				}
				decision = planned.Kind()
			}
			switch decision {
			case sourceplan.KindCertifiedGraph:
				if artifact == nil ||
					!artifact.HasPackageFile(
						loadedPackage.ID(),
						file.ID(),
					) {
					return nil, nil, fmt.Errorf(
						"provider graph omits %s", file.ID(),
					)
				}
				projection.certifiedFiles = append(
					projection.certifiedFiles,
					certifiedFileProjection{
						id:         file.ID(),
						byteDigest: file.ByteDigest().String(),
					},
				)
			case sourceplan.KindLocalSyntax:
				syntax := file.PhysicalSyntax()
				if syntax == nil {
					return nil, nil, fmt.Errorf(
						"local structural source %s has no syntax", file.ID(),
					)
				}
				built, err := buildFile(
					file, syntax, &graph.work, index,
				)
				if err != nil {
					return nil, nil, err
				}
				appendFileGraph(&pkg, built)
				localCgo = localCgo || file.CgoOriginal()
			default:
				return nil, nil, fmt.Errorf(
					"source plan has invalid decision for %s", file.ID(),
				)
			}
		}
		localSynthetic, certifiedSynthetic, err :=
			structuralSyntheticSource(
				plan, artifact, loadedPackage.ID(),
			)
		if err != nil {
			return nil, nil, err
		}
		projection.certifiedSynthetic = certifiedSynthetic
		if localCgo || plan == nil {
			if err := attachCgo(
				universe,
				loadedPackage,
				&pkg,
				index,
				&graph.work,
				localSynthetic,
			); err != nil {
				return nil, nil, err
			}
		}
		if loadedPackage.Disposition() == source.DispositionOrdinarySource {
			if err := addPackageInitialization(&pkg, &graph.work); err != nil {
				return nil, nil, err
			}
		}
		if err := validatePackageGraph(pkg); err != nil {
			return nil, nil, err
		}
		graph.packages = append(graph.packages, pkg)
		graph.projections = append(graph.projections, projection)
	}
	if selectedPackages != nil &&
		len(graph.projections) != len(selectedPackages) {
		return nil, nil, fmt.Errorf(
			"package graph selection resolved %d of %d packages",
			len(graph.projections), len(selectedPackages),
		)
	}
	sortGraphPackages(graph)
	if err := sealGraph(graph); err != nil {
		return nil, nil, err
	}
	if err := sealDefinitionCensus(graph); err != nil {
		return nil, nil, err
	}
	if err := Validate(graph); err != nil {
		return nil, nil, err
	}
	return graph, index, nil
}

func structuralSyntheticSource(
	plan *sourceplan.Plan,
	artifact *ProviderArtifact,
	pkg identity.PackageID,
) (bool, bool, error) {
	if plan == nil {
		return true, false, nil
	}
	decision, planned := plan.SyntheticFor(pkg)
	if !planned {
		return false, false, nil
	}
	switch decision.Kind() {
	case sourceplan.KindLocalSyntax:
		return true, false, nil
	case sourceplan.KindCertifiedGraph:
		if artifact == nil || !artifact.HasSyntheticPackage(pkg) {
			return false, false, fmt.Errorf(
				"certified synthetic owner %s has no artifact", pkg,
			)
		}
		return false, true, nil
	default:
		return false, false, fmt.Errorf(
			"invalid synthetic source decision for %s", pkg,
		)
	}
}

func appendFileGraph(pkg *PackageGraph, file FileGraph) {
	pkg.files = append(pkg.files, file)
}

func addPackageInitialization(pkg *PackageGraph, work *Work) error {
	ownerID, err := SyntheticOwner(
		pkg.id, SyntheticOwnerPackageInitialization,
	)
	if err != nil {
		return err
	}
	definitionID, err := identity.NewImplicitDefinitionID(
		pkg.id, identity.ImplicitDefinitionPackageInit,
	)
	if err != nil {
		return err
	}
	headerID, err := identity.NewHeaderRegionID(definitionID)
	if err != nil {
		return err
	}
	boundaryID, err := identity.NewExecutionBoundaryID(definitionID)
	if err != nil {
		return err
	}
	pkg.synthetic = append(pkg.synthetic, OwnerRegion{id: ownerID})
	pkg.ownedDefinitions = append(
		pkg.ownedDefinitions,
		ImplementationDefinition{
			id: definitionID, owner: ownerID, header: headerID,
			boundary: boundaryID, name: "package initialization",
		},
	)
	pkg.ownedSites = append(pkg.ownedSites, DefinitionSite{
		kind: DefinitionSiteSynthetic, definition: definitionID,
		owner: ownerID,
	})
	pkg.ownedHeaders = append(pkg.ownedHeaders, HeaderRegion{
		id:     headerID,
		digest: digestStrings(definitionID.String(), "header"),
	})
	pkg.ownedBoundaries = append(pkg.ownedBoundaries, ExecutionBoundary{
		id: boundaryID, kind: BoundaryImplicit,
		implicit:       identity.ImplicitDefinitionPackageInit,
		combinedDigest: digestStrings(definitionID.String(), "execution"),
	})
	work.RecordAppends += 5
	return nil
}

func sortGraphPackages(graph *Graph) {
	type entry struct {
		pkg        PackageGraph
		projection packageProjection
	}
	entries := make([]entry, len(graph.packages))
	for index := range graph.packages {
		entries[index] = entry{
			pkg:        graph.packages[index],
			projection: graph.projections[index],
		}
	}
	sort.Slice(entries, func(i, j int) bool {
		graph.work.SortComparisons++
		return entries[i].pkg.id.Compare(entries[j].pkg.id) < 0
	})
	for index := range entries {
		graph.packages[index] = entries[index].pkg
		graph.projections[index] = entries[index].projection
	}
}

func sealGraph(graph *Graph) error {
	graph.occurrenceStores = map[identity.FileID]*OccurrenceStore{}
	graph.byDefinition = map[identity.DefinitionID]*ImplementationDefinition{}
	graph.byBoundary = map[identity.DefinitionID]*ExecutionBoundary{}
	for packageIndex := range graph.packages {
		pkg := &graph.packages[packageIndex]
		for fileIndex := range pkg.files {
			file := &pkg.files[fileIndex]
			fileID := file.owner.id.file
			if file.occurrences == nil || fileID.IsZero() {
				return fmt.Errorf(
					"file graph has no canonical occurrence store",
				)
			}
			if _, duplicate := graph.occurrenceStores[fileID]; duplicate {
				return fmt.Errorf(
					"file occurrence store %s is retained more than once",
					fileID,
				)
			}
			graph.occurrenceStores[fileID] = file.occurrences
			if err := file.VisitOccurrenceRefs(func(
				occurrence OccurrenceRef,
			) error {
				graph.work.IdentityProbes++
				graph.occurrenceOrder = append(
					graph.occurrenceOrder,
					occurrence,
				)
				graph.work.RecordAppends++
				return nil
			}); err != nil {
				return err
			}
			for definitionIndex := range file.definitions {
				if err := indexDefinition(
					graph, &file.definitions[definitionIndex],
				); err != nil {
					return err
				}
			}
			for boundaryIndex := range file.boundaries {
				if err := indexBoundary(
					graph, &file.boundaries[boundaryIndex],
				); err != nil {
					return err
				}
			}
		}
		for definitionIndex := range pkg.ownedDefinitions {
			if err := indexDefinition(
				graph, &pkg.ownedDefinitions[definitionIndex],
			); err != nil {
				return err
			}
		}
		for boundaryIndex := range pkg.ownedBoundaries {
			if err := indexBoundary(
				graph, &pkg.ownedBoundaries[boundaryIndex],
			); err != nil {
				return err
			}
		}
	}
	sort.Slice(graph.occurrenceOrder, func(i, j int) bool {
		graph.work.SortComparisons++
		return graph.occurrenceOrder[i].ID().Compare(
			graph.occurrenceOrder[j].ID(),
		) < 0
	})
	sort.Slice(graph.definitionIDs, func(i, j int) bool {
		graph.work.SortComparisons++
		return graph.definitionIDs[i].Compare(
			graph.definitionIDs[j],
		) < 0
	})
	return nil
}

func indexDefinition(
	graph *Graph,
	definition *ImplementationDefinition,
) error {
	graph.work.IdentityProbes++
	if _, duplicate := graph.byDefinition[definition.id]; duplicate {
		return fmt.Errorf("duplicate definition %s", definition.id)
	}
	graph.byDefinition[definition.id] = definition
	graph.definitionIDs = append(graph.definitionIDs, definition.id)
	graph.work.RecordAppends++
	return nil
}

func indexBoundary(
	graph *Graph,
	boundary *ExecutionBoundary,
) error {
	graph.work.IdentityProbes++
	definition := boundary.id.Definition()
	if _, duplicate := graph.byBoundary[definition]; duplicate {
		return fmt.Errorf("duplicate execution boundary %s", definition)
	}
	graph.byBoundary[definition] = boundary
	return nil
}
