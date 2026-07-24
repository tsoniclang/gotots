package frontend

import (
	"crypto/sha256"
	"fmt"
	"go/ast"
	"sort"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
	"github.com/tsoniclang/gotots/internal/language/executable"
	"github.com/tsoniclang/gotots/internal/language/selectionfacts"
	"github.com/tsoniclang/gotots/internal/language/semantic"
	"github.com/tsoniclang/gotots/internal/language/structure"
	"github.com/tsoniclang/gotots/internal/scope"
	"github.com/tsoniclang/gotots/internal/scope/sourceplan"
	"github.com/tsoniclang/gotots/internal/source"
)

type occurrenceInput struct {
	occurrence      structure.OccurrenceRef
	node            ast.Node
	owner           packageDefinitionRef
	domain          catalog.ResolutionDomain
	children        []packageOccurrenceRef
	checkedUnmapped bool
	context         occurrenceContext
	contextAssigned bool
	contextVisiting bool
}

type packageInput struct {
	id          identity.PackageID
	provenance  semantic.PackageProvenance
	loaded      *source.LoadedPackage
	graph       structure.PackageGraph
	index       *structure.TransientIndex
	authority   semantic.Authority
	occurrences *occurrenceStore
	order       []packageOccurrenceRef
	definitions *definitionStore
	containment *definitionContainment
	work        Work
}

func (input *packageInput) occurrence(
	id identity.OccurrenceID,
) *occurrenceInput {
	return input.occurrences.get(id)
}

func (input *packageInput) occurrenceReference(
	id identity.OccurrenceID,
) packageOccurrenceRef {
	return input.occurrences.reference(id)
}

func (input *packageInput) occurrenceRecord(
	reference packageOccurrenceRef,
) *occurrenceInput {
	return input.occurrences.record(reference)
}

func (input *packageInput) definition(
	id identity.DefinitionID,
) *definitionInput {
	return input.definitions.get(id)
}

func (input *packageInput) definitionID(
	reference packageDefinitionRef,
) identity.DefinitionID {
	return input.definitions.id(reference)
}

func (input *packageInput) occurrenceOwner(
	record *occurrenceInput,
) identity.DefinitionID {
	if record == nil {
		return identity.DefinitionID{}
	}
	return input.definitionID(record.owner)
}

type stageInput struct {
	universe      *source.Universe
	graph         *structure.Graph
	index         *structure.TransientIndex
	facts         *selectionfacts.Artifact
	selections    *scope.DefinitionSelections
	executable    *executable.Inventory
	plan          *sourceplan.Plan
	loaded        map[identity.PackageID]*source.LoadedPackage
	packageByPath map[string]identity.PackageID
	allLocal      bool
}

func newStageInput(
	universe *source.Universe,
	graph *structure.Graph,
	index *structure.TransientIndex,
	facts *selectionfacts.Artifact,
	selections *scope.DefinitionSelections,
	executableInventory *executable.Inventory,
	plan *sourceplan.Plan,
	allLocal bool,
) (*stageInput, error) {
	expectedPurpose := sourceplan.PurposeCompilation
	if allLocal {
		expectedPurpose = sourceplan.PurposeProviderProduction
	}
	if universe == nil ||
		!universe.Hydrated() ||
		graph == nil ||
		index == nil ||
		facts == nil ||
		selections == nil ||
		executableInventory == nil ||
		plan == nil ||
		plan.Purpose() != expectedPurpose {
		return nil, fmt.Errorf(
			"frontend requires every live Stage-1 input before finalization",
		)
	}
	out := &stageInput{
		universe: universe, graph: graph, index: index, facts: facts,
		selections: selections, executable: executableInventory,
		plan: plan, allLocal: allLocal,
		loaded:        map[identity.PackageID]*source.LoadedPackage{},
		packageByPath: map[string]identity.PackageID{},
	}
	for _, pkg := range universe.Packages() {
		if _, duplicate := out.loaded[pkg.ID()]; duplicate {
			return nil, fmt.Errorf(
				"semantic source package %s is duplicated",
				pkg.ID(),
			)
		}
		out.loaded[pkg.ID()] = pkg
		path := pkg.ID().ImportPath()
		if existing := out.packageByPath[path]; !existing.IsZero() &&
			existing != pkg.ID() {
			return nil, fmt.Errorf(
				"checker import path %s has package identities %s and %s",
				path, existing, pkg.ID(),
			)
		}
		out.packageByPath[path] = pkg.ID()
	}
	return out, nil
}

func definitionUsesLocalSemantics(
	plan *sourceplan.Plan,
	pkg *source.LoadedPackage,
	definition identity.DefinitionID,
) bool {
	if file := definition.File(); !file.IsZero() {
		decision, present := plan.For(file)
		return present && decision.Kind() == sourceplan.KindLocalSyntax
	}
	if definition.SyntheticRole().Valid() {
		decision, present := plan.SyntheticFor(pkg.ID())
		return present && decision.Kind() == sourceplan.KindLocalSyntax
	}
	if definition.ImplicitOp().Valid() {
		return !packageUsesCertifiedSemantics(plan, pkg)
	}
	return false
}

