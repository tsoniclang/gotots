package emit

import (
	"go/ast"
	"go/types"
	"sort"

	"github.com/tsoniclang/gotots/internal/emit/api"
	artifactstate "github.com/tsoniclang/gotots/internal/emit/artifact"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	emitordering "github.com/tsoniclang/gotots/internal/emit/ordering"
	providerboundary "github.com/tsoniclang/gotots/internal/emit/value/providerboundary"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func (s *programSession) ObserveCooperativeCallable(
	context api.Context,
	facet api.CallableFacet,
) (api.CooperativeCallableObservation, error) {
	return s.observeCooperativeCallable(
		context.ArtifactOwner(),
		&context,
		facet,
	)
}

func (s *programSession) ExactCallableFieldAssignments(
	field *types.Var,
) ([]ast.Expr, bool) {
	if s == nil || s.callableFields == nil {
		return nil, false
	}
	return s.callableFields.Assignments(field)
}

func (s *programSession) observeCooperativeCallable(
	consumer api.ArtifactOwner,
	context *api.Context,
	facet api.CallableFacet,
) (api.CooperativeCallableObservation, error) {
	if !consumer.Valid() || !facet.Valid() {
		return api.CooperativeCallableObservation{}, &ScheduleError{
			Reason: "cooperative callable facet is invalid",
		}
	}
	if context != nil && context.ConcurrencySemantics() != s.concurrency {
		return api.CooperativeCallableObservation{}, &ScheduleError{
			Reason: "callable observation execution profile is inconsistent",
		}
	}
	if s.concurrency == api.ConcurrencySemanticsDisabled {
		if context == nil {
			return api.NewCooperativeCallableObservation(false)
		}
		return s.observeDisabledCallable(*context, facet)
	}
	if observation, selected, err :=
		s.observeProviderCallable(context, facet); err != nil || selected {
		return observation, err
	}
	cooperative := false
	for _, requirement := range s.requirements.SelectedFor(facet.Owner()) {
		selected, selectedCooperative := requirement.CooperativeCallable()
		if !selectedCooperative {
			continue
		}
		if selected.Owner() != facet.Owner() {
			return api.CooperativeCallableObservation{}, &ScheduleError{
				Object: facet.Owner().Name(),
				Reason: "cooperative callable has inconsistent ownership",
			}
		}
		if selected == facet {
			cooperative = true
			break
		}
	}
	requests, err := callableEffectDependency(consumer, facet)
	if err != nil {
		return api.CooperativeCallableObservation{}, err
	}
	return api.NewCooperativeCallableObservation(cooperative, requests...)
}

func (s *programSession) observeDisabledCallable(
	context api.Context,
	facet api.CallableFacet,
) (api.CooperativeCallableObservation, error) {
	observation, selected, err := s.observeProviderCallable(&context, facet)
	if err != nil {
		return api.CooperativeCallableObservation{}, err
	}
	if selected {
		if err := providerboundary.RequireSynchronousSuspension(
			context,
			facet.Owner().Name(),
			observation.Cooperative(),
		); err != nil {
			return api.CooperativeCallableObservation{}, err
		}
		return api.NewCooperativeCallableObservation(
			false,
			observation.Requests()...,
		)
	}
	return api.NewCooperativeCallableObservation(false)
}

func (s *programSession) observeProviderCallable(
	context *api.Context,
	facet api.CallableFacet,
) (api.CooperativeCallableObservation, bool, error) {
	source, ok := facet.Owner().Source()
	if !ok || source == nil ||
		s.source.EnvironmentForTypes(source.Pkg()) == nil {
		return api.CooperativeCallableObservation{}, false, nil
	}
	function, callable := source.(*types.Func)
	if !callable {
		return api.CooperativeCallableObservation{}, false, nil
	}
	var profileRequests []api.RootRequest
	if context != nil && function.Signature().Recv() != nil {
		effect, selected, requests, err :=
			providerboundary.ResolveStatefulMethodEffect(*context, function)
		if err != nil {
			return api.CooperativeCallableObservation{}, false, err
		}
		profileRequests = requests
		if selected {
			observation, err := api.NewCooperativeCallableObservation(
				effect.MaySuspend(),
				profileRequests...,
			)
			return observation, true, err
		}
	}
	effect, providerOwned, err := s.registry.ProviderCallableEffect(function)
	if err != nil || !providerOwned {
		return api.CooperativeCallableObservation{}, false, err
	}
	observation, err := api.NewCooperativeCallableObservation(
		effect.MaySuspend(),
		profileRequests...,
	)
	return observation, true, err
}

func callableEffectDependency(
	consumer api.ArtifactOwner,
	facet api.CallableFacet,
) ([]api.RootRequest, error) {
	if consumer == facet.Owner() {
		return nil, nil
	}
	switch facet.Kind() {
	case api.CallableFacetSource,
		api.CallableFacetABI,
		api.CallableFacetInterfaceMethod,
		api.CallableFacetGenericCapability,
		api.CallableFacetGenericOperation:
		request, err := api.NewOwnedArtifactDependencyRequest(
			facet.Owner(),
			api.ArtifactFacetCallableSignature,
		)
		if err != nil {
			return nil, err
		}
		return []api.RootRequest{request}, nil
	case api.CallableFacetFunctionLiteral,
		api.CallableFacetPackageInitializer:
		return nil, &ScheduleError{
			Object: facet.Owner().Name(),
			Reason: "lexical callable facet escaped its source artifact",
		}
	default:
		return nil, &ScheduleError{
			Object: facet.Owner().Name(),
			Reason: "cooperative callable facet kind is invalid",
		}
	}
}

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
		contract, err := s.callableContract(false)
		if err != nil {
			return err
		}
		if err := s.commitArtifactContract(owner, contract, nil); err != nil {
			return err
		}
		s.artifacts.DiscardDirty(owner)
		return nil
	}
	cooperative, err := callableContractRequirements(
		selected,
		artifact,
	)
	if err != nil {
		return err
	}
	contract, err := s.callableContract(cooperative)
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
	contract, err := s.callableContract(false)
	if err != nil {
		return err
	}
	return s.commitArtifactContract(owner, contract, nil)
}

