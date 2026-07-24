package stagecheck

import (
	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
	"github.com/tsoniclang/gotots/internal/language/executable"
	"github.com/tsoniclang/gotots/internal/language/structure"
	"github.com/tsoniclang/gotots/internal/scope"
	"github.com/tsoniclang/gotots/internal/scope/sourceplan"
	"github.com/tsoniclang/gotots/internal/source"
)

type semanticPackageExpectation struct {
	id             identity.PackageID
	pkg            structure.PackageGraph
	loaded         *source.LoadedPackage
	definitions    map[identity.DefinitionID]structure.ImplementationDefinition
	executable     map[identity.DefinitionID]bool
	regions        map[identity.DefinitionID]executable.Region
	parents        map[identity.DefinitionID]identity.DefinitionID
	initializers   map[identity.DefinitionID][]identity.OccurrenceID
	occurrences    *semanticOccurrenceStore
	definitionRefs map[identity.DefinitionID]semanticDefinitionRef
	definitionIDs  []identity.DefinitionID
	domainCount    int
	order          []semanticOccurrenceRef
	localFiles     map[identity.FileID]bool
}

type semanticDefinitionRef uint32

func (reference semanticDefinitionRef) valid() bool {
	return reference != 0
}

type semanticExpectedOccurrence struct {
	structure.OccurrenceRef
	domain          catalog.ResolutionDomain
	owner           semanticDefinitionRef
	structuralOwner semanticDefinitionRef
}

func (expected semanticPackageExpectation) occurrence(
	occurrence identity.OccurrenceID,
) semanticExpectedOccurrence {
	record, present := expected.occurrences.get(occurrence)
	if !present {
		return semanticExpectedOccurrence{}
	}
	return *record
}

func (expected semanticPackageExpectation) occurrenceRecord(
	reference semanticOccurrenceRef,
) semanticExpectedOccurrence {
	record := expected.occurrences.record(reference)
	if record == nil {
		return semanticExpectedOccurrence{}
	}
	return *record
}

func (expected semanticPackageExpectation) hasOccurrence(
	occurrence identity.OccurrenceID,
) bool {
	_, present := expected.occurrences.get(occurrence)
	return present
}

func (expected semanticPackageExpectation) occurrenceOwner(
	occurrence identity.OccurrenceID,
) identity.DefinitionID {
	record := expected.occurrence(occurrence)
	return expected.definitionID(record.owner)
}

func (expected semanticPackageExpectation) structuralOccurrenceOwner(
	occurrence identity.OccurrenceID,
) identity.DefinitionID {
	record := expected.occurrence(occurrence)
	return expected.definitionID(record.structuralOwner)
}

func (expected semanticPackageExpectation) definitionID(
	reference semanticDefinitionRef,
) identity.DefinitionID {
	if !reference.valid() ||
		int(reference) > len(expected.definitionIDs) {
		return identity.DefinitionID{}
	}
	return expected.definitionIDs[reference-1]
}

func (expected *semanticPackageExpectation) admitDefinition(
	id identity.DefinitionID,
) semanticDefinitionRef {
	if id.IsZero() {
		return 0
	}
	if reference := expected.definitionRefs[id]; reference.valid() {
		return reference
	}
	expected.definitionIDs = append(expected.definitionIDs, id)
	reference := semanticDefinitionRef(len(expected.definitionIDs))
	expected.definitionRefs[id] = reference
	return reference
}