func packageUsesCertifiedSemantics(
	plan *sourceplan.Plan,
	pkg *source.LoadedPackage,
) bool {
	for _, file := range pkg.Files() {
		decision, present := plan.For(file.ID())
		if present && decision.Kind() == sourceplan.KindCertifiedGraph {
			return true
		}
	}
	if decision, present := plan.SyntheticFor(pkg.ID()); present {
		return decision.Kind() == sourceplan.KindCertifiedGraph
	}
	return false
}

func (input *packageInput) buildOccurrences(
	index *structure.TransientIndex,
	executableInventory *executable.Inventory,
) error {
	files := input.graph.Files()
	structuralCount := 0
	var packageFiles []identity.FileID
	for _, file := range files {
		structuralCount += file.OccurrenceCount()
		packageFiles = append(
			packageFiles, file.Owner().ID().File(),
		)
	}
	additional, err := executableInventory.
		AdditionalOccurrenceRefsForFiles(packageFiles)
	if err != nil {
		return err
	}
	if err := input.occurrences.reserve(
		structuralCount + len(additional),
	); err != nil {
		return err
	}
	appendOccurrence := func(occurrence structure.OccurrenceRef) error {
		node, local := index.OccurrenceNode(occurrence.ID())
		if !local {
			return nil
		}
		if existing := input.occurrence(occurrence.ID()); existing != nil {
			if !existing.occurrence.Equal(occurrence) ||
				existing.node != node {
				return fmt.Errorf(
					"occurrence %s has conflicting Stage-2 input",
					occurrence.ID(),
				)
			}
			return nil
		}
		record := &occurrenceInput{
			occurrence: occurrence,
			node:       node,
			checkedUnmapped: index.CheckedUnmapped(
				occurrence.ID(),
			),
		}
		reference, err := input.occurrences.put(
			occurrence.ID(), record,
		)
		if err != nil {
			return err
		}
		input.order = append(input.order, reference)
		return nil
	}
	for _, file := range files {
		if err := file.VisitOccurrenceRefs(
			appendOccurrence,
		); err != nil {
			return err
		}
	}
	for _, occurrence := range additional {
		if err := appendOccurrence(occurrence); err != nil {
			return err
		}
	}
	for _, file := range files {
		for _, member := range file.Owner().Members() {
			if err := input.assignOccurrenceOwner(
				member,
				identity.DefinitionID{},
				catalog.ResolutionDomainOwner,
			); err != nil {
				return err
			}
		}
	}
	for _, owner := range input.graph.SyntheticOwners() {
		for _, member := range owner.Members() {
			if err := input.assignOccurrenceOwner(
				member,
				identity.DefinitionID{},
				catalog.ResolutionDomainOwner,
			); err != nil {
				return err
			}
		}
	}
	for _, header := range input.graph.Headers() {
		definition := header.ID().Definition()
		if input.definition(definition) == nil {
			continue
		}
		for _, member := range header.Members() {
			if err := input.assignOccurrenceOwner(
				member, definition,
				catalog.ResolutionDomainHeader,
			); err != nil {
				return err
			}
		}
	}
	for _, boundary := range input.graph.Boundaries() {
		definition := boundary.ID().Definition()
		if input.definition(definition) == nil {
			continue
		}
		for _, entry := range boundary.Entries() {
			if err := input.assignOccurrenceOwner(
				entry.ID(), definition,
				catalog.ResolutionDomainBoundary,
			); err != nil {
				return err
			}
		}
	}
	if err := input.definitions.visit(func(
		_ packageDefinitionRef,
		definition *definitionInput,
	) error {
		if !definition.hasRegion {
			return nil
		}
		definitionID := definition.definition.ID()
		if err := definition.region.VisitMembers(func(
			_ int,
			member identity.OccurrenceID,
		) error {
			if err := input.assignOccurrenceOwner(
				member, definitionID,
				catalog.ResolutionDomainExecutable,
			); err != nil {
				return err
			}
			return nil
		}); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}
	filtered := input.order[:0]
	for _, reference := range input.order {
		record := input.occurrenceRecord(reference)
		if !record.domain.Valid() {
			input.occurrences.remove(record.occurrence.ID())
			continue
		}
		filtered = append(filtered, reference)
	}
	input.order = filtered
	return input.assignChildren()
}

func (input *packageInput) assignChildren() error {
	input.work.InputOccurrences += input.occurrences.count()
	if err := input.occurrences.visit(func(
		reference packageOccurrenceRef,
		record *occurrenceInput,
	) error {
		parent := input.occurrenceReference(record.occurrence.Parent())
		parentRecord := input.occurrenceRecord(parent)
		if parentRecord != nil {
			input.work.ChildEdgeAssignments++
			parentRecord.children = append(
				parentRecord.children, reference,
			)
		}
		return nil
	}); err != nil {
		return err
	}
	for _, parentReference := range input.order {
		parent := input.occurrenceRecord(parentReference)
		rank := map[catalog.Edge]int{}
		for index, edge := range catalog.EdgesOf(
			parent.occurrence.Kind(),
		) {
			rank[edge] = index
		}
		input.work.CanonicalSortInputs += len(parent.children)
		sort.Slice(parent.children, func(left, right int) bool {
			leftRecord := input.occurrenceRecord(
				parent.children[left],
			).occurrence
			rightRecord := input.occurrenceRecord(
				parent.children[right],
			).occurrence
			if rank[leftRecord.Edge()] != rank[rightRecord.Edge()] {
				return rank[leftRecord.Edge()] <
					rank[rightRecord.Edge()]
			}
			return leftRecord.Ordinal() < rightRecord.Ordinal()
		})
	}
	return nil
}

func (input *packageInput) assignOccurrenceOwner(
	occurrence identity.OccurrenceID,
	definition identity.DefinitionID,
	domain catalog.ResolutionDomain,
) error {
	record := input.occurrence(occurrence)
	if record == nil {
		return nil
	}
	definitionReference := input.definitions.reference(definition)
	if !definition.IsZero() && !definitionReference.valid() {
		return fmt.Errorf(
			"occurrence %s names absent definition %s",
			occurrence, definition,
		)
	}
	if existing := record.owner; existing.valid() &&
		existing != definitionReference {
		current := record.domain
		switch {
		case resolutionDomainPrecedence(domain) >
			resolutionDomainPrecedence(current):
		case resolutionDomainPrecedence(domain) <
			resolutionDomainPrecedence(current):
			return nil
		default:
			return fmt.Errorf(
				"occurrence %s is owned by definitions %s and %s in domain %s",
				occurrence,
				input.definitionID(existing),
				definition,
				domain,
			)
		}
	}
	if current := record.domain; current.Valid() &&
		current != domain {
		if resolutionDomainPrecedence(domain) <
			resolutionDomainPrecedence(current) {
			return nil
		}
	}
	record.owner = definitionReference
	record.domain = domain
	return nil
}

func resolutionDomainPrecedence(
	domain catalog.ResolutionDomain,
) int {
	switch domain {
	case catalog.ResolutionDomainOwner:
		return 1
	case catalog.ResolutionDomainHeader:
		return 2
	case catalog.ResolutionDomainBoundary:
		return 3
	case catalog.ResolutionDomainExecutable:
		return 4
	default:
		return 0
	}
}

func checkerAuthority(
	universe *source.Universe,
	pkg structure.PackageGraph,
	loaded *source.LoadedPackage,
	facts *selectionfacts.Artifact,
) (semantic.Authority, error) {
	selectionDigest, err := packageSelectionDigest(pkg, facts)
	if err != nil {
		return semantic.Authority{}, err
	}
	structureDigest := structure.PackageDigest(pkg)
	if loaded.Disposition() == source.DispositionBuiltinUniverse {
		structureDigest = catalog.StructureDigest()
	}
	return semantic.NewCheckerAuthority(
		universe.Toolchain().BinaryDigest(),
		universe.Toolchain().BuildConfigurationDigest(),
		loaded.ProviderInputFingerprint(),
		structureDigest,
		selectionDigest,
	)
}

func packageSelectionDigest(
	pkg structure.PackageGraph,
	facts *selectionfacts.Artifact,
) (string, error) {
	definitions := map[identity.DefinitionID]bool{}
	if err := pkg.VisitDefinitions(func(
		definition structure.ImplementationDefinition,
	) error {
		definitions[definition.ID()] = true
		return nil
	}); err != nil {
		return "", err
	}
	hash := sha256.New()
	fmt.Fprintln(hash, "gotots-semantic-selection/v1")
	_ = facts.VisitFacts(func(fact selectionfacts.Fact) error {
		if !definitions[fact.ID().Definition()] {
			return nil
		}
		fmt.Fprintf(
			hash, "%s|%t|%s|%s\n",
			fact.ID(), fact.Value(),
			fact.ProducerDigest(), fact.EvidenceDigest(),
		)
		return nil
	})
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func semanticProvenance(
	provenance source.Provenance,
) semantic.PackageProvenance {
	switch provenance {
	case source.ProvenanceWorkspaceModule:
		return semantic.ProvenanceWorkspaceModule
	case source.ProvenanceModuleDependency:
		return semantic.ProvenanceModuleDependency
	case source.ProvenanceStandardLibrary:
		return semantic.ProvenanceStandardLibrary
	case source.ProvenanceToolchainPackage:
		return semantic.ProvenanceToolchainPackage
	case source.ProvenanceLanguagePseudo:
		return semantic.ProvenanceLanguagePseudo
	default:
		return semantic.ProvenanceInvalid
	}
}
