package emit

import (
	"crypto/sha256"
	"encoding/hex"
	"go/ast"
	"go/types"
	"sort"
	"strconv"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/type/typeidentity"
)

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
		if object == nil {
			return "", &api.NameError{
				Reason: "generic operation named type is nil",
			}
		}
		if object.Pkg() == nil {
			return typeidentity.NamedObjectKey(object)
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
