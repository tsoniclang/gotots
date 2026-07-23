package executable

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/structure"
	"github.com/tsoniclang/gotots/internal/scope"
	"github.com/tsoniclang/gotots/internal/scope/contract"
)

// Validate proves the executable artifact is the exact full-selection overlay
// of the depth-independent structural graph.
func Validate(
	graph *structure.Graph,
	selections *scope.DefinitionSelections,
	inventory *Inventory,
) error {
	if graph == nil || selections == nil || inventory == nil {
		return fmt.Errorf("executable validation requires all Stage-1 artifacts")
	}
	definitions := map[identity.DefinitionID]structure.ImplementationDefinition{}
	sites := map[identity.DefinitionID]structure.DefinitionSite{}
	boundaries := map[identity.DefinitionID]structure.ExecutionBoundary{}
	census := map[identity.DefinitionID]bool{}
	for _, record := range graph.DefinitionCensus() {
		census[record.ID()] = true
	}
	if err := graph.VisitResidentPackages(func(
		pkg structure.PackageGraph,
	) error {
		for _, definition := range pkg.Definitions() {
			definitions[definition.ID()] = definition
		}
		for _, site := range pkg.Sites() {
			sites[site.Definition()] = site
		}
		for _, boundary := range pkg.Boundaries() {
			boundaries[boundary.ID().Definition()] = boundary
		}
		return nil
	}); err != nil {
		return err
	}
	if err := validateInventoryIndexes(inventory); err != nil {
		return err
	}
	for id := range inventory.byOccurrence {
		if _, duplicated := graph.ResidentOccurrence(id); duplicated {
			return fmt.Errorf(
				"occurrence %s is duplicated across structural and executable stores",
				id,
			)
		}
	}
	full := map[identity.DefinitionID]bool{}
	for _, selection := range selections.Records() {
		if !census[selection.Definition()] {
			return fmt.Errorf(
				"selection names absent definition %s",
				selection.Definition(),
			)
		}
		if selection.Depth() == contract.DepthFullSemantic {
			full[selection.Definition()] = true
		}
	}
	if len(selections.Records()) != len(census) {
		return fmt.Errorf(
			"selection/definition cardinality is %d/%d",
			len(selections.Records()), len(census),
		)
	}
	memberOwner := map[identity.OccurrenceID]identity.DefinitionID{}
	referenceCount := map[identity.DefinitionID]int{}
	for _, regionID := range inventory.regionIDs {
		region := inventory.byID[regionID]
		definition := region.Definition()
		if !full[definition] {
			return fmt.Errorf(
				"non-full definition %s owns an executable region",
				definition,
			)
		}
		if err := validateRegion(
			graph,
			inventory,
			definitions,
			sites,
			boundaries[definition],
			region,
			memberOwner,
			referenceCount,
		); err != nil {
			return err
		}
	}
	if len(inventory.regionIDs) != len(full) {
		return fmt.Errorf(
			"full-selection/executable-region cardinality is %d/%d",
			len(full), len(inventory.regionIDs),
		)
	}
	for definition := range full {
		if _, present := inventory.byID[definition]; !present {
			return fmt.Errorf(
				"full definition %s has no executable region", definition,
			)
		}
	}
	for definition, site := range sites {
		want := 0
		if !site.ParentDefinition().IsZero() &&
			full[site.ParentDefinition()] {
			want = 1
		}
		if referenceCount[definition] != want {
			return fmt.Errorf(
				"definition %s has %d executable references, want %d",
				definition, referenceCount[definition], want,
			)
		}
	}
	for id := range inventory.byOccurrence {
		if memberOwner[id].IsZero() {
			return fmt.Errorf(
				"additional occurrence %s belongs to no executable region",
				id,
			)
		}
	}
	return nil
}

func validateInventoryIndexes(inventory *Inventory) error {
	if len(inventory.regionIDs) != len(inventory.byID) ||
		len(inventory.additionalIDs) != len(inventory.byOccurrence) {
		return fmt.Errorf("executable indexes have unequal cardinalities")
	}
	previous := ""
	for _, regionID := range inventory.regionIDs {
		region := inventory.byID[regionID]
		if region.id.IsZero() ||
			previous >= region.Definition().String() {
			return fmt.Errorf(
				"executable region order is noncanonical at %s",
				region.Definition(),
			)
		}
		previous = region.Definition().String()
		indexed, present := inventory.byID[region.Definition()]
		if !present || indexed.id != region.id {
			return fmt.Errorf(
				"executable region index disagrees at %s",
				region.Definition(),
			)
		}
	}
	previous = ""
	for _, id := range inventory.additionalIDs {
		occurrence := inventory.byOccurrence[id]
		if previous >= occurrence.ID().String() {
			return fmt.Errorf(
				"additional occurrence order is noncanonical at %s",
				occurrence.ID(),
			)
		}
		previous = occurrence.ID().String()
		indexed, present := inventory.byOccurrence[occurrence.ID()]
		if !present || indexed != occurrence {
			return fmt.Errorf(
				"additional occurrence index disagrees at %s",
				occurrence.ID(),
			)
		}
	}
	return nil
}

