package emit

import (
	"crypto/sha256"
	"encoding/hex"
	"go/types"
	"strconv"
	"strings"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/type/typeidentity"
)

type genericOperationIdentity struct {
	owner    types.Object
	consumer api.GenericOperationConsumer
	key      string
}

type genericConcretizationIdentity struct {
	owner *types.Func
	key   string
}

func (s *programSession) GenericCallableRequiresConcretization(
	owner *types.Func,
) (bool, error) {
	if owner == nil || owner.Origin() != owner {
		return false, &ScheduleError{
			Reason: "generic concretization owner is invalid",
		}
	}
	if _, sourceOwned := s.sites[owner]; !sourceOwned {
		if s.source.EnvironmentForTypes(owner.Pkg()) != nil {
			_, kernelOwned, err := s.registry.ProviderGenericKernel(owner)
			return kernelOwned, err
		}
		return false, &ScheduleError{
			Object: owner.Name(),
			Reason: "generic concretization owner has no declaration",
		}
	}
	artifactOwner := api.MustSourceArtifactOwner(owner)
	return api.GenericKernelRequired(
		owner,
		s.requirements.appliedFor(artifactOwner),
	)
}

func (s *programSession) ResolveGenericConcretization(
	owner *types.Func,
	arguments api.TypeArgumentList,
	signature *types.Signature,
	placement api.GeneratedArtifactPlacement,
	lexicalOwner api.ArtifactOwner,
	anchor *types.TypeName,
) (*api.GenericConcretization, error) {
	if owner == nil || owner.Origin() != owner || arguments.Len() == 0 ||
		signature == nil ||
		(placement != api.GeneratedArtifactPlacementCompilation &&
			placement != api.GeneratedArtifactPlacementLexical) {
		return nil, &ScheduleError{
			Reason: "generic concretization identity is invalid",
		}
	}
	if _, sourceOwned := s.sites[owner]; !sourceOwned {
		if s.source.EnvironmentForTypes(owner.Pkg()) == nil {
			return nil, &ScheduleError{
				Object: owner.Name(),
				Reason: "generic concretization has no selected declaration",
			}
		}
		_, kernelOwned, err := s.registry.ProviderGenericKernel(owner)
		if err != nil {
			return nil, err
		}
		if !kernelOwned {
			return nil, &ScheduleError{
				Object: owner.Name(),
				Reason: "generic concretization has no certified provider kernel",
			}
		}
	}
	sourceSignature, ok := owner.Type().(*types.Signature)
	if !ok || len(api.GenericDeclarationParameters(owner)) != arguments.Len() ||
		(sourceSignature.Recv() == nil) != (signature.Recv() == nil) {
		return nil, &ScheduleError{
			Object: owner.Name(),
			Reason: "generic concretization arguments do not match the source declaration",
		}
	}
	selected := make([]types.Type, 0, arguments.Len())
	var identity strings.Builder
	ownerIdentity, err := typeidentity.SourceObjectKey(owner)
	if err != nil {
		return nil, err
	}
	identity.WriteString("generic-concretization|")
	identity.WriteString(strconv.Itoa(len(ownerIdentity)))
	identity.WriteByte(':')
	identity.WriteString(ownerIdentity)
	identity.WriteByte('|')
	namedIdentity := s.genericConcretizationNamedIdentity(
		placement,
		lexicalOwner,
	)
	for index := range arguments.Len() {
		argument := arguments.At(index)
		if api.ContainsGenericTypeParameter(argument) {
			return nil, &ScheduleError{
				Object: owner.Name(),
				Reason: "generic concretization argument " +
					strconv.Itoa(index) + " remains open: " +
					types.TypeString(argument, nil),
			}
		}
		descriptor, err := typeidentity.BuildDescriptor(
			argument,
			namedIdentity,
		)
		if err != nil {
			return nil, err
		}
		selected = append(selected, argument)
		identity.WriteString(strconv.Itoa(len(descriptor)))
		identity.WriteByte(':')
		identity.WriteString(descriptor)
	}
	digest := sha256.Sum256([]byte(identity.String()))
	key := hex.EncodeToString(digest[:])
	selection := genericConcretizationIdentity{owner: owner, key: key}
	if existing := s.genericConcretizations[selection]; existing != nil {
		if !types.Identical(existing.Signature(), signature) {
			return nil, &ScheduleError{
				Object: owner.Name(),
				Reason: "generic concretization key joined different signatures",
			}
		}
		if existing.Placement() != placement ||
			existing.LexicalOwner() != lexicalOwner ||
			existing.LexicalAnchor() != anchor {
			return nil, &ScheduleError{
				Object: owner.Name(),
				Reason: "generic concretization key joined different placements",
			}
		}
		return existing, nil
	}
	expected, err := api.InstantiateGenericCallable(owner, arguments)
	if err != nil {
		return nil, err
	}
	if !types.Identical(expected, signature) {
		return nil, &ScheduleError{
			Object: owner.Name(),
			Reason: "generic concretization signature differs from its exact instance",
		}
	}
	concretization, err := api.NewGenericConcretization(
		owner,
		selected,
		signature,
		key,
		"$concrete_"+key[:20],
		placement,
		lexicalOwner,
		anchor,
	)
	if err != nil {
		return nil, err
	}
	s.genericConcretizations[selection] = concretization
	return concretization, nil
}

func (s *programSession) genericConcretizationNamedIdentity(
	placement api.GeneratedArtifactPlacement,
	lexicalOwner api.ArtifactOwner,
) typeidentity.NamedObjectIdentity {
	return func(object *types.TypeName) (string, error) {
		if object == nil {
			return "", &api.NameError{
				Reason: "generic concretization named type is nil",
			}
		}
		if object.Pkg() == nil ||
			object.Parent() == object.Pkg().Scope() {
			return typeidentity.NamedObjectKey(object)
		}
		if placement != api.GeneratedArtifactPlacementLexical ||
			!lexicalOwner.Valid() ||
			lexicalOwner.Package() != object.Pkg() {
			return "", &api.NameError{
				Name:   object.Name(),
				Reason: "generic concretization local type has no lexical placement",
			}
		}
		return typeidentity.LexicalNamedObjectKey(
			object,
			lexicalOwner,
			object.Pkg().Scope(),
		)
	}
}
