package naming

import (
	"crypto/sha256"
	"encoding/hex"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/type/typeidentity"
	"github.com/tsoniclang/gotots/internal/output"
)

const deferredCallableRegistryNameHexLength = 20

func (n *File) DeferredCallableRegistry(
	signature *types.Signature,
) (api.NameReference, error) {
	signature, err := deferredCallableSignature(signature)
	if err != nil {
		return api.NameReference{}, err
	}
	if reason := deferredRegistrySignatureReason(signature); reason != "" {
		return api.NameReference{}, &api.NameError{
			Name:   types.TypeString(signature, packageQualifier),
			Reason: "deferred-callable registry signature is not closed: " + reason,
		}
	}
	signatureKey, err := typeidentity.BuildKey(
		signature,
		n.generatedNamedObjectIdentity,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	digest := sha256.Sum256([]byte("deferred-callable|" + signatureKey))
	artifactKey := hex.EncodeToString(digest[:])
	binding, err := n.owner.registry.internDeferredCallableRegistry(
		artifactKey,
		signature,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	definition, err := api.NewDeferredCallableRegistryRequest(binding.owner)
	if err != nil {
		return api.NameReference{}, err
	}
	return n.generatedValueReference(
		binding.owner,
		binding.name,
		definition,
		api.ArtifactFacetValueSurface,
	)
}

func deferredCallableSignature(
	signature *types.Signature,
) (*types.Signature, error) {
	if signature == nil || api.ContainsGenericTypeParameter(signature) {
		return nil, &api.NameError{
			Name:   types.TypeString(signature, packageQualifier),
			Reason: "deferred-callable registry signature is not closed",
		}
	}
	if signature.Recv() == nil {
		return signature, nil
	}
	return types.NewSignatureType(
		nil,
		nil,
		nil,
		signature.Params(),
		signature.Results(),
		signature.Variadic(),
	), nil
}

func deferredRegistrySignatureReason(signature *types.Signature) string {
	if signature == nil {
		return "nil"
	}
	if signature.Recv() != nil {
		return "receiver is present"
	}
	if api.ContainsGenericTypeParameter(signature) {
		return "generic type parameter is present"
	}
	return ""
}

func packageQualifier(source *types.Package) string {
	if source == nil {
		return ""
	}
	return source.Path()
}

func (r *Registry) internDeferredCallableRegistry(
	artifactKey string,
	signature *types.Signature,
) (deferredCallableRegistryBinding, error) {
	if r == nil ||
		len(artifactKey) != sha256.Size*2 ||
		signature == nil ||
		signature.Recv() != nil ||
		api.ContainsGenericTypeParameter(signature) {
		return deferredCallableRegistryBinding{}, &api.NameError{
			Reason: "deferred-callable registry identity is invalid",
		}
	}
	if existing, ok := r.deferredCallableRegistries[artifactKey]; ok {
		existingSignature, valid := existing.owner.DeferredCallableRegistry()
		if !valid || !types.Identical(existingSignature, signature) {
			return deferredCallableRegistryBinding{}, &api.NameError{
				Name:   existing.name,
				Reason: "deferred-callable registry key joined non-identical signatures",
			}
		}
		return existing, nil
	}
	name := "$goDeferred_" +
		artifactKey[len(artifactKey)-deferredCallableRegistryNameHexLength:]
	if err := reserveGeneratedName(
		r.deferredCallableRegistryNames,
		name,
		artifactKey,
		"deferred-callable registry",
	); err != nil {
		return deferredCallableRegistryBinding{}, err
	}
	outputPath, err := output.DeferredCallableRegistryPath(artifactKey)
	if err != nil {
		return deferredCallableRegistryBinding{}, err
	}
	owner, err := api.NewCompilationGeneratedArtifact(
		api.GeneratedArtifactDeferredCallableRegistry,
		signature,
		artifactKey,
		name,
		outputPath,
	)
	if err != nil {
		return deferredCallableRegistryBinding{}, err
	}
	binding := deferredCallableRegistryBinding{owner: owner, name: name}
	r.deferredCallableRegistries[artifactKey] = binding
	return binding, nil
}
