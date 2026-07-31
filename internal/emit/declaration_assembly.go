package emit

import (
	"fmt"
	"go/types"

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

func (s *programSession) scheduleDeclarationRequirement(
	requirement api.DeclarationRequirement,
) error {
	if err := s.prepareDeclarationRequirement(requirement); err != nil {
		return err
	}
	s.requirements.enqueue(requirement)
	return nil
}

func (s *programSession) prepareDeclarationRequirement(
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
		_, sourceDeclared := s.sites[sourceOwner]
		environmentDeclared := s.environmentArtifactSource(sourceOwner)
		if !sourceDeclared && !environmentDeclared {
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
	if generated, generatedOwned := owner.Generated(); generatedOwned &&
		(generated.Kind() == api.GeneratedArtifactCallableABI ||
			generated.Kind() ==
				api.GeneratedArtifactInterfaceMethodCallable) {
		if err := s.ensureCallableContractBaseline(generated); err != nil {
			return err
		}
	}
	if generated, generatedOwned := owner.Generated(); generatedOwned &&
		generated.Kind() == api.GeneratedArtifactPointerRepresentation {
		if err := s.ensurePointerRepresentationBaseline(generated); err != nil {
			return err
		}
	}
	return nil
}

func (s *programSession) applyDeclarationRequirements(
	owner api.ArtifactOwner,
	requirements []api.DeclarationRequirement,
	removed bool,
) error {
	if s.sealed {
		return &ScheduleError{
			Reason: "declaration requirements applied after target files were sealed",
		}
	}
	if !owner.Valid() {
		return &ScheduleError{Reason: "declaration requirement owner is invalid"}
	}
	if removed {
		if s.requirementRemovalOwner.Valid() {
			return &ScheduleError{
				Object: owner.Name(),
				Reason: "declaration requirement removal transaction is nested",
			}
		}
		s.requirementRemovalOwner = owner
		defer func() {
			s.requirementRemovalOwner = api.ArtifactOwner{}
		}()
	}
	if sourceOwner, sourceOwned := owner.Source(); sourceOwned &&
		s.environmentArtifactSource(sourceOwner) {
		return s.applyEnvironmentRequirementSet(sourceOwner, requirements)
	}
	if generatedOwner, ok := owner.Generated(); ok {
		for _, requirement := range requirements {
			selectedOwner, generated := requirement.GeneratedArtifact()
			facet, cooperative := requirement.CooperativeCallable()
			if !requirement.Valid() ||
				requirement.Owner() != owner ||
				(!generated && !cooperative) ||
				(generated && selectedOwner != generatedOwner) ||
				(cooperative &&
					!generatedCallableFacetMatches(
						facet,
						generatedOwner,
					)) {
				return &ScheduleError{
					Object: owner.Name(),
					Reason: "generated-artifact requirement batch has mixed or invalid ownership",
				}
			}
			if !s.requirements.wasApplied(requirement) {
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
			if !s.requirements.wasApplied(requirement) {
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
		if !s.requirements.wasApplied(requirement) {
			return &ScheduleError{
				Object: owner.Name(),
				Reason: "declaration requirement was not accepted by its owner",
			}
		}
	}
	return s.reconstructScheduledArtifact(owner)
}

func generatedCallableFacetMatches(
	facet api.CallableFacet,
	artifact *api.GeneratedArtifact,
) bool {
	if selected, ok := facet.ABI(); ok {
		return selected == artifact
	}
	if selected, ok := facet.GenericCapability(); ok {
		return selected == artifact
	}
	if selected, ok := facet.InterfaceMethod(); ok {
		return selected == artifact
	}
	return false
}

func requirementOwnerName(requirement api.DeclarationRequirement) string {
	return requirement.Owner().Name()
}

type artifactRevision struct {
	statements        []tsgo.Statement
	placement         *targetplacement.Owner
	dependencies      []api.ArtifactDependency
	requirements      []api.DeclarationRequirement
	eagerDependencies []api.ArtifactOwner
	contract          artifactstate.Contract
	classContribution *classMemberContribution
	temporaryStart    emitnaming.TemporarySnapshot
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
		site.Declaration,
		site.SourceFile.Syntax(),
		site.OutputPath,
	)
	if err != nil {
		return artifactRevision{}, err
	}
	defer finish()

	requirements := s.requirements.appliedFor(artifactOwner)
	handlerRequirements, selectedMethods, err :=
		s.partitionClassMethodRequirements(owner, requirements)
	if err != nil {
		return artifactRevision{}, err
	}
	context, err := builder.context.WithSourceArtifactOwner(artifactOwner)
	if err != nil {
		return artifactRevision{}, err
	}
	context, err = emitnaming.WithLexicalTypeRequirements(
		context,
		site.Declaration,
		artifactOwner,
		handlerRequirements,
	)
	if err != nil {
		return artifactRevision{}, err
	}
	result, err := builder.emitter.declarationObject(
		context,
		site.Declaration,
		owner,
		handlerRequirements,
	)
	if err != nil {
		return artifactRevision{}, err
	}
	requests, err := s.classArtifactRequests(
		owner,
		selectedMethods,
		result.Requests(),
	)
	if err != nil {
		return artifactRevision{}, err
	}
	var contribution *classMemberContribution
	if classOwner, members, ok := result.ClassMemberContribution(); ok {
		method, methodOK := owner.(*types.Func)
		if !methodOK ||
			method.Origin() != method ||
			api.MethodReceiverTypeName(method) != classOwner {
			return artifactRevision{}, &ScheduleError{
				Object: owner.Name(),
				Reason: "class-member contribution has a foreign owner",
			}
		}
		contribution = &classMemberContribution{
			owner:   classOwner,
			method:  method,
			members: members,
		}
		request, requestErr := api.NewClassMethodRequest(
			classOwner,
			method,
		)
		if requestErr != nil {
			return artifactRevision{}, requestErr
		}
		requests = append(requests, request)
	}
	placement, dependencies, declarationRequirements, err :=
		s.consumeArtifactRequests(
			artifactOwner,
			requests,
		)
	if err != nil {
		return artifactRevision{}, err
	}
	statements := result.Declarations()
	if len(selectedMethods) != 0 {
		statements, err = s.attachClassMemberContributions(
			builder,
			owner,
			statements,
			selectedMethods,
		)
		if err != nil {
			return artifactRevision{}, err
		}
	}
	var contract artifactstate.Contract
	switch result.Disposition() {
	case api.DeclarationDispositionMaterialized:
		contract, err = artifactstate.ProjectContract(s.factory, statements)
	case api.DeclarationDispositionCoverageOnly:
		contract, err = artifactstate.ProjectCoverageContract(
			s.factory,
			statements,
		)
	case api.DeclarationDispositionClassMemberContribution:
		if contribution == nil || len(statements) != 0 {
			err = &ScheduleError{
				Object: owner.Name(),
				Reason: "class-member artifact lost its contribution",
			}
			break
		}
		contract, err = artifactstate.ProjectClassMemberContract(
			s.factory,
			contribution.members,
		)
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
		statements:        statements,
		placement:         placement,
		dependencies:      dependencies,
		requirements:      declarationRequirements,
		eagerDependencies: eagerDeclarationDependencies(owner, dependencies),
		contract:          contract,
		classContribution: contribution,
		temporaryStart:    temporaryStart,
	}, nil
}
