package naming

import (
	"crypto/sha256"
	"encoding/hex"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/type/typeidentity"
	"github.com/tsoniclang/gotots/internal/output"
)

const (
	genericCapabilityTargetNameHexLength = 20
	callableABITargetNameHexLength       = 20
)

func (n *File) GenericCapability(
	selection api.GenericOperationSelection,
	signature *types.Signature,
) (api.GenericCapabilityReference, error) {
	if !selection.Valid() || signature == nil {
		return api.GenericCapabilityReference{}, &api.NameError{
			Reason: "generic-capability contract is invalid",
		}
	}
	signatureKey, err := typeidentity.BuildKey(
		signature,
		n.generatedNamedObjectIdentity,
	)
	if err != nil {
		return api.GenericCapabilityReference{}, err
	}
	operationKey, err := selection.IdentityPrefix()
	if err != nil {
		return api.GenericCapabilityReference{}, err
	}
	digest := sha256.Sum256(
		[]byte(operationKey + "|" + signatureKey),
	)
	artifactKey := hex.EncodeToString(digest[:])
	placement, err := n.generatedArtifactPlacement(signature)
	if err != nil {
		return api.GenericCapabilityReference{}, err
	}
	binding, err := n.owner.registry.internGenericCapability(
		artifactKey,
		selection,
		signature,
		placement,
	)
	if err != nil {
		return api.GenericCapabilityReference{}, err
	}
	definition, err := api.NewGenericCapabilityRequest(binding.owner)
	if err != nil {
		return api.GenericCapabilityReference{}, err
	}
	reference, err := n.generatedValueReference(
		binding.owner,
		binding.name,
		definition,
		api.ArtifactFacetCallableSignature,
	)
	if err != nil {
		return api.GenericCapabilityReference{}, err
	}
	return api.NewGenericCapabilityReference(
		binding.owner,
		reference.Name(),
		reference.Requests()...,
	)
}

func (r *Registry) internGenericCapability(
	artifactKey string,
	selection api.GenericOperationSelection,
	signature *types.Signature,
	placement generatedArtifactPlacement,
) (genericCapabilityBinding, error) {
	if r == nil ||
		!selection.Valid() ||
		signature == nil ||
		artifactKey == "" ||
		!placement.kind.Valid() {
		return genericCapabilityBinding{}, &api.NameError{
			Reason: "generic-capability canonicalization input is invalid",
		}
	}
	if existing, ok := r.genericCapabilities[artifactKey]; ok {
		existingSignature, existingOperation, valid :=
			existing.owner.GenericCapability()
		if !valid ||
			existingOperation != selection ||
			!types.Identical(existingSignature, signature) {
			return genericCapabilityBinding{}, &api.NameError{
				Name:   existing.name,
				Reason: "generic-capability key joined non-identical contracts",
			}
		}
		if !sameGeneratedPlacement(existing.owner, placement) {
			return genericCapabilityBinding{}, &api.NameError{
				Name:   existing.name,
				Reason: "generic-capability placement is inconsistent",
			}
		}
		return existing, nil
	}
	if len(artifactKey) < genericCapabilityTargetNameHexLength {
		return genericCapabilityBinding{}, &api.NameError{
			Reason: "generic-capability artifact key is invalid",
		}
	}
	name := "$goCapability_" +
		artifactKey[len(artifactKey)-genericCapabilityTargetNameHexLength:]
	if err := reserveGeneratedName(
		r.genericCapabilityNames,
		name,
		artifactKey,
		"generic-capability",
	); err != nil {
		return genericCapabilityBinding{}, err
	}
	owner, err := newGenericCapabilityArtifact(
		selection,
		signature,
		artifactKey,
		name,
		placement,
	)
	if err != nil {
		return genericCapabilityBinding{}, err
	}
	binding := genericCapabilityBinding{owner: owner, name: name}
	r.genericCapabilities[artifactKey] = binding
	return binding, nil
}

