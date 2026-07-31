package emit

import (
	"crypto/sha256"
	"encoding/hex"
	"go/ast"
	"go/types"
	"sort"
	"strconv"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/emit/api"
	artifactstate "github.com/tsoniclang/gotots/internal/emit/artifact"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	"github.com/tsoniclang/gotots/internal/emit/type/typeidentity"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type genericOperationIdentity struct {
	owner    types.Object
	consumer api.GenericOperationConsumer
	key      string
}

type genericCallableProfileIdentity struct {
	owner *types.Func
	key   string
}

func (s *programSession) ResolveGenericCallableProfile(
	owner *types.Func,
	selection api.GenericCallableProfileSelection,
) (*api.GenericCallableProfile, error) {
	if owner == nil {
		return nil, &ScheduleError{
			Reason: "generic callable profile owner is nil",
		}
	}
	owner = owner.Origin()
	_, sourceOwned := s.sites[owner]
	environmentOwned :=
		s.source.EnvironmentForTypes(owner.Pkg()) != nil
	if (!sourceOwned && !environmentOwned) ||
		len(api.GenericDeclarationParameters(owner)) == 0 ||
		!selection.Valid() ||
		!selection.Cooperative() {
		return nil, &ScheduleError{
			Object: owner.Name(),
			Reason: "generic callable profile identity is invalid",
		}
	}
	identity := genericCallableProfileIdentity{
		owner: owner,
		key:   selection.Key(),
	}
	if existing := s.genericProfiles[identity]; existing != nil {
		return existing, nil
	}
	digest := sha256.Sum256([]byte(
		"generic-callable-profile|" + selection.Key(),
	))
	suffix := "$cooperative_" + hex.EncodeToString(digest[:10])
	profile, err := api.NewGenericCallableProfile(
		owner,
		selection,
		suffix,
	)
	if err != nil {
		return nil, err
	}
	s.genericProfiles[identity] = profile
	return profile, nil
}

func (s *programSession) ResolveGenericOperationSet(
	declaration types.Object,
	consumer api.GenericOperationConsumer,
) (api.GenericOperationSet, bool, error) {
	owner := api.GenericDeclarationOrigin(declaration)
	if owner == nil ||
		!consumer.Valid() ||
		len(api.GenericDeclarationParameters(owner)) == 0 {
		return api.GenericOperationSet{}, false, nil
	}
	if _, ok := s.sites[owner]; !ok {
		if s.source.EnvironmentForTypes(owner.Pkg()) != nil {
			return s.providerGenericOperationSet(owner, consumer)
		}
		return api.GenericOperationSet{}, false, &ScheduleError{
			Object: owner.Name(),
			Reason: "generic operation set has no source declaration",
		}
	}
	var operations []*api.GenericOperationContract
	for _, requirement := range s.requirements.appliedFor(
		api.MustSourceArtifactOwner(owner),
	) {
		requirementOwner, operation, generic :=
			requirement.GenericOperation()
		if !generic {
			continue
		}
		if requirementOwner != owner ||
			operation.Consumer() != consumer {
			if requirementOwner == owner {
				continue
			}
			return api.GenericOperationSet{}, false, &ScheduleError{
				Object: owner.Name(),
				Reason: "generic operation has inconsistent ownership",
			}
		}
		operations = append(operations, operation)
	}
	sort.Slice(operations, func(left, right int) bool {
		return operations[left].Key() < operations[right].Key()
	})
	operationSet, err := api.NewGenericOperationSet(
		owner,
		consumer,
		operations,
	)
	return operationSet, err == nil, err
}

func (s *programSession) ResolveGenericOperation(
	declaration types.Object,
	consumer api.GenericOperationConsumer,
	selection api.GenericOperationSelection,
	signature *types.Signature,
) (*api.GenericOperationContract, error) {
	owner := api.GenericDeclarationOrigin(declaration)
	if owner == nil {
		return nil, &ScheduleError{
			Reason: "generic operation owner is nil",
		}
	}
	if !consumer.Valid() || !selection.Valid() || signature == nil {
		return nil, &ScheduleError{
			Object: owner.Name(),
			Reason: "generic operation identity is invalid",
		}
	}
	if _, ok := s.sites[owner]; !ok ||
		len(api.GenericDeclarationParameters(owner)) == 0 {
		return nil, &ScheduleError{
			Object: owner.Name(),
			Reason: "generic operation owner has no generic declaration",
		}
	}
	return s.internGenericOperation(
		owner,
		consumer,
		selection,
		signature,
	)
}

func (s *programSession) genericOperationKey(
	owner types.Object,
	consumer api.GenericOperationConsumer,
	selection api.GenericOperationSelection,
	signature *types.Signature,
) (string, error) {
	signatureKey, err := typeidentity.BuildParameterizedKey(
		signature,
		s.genericOperationNamedIdentity(owner),
		genericOperationParameterIdentity(owner),
	)
	if err != nil {
		return "", err
	}
	prefix, err := selection.IdentityPrefix()
	if err != nil {
		return "", err
	}
	return consumer.Identity() + "|" + prefix + "|" + signatureKey, nil
}

func (s *programSession) genericOperationNamedIdentity(
	owner types.Object,
) typeidentity.NamedObjectIdentity {
	return func(object *types.TypeName) (string, error) {
		if object == nil || object.Pkg() == nil {
			return "", &api.NameError{
				Reason: "generic operation named type has no package identity",
			}
		}
		if object.Parent() == object.Pkg().Scope() {
			return typeidentity.NamedObjectKey(object)
		}
		site, ok := s.sites[owner]
		if !ok {
			return "", &api.NameError{
				Name:   object.Name(),
				Reason: "generic operation local type has no owning declaration",
			}
		}
		function, functionOwner := site.Declaration.(*ast.FuncDecl)
		var root *types.Scope
		if functionOwner {
			root = site.Source.TypesInfo().Scopes[function.Type]
		}
		if owner.Pkg() != object.Pkg() || root == nil {
			return "", &api.NameError{
				Name:   object.Name(),
				Reason: "generic operation local type has no owning declaration",
			}
		}
		return typeidentity.LexicalNamedObjectKey(
			object,
			api.MustSourceArtifactOwner(owner),
			root,
		)
	}
}

func genericOperationParameterIdentity(
	owner types.Object,
) typeidentity.TypeParameterIdentity {
	identities := make(map[*types.TypeParam]string)
	switch owner := owner.(type) {
	case *types.Func:
		signature, _ := owner.Type().(*types.Signature)
		if signature != nil {
			for index := range signature.RecvTypeParams().Len() {
				identities[signature.RecvTypeParams().At(index)] =
					"receiver|" + strconv.Itoa(index)
			}
			for index := range signature.TypeParams().Len() {
				identities[signature.TypeParams().At(index)] =
					"function|" + strconv.Itoa(index)
			}
		}
	case *types.TypeName:
		for index, parameter := range api.GenericDeclarationParameters(owner) {
			identities[parameter] = "type|" + strconv.Itoa(index)
		}
	}
	return func(parameter *types.TypeParam) (string, error) {
		identity := identities[parameter]
		if identity == "" {
			name := ""
			if parameter != nil && parameter.Obj() != nil {
				name = parameter.Obj().Name()
			}
			return "", &api.NameError{
				Name:   name,
				Reason: "generic operation type parameter is foreign to its owner",
			}
		}
		return identity, nil
	}
}

func (s *programSession) ObserveCooperativeCallable(
	consumer api.ArtifactOwner,
	facet api.CallableFacet,
) (api.CooperativeCallableObservation, error) {
	if !consumer.Valid() || !facet.Valid() {
		return api.CooperativeCallableObservation{}, &ScheduleError{
			Reason: "cooperative callable facet is invalid",
		}
	}
	if profile, profiled := facet.GenericProfile(); profiled {
		effect, providerOwned, err :=
			s.registry.ProviderGenericCallableEffect(
				profile.Owner(),
				profile.Selection().Key(),
			)
		if err != nil {
			return api.CooperativeCallableObservation{}, err
		}
		if providerOwned {
			return api.NewCooperativeCallableObservation(
				effect == gostdlib.EffectAsynchronous,
			)
		}
	}
	if source, ok := facet.Owner().Source(); ok && source != nil &&
		s.source.EnvironmentForTypes(source.Pkg()) != nil {
		function, callable := source.(*types.Func)
		if callable {
			effect, providerOwned, err :=
				s.registry.ProviderCallableEffect(function)
			if err != nil {
				return api.CooperativeCallableObservation{}, err
			}
			if providerOwned {
				return api.NewCooperativeCallableObservation(
					effect == gostdlib.EffectAsynchronous,
				)
			}
		}
	}
	cooperative := false
	if profile, artifact, profiled := facet.GenericProfileABI(); profiled {
		selected, found := profile.Selection().ABI(artifact)
		cooperative = found && selected
	}
	for _, requirement := range s.requirements.appliedFor(
		facet.Owner(),
	) {
		selected, selectedCooperative :=
			requirement.CooperativeCallable()
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
	var requests []api.RootRequest
	if consumer != facet.Owner() {
		switch facet.Kind() {
		case api.CallableFacetSource,
			api.CallableFacetABI,
			api.CallableFacetInterfaceMethod,
			api.CallableFacetGenericCapability,
			api.CallableFacetGenericOperation,
			api.CallableFacetGenericProfile:
			request, err := api.NewOwnedArtifactDependencyRequest(
				facet.Owner(),
				api.ArtifactFacetCallableSignature,
			)
			if err != nil {
				return api.CooperativeCallableObservation{}, err
			}
			requests = append(requests, request)
		case api.CallableFacetFunctionLiteral,
			api.CallableFacetPackageInitializer:
			return api.CooperativeCallableObservation{}, &ScheduleError{
				Object: facet.Owner().Name(),
				Reason: "lexical callable facet escaped its source artifact",
			}
		default:
			return api.CooperativeCallableObservation{}, &ScheduleError{
				Object: facet.Owner().Name(),
				Reason: "cooperative callable facet kind is invalid",
			}
		}
	}
	return api.NewCooperativeCallableObservation(
		cooperative,
		requests...,
	)
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
	cooperative, err := callableContractRequirements(
		s.requirements.appliedFor(owner),
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
	if _, _, profiled := facet.GenericProfileABI(); profiled {
		return nil, false
	}
	if artifact, ok := facet.ABI(); ok {
		return artifact, true
	}
	return facet.InterfaceMethod()
}

func compareGenericProfileABIs(
	left api.CallableFacet,
	right api.CallableFacet,
) (int, bool) {
	leftProfile, leftABI, leftOK := left.GenericProfileABI()
	rightProfile, rightABI, rightOK := right.GenericProfileABI()
	if leftOK != rightOK {
		if leftOK {
			return 1, true
		}
		return -1, true
	}
	if !leftOK {
		return 0, false
	}
	switch {
	case leftProfile.Key() < rightProfile.Key():
		return -1, true
	case leftProfile.Key() > rightProfile.Key():
		return 1, true
	case leftABI.ArtifactKey() < rightABI.ArtifactKey():
		return -1, true
	case leftABI.ArtifactKey() > rightABI.ArtifactKey():
		return 1, true
	default:
		return 0, true
	}
}
