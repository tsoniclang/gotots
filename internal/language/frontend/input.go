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
	occurrence      structure.Occurrence
	node            ast.Node
	owner           identity.DefinitionID
	domain          catalog.ResolutionDomain
	children        []identity.OccurrenceID
	checkedUnmapped bool
}

type packageInput struct {
	id          identity.PackageID
	provenance  semantic.PackageProvenance
	loaded      *source.LoadedPackage
	graph       structure.PackageGraph
	index       *structure.TransientIndex
	authority   semantic.Authority
	occurrences map[identity.OccurrenceID]*occurrenceInput
	order       []identity.OccurrenceID
	definitions map[identity.DefinitionID]structure.ImplementationDefinition
	parents     map[identity.DefinitionID]identity.DefinitionID
	regions     map[identity.DefinitionID]executable.Region
	selections  map[identity.DefinitionID]scope.DefinitionSelection
}

type stageInput struct {
	universe    *source.Universe
	graph       *structure.Graph
	index       *structure.TransientIndex
	facts       *selectionfacts.Artifact
	selections  *scope.DefinitionSelections
	executable  *executable.Inventory
	plan        *sourceplan.Plan
	byPackage   map[identity.PackageID]*packageInput
	packageList []*packageInput
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
		plan:      plan,
		byPackage: map[identity.PackageID]*packageInput{},
	}
	loaded := map[identity.PackageID]*source.LoadedPackage{}
	for _, pkg := range universe.Packages() {
		loaded[pkg.ID()] = pkg
	}
	selectionByID := map[identity.DefinitionID]scope.DefinitionSelection{}
	for _, selection := range selections.Records() {
		selectionByID[selection.Definition()] = selection
	}
	regions := map[identity.DefinitionID]executable.Region{}
	for _, region := range executableInventory.Regions() {
		regions[region.Definition()] = region
	}
	err := graph.VisitResidentPackages(func(pkg structure.PackageGraph) error {
		loadedPackage := loaded[pkg.ID()]
		if loadedPackage == nil {
			return fmt.Errorf(
				"semantic package %s is absent from source universe",
				pkg.ID(),
			)
		}
		input := &packageInput{
			id: pkg.ID(), provenance: semanticProvenance(
				loadedPackage.Provenance(),
			),
			loaded: loadedPackage, graph: pkg, index: index,
			occurrences: map[identity.OccurrenceID]*occurrenceInput{},
			definitions: map[identity.DefinitionID]structure.ImplementationDefinition{},
			parents:     map[identity.DefinitionID]identity.DefinitionID{},
			regions:     map[identity.DefinitionID]executable.Region{},
			selections:  map[identity.DefinitionID]scope.DefinitionSelection{},
		}
		for _, definition := range pkg.Definitions() {
			if !allLocal && !definitionUsesLocalSemantics(
				plan, loadedPackage, definition.ID(),
			) {
				continue
			}
			input.definitions[definition.ID()] = definition
			selection, present := selectionByID[definition.ID()]
			if !present {
				return fmt.Errorf(
					"local definition %s has no selection",
					definition.ID(),
				)
			}
			input.selections[definition.ID()] = selection
			if region, present := regions[definition.ID()]; present {
				input.regions[definition.ID()] = region
			}
		}
		for _, site := range pkg.Sites() {
			if _, local := input.definitions[site.Definition()]; local {
				input.parents[site.Definition()] =
					site.ParentDefinition()
			}
		}
		if err := input.buildOccurrences(index, executableInventory); err != nil {
			return err
		}
		if len(input.definitions) == 0 &&
			len(input.occurrences) == 0 &&
			!(input.provenance == semantic.ProvenanceLanguagePseudo &&
				input.id.ImportPath() == "builtin") {
			return nil
		}
		authority, err := checkerAuthority(
			universe, pkg, loadedPackage, facts,
		)
		if err != nil {
			return err
		}
		input.authority = authority
		out.byPackage[input.id] = input
		out.packageList = append(out.packageList, input)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(out.packageList, func(left, right int) bool {
		return out.packageList[left].id.String() <
			out.packageList[right].id.String()
	})
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
	domains := map[identity.OccurrenceID]catalog.ResolutionDomain{}
	owners := map[identity.OccurrenceID]identity.DefinitionID{}
	for _, file := range input.graph.Files() {
		for _, member := range file.Owner().Members() {
			if err := assignOccurrenceOwner(
				domains,
				owners,
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
			if err := assignOccurrenceOwner(
				domains,
				owners,
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
		if _, local := input.definitions[definition]; !local {
			continue
		}
		for _, member := range header.Members() {
			if err := assignOccurrenceOwner(
				domains, owners, member, definition,
				catalog.ResolutionDomainHeader,
			); err != nil {
				return err
			}
		}
	}
	for _, boundary := range input.graph.Boundaries() {
		definition := boundary.ID().Definition()
		if _, local := input.definitions[definition]; !local {
			continue
		}
		for _, entry := range boundary.Entries() {
			if err := assignOccurrenceOwner(
				domains, owners, entry.ID(), definition,
				catalog.ResolutionDomainBoundary,
			); err != nil {
				return err
			}
		}
	}
	for definition, region := range input.regions {
		for _, member := range region.Members() {
			if err := assignOccurrenceOwner(
				domains, owners, member, definition,
				catalog.ResolutionDomainExecutable,
			); err != nil {
				return err
			}
		}
	}
	appendOccurrence := func(occurrence structure.Occurrence) error {
		domain := domains[occurrence.ID()]
		if !domain.Valid() {
			return nil
		}
		node, local := index.OccurrenceNode(occurrence.ID())
		if !local {
			return nil
		}
		if existing := input.occurrences[occurrence.ID()]; existing != nil {
			if existing.occurrence != occurrence || existing.node != node {
				return fmt.Errorf(
					"occurrence %s has conflicting Stage-2 input",
					occurrence.ID(),
				)
			}
			return nil
		}
		input.occurrences[occurrence.ID()] = &occurrenceInput{
			occurrence: occurrence,
			node:       node,
			owner:      owners[occurrence.ID()],
			domain:     domain,
			checkedUnmapped: index.CheckedUnmapped(
				occurrence.ID(),
			),
		}
		input.order = append(input.order, occurrence.ID())
		return nil
	}
	for _, file := range input.graph.Files() {
		for _, occurrence := range file.Occurrences() {
			if err := appendOccurrence(occurrence); err != nil {
				return err
			}
		}
	}
	packageFiles := map[identity.FileID]bool{}
	for _, file := range input.graph.Files() {
		packageFiles[file.Owner().ID().File()] = true
	}
	for _, occurrence := range executableInventory.AdditionalOccurrences() {
		if !packageFiles[occurrence.ID().Span().File()] {
			continue
		}
		if err := appendOccurrence(occurrence); err != nil {
			return err
		}
	}
	for _, id := range input.order {
		record := input.occurrences[id]
		parent := input.occurrences[record.occurrence.Parent()]
		if parent != nil {
			parent.children = append(parent.children, id)
		}
	}
	return nil
}

func assignOccurrenceOwner(
	domains map[identity.OccurrenceID]catalog.ResolutionDomain,
	owners map[identity.OccurrenceID]identity.DefinitionID,
	occurrence identity.OccurrenceID,
	definition identity.DefinitionID,
	domain catalog.ResolutionDomain,
) error {
	if existing := owners[occurrence]; !existing.IsZero() &&
		existing != definition {
		current := domains[occurrence]
		switch {
		case resolutionDomainPrecedence(domain) >
			resolutionDomainPrecedence(current):
		case resolutionDomainPrecedence(domain) <
			resolutionDomainPrecedence(current):
			return nil
		default:
			return fmt.Errorf(
				"occurrence %s is owned by definitions %s and %s in domain %s",
				occurrence, existing, definition, domain,
			)
		}
	}
	if current := domains[occurrence]; current.Valid() &&
		current != domain {
		if resolutionDomainPrecedence(domain) <
			resolutionDomainPrecedence(current) {
			return nil
		}
	}
	owners[occurrence] = definition
	domains[occurrence] = domain
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
	selectionDigest := packageSelectionDigest(pkg, facts)
	return semantic.NewCheckerAuthority(
		universe.Toolchain().BinaryDigest(),
		universe.Toolchain().BuildConfigurationDigest(),
		loaded.ProviderInputFingerprint(),
		structure.PackageDigest(pkg),
		selectionDigest,
	)
}

func packageSelectionDigest(
	pkg structure.PackageGraph,
	facts *selectionfacts.Artifact,
) string {
	definitions := map[identity.DefinitionID]bool{}
	for _, definition := range pkg.Definitions() {
		definitions[definition.ID()] = true
	}
	hash := sha256.New()
	fmt.Fprintln(hash, "gotots-semantic-selection/v1")
	for _, fact := range facts.Facts() {
		if !definitions[fact.ID().Definition()] {
			continue
		}
		fmt.Fprintf(
			hash, "%s|%t|%s|%s\n",
			fact.ID(), fact.Value(),
			fact.ProducerDigest(), fact.EvidenceDigest(),
		)
	}
	return fmt.Sprintf("%x", hash.Sum(nil))
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