func newSemanticPackageExpectation(
	pkg structure.PackageGraph,
	loaded *source.LoadedPackage,
	selections map[identity.DefinitionID]scope.DefinitionSelection,
	executableInventory *executable.Inventory,
	plan *sourceplan.Plan,
	localOnly bool,
) (semanticPackageExpectation, error) {
	if loaded == nil {
		return semanticPackageExpectation{}, semanticVerificationError(
			"package", "structural package is absent from source universe",
		)
	}
	files := pkg.Files()
	occurrenceCapacity := 0
	var packageFiles []identity.FileID
	for _, file := range files {
		occurrenceCapacity += file.OccurrenceCount()
		packageFiles = append(
			packageFiles, file.Owner().ID().File(),
		)
	}
	additionalCount, err := executableInventory.
		AdditionalOccurrenceCountForFiles(packageFiles)
	if err != nil {
		return semanticPackageExpectation{}, err
	}
	occurrenceCapacity += additionalCount
	out := semanticPackageExpectation{
		id: pkg.ID(), pkg: pkg, loaded: loaded,
		definitions:  map[identity.DefinitionID]structure.ImplementationDefinition{},
		executable:   map[identity.DefinitionID]bool{},
		regions:      map[identity.DefinitionID]executable.Region{},
		parents:      map[identity.DefinitionID]identity.DefinitionID{},
		initializers: map[identity.DefinitionID][]identity.OccurrenceID{},
		occurrences: newSemanticOccurrenceStore(
			occurrenceCapacity,
		),
		definitionRefs: map[identity.DefinitionID]semanticDefinitionRef{},
		localFiles:     map[identity.FileID]bool{},
	}
	for _, file := range files {
		out.localFiles[file.Owner().ID().File()] = true
		if err := file.VisitOccurrenceRefs(func(
			reference structure.OccurrenceRef,
		) error {
			return out.addOccurrenceRef(reference)
		}); err != nil {
			return semanticPackageExpectation{}, err
		}
		if err := out.assign(
			file.Owner().Members(),
			catalog.ResolutionDomainOwner,
			identity.DefinitionID{},
		); err != nil {
			return semanticPackageExpectation{}, err
		}
	}
	for _, owner := range pkg.SyntheticOwners() {
		if err := out.assign(
			owner.Members(),
			catalog.ResolutionDomainOwner,
			identity.DefinitionID{},
		); err != nil {
			return semanticPackageExpectation{}, err
		}
	}
	for _, site := range pkg.Sites() {
		if existing, present := out.parents[site.Definition()]; present &&
			existing != site.ParentDefinition() {
			return semanticPackageExpectation{},
				semanticVerificationError(
					"definition",
					"definition site has two parents "+
						site.Definition().String(),
				)
		}
		out.parents[site.Definition()] = site.ParentDefinition()
	}
	if err := pkg.VisitDefinitions(func(
		definition structure.ImplementationDefinition,
	) error {
		if localOnly &&
			!semanticDefinitionUsesLocal(
				plan, loaded, definition.ID(),
			) {
			return nil
		}
		if _, duplicate := out.definitions[definition.ID()]; duplicate {
			return semanticVerificationError(
				"definition",
				"duplicate structural definition "+
					definition.ID().String(),
			)
		}
		_, present := selections[definition.ID()]
		if !present {
			return semanticVerificationError(
				"definition",
				"missing selection "+definition.ID().String(),
			)
		}
		out.definitions[definition.ID()] = definition
		out.admitDefinition(definition.ID())
		return nil
	}); err != nil {
		return semanticPackageExpectation{}, err
	}
	for _, header := range pkg.Headers() {
		if err := out.assignStructuralOwner(
			header.Members(), header.ID().Definition(),
		); err != nil {
			return semanticPackageExpectation{}, err
		}
		_, local := out.definitions[header.ID().Definition()]
		if localOnly && !local {
			continue
		}
		if err := out.assign(
			header.Members(),
			catalog.ResolutionDomainHeader,
			header.ID().Definition(),
		); err != nil {
			return semanticPackageExpectation{}, err
		}
	}
	for _, boundary := range pkg.Boundaries() {
		var members []identity.OccurrenceID
		for _, entry := range boundary.Entries() {
			members = append(members, entry.ID())
		}
		_, local := out.definitions[boundary.ID().Definition()]
		if localOnly && !local {
			continue
		}
		out.initializers[boundary.ID().Definition()] = append(
			[]identity.OccurrenceID(nil), members...,
		)
		if err := out.assign(
			members,
			catalog.ResolutionDomainBoundary,
			boundary.ID().Definition(),
		); err != nil {
			return semanticPackageExpectation{}, err
		}
	}
	for definition := range out.definitions {
		region, present := executableInventory.For(definition)
		if !present {
			continue
		}
		out.executable[definition] = true
		out.regions[definition] = region
		if err := region.VisitMembers(func(
			_ int,
			member identity.OccurrenceID,
		) error {
			occurrence, present, err :=
				executableInventory.AdditionalOccurrenceRef(member)
			if err != nil {
				return err
			}
			if present {
				if err := out.addOccurrenceRef(occurrence); err != nil {
					return err
				}
			}
			return out.assignOccurrence(
				member,
				catalog.ResolutionDomainExecutable,
				definition,
			)
		}); err != nil {
			return semanticPackageExpectation{}, err
		}
	}
	filtered := out.order[:0]
	for _, reference := range out.order {
		record := out.occurrenceRecord(reference)
		if !record.domain.Valid() {
			out.occurrences.remove(record.ID())
			continue
		}
		filtered = append(filtered, reference)
	}
	out.order = filtered
	return out, nil
}

