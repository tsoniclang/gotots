package structure

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/identity"
)

func validatePackageStorage(pkg PackageGraph) error {
	files := map[identity.FileID]bool{}
	for _, file := range pkg.files {
		fileID := file.owner.id.file
		if fileID.IsZero() || files[fileID] {
			return fmt.Errorf(
				"package %s has zero or duplicate file graph %s",
				pkg.id, fileID,
			)
		}
		files[fileID] = true
		for _, definition := range file.definitions {
			if definition.id.File() != fileID {
				return fmt.Errorf(
					"file %s stores definition %s owned elsewhere",
					fileID, definition.id,
				)
			}
		}
	}
	owners := map[OwnerRegionID]bool{}
	for _, owner := range pkg.synthetic {
		if owners[owner.id] {
			return fmt.Errorf(
				"package %s repeats synthetic owner %s", pkg.id, owner.id,
			)
		}
		owners[owner.id] = true
	}
	ownerUse := map[OwnerRegionID]int{}
	for _, definition := range pkg.ownedDefinitions {
		if !definition.id.File().IsZero() ||
			definition.id.Package() != pkg.id ||
			!owners[definition.owner] {
			return fmt.Errorf(
				"package %s stores invalid package-owned definition %s",
				pkg.id, definition.id,
			)
		}
		switch {
		case definition.id.ImplicitOp().Valid():
			if definition.owner.synthetic !=
				SyntheticOwnerPackageInitialization {
				return fmt.Errorf(
					"implicit definition %s has wrong owner %s",
					definition.id, definition.owner,
				)
			}
		case definition.id.SyntheticRole().Valid():
			if definition.owner.synthetic != SyntheticOwnerCgoGenerated {
				return fmt.Errorf(
					"synthetic definition %s has wrong owner %s",
					definition.id, definition.owner,
				)
			}
		default:
			return fmt.Errorf(
				"package-owned definition %s has no closed identity role",
				definition.id,
			)
		}
		ownerUse[definition.owner]++
	}
	for owner := range owners {
		if ownerUse[owner] == 0 {
			return fmt.Errorf(
				"package %s has unused synthetic owner %s", pkg.id, owner,
			)
		}
	}
	if len(pkg.ownedDefinitions) != len(pkg.ownedSites) ||
		len(pkg.ownedDefinitions) != len(pkg.ownedHeaders) ||
		len(pkg.ownedDefinitions) != len(pkg.ownedBoundaries) {
		return fmt.Errorf(
			"package %s has incoherent package-owned record counts",
			pkg.id,
		)
	}
	return nil
}

func validateSealedIndexes(graph *Graph) error {
	if len(graph.occurrenceIDs) != len(graph.byOccurrence) ||
		len(graph.definitionIDs) != len(graph.byDefinition) ||
		len(graph.byBoundary) != len(graph.byDefinition) {
		return fmt.Errorf("sealed structural indexes have unequal cardinalities")
	}
	var previousOccurrence identity.OccurrenceID
	for _, id := range graph.occurrenceIDs {
		occurrence := graph.byOccurrence[id]
		if !previousOccurrence.IsZero() &&
			previousOccurrence.Compare(occurrence.id) >= 0 {
			return fmt.Errorf(
				"canonical occurrence index is not strictly ordered at %s",
				occurrence.id,
			)
		}
		previousOccurrence = occurrence.id
		indexed, present := graph.byOccurrence[occurrence.id]
		if !present || indexed != occurrence {
			return fmt.Errorf(
				"canonical occurrence index disagrees at %s", occurrence.id,
			)
		}
	}
	var previousDefinition identity.DefinitionID
	for _, id := range graph.definitionIDs {
		definition := graph.byDefinition[id]
		if !previousDefinition.IsZero() &&
			previousDefinition.Compare(definition.id) >= 0 {
			return fmt.Errorf(
				"definition index is not strictly ordered at %s", definition.id,
			)
		}
		previousDefinition = definition.id
		indexed, present := graph.byDefinition[definition.id]
		if !present || indexed != definition {
			return fmt.Errorf(
				"definition index disagrees at %s", definition.id,
			)
		}
		boundary, present := graph.byBoundary[definition.id]
		if !present || boundary.id.Definition() != definition.id {
			return fmt.Errorf(
				"boundary index disagrees at %s", definition.id,
			)
		}
	}
	for index := 1; index < len(graph.packages); index++ {
		if graph.packages[index-1].id.Compare(
			graph.packages[index].id,
		) >= 0 {
			return fmt.Errorf("package graph order is not canonical")
		}
	}
	return nil
}

func validateDefinitionForest(
	sites map[identity.DefinitionID]DefinitionSite,
	definitions map[identity.DefinitionID]ImplementationDefinition,
) error {
	const (
		active = iota + 1
		done
	)
	state := map[identity.DefinitionID]int{}
	var visit func(identity.DefinitionID) error
	visit = func(definition identity.DefinitionID) error {
		switch state[definition] {
		case active:
			return fmt.Errorf(
				"definition containment forest cycles at %s", definition,
			)
		case done:
			return nil
		}
		state[definition] = active
		site, present := sites[definition]
		if !present {
			return fmt.Errorf("definition %s has no containment site", definition)
		}
		if !site.parentDefinition.IsZero() {
			parent, exists := definitions[site.parentDefinition]
			child := definitions[definition]
			if !exists ||
				parent.owner != child.owner ||
				parent.id.File() != child.id.File() {
				return fmt.Errorf(
					"definition %s has invalid forest parent %s",
					definition, site.parentDefinition,
				)
			}
			if err := visit(site.parentDefinition); err != nil {
				return err
			}
		}
		state[definition] = done
		return nil
	}
	for definition := range definitions {
		if err := visit(definition); err != nil {
			return err
		}
	}
	return nil
}

func validateCheckedMappings(
	file FileGraph,
	all map[identity.OccurrenceID]*Occurrence,
) error {
	definitions := map[identity.DefinitionID]bool{}
	for _, definition := range file.definitions {
		definitions[definition.id] = true
	}
	seen := map[identity.DefinitionID]bool{}
	for _, mapping := range file.mappings {
		if !definitions[mapping.definition] ||
			seen[mapping.definition] ||
			mapping.definition.File() != file.owner.id.file ||
			mapping.originLine <= 0 ||
			mapping.originColumn <= 0 ||
			!mapping.originMatch.Valid() ||
			!validSHA256(mapping.checkedDigest) {
			return fmt.Errorf(
				"file %s has invalid checked mapping for %s",
				file.owner.id.file, mapping.definition,
			)
		}
		if _, present := all[mapping.definition.Root()]; !present {
			return fmt.Errorf(
				"checked mapping definition %s has no root occurrence",
				mapping.definition,
			)
		}
		seen[mapping.definition] = true
	}
	return nil
}