func newGenericCapabilityArtifact(
	selection api.GenericOperationSelection,
	signature *types.Signature,
	artifactKey string,
	name string,
	placement generatedArtifactPlacement,
) (*api.GeneratedArtifact, error) {
	if placement.kind == api.GeneratedArtifactPlacementLexical {
		return api.NewLexicalGenericCapabilityArtifact(
			selection,
			signature,
			artifactKey,
			name,
			placement.lexicalOwner,
			placement.anchor,
		)
	}
	outputPath, err := output.GenericCapabilityPath(artifactKey)
	if err != nil {
		return nil, err
	}
	return api.NewCompilationGenericCapabilityArtifact(
		selection,
		signature,
		artifactKey,
		name,
		outputPath,
	)
}

func (n *File) CallableABI(
	signature *types.Signature,
) (api.CallableABIReference, error) {
	if signature == nil || signature.Recv() != nil {
		return api.CallableABIReference{}, &api.NameError{
			Reason: "callable ABI signature is invalid",
		}
	}
	signatureKey, err := typeidentity.BuildParameterizedKey(
		signature,
		n.generatedNamedObjectIdentity,
		func(parameter *types.TypeParam) (string, error) {
			if parameter == nil || parameter.Obj() == nil {
				return "", &api.NameError{
					Reason: "callable ABI has an unbound type parameter",
				}
			}
			return n.generatedNamedObjectIdentity(parameter.Obj())
		},
	)
	if err != nil {
		return api.CallableABIReference{}, err
	}
	digest := sha256.Sum256([]byte("callable-abi|" + signatureKey))
	artifactKey := hex.EncodeToString(digest[:])
	binding, err := n.owner.registry.internCallableABI(
		artifactKey,
		signature,
	)
	if err != nil {
		return api.CallableABIReference{}, err
	}
	definition, err := api.NewCallableABIRequest(binding.owner)
	if err != nil {
		return api.CallableABIReference{}, err
	}
	requests := []api.RootRequest{definition}
	owner := api.MustGeneratedArtifactOwner(binding.owner)
	if n.artifactOwner.Valid() && n.artifactOwner != owner {
		dependency, dependencyErr :=
			api.NewGeneratedArtifactDependencyRequest(
				binding.owner,
				api.ArtifactFacetCallableSignature,
			)
		if dependencyErr != nil {
			return api.CallableABIReference{}, dependencyErr
		}
		requests = append(requests, dependency)
	}
	return api.NewCallableABIReference(
		binding.owner,
		requests...,
	)
}

func (r *Registry) internCallableABI(
	artifactKey string,
	signature *types.Signature,
) (callableABIBinding, error) {
	if r == nil ||
		len(artifactKey) != sha256.Size*2 ||
		signature == nil ||
		signature.Recv() != nil {
		return callableABIBinding{}, &api.NameError{
			Reason: "callable ABI canonicalization input is invalid",
		}
	}
	if existing, ok := r.callableABIs[artifactKey]; ok {
		existingSignature, valid := existing.owner.CallableABI()
		if !valid || !types.Identical(existingSignature, signature) {
			return callableABIBinding{}, &api.NameError{
				Name:   existing.name,
				Reason: "callable ABI key joined non-identical signatures",
			}
		}
		return existing, nil
	}
	name := "$goCallable_" +
		artifactKey[len(artifactKey)-callableABITargetNameHexLength:]
	if err := reserveGeneratedName(
		r.callableABINames,
		name,
		artifactKey,
		"callable ABI",
	); err != nil {
		return callableABIBinding{}, err
	}
	owner, err := api.NewContractGeneratedArtifact(
		api.GeneratedArtifactCallableABI,
		signature,
		artifactKey,
		name,
	)
	if err != nil {
		return callableABIBinding{}, err
	}
	binding := callableABIBinding{owner: owner, name: name}
	r.callableABIs[artifactKey] = binding
	return binding, nil
}
