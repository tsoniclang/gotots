package naming

import (
	"crypto/sha256"
	"encoding/hex"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/type/typeidentity"
	"github.com/tsoniclang/gotots/internal/output"
)

const genericCapabilityTargetNameHexLength = 20

func (n *File) GenericCapability(
	selection api.GenericOperationSelection,
	signature *types.Signature,
) (api.NameReference, error) {
	if !selection.Valid() || signature == nil {
		return api.NameReference{}, &api.NameError{
			Reason: "generic-capability contract is invalid",
		}
	}
	signatureKey, err := typeidentity.BuildKey(
		signature,
		n.generatedNamedObjectIdentity,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	operationKey, err := selection.IdentityPrefix()
	if err != nil {
		return api.NameReference{}, err
	}
	digest := sha256.Sum256(
		[]byte(operationKey + "|" + signatureKey),
	)
	artifactKey := hex.EncodeToString(digest[:])
	placement, err := n.generatedArtifactPlacement(signature)
	if err != nil {
		return api.NameReference{}, err
	}
	binding, err := n.owner.registry.internGenericCapability(
		artifactKey,
		selection,
		signature,
		placement,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	definition, err := api.NewGenericCapabilityRequest(binding.owner)
	if err != nil {
		return api.NameReference{}, err
	}
	return n.generatedValueReference(
		binding.owner,
		binding.name,
		definition,
		api.ArtifactFacetCallableSignature,
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
