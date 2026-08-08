package emit

import (
	"fmt"
	"go/types"
	"slices"
	"strings"

	environmentidentity "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/emit/api"
	artifactstate "github.com/tsoniclang/gotots/internal/emit/artifact"
	environmentcontract "github.com/tsoniclang/gotots/internal/emit/environmentcontract"
	emitnaming "github.com/tsoniclang/gotots/internal/emit/naming"
	targetplacement "github.com/tsoniclang/gotots/internal/emit/placement"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

// declarationRecord is the canonical root scheduling record for one
// declaration. It owns the declaration lifecycle together with the joined
// closed use demands, the sole implementation route, the typed provider
// selections, and whether the settled route emits a declaration artifact.
// There is no other use ledger.
type declarationRecord struct {
	queued     bool
	emitted    bool
	emitting   bool
	demands    uint16
	route      environmentidentity.ImplementationRoute
	selections []gostdlib.UseSelection
}

var declarationDemandOrder = []environmentidentity.UseDemand{
	environmentidentity.UseDemandTypeContract,
	environmentidentity.UseDemandValue,
	environmentidentity.UseDemandCallable,
	environmentidentity.UseDemandState,
	environmentidentity.UseDemandInitializer,
	environmentidentity.UseDemandInterfaceCapability,
	environmentidentity.UseDemandCallbackCapability,
	environmentidentity.UseDemandRuntimeFacet,
}

func (r *declarationRecord) joinDemand(
	demand environmentidentity.UseDemand,
) {
	r.demands |= 1 << uint16(demand)
}

func (r *declarationRecord) demandList() []environmentidentity.UseDemand {
	result := make(
		[]environmentidentity.UseDemand,
		0,
		len(declarationDemandOrder),
	)
	for _, demand := range declarationDemandOrder {
		if r.demands&(1<<uint16(demand)) != 0 {
			result = append(result, demand)
		}
	}
	return result
}

func (r *declarationRecord) joinSelection(selection gostdlib.UseSelection) {
	if selection.Kind() == gostdlib.UseSelectionNone {
		return
	}
	if slices.Contains(r.selections, selection) {
		return
	}
	r.selections = append(r.selections, selection)
	slices.SortFunc(r.selections, compareUseSelections)
}

// compareUseSelections orders typed provider selections structurally so the
// settled evidence is deterministic without flattening identity to strings.
func compareUseSelections(left, right gostdlib.UseSelection) int {
	if left.Kind() != right.Kind() {
		if left.Kind() < right.Kind() {
			return -1
		}
		return 1
	}
	leftKind, leftCapability, _ := left.Facet()
	rightKind, rightCapability, _ := right.Facet()
	if leftKind != rightKind {
		return strings.Compare(string(leftKind), string(rightKind))
	}
	if leftCapability != rightCapability {
		return strings.Compare(
			string(leftCapability),
			string(rightCapability),
		)
	}
	leftKey, _ := left.ProfileKey()
	rightKey, _ := right.ProfileKey()
	return strings.Compare(leftKey, rightKey)
}

// joinRoute settles the sole implementation route: the first observation
// sets it, repeated identical observations succeed, and a different route
// fails immediately.
func (r *declarationRecord) joinRoute(
	object types.Object,
	route environmentidentity.ImplementationRoute,
) error {
	if r.route == environmentidentity.RouteInvalid {
		r.route = route
		return nil
	}
	if r.route == route {
		return nil
	}
	return &ScheduleError{
		Object: object.Name(),
		Reason: "environment declaration selected implementation route " +
			route.String() + " but already settled " + r.route.String(),
	}
}

type scheduler struct {
	queue   []types.Object
	records map[types.Object]*declarationRecord
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
		records: make(map[types.Object]*declarationRecord),
	}
}

func (s *scheduler) record(object types.Object) *declarationRecord {
	existing := s.records[object]
	if existing == nil {
		existing = &declarationRecord{}
		s.records[object] = existing
	}
	return existing
}

func (s *scheduler) enqueue(object types.Object) {
	record := s.record(object)
	record.emitting = true
	if record.emitted || record.queued {
		return
	}
	record.queued = true
	s.queue = append(s.queue, object)
}

func (s *scheduler) next() (types.Object, bool) {
	if len(s.queue) == 0 {
		return nil, false
	}
	object := s.queue[0]
	s.queue = s.queue[1:]
	record := s.records[object]
	record.queued = false
	record.emitted = true
	return object, true
}

func (s *scheduler) hasPending() bool {
	if len(s.queue) != 0 {
		return true
	}
	for _, record := range s.records {
		if record.queued {
			return true
		}
	}
	return false
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
		if err := s.RequireUse(
			sourceOwner,
			environmentcontract.RequirementUseDemand(requirement),
			gostdlib.NoUseSelection(),
		); err != nil {
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
		if removed && len(requirements) == 0 &&
			generatedOwner.Placement() == api.GeneratedArtifactPlacementCompilation {
			return s.retireCompilationGeneratedArtifact(generatedOwner)
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
		publicName, nameErr := names.Declare(owner)
		if nameErr != nil {
			err = nameErr
			break
		}
		contract, err = artifactstate.ProjectSourceContract(
			s.factory,
			publicName,
			result.AdditionalPackageBindings(),
			statements,
		)
	case api.DeclarationDispositionCoverageOnly:
		contract, err = artifactstate.ProjectCoverageContract(
			s.factory,
			statements,
		)
	case api.DeclarationDispositionClassMemberContribution:
		if contribution == nil {
			err = &ScheduleError{
				Object: owner.Name(),
				Reason: "class-member artifact lost its contribution",
			}
			break
		}
		contract, err = artifactstate.ProjectClassMemberArtifactContract(
			s.factory,
			contribution.members,
			statements,
		)
	default:
		err = &ScheduleError{
			Object: owner.Name(),
			Reason: "declaration emission disposition is invalid",
		}
	}
	if err == nil {
		if function, callable := owner.(*types.Func); callable {
			recovery, recoveryErr := sourceCallableRecoveryRequirement(
				function,
				requirements,
			)
			if recoveryErr != nil {
				err = recoveryErr
			} else {
				contract, err = artifactstate.WithCallableRecovery(
					contract,
					s.factory,
					recovery,
				)
			}
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
