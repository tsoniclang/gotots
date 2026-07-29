package emit

import (
	"crypto/sha256"
	"encoding/hex"
	"go/ast"
	"go/types"
	"sort"
	"strconv"

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
	key, err := s.genericOperationKey(
		owner,
		consumer,
		selection,
		signature,
	)
	if err != nil {
		return nil, err
	}
	identity := genericOperationIdentity{
		owner:    owner,
		consumer: consumer,
		key:      key,
	}
	if existing := s.genericOperations[identity]; existing != nil {
		if existing.Consumer() != consumer ||
			existing.Selection() != selection ||
			!types.Identical(existing.Signature(), signature) {
			return nil, &ScheduleError{
				Object: owner.Name(),
				Reason: "generic operation identity changed semantic contract",
			}
		}
		return existing, nil
	}
	digest := sha256.Sum256([]byte(key))
	targetName := "$go$" + selection.Operation().Identifier() + "_" +
		hex.EncodeToString(digest[:10])
	contract, err := api.NewGenericOperationContract(
		owner,
		key,
		targetName,
		consumer,
		selection,
		signature,
	)
	if err != nil {
		return nil, err
	}
	s.genericOperations[identity] = contract
	return contract, nil
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
		function, functionOwner := site.declaration.(*ast.FuncDecl)
		var root *types.Scope
		if functionOwner {
			root = site.source.TypesInfo().Scopes[function.Type]
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
	cooperative := false
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
		case api.CallableFacetSource, api.CallableFacetABI:
			request, err := api.NewOwnedArtifactDependencyRequest(
				facet.Owner(),
				api.ArtifactFacetCallableSignature,
			)
			if err != nil {
				return api.CooperativeCallableObservation{}, err
			}
			requests = append(requests, request)
		case api.CallableFacetFunctionLiteral:
			return api.CooperativeCallableObservation{}, &ScheduleError{
				Object: facet.Owner().Name(),
				Reason: "function-literal facet escaped its source artifact",
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

func (s *programSession) validateCallableABIArtifact(
	artifact *api.GeneratedArtifact,
) error {
	signature, ok := artifact.CallableABI()
	binding, found := s.registry.GeneratedArtifact(
		api.GeneratedArtifactCallableABI,
		artifact.ArtifactKey(),
	)
	boundSignature, bound := binding.CallableABI()
	if !ok ||
		!found ||
		binding != artifact ||
		!bound ||
		!types.Identical(boundSignature, signature) ||
		artifact.Placement() != api.GeneratedArtifactPlacementContract {
		return &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "callable ABI artifact has no exact canonical binding",
		}
	}
	return nil
}

func (s *programSession) reconstructCallableABIArtifact(
	artifact *api.GeneratedArtifact,
) error {
	if s.sealed {
		return &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "callable ABI reconstructed after target files were sealed",
		}
	}
	if err := s.validateCallableABIArtifact(artifact); err != nil {
		return err
	}
	owner := api.MustGeneratedArtifactOwner(artifact)
	cooperative, err := callableABIRequirements(
		s.requirements.appliedFor(owner),
		artifact,
	)
	if err != nil {
		return err
	}
	contract, err := s.callableABIContract(cooperative)
	if err != nil {
		return err
	}
	if err := s.artifacts.Commit(owner, contract, nil); err != nil {
		return err
	}
	s.artifacts.DiscardDirty(owner)
	return nil
}

func (s *programSession) ensureCallableABIBaseline(
	artifact *api.GeneratedArtifact,
) error {
	owner := api.MustGeneratedArtifactOwner(artifact)
	if s.artifacts.FacetRevision(
		owner,
		api.ArtifactFacetCallableSignature,
	) != 0 {
		return nil
	}
	if err := s.validateCallableABIArtifact(artifact); err != nil {
		return err
	}
	contract, err := s.callableABIContract(false)
	if err != nil {
		return err
	}
	return s.artifacts.Commit(owner, contract, nil)
}

func (s *programSession) callableABIContract(
	cooperative bool,
) (artifactstate.Contract, error) {
	var result tsgo.TypeNode = s.factory.KeywordTypeNode(
		tsgo.KeywordTypeSyntaxKindVoidKeyword,
	)
	if cooperative {
		result = callable.PromiseResult(s.factory, result)
	}
	encoded, err := tsgo.EncodeNode(
		s.factory.FunctionTypeNode(nil, nil, result),
	)
	if err != nil {
		return nil, err
	}
	return artifactstate.Contract{
		api.ArtifactFacetCallableSignature: encoded,
	}, nil
}

func callableABIRequirements(
	requirements []api.DeclarationRequirement,
	artifact *api.GeneratedArtifact,
) (bool, error) {
	definitions := 0
	cooperative := false
	for _, requirement := range requirements {
		if selected, ok := requirement.CallableABI(); ok {
			if selected != artifact {
				return false, &ScheduleError{
					Object: artifact.TargetName(),
					Reason: "callable ABI received a foreign definition",
				}
			}
			definitions++
			continue
		}
		facet, ok := requirement.CooperativeCallable()
		selected, abi := facet.ABI()
		if !ok || !abi || selected != artifact || cooperative {
			return false, &ScheduleError{
				Object: artifact.TargetName(),
				Reason: "callable ABI received a foreign requirement",
			}
		}
		cooperative = true
	}
	if definitions != 1 {
		return false, &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "callable ABI requires exactly one definition request",
		}
	}
	return cooperative, nil
}
