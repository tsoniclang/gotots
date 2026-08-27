package emit

import (
	"go/types"
	"sort"

	"github.com/tsoniclang/gotots/internal/contracts/sourceimplementation"
	"github.com/tsoniclang/gotots/internal/emit/api"
	artifactstate "github.com/tsoniclang/gotots/internal/emit/artifact"
	"github.com/tsoniclang/gotots/internal/emit/callableimplementation"
	emitordering "github.com/tsoniclang/gotots/internal/emit/ordering"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func (s *programSession) validateCallableContractArtifact(
	artifact *api.GeneratedArtifact,
) error {
	signature, ok := callableContractSignature(artifact)
	binding, found := s.registry.GeneratedArtifact(
		artifact.Kind(),
		artifact.ArtifactKey(),
	)
	boundSignature, bound := callableContractSignature(binding)
	if !ok ||
		!found ||
		binding != artifact ||
		!bound ||
		!types.Identical(boundSignature, signature) ||
		artifact.Placement() != api.GeneratedArtifactPlacementContract {
		return &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "callable contract artifact has no exact canonical binding",
		}
	}
	return nil
}

func (s *programSession) reconstructCallableContractArtifact(
	artifact *api.GeneratedArtifact,
) error {
	if s.sealed {
		return &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "callable contract reconstructed after target files were sealed",
		}
	}
	if err := s.validateCallableContractArtifact(artifact); err != nil {
		return err
	}
	owner := api.MustGeneratedArtifactOwner(artifact)
	selected := s.requirements.SelectedFor(owner)
	if len(selected) == 0 && s.requirementRemovalOwner == owner {
		contract, err := s.callableContract()
		if err != nil {
			return err
		}
		if err := s.commitArtifactContract(owner, contract, nil); err != nil {
			return err
		}
		s.artifacts.DiscardDirty(owner)
		return nil
	}
	if err := callableContractRequirements(selected, artifact); err != nil {
		return err
	}
	contract, err := s.callableContract()
	if err != nil {
		return err
	}
	if err := s.commitArtifactContract(owner, contract, nil); err != nil {
		return err
	}
	s.artifacts.DiscardDirty(owner)
	return nil
}

func (s *programSession) ensureCallableContractBaseline(
	artifact *api.GeneratedArtifact,
) error {
	owner := api.MustGeneratedArtifactOwner(artifact)
	if s.artifacts.FacetRevision(
		owner,
		api.ArtifactFacetCallableSignature,
	) != 0 {
		return nil
	}
	if err := s.validateCallableContractArtifact(artifact); err != nil {
		return err
	}
	contract, err := s.callableContract()
	if err != nil {
		return err
	}
	return s.commitArtifactContract(owner, contract, nil)
}

func (s *programSession) callableContract() (artifactstate.Contract, error) {
	encoded, err := s.callableSignatureFacet()
	if err != nil {
		return artifactstate.Contract{}, err
	}
	return artifactstate.NewContractFacet(
		api.ArtifactFacetCallableSignature,
		encoded,
	)
}

func (s *programSession) callableSignatureFacet() ([]byte, error) {
	result := s.factory.KeywordTypeNode(
		tsgo.KeywordTypeSyntaxKindVoidKeyword,
	)
	return tsgo.EncodeNode(
		s.factory.FunctionTypeNode(nil, nil, result),
	)
}

func callableContractRequirements(
	requirements []api.DeclarationRequirement,
	artifact *api.GeneratedArtifact,
) error {
	definitions := 0
	for _, requirement := range requirements {
		if selected, ok := callableContractDefinition(
			requirement,
			artifact.Kind(),
		); ok {
			if selected != artifact {
				return &ScheduleError{
					Object: artifact.TargetName(),
					Reason: "callable contract received a foreign definition",
				}
			}
			definitions++
			continue
		}
		return &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "callable contract received a foreign requirement",
		}
	}
	if definitions != 1 {
		return &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "callable contract requires exactly one definition request",
		}
	}
	return nil
}

func callableContractSignature(
	artifact *api.GeneratedArtifact,
) (*types.Signature, bool) {
	if artifact == nil {
		return nil, false
	}
	switch artifact.Kind() {
	case api.GeneratedArtifactCallableABI:
		return artifact.CallableABI()
	case api.GeneratedArtifactInterfaceMethodCallable:
		return artifact.InterfaceMethodCallableSignature()
	default:
		return nil, false
	}
}

