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
		if err := pkg.VisitDefinitions(func(
			definition structure.ImplementationDefinition,
		) error {
			definitions[definition.ID()] = definition
			return nil
		}); err != nil {
			return err
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
	if err := inventory.occurrences.visitPayloads(func(
		_ occurrenceRef,
		occurrence structure.OccurrenceRef,
	) error {
		id := occurrence.ID()
		if _, duplicated := graph.ResidentOccurrence(id); duplicated {
			return fmt.Errorf(
				"occurrence %s is duplicated across structural and executable stores",
				id,
			)
		}
		return nil
	}); err != nil {
		return err
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
	state := newRegionValidationState(inventory.occurrences.length())
	referenceCount := map[identity.DefinitionID]int{}
	for regionIndex, regionID := range inventory.regionIDs {
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
			uint32(regionIndex+1),
			&state,
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
	if err := inventory.occurrences.visitPayloads(func(
		_ occurrenceRef,
		occurrence structure.OccurrenceRef,
	) error {
		member := inventory.occurrences.reference(occurrence.ID())
		if !member.valid() || state.memberOwner[member] == 0 {
			return fmt.Errorf(
				"additional occurrence %s belongs to no executable region",
				occurrence.ID(),
			)
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}

type regionValidationState struct {
	memberOwner []uint32
	local       []uint32
	entry       []uint32
	generation  uint32
}

func newRegionValidationState(
	memberCount int,
) regionValidationState {
	return regionValidationState{
		memberOwner: make([]uint32, memberCount+1),
		local:       make([]uint32, memberCount+1),
		entry:       make([]uint32, memberCount+1),
	}
}

func (state *regionValidationState) nextRegion() {
	state.generation++
	if state.generation == 0 {
		clear(state.local)
		clear(state.entry)
		state.generation = 1
	}
}

func validateInventoryIndexes(inventory *Inventory) error {
	if len(inventory.regionIDs) != len(inventory.byID) ||
		len(inventory.additional) != inventory.occurrences.payloadLength() {
		return fmt.Errorf("executable indexes have unequal cardinalities")
	}
	var previousDefinition identity.DefinitionID
	for _, regionID := range inventory.regionIDs {
		region := inventory.byID[regionID]
		if region.id.IsZero() ||
			(!previousDefinition.IsZero() &&
				previousDefinition.Compare(region.Definition()) >= 0) {
			return fmt.Errorf(
				"executable region order is noncanonical at %s",
				region.Definition(),
			)
		}
		previousDefinition = region.Definition()
		indexed, present := inventory.byID[region.Definition()]
		if !present || indexed.id != region.id {
			return fmt.Errorf(
				"executable region index disagrees at %s",
				region.Definition(),
			)
		}
	}
	var previousOccurrence identity.OccurrenceID
	expectedRanges := map[identity.FileID]occurrenceRange{}
	seen := make([]bool, inventory.occurrences.length()+1)
	for position, reference := range inventory.additional {
		occurrence, present := inventory.occurrences.payloadFor(reference)
		if !present || seen[reference] {
			return fmt.Errorf(
				"additional occurrence reference %d is absent or repeated",
				reference,
			)
		}
		seen[reference] = true
		if !previousOccurrence.IsZero() &&
			previousOccurrence.Compare(occurrence.ID()) >= 0 {
			return fmt.Errorf(
				"additional occurrence order is noncanonical at %s",
				occurrence.ID(),
			)
		}
		previousOccurrence = occurrence.ID()
		indexed := inventory.occurrences.reference(occurrence.ID())
		if indexed != reference {
			return fmt.Errorf(
				"additional occurrence index disagrees at %s",
				occurrence.ID(),
			)
		}
		file := occurrence.ID().Span().File()
		expectedRange, present := expectedRanges[file]
		if !present {
			expectedRange.start = position
		}
		expectedRange.end = position + 1
		expectedRanges[file] = expectedRange
	}
	if len(inventory.additionalByFile) != len(expectedRanges) {
		return fmt.Errorf(
			"additional occurrence file index has unequal cardinality",
		)
	}
	for file, expected := range expectedRanges {
		if actual, present := inventory.additionalByFile[file]; !present || actual != expected {
			return fmt.Errorf(
				"additional occurrence file range disagrees for %s",
				file,
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
	regionOrdinal uint32,
	state *regionValidationState,
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
	state.nextRegion()
	for _, entry := range boundary.Entries() {
		reference := inventory.occurrences.reference(entry.ID())
		if reference.valid() {
			state.entry[reference] = state.generation
		}
	}
	for _, member := range region.members {
		memberID, err := inventory.occurrences.id(member)
		if err != nil {
			return err
		}
		if prior := state.memberOwner[member]; prior != 0 {
			return fmt.Errorf(
				"occurrence %s belongs to executable regions %s and %s",
				memberID,
				inventory.regionIDs[prior-1],
				definition.ID(),
			)
		}
		occurrence, found := executableOccurrence(
			graph, inventory, memberID,
		)
		if !found {
			return fmt.Errorf(
				"region %s member %s has no canonical occurrence",
				region.id, memberID,
			)
		}
		if state.local[member] == state.generation {
			return fmt.Errorf(
				"region %s repeats occurrence %s", region.id, memberID,
			)
		}
		parent := inventory.occurrences.reference(occurrence.Parent())
		if state.entry[member] != state.generation &&
			(!parent.valid() ||
				state.local[parent] != state.generation) {
			return fmt.Errorf(
				"region %s member %s precedes or omits parent %s",
				region.id, memberID, occurrence.Parent(),
			)
		}
		state.local[member] = state.generation
		state.memberOwner[member] = regionOrdinal
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
				state.local[inventory.occurrences.reference(
					reference.parent,
				)] != state.generation) {
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