func validateRegion(
	graph *structure.Graph,
	inventory *Inventory,
	definitions map[identity.DefinitionID]structure.ImplementationDefinition,
	sites map[identity.DefinitionID]structure.DefinitionSite,
	boundary structure.ExecutionBoundary,
	region Region,
	memberOwner map[identity.OccurrenceID]identity.DefinitionID,
	referenceCount map[identity.DefinitionID]int,
) error {
	definition, present := definitions[region.Definition()]
	if !present || boundary.ID().Definition() != definition.ID() {
		return fmt.Errorf(
			"executable region names absent or incoherent definition %s",
			region.Definition(),
		)
	}
	if definition.Kind() == identity.DefinitionImplicit {
		return validateImplicitRegion(definition, region)
	}
	if len(region.implicit) != 0 {
		return fmt.Errorf(
			"source definition %s contains implicit executable operations",
			definition.ID(),
		)
	}
	entries := map[identity.OccurrenceID]bool{}
	for _, entry := range boundary.Entries() {
		entries[entry.ID()] = true
	}
	local := map[identity.OccurrenceID]bool{}
	for _, member := range region.members {
		if prior := memberOwner[member]; !prior.IsZero() {
			return fmt.Errorf(
				"occurrence %s belongs to executable regions %s and %s",
				member, prior, definition.ID(),
			)
		}
		occurrence, found := executableOccurrence(
			graph, inventory, member,
		)
		if !found {
			return fmt.Errorf(
				"region %s member %s has no canonical occurrence",
				region.id, member,
			)
		}
		if local[member] {
			return fmt.Errorf(
				"region %s repeats occurrence %s", region.id, member,
			)
		}
		if !entries[member] && !local[occurrence.Parent()] {
			return fmt.Errorf(
				"region %s member %s precedes or omits parent %s",
				region.id, member, occurrence.Parent(),
			)
		}
		local[member] = true
		memberOwner[member] = definition.ID()
	}
	seenReferences := map[identity.DefinitionID]bool{}
	for _, reference := range region.references {
		child, found := definitions[reference.child]
		site := sites[reference.child]
		if !found ||
			site.ParentDefinition() != definition.ID() ||
			seenReferences[reference.child] {
			return fmt.Errorf(
				"region %s has invalid or duplicate child reference %s",
				region.id, reference.child,
			)
		}
		root, found := graph.ResidentOccurrence(child.ID().Root())
		if !found ||
			root.Parent() != reference.parent ||
			root.Edge() != reference.edge ||
			root.Ordinal() != reference.ordinal ||
			(reference.parent != definition.ID().Root() &&
				!local[reference.parent]) {
			return fmt.Errorf(
				"region %s has incoherent child reference %s",
				region.id, reference.child,
			)
		}
		seenReferences[reference.child] = true
		referenceCount[reference.child]++
	}
	return nil
}

func validateImplicitRegion(
	definition structure.ImplementationDefinition,
	region Region,
) error {
	if len(region.members) != 0 ||
		len(region.references) != 0 ||
		len(region.implicit) != 1 {
		return fmt.Errorf(
			"implicit definition %s has a source-shaped executable region",
			definition.ID(),
		)
	}
	operation := region.implicit[0]
	if definition.ID().ImplicitOp() !=
		identity.ImplicitDefinitionPackageInit ||
		operation.kind !=
			ImplicitOperationCoordinatePackageInitialization ||
		operation.pkg != definition.ID().Package() {
		return fmt.Errorf(
			"implicit definition %s has an invalid operation graph",
			definition.ID(),
		)
	}
	return nil
}

func executableOccurrence(
	graph *structure.Graph,
	inventory *Inventory,
	id identity.OccurrenceID,
) (structure.Occurrence, bool) {
	if occurrence, present := graph.ResidentOccurrence(id); present {
		return occurrence, true
	}
	return inventory.AdditionalOccurrence(id)
}