func callableContractDefinition(
	requirement api.DeclarationRequirement,
	kind api.GeneratedArtifactKind,
) (*api.GeneratedArtifact, bool) {
	switch kind {
	case api.GeneratedArtifactCallableABI:
		return requirement.CallableABI()
	case api.GeneratedArtifactInterfaceMethodCallable:
		return requirement.InterfaceMethodCallable()
	default:
		return nil, false
	}
}

func (s *programSession) commitArtifactRevision(
	owner api.ArtifactOwner,
	contract artifactstate.Contract,
	dependencies []api.ArtifactDependency,
	requestRoots []api.RootRequest,
) error {
	if err := s.commitArtifactContract(owner, contract, dependencies); err != nil {
		return err
	}
	return s.requirements.Replace(owner, requestRoots)
}

func (s *programSession) commitArtifactContract(
	owner api.ArtifactOwner,
	contract artifactstate.Contract,
	dependencies []api.ArtifactDependency,
) error {
	if s.requirementRemovalOwner == owner {
		return s.artifacts.CommitHistoricalReplacement(
			owner,
			contract,
			dependencies,
		)
	}
	return s.artifacts.Commit(owner, contract, dependencies)
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
	if err := s.commitArtifactRevision(
		artifactOwner,
		revision.contract,
		revision.dependencies,
		revision.requestRoots,
	); err != nil {
		return err
	}
	s.commitClassMemberContribution(owner, revision.classContribution)
	s.artifacts.DiscardDirty(artifactOwner)
	declaration.statements = revision.statements
	declaration.placement = revision.placement
	declaration.eagerDependencies = revision.eagerDependencies
	declaration.reconstructions++
	return nil
}

func eagerDeclarationDependencies(
	consumer types.Object,
	dependencies []api.ArtifactDependency,
) []api.ArtifactOwner {
	if _, eager := consumer.(*types.Const); !eager {
		return nil
	}
	seen := make(map[api.ArtifactOwner]struct{})
	result := make([]api.ArtifactOwner, 0, len(dependencies))
	for _, dependency := range dependencies {
		provider := dependency.Provider()
		if _, sourceOwned := provider.Source(); !sourceOwned ||
			provider == api.MustSourceArtifactOwner(consumer) {
			continue
		}
		if _, duplicate := seen[provider]; duplicate {
			continue
		}
		seen[provider] = struct{}{}
		result = append(result, provider)
	}
	sort.Slice(result, func(left, right int) bool {
		return emitordering.CompareArtifactOwners(
			result[left],
			result[right],
		) < 0
	})
	return result
}

func (s *programSession) reconstructScheduledArtifact(
	owner api.ArtifactOwner,
) error {
	if accepted, err := s.acceptSourceImplementationReconstruction(owner); accepted {
		return err
	}
	if _, ok := owner.PackageAssembly(); ok {
		return s.reconstructPackageExports(owner)
	}
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
	if s.environmentArtifactSource(source) {
		return s.reconstructEnvironmentArtifact(source)
	}
	if variable, ok := source.(*types.Var); ok {
		return s.reconstructPackageStorage(owner, variable)
	}
	return s.reconstructArtifact(source)
}

func validateImplementationOwnership(
	program *load.Program,
	packages *sourceimplementation.Certificate,
	callables *callableimplementation.Certificate,
) error {
	if program == nil || packages == nil || callables == nil {
		return nil
	}
	for _, module := range callables.Modules() {
		sourcePackage := program.PackageByPath(module.PackagePath())
		if sourcePackage == nil {
			continue
		}
		if _, selected := packages.ForPackage(sourcePackage); selected {
			return &ScheduleError{
				Object: module.PackagePath(),
				Reason: "package and callable implementations overlap",
			}
		}
	}
	return nil
}

