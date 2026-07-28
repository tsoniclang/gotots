package emit

import (
	"fmt"
	"go/types"
	"sort"

	"github.com/tsoniclang/gotots/internal/emit/api"
	artifactstate "github.com/tsoniclang/gotots/internal/emit/artifact"
	emitnaming "github.com/tsoniclang/gotots/internal/emit/naming"
	targetplacement "github.com/tsoniclang/gotots/internal/emit/placement"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type scheduler struct {
	queue   []types.Object
	pending map[types.Object]struct{}
	emitted map[types.Object]struct{}
}

type ScheduleError struct {
	Object string
	Reason string
}

func (e *ScheduleError) Error() string {
	if e.Object == "" {
		return "schedule declaration: " + e.Reason
	}
	return fmt.Sprintf("schedule declaration %q: %s", e.Object, e.Reason)
}

func newScheduler() *scheduler {
	return &scheduler{
		pending: make(map[types.Object]struct{}),
		emitted: make(map[types.Object]struct{}),
	}
}

func (s *scheduler) enqueue(object types.Object) {
	if _, done := s.emitted[object]; done {
		return
	}
	if _, queued := s.pending[object]; queued {
		return
	}
	s.pending[object] = struct{}{}
	s.queue = append(s.queue, object)
}

func (s *scheduler) next() (types.Object, bool) {
	if len(s.queue) == 0 {
		return nil, false
	}
	object := s.queue[0]
	s.queue = s.queue[1:]
	delete(s.pending, object)
	s.emitted[object] = struct{}{}
	return object, true
}

func (s *scheduler) hasPending() bool {
	return len(s.queue) != 0 || len(s.pending) != 0
}

type declarationRequirementScheduler struct {
	pending map[api.DeclarationRequirement]struct{}
	applied map[api.DeclarationRequirement]struct{}
}

func newDeclarationRequirementScheduler() *declarationRequirementScheduler {
	return &declarationRequirementScheduler{
		pending: make(map[api.DeclarationRequirement]struct{}),
		applied: make(map[api.DeclarationRequirement]struct{}),
	}
}

func (s *declarationRequirementScheduler) enqueue(
	requirement api.DeclarationRequirement,
) {
	if _, done := s.applied[requirement]; done {
		return
	}
	if _, queued := s.pending[requirement]; queued {
		return
	}
	s.pending[requirement] = struct{}{}
}

func (s *declarationRequirementScheduler) nextBatch() (
	[]api.DeclarationRequirement,
	bool,
) {
	if len(s.pending) == 0 {
		return nil, false
	}
	var selected api.DeclarationRequirement
	first := true
	for requirement := range s.pending {
		if first || compareDeclarationRequirements(requirement, selected) < 0 {
			selected = requirement
			first = false
		}
	}
	requirements := make([]api.DeclarationRequirement, 0, len(s.pending))
	for requirement := range s.pending {
		if requirement.Owner() != selected.Owner() {
			continue
		}
		requirements = append(requirements, requirement)
	}
	sort.Slice(requirements, func(left, right int) bool {
		return compareDeclarationRequirements(
			requirements[left],
			requirements[right],
		) < 0
	})
	for _, requirement := range requirements {
		delete(s.pending, requirement)
		s.applied[requirement] = struct{}{}
	}
	return requirements, true
}

func (s *declarationRequirementScheduler) hasPending() bool {
	return len(s.pending) != 0
}

func (s *declarationRequirementScheduler) appliedFor(
	owner api.ArtifactOwner,
) []api.DeclarationRequirement {
	requirements := make([]api.DeclarationRequirement, 0)
	for requirement := range s.applied {
		if requirement.Owner() == owner {
			requirements = append(requirements, requirement)
		}
	}
	sort.Slice(requirements, func(left, right int) bool {
		return compareDeclarationRequirements(
			requirements[left],
			requirements[right],
		) < 0
	})
	return requirements
}

func (s *programSession) scheduleDeclarationRequirement(
	requirement api.DeclarationRequirement,
) error {
	if s.sealed {
		return &ScheduleError{
			Object: requirementOwnerName(requirement),
			Reason: "declaration requirement requested after target files were sealed",
		}
	}
	if !requirement.Valid() {
		return &ScheduleError{Reason: "declaration requirement is invalid"}
	}
	if artifact, generated := requirement.GeneratedArtifact(); generated {
		if err := s.validateGeneratedArtifact(artifact); err != nil {
			return err
		}
	}
	owner := requirement.Owner()
	if sourceOwner, sourceOwned := owner.Source(); sourceOwned {
		if _, ok := s.sites[sourceOwner]; !ok {
			return &ScheduleError{
				Object: requirementOwnerName(requirement),
				Reason: "declaration requirement owner has no source declaration",
			}
		}
		if err := s.require(sourceOwner); err != nil {
			return err
		}
	} else if sourceTypes, _, initializerOwned := owner.PackageInitializer(); initializerOwned {
		sourcePackage := s.source.PackageForTypes(sourceTypes)
		if sourcePackage == nil {
			return &ScheduleError{
				Object: owner.Name(),
				Reason: "package initializer owner has no source package",
			}
		}
		if err := s.requirePackage(sourcePackage); err != nil {
			return err
		}
	} else if _, generatedOwned := owner.Generated(); !generatedOwned {
		return &ScheduleError{
			Object: requirementOwnerName(requirement),
			Reason: "declaration requirement owner is invalid",
		}
	}
	s.requirements.enqueue(requirement)
	return nil
}

func (s *programSession) applyDeclarationRequirements(
	requirements []api.DeclarationRequirement,
) error {
	if s.sealed {
		return &ScheduleError{
			Reason: "declaration requirements applied after target files were sealed",
		}
	}
	if len(requirements) == 0 {
		return &ScheduleError{Reason: "declaration requirement batch is empty"}
	}
	owner := requirements[0].Owner()
	if !owner.Valid() {
		return &ScheduleError{Reason: "declaration requirement owner is invalid"}
	}
	if generatedOwner, ok := owner.Generated(); ok {
		for _, requirement := range requirements {
			selectedOwner, generated := requirement.GeneratedArtifact()
			if !requirement.Valid() ||
				!generated ||
				selectedOwner != generatedOwner ||
				requirement.Owner() != owner {
				return &ScheduleError{
					Object: owner.Name(),
					Reason: "generated-artifact requirement batch has mixed or invalid ownership",
				}
			}
			if _, accepted := s.requirements.applied[requirement]; !accepted {
				return &ScheduleError{
					Object: owner.Name(),
					Reason: "generated-artifact requirement was not accepted by its owner",
				}
			}
		}
		return s.reconstructGeneratedArtifact(generatedOwner)
	}
	if _, _, initializerOwned := owner.PackageInitializer(); initializerOwned {
		for _, requirement := range requirements {
			if !requirement.Valid() || requirement.Owner() != owner {
				return &ScheduleError{
					Object: owner.Name(),
					Reason: "package initializer requirement batch has mixed or invalid ownership",
				}
			}
			if _, accepted := s.requirements.applied[requirement]; !accepted {
				return &ScheduleError{
					Object: owner.Name(),
					Reason: "package initializer requirement was not accepted by its owner",
				}
			}
		}
		return s.reconstructPackageInitializer(owner)
	}
	sourceOwner, sourceOwned := owner.Source()
	if !sourceOwned {
		return &ScheduleError{
			Object: owner.Name(),
			Reason: "declaration requirement owner is invalid",
		}
	}
	if _, ok := s.sites[sourceOwner]; !ok {
		return &ScheduleError{
			Object: owner.Name(),
			Reason: "declaration requirement owner lost its source declaration",
		}
	}
	for _, requirement := range requirements {
		if !requirement.Valid() || requirement.Owner() != owner {
			return &ScheduleError{
				Object: owner.Name(),
				Reason: "declaration requirement batch has mixed or invalid ownership",
			}
		}
		if _, accepted := s.requirements.applied[requirement]; !accepted {
			return &ScheduleError{
				Object: owner.Name(),
				Reason: "declaration requirement was not accepted by its owner",
			}
		}
	}
	return s.reconstructScheduledArtifact(owner)
}

func requirementOwnerName(requirement api.DeclarationRequirement) string {
	return requirement.Owner().Name()
}

type artifactRevision struct {
	statements     []tsgo.Statement
	placement      *targetplacement.Owner
	dependencies   []api.ArtifactDependency
	contract       artifactstate.Contract
	temporaryStart emitnaming.TemporarySnapshot
}

func (s *programSession) buildArtifactRevision(
	builder *targetFileBuilder,
	site declarationSite,
	owner types.Object,
	temporaryStart emitnaming.TemporarySnapshot,
	reconstruction bool,
) (artifactRevision, error) {
	names, ok := builder.context.Names().(*emitnaming.File)
	if !ok {
		return artifactRevision{}, &ScheduleError{
			Object: owner.Name(),
			Reason: "target artifact has no concrete name owner",
		}
	}
	if !reconstruction {
		temporaryStart = names.SnapshotTemporaries()
	} else {
		current := names.SnapshotTemporaries()
		names.RestoreTemporaries(temporaryStart)
		defer names.RestoreTemporaries(current)
	}
	artifactOwner := api.MustSourceArtifactOwner(owner)
	finish, err := names.BeginArtifact(
		artifactOwner,
		site.declaration,
		site.sourceFile.Syntax(),
		site.outputPath,
	)
	if err != nil {
		return artifactRevision{}, err
	}
	defer finish()

	requirements := s.requirements.appliedFor(artifactOwner)
	context, err := emitnaming.WithLexicalGeneratedArtifacts(
		builder.context,
		site.declaration,
		artifactOwner,
		requirements,
	)
	if err != nil {
		return artifactRevision{}, err
	}
	result, err := builder.emitter.declarationObject(
		context,
		site.declaration,
		owner,
		requirements,
	)
	if err != nil {
		return artifactRevision{}, err
	}
	placement, dependencies, err := s.consumeArtifactRequests(
		artifactOwner,
		result.Requests(),
	)
	if err != nil {
		return artifactRevision{}, err
	}
	statements := result.Declarations()
	var contract artifactstate.Contract
	switch result.Disposition() {
	case api.DeclarationDispositionMaterialized:
		contract, err = artifactstate.ProjectContract(s.factory, statements)
	case api.DeclarationDispositionCoverageOnly:
		contract, err = artifactstate.ProjectCoverageContract(statements)
	default:
		err = &ScheduleError{
			Object: owner.Name(),
			Reason: "declaration emission disposition is invalid",
		}
	}
	if err != nil {
		return artifactRevision{}, err
	}
	return artifactRevision{
		statements:     statements,
		placement:      placement,
		dependencies:   dependencies,
		contract:       contract,
		temporaryStart: temporaryStart,
	}, nil
}

func (s *programSession) consumeArtifactRequests(
	consumer api.ArtifactOwner,
	requests []api.RootRequest,
) (*targetplacement.Owner, []api.ArtifactDependency, error) {
	placement := targetplacement.New()
	imports := make([]api.RootRequest, 0, len(requests))
	dependencies := make([]api.ArtifactDependency, 0, len(requests))
	for _, request := range requests {
		switch request.Kind() {
		case api.RootRequestImport:
			imports = append(imports, request)
		case api.RootRequestDeclarationRequirement:
			requirement, ok := request.DeclarationRequirement()
			if !ok {
				return nil, nil, &ScheduleError{
					Object: consumer.Name(),
					Reason: "declaration requirement is invalid",
				}
			}
			if err := s.scheduleDeclarationRequirement(requirement); err != nil {
				return nil, nil, err
			}
		case api.RootRequestArtifactDependency:
			dependency, ok := request.ArtifactDependency()
			if !ok {
				return nil, nil, &ScheduleError{
					Object: consumer.Name(),
					Reason: "artifact dependency is invalid",
				}
			}
			sourceObject, sourceProvider := dependency.Provider().Source()
			if sourceProvider {
				_, sourceProvider = s.sites[sourceObject]
			}
			generated, generatedProvider := dependency.Provider().Generated()
			if generatedProvider {
				generatedProvider =
					s.validateGeneratedArtifact(generated) == nil &&
						generated.Placement() ==
							api.GeneratedArtifactPlacementCompilation
			}
			if !sourceProvider && !generatedProvider {
				return nil, nil, &ScheduleError{
					Object: dependency.Provider().Name(),
					Reason: "artifact dependency provider has no reconstructible declaration",
				}
			}
			if sourceProvider {
				if err := s.require(sourceObject); err != nil {
					return nil, nil, err
				}
			}
			dependencies = append(dependencies, dependency)
		default:
			return nil, nil, &ScheduleError{
				Object: consumer.Name(),
				Reason: "root request kind is invalid",
			}
		}
	}
	if err := placement.Apply(imports); err != nil {
		return nil, nil, err
	}
	return placement, dependencies, nil
}

func (s *programSession) reconstructArtifact(owner types.Object) error {
	if s.sealed {
		return &ScheduleError{
			Object: owner.Name(),
			Reason: "target artifact reconstructed after target files were sealed",
		}
	}
	site, ok := s.sites[owner]
	if !ok {
		return &ScheduleError{
			Object: owner.Name(),
			Reason: "dirty target artifact lost its source declaration",
		}
	}
	builder, err := s.builder(site)
	if err != nil {
		return err
	}
	artifactOwner := api.MustSourceArtifactOwner(owner)
	index, ok := builder.indexByOwner[artifactOwner]
	if !ok {
		return &ScheduleError{
			Object: owner.Name(),
			Reason: "dirty target artifact was not emitted first",
		}
	}
	declaration := &builder.declarations[index]
	revision, err := s.buildArtifactRevision(
		builder,
		site,
		owner,
		declaration.temporaryStart,
		true,
	)
	if err != nil {
		return err
	}
	if err := s.artifacts.Commit(
		artifactOwner,
		revision.contract,
		revision.dependencies,
	); err != nil {
		return err
	}
	s.artifacts.DiscardDirty(artifactOwner)
	declaration.statements = revision.statements
	declaration.placement = revision.placement
	declaration.reconstructions++
	return nil
}

func (s *programSession) reconstructScheduledArtifact(
	owner api.ArtifactOwner,
) error {
	if generated, ok := owner.Generated(); ok {
		return s.reconstructGeneratedArtifact(generated)
	}
	if _, _, ok := owner.PackageInitializer(); ok {
		return s.reconstructPackageInitializer(owner)
	}
	source, ok := owner.Source()
	if !ok {
		return &ScheduleError{Reason: "dirty target artifact owner is invalid"}
	}
	return s.reconstructArtifact(source)
}
