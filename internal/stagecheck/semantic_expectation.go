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
	id           identity.PackageID
	pkg          structure.PackageGraph
	loaded       *source.LoadedPackage
	definitions  map[identity.DefinitionID]structure.ImplementationDefinition
	selections   map[identity.DefinitionID]scope.DefinitionSelection
	executable   map[identity.DefinitionID]bool
	regions      map[identity.DefinitionID]executable.Region
	parents      map[identity.DefinitionID]identity.DefinitionID
	initializers map[identity.DefinitionID][]identity.OccurrenceID
	occurrences  *semanticOccurrenceStore
	domainCount  int
	order        []identity.OccurrenceID
	localFiles   map[identity.FileID]bool
}

type semanticExpectedOccurrence struct {
	structure.Occurrence
	domain          catalog.ResolutionDomain
	owner           identity.DefinitionID
	structuralOwner identity.DefinitionID
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
	return record.owner
}

func (expected semanticPackageExpectation) structuralOccurrenceOwner(
	occurrence identity.OccurrenceID,
) identity.DefinitionID {
	record := expected.occurrence(occurrence)
	return record.structuralOwner
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
	out := semanticPackageExpectation{
		id: pkg.ID(), pkg: pkg, loaded: loaded,
		definitions:  map[identity.DefinitionID]structure.ImplementationDefinition{},
		selections:   map[identity.DefinitionID]scope.DefinitionSelection{},
		executable:   map[identity.DefinitionID]bool{},
		regions:      map[identity.DefinitionID]executable.Region{},
		parents:      map[identity.DefinitionID]identity.DefinitionID{},
		initializers: map[identity.DefinitionID][]identity.OccurrenceID{},
		occurrences:  newSemanticOccurrenceStore(),
		localFiles:   map[identity.FileID]bool{},
	}
	for _, file := range pkg.Files() {
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
	for _, definition := range pkg.Definitions() {
		if localOnly &&
			!semanticDefinitionUsesLocal(
				plan, loaded, definition.ID(),
			) {
			continue
		}
		if _, duplicate := out.definitions[definition.ID()]; duplicate {
			return semanticPackageExpectation{}, semanticVerificationError(
				"definition",
				"duplicate structural definition "+
					definition.ID().String(),
			)
		}
		selection, present := selections[definition.ID()]
		if !present {
			return semanticPackageExpectation{}, semanticVerificationError(
				"definition",
				"missing selection "+definition.ID().String(),
			)
		}
		out.definitions[definition.ID()] = definition
		out.selections[definition.ID()] = selection
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
			if occurrence, present :=
				executableInventory.AdditionalOccurrence(member); present {
				if err := out.addOccurrence(occurrence); err != nil {
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
	return out, nil
}

func builtinSemanticExpectation(
	loaded *source.LoadedPackage,
) semanticPackageExpectation {
	return semanticPackageExpectation{
		id: loaded.ID(), loaded: loaded,
		definitions:  map[identity.DefinitionID]structure.ImplementationDefinition{},
		selections:   map[identity.DefinitionID]scope.DefinitionSelection{},
		executable:   map[identity.DefinitionID]bool{},
		regions:      map[identity.DefinitionID]executable.Region{},
		parents:      map[identity.DefinitionID]identity.DefinitionID{},
		initializers: map[identity.DefinitionID][]identity.OccurrenceID{},
		occurrences:  newSemanticOccurrenceStore(),
		localFiles:   map[identity.FileID]bool{},
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
		if existing := record.structuralOwner; !existing.IsZero() &&
			existing != owner {
			return semanticVerificationError(
				"occurrence",
				"structural occurrence has two definition owners "+
					occurrence.String(),
			)
		}
		record.structuralOwner = owner
	}
	return nil
}

func (expected *semanticPackageExpectation) addOccurrence(
	occurrence structure.Occurrence,
) error {
	if existing, present := expected.occurrences.get(occurrence.ID()); present {
		if existing.Occurrence != occurrence {
			return semanticVerificationError(
				"occurrence",
				"conflicting payload "+occurrence.ID().String(),
			)
		}
		return nil
	}
	record := &semanticExpectedOccurrence{
		Occurrence: occurrence,
	}
	if err := expected.occurrences.put(occurrence.ID(), record); err != nil {
		return err
	}
	expected.order = append(expected.order, occurrence.ID())
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
	canonical, err := structure.NewOccurrence(
		occurrence.ID(),
		occurrence.Kind(),
		occurrence.Parent(),
		occurrence.Edge(),
		occurrence.Ordinal(),
		occurrence.Span(),
		occurrence.Display(),
		occurrence.Token(),
	)
	if err != nil {
		return err
	}
	return expected.addOccurrence(canonical)
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
	if existing >= domain {
		if existing == domain && record.owner != owner {
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
	record.owner = owner
	return nil
}