func (s *programSession) ResolveCallableImplementation(
	context api.Context,
	owner *types.Func,
	kernel bool,
) (api.CallableImplementationSelection, bool, error) {
	if s == nil || s.callableImplementations == nil {
		return api.CallableImplementationSelection{}, false, nil
	}
	current, ok := context.FunctionArtifactOwner()
	if !ok || current != owner || owner == nil || owner.Origin() != owner {
		return api.CallableImplementationSelection{}, false, &ScheduleError{
			Object: ownerName(owner),
			Reason: "callable implementation consumer has a foreign artifact owner",
		}
	}
	selected, ok := s.callableImplementations.ForFunction(owner)
	if !ok {
		return api.CallableImplementationSelection{}, false, nil
	}
	expected := callableimplementation.VariantSource
	targetVariant := api.CallableImplementationVariantSource
	if kernel {
		expected = callableimplementation.VariantKernel
		targetVariant = api.CallableImplementationVariantKernel
	}
	if selected.Variant() != expected {
		return api.CallableImplementationSelection{}, false, &ScheduleError{
			Object: selected.SourceIdentity(),
			Reason: "callable implementation variant differs from emitted ABI",
		}
	}
	selection, err := api.NewCallableImplementationSelection(
		selected.SourceIdentity(),
		selected.Module().OutputPath(),
		selected.Export(),
		targetVariant,
	)
	return selection, err == nil, err
}

func (s *programSession) AcceptCallableImplementation(
	selection api.CallableImplementationSelection,
	target api.CallableImplementationTarget,
) error {
	if s == nil || s.callableImplementations == nil ||
		!selection.Valid() || !target.Valid() {
		return &ScheduleError{Reason: "callable implementation target is invalid"}
	}
	selected, ok := s.callableImplementations.ImplementationByIdentity(
		selection.SourceIdentity(),
	)
	if !ok || selected.Module().OutputPath() != selection.OutputPath() ||
		selected.Export() != selection.Export() {
		return &ScheduleError{
			Object: selection.SourceIdentity(),
			Reason: "callable implementation selection differs from its certificate",
		}
	}
	if _, duplicate := s.callableImplementationTargets[selection.SourceIdentity()]; duplicate {
		return &ScheduleError{
			Object: selection.SourceIdentity(),
			Reason: "callable implementation target was accepted more than once",
		}
	}
	s.callableImplementationTargets[selection.SourceIdentity()] = target
	return nil
}

func (s *programSession) planCallableImplementations() error {
	if s.callableImplementations == nil {
		if len(s.callableImplementationTargets) != 0 {
			return &ScheduleError{Reason: "unselected callable implementation target survived"}
		}
		return nil
	}
	implementations := s.callableImplementations.Implementations()
	if len(s.callableImplementationTargets) != len(implementations) {
		return &ScheduleError{
			Reason: "not every selected callable implementation was consumed exactly once",
		}
	}
	targets := make([]callableimplementation.GeneratedTarget, 0, len(implementations))
	for _, implementation := range implementations {
		target, ok := s.callableImplementationTargets[implementation.SourceIdentity()]
		if !ok {
			return &ScheduleError{
				Object: implementation.SourceIdentity(),
				Reason: "selected callable implementation was not consumed",
			}
		}
		variant := callableimplementation.VariantSource
		if implementation.Variant() == callableimplementation.VariantKernel {
			variant = callableimplementation.VariantKernel
		}
		var (
			planned callableimplementation.GeneratedTarget
			err     error
		)
		switch target.Kind() {
		case api.CallableImplementationTargetModuleFunction:
			planned, err = callableimplementation.NewGeneratedModuleTarget(
				implementation.SourceIdentity(),
				variant,
				target.OutputPath(),
				target.Export(),
			)
		case api.CallableImplementationTargetStaticMethod:
			planned, err = callableimplementation.NewGeneratedStaticMethodTarget(
				implementation.SourceIdentity(),
				variant,
				target.OutputPath(),
				target.ClassName(),
				target.MemberName(),
			)
		default:
			err = &ScheduleError{
				Object: implementation.SourceIdentity(),
				Reason: "callable implementation target kind is invalid",
			}
		}
		if err != nil {
			return err
		}
		targets = append(targets, planned)
	}
	plan, err := s.callableImplementations.PlanGeneratedContracts(targets)
	if err != nil {
		return err
	}
	s.callableImplementationPlan = plan
	return nil
}

func ownerName(owner *types.Func) string {
	if owner == nil {
		return ""
	}
	return owner.FullName()
}