func (s *programSession) callableContract(
	cooperative bool,
) (artifactstate.Contract, error) {
	encoded, err := s.callableSignatureFacet(cooperative)
	if err != nil {
		return artifactstate.Contract{}, err
	}
	return artifactstate.NewContractFacet(
		api.ArtifactFacetCallableSignature,
		encoded,
	)
}

func (s *programSession) callableSignatureFacet(
	cooperative bool,
) ([]byte, error) {
	var result tsgo.TypeNode = s.factory.KeywordTypeNode(
		tsgo.KeywordTypeSyntaxKindVoidKeyword,
	)
	if cooperative {
		result = callable.PromiseResult(s.factory, result)
	}
	return tsgo.EncodeNode(
		s.factory.FunctionTypeNode(nil, nil, result),
	)
}

func callableContractRequirements(
	requirements []api.DeclarationRequirement,
	artifact *api.GeneratedArtifact,
) (bool, error) {
	definitions := 0
	cooperative := false
	for _, requirement := range requirements {
		if selected, ok := callableContractDefinition(
			requirement,
			artifact.Kind(),
		); ok {
			if selected != artifact {
				return false, &ScheduleError{
					Object: artifact.TargetName(),
					Reason: "callable contract received a foreign definition",
				}
			}
			definitions++
			continue
		}
		facet, ok := requirement.CooperativeCallable()
		selected, callable := callableContractFacet(facet)
		if !ok || !callable || selected != artifact || cooperative {
			return false, &ScheduleError{
				Object: artifact.TargetName(),
				Reason: "callable contract received a foreign requirement",
			}
		}
		cooperative = true
	}
	if definitions != 1 {
		return false, &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "callable contract requires exactly one definition request",
		}
	}
	return cooperative, nil
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

func callableContractFacet(
	facet api.CallableFacet,
) (*api.GeneratedArtifact, bool) {
	if artifact, ok := facet.ABI(); ok {
		return artifact, true
	}
	return facet.InterfaceMethod()
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