func builtinSemanticExpectation(
	loaded *source.LoadedPackage,
) semanticPackageExpectation {
	return semanticPackageExpectation{
		id: loaded.ID(), loaded: loaded,
		definitions:    map[identity.DefinitionID]structure.ImplementationDefinition{},
		executable:     map[identity.DefinitionID]bool{},
		regions:        map[identity.DefinitionID]executable.Region{},
		parents:        map[identity.DefinitionID]identity.DefinitionID{},
		initializers:   map[identity.DefinitionID][]identity.OccurrenceID{},
		occurrences:    newSemanticOccurrenceStore(0),
		definitionRefs: map[identity.DefinitionID]semanticDefinitionRef{},
		localFiles:     map[identity.FileID]bool{},
	}
}

func (expected *semanticPackageExpectation) assignStructuralOwner(
	occurrences []identity.OccurrenceID,
	owner identity.DefinitionID,
) error {
	for _, occurrence := range occurrences {
		record, present := expected.occurrences.get(occurrence)
		if !present {
			return semanticVerificationError(
				"occurrence",
				"structural owner names absent occurrence "+
					occurrence.String(),
			)
		}
		ownerReference := expected.admitDefinition(owner)
		if existing := record.structuralOwner; existing.valid() &&
			existing != ownerReference {
			return semanticVerificationError(
				"occurrence",
				"structural occurrence has two definition owners "+
					occurrence.String(),
			)
		}
		record.structuralOwner = ownerReference
	}
	return nil
}

func (expected *semanticPackageExpectation) addOccurrenceRef(
	occurrence structure.OccurrenceRef,
) error {
	if existing, present := expected.occurrences.get(occurrence.ID()); present {
		if existing.ID() != occurrence.ID() ||
			existing.Kind() != occurrence.Kind() ||
			existing.Parent() != occurrence.Parent() ||
			existing.Edge() != occurrence.Edge() ||
			existing.Ordinal() != occurrence.Ordinal() ||
			existing.Span() != occurrence.Span() ||
			existing.Display() != occurrence.Display() ||
			existing.Token() != occurrence.Token() {
			return semanticVerificationError(
				"occurrence",
				"conflicting payload "+occurrence.ID().String(),
			)
		}
		return nil
	}
	record := &semanticExpectedOccurrence{
		OccurrenceRef: occurrence,
	}
	reference, err := expected.occurrences.put(
		occurrence.ID(), record,
	)
	if err != nil {
		return err
	}
	expected.order = append(expected.order, reference)
	return nil
}

func (expected *semanticPackageExpectation) assign(
	occurrences []identity.OccurrenceID,
	domain catalog.ResolutionDomain,
	owner identity.DefinitionID,
) error {
	for _, occurrence := range occurrences {
		if err := expected.assignOccurrence(
			occurrence, domain, owner,
		); err != nil {
			return err
		}
	}
	return nil
}

func (expected *semanticPackageExpectation) assignOccurrence(
	occurrence identity.OccurrenceID,
	domain catalog.ResolutionDomain,
	owner identity.DefinitionID,
) error {
	record, present := expected.occurrences.get(occurrence)
	if !present {
		return semanticVerificationError(
			"domain",
			"semantic domain names absent occurrence "+
				occurrence.String(),
		)
	}
	existing := record.domain
	ownerReference := expected.admitDefinition(owner)
	if existing >= domain {
		if existing == domain && record.owner != ownerReference {
			return semanticVerificationError(
				"domain",
				"occurrence has two owners "+occurrence.String(),
			)
		}
		return nil
	}
	if !existing.Valid() {
		expected.domainCount++
	}
	record.domain = domain
	record.owner = ownerReference
	return nil
}
