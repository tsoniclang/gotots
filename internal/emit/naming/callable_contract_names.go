package naming

import (
	"crypto/sha256"
	"encoding/hex"
	"go/types"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/generic/semanticname"
	"github.com/tsoniclang/gotots/internal/emit/type/typeidentity"
	"github.com/tsoniclang/gotots/internal/output"
)

func (r *Registry) ProviderCallableEffect(
	owner *types.Func,
) (gostdlib.EffectKind, bool, error) {
	if r == nil || owner == nil {
		return gostdlib.EffectInvalid, false, &api.NameError{
			Reason: "provider callable identity is invalid",
		}
	}
	owner = owner.Origin()
	binding, ok := r.byObject[owner]
	if !ok || binding.kind == targetBindingMissingProvider ||
		binding.kind != targetBindingProvider {
		return gostdlib.EffectInvalid, false, nil
	}
	if !binding.providerEffect.Valid() {
		return gostdlib.EffectInvalid, true, &api.NameError{
			Name:   owner.Name(),
			Reason: "certified provider callable effect is absent",
		}
	}
	return binding.providerEffect, true, nil
}

func (r *Registry) ProviderDefinedCallableEffect(
	owner *types.TypeName,
) (gostdlib.EffectKind, bool, error) {
	if r == nil || owner == nil || owner.IsAlias() {
		return gostdlib.EffectInvalid, false, &api.NameError{
			Reason: "provider defined-callable identity is invalid",
		}
	}
	if _, callable := owner.Type().Underlying().(*types.Signature); !callable {
		return gostdlib.EffectInvalid, false, nil
	}
	binding, ok := r.byObject[owner]
	if !ok || binding.kind == targetBindingMissingProvider ||
		binding.kind != targetBindingProvider {
		return gostdlib.EffectInvalid, false, nil
	}
	switch binding.providerDefinedValue {
	case gostdlib.DefinedValueRepresentationCanonical:
		if !binding.providerEffect.Valid() {
			return gostdlib.EffectInvalid, true, &api.NameError{
				Name:   owner.Name(),
				Reason: "certified provider canonical-callable effect is absent",
			}
		}
		return binding.providerEffect, true, nil
	case gostdlib.DefinedValueRepresentationOperations:
		if r.provider == nil || !r.provider.Valid() {
			return gostdlib.EffectInvalid, true, &api.NameError{
				Name:   owner.Name(),
				Reason: "standard-library provider certificate is invalid",
			}
		}
		contract, err := environmentcontract.Describe(owner)
		if err != nil {
			return gostdlib.EffectInvalid, true, err
		}
		facet, ok := r.provider.Facet(
			contract.Identity(),
			gostdlib.FacetDefinedValueOperations,
			gostdlib.FacetCapabilityProject,
		)
		if !ok || facet.SourceIdentity() != contract.Identity() ||
			facet.Kind() != gostdlib.FacetDefinedValueOperations ||
			!facet.Effect().Valid() {
			return gostdlib.EffectInvalid, true, &api.NameError{
				Name:   owner.Name(),
				Reason: "certified provider operation-callable effect is absent",
			}
		}
		return facet.Effect(), true, nil
	default:
		return gostdlib.EffectInvalid, true, &api.NameError{
			Name:   owner.Name(),
			Reason: "provider defined-callable representation is uncertified",
		}
	}
}

func (n *File) ProviderCallableEffect(
	owner *types.Func,
) (gostdlib.EffectKind, bool, error) {
	if n == nil || n.owner == nil || n.owner.registry == nil {
		return gostdlib.EffectInvalid, false, &api.NameError{
			Reason: "provider callable-effect registry is unavailable",
		}
	}
	return n.owner.registry.ProviderCallableEffect(owner)
}

func (n *File) ProviderDefinedCallableEffect(
	owner *types.TypeName,
) (gostdlib.EffectKind, bool, error) {
	if n == nil || n.owner == nil || n.owner.registry == nil {
		return gostdlib.EffectInvalid, false, &api.NameError{
			Reason: "provider defined-callable registry is unavailable",
		}
	}
	return n.owner.registry.ProviderDefinedCallableEffect(owner)
}

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
		n.generatedNamedObjectToken,
		n.generatedPackageToken,
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
	namedToken semanticname.NamedTypeToken,
	packageToken semanticname.PackageToken,
) (genericCapabilityBinding, error) {
	if r == nil ||
		!selection.Valid() ||
		signature == nil ||
		artifactKey == "" ||
		!placement.kind.Valid() ||
		namedToken == nil || packageToken == nil {
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
	name, err := semanticname.OperationNameWithIdentityTokens(
		selection.Operation().Identifier(),
		selectedMethod(selection),
		signature,
		namedToken,
		packageToken,
	)
	if err != nil {
		return genericCapabilityBinding{}, err
	}
	module := ""
	if placement.kind == api.GeneratedArtifactPlacementCompilation {
		module, err = semanticname.CapabilityModule(
			selection.Operation().Identifier(),
		)
		if err != nil {
			return genericCapabilityBinding{}, err
		}
	}
	if err := reserveGenericGeneratedName(
		r.genericCapabilityNames,
		genericGeneratedNameScope{
			placement:    placement.kind,
			lexicalOwner: placement.lexicalOwner,
			anchor:       placement.anchor,
			module:       module,
			name:         name,
		},
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
		module,
	)
	if err != nil {
		return genericCapabilityBinding{}, err
	}
	binding := genericCapabilityBinding{owner: owner, name: name}
	r.genericCapabilities[artifactKey] = binding
	return binding, nil
}

func selectedMethod(selection api.GenericOperationSelection) *types.Func {
	method, _ := selection.Method()
	return method
}

func newGenericCapabilityArtifact(
	selection api.GenericOperationSelection,
	signature *types.Signature,
	artifactKey string,
	name string,
	placement generatedArtifactPlacement,
	module string,
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
	outputPath, err := output.GenericCapabilityPath(module)
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
	return n.callableABI(signature, nil)
}

func (n *File) DeferredCallable(
	owner *types.Func,
	variantSuffix string,
) (api.NameReference, error) {
	if owner == nil {
		return api.NameReference{}, &api.NameError{
			Reason: "deferred callable owner is nil",
		}
	}
	return n.derivedSourceReference(
		owner.Origin(),
		variantSuffix+api.DeferredEntrySuffix,
		api.ArtifactFacetCallableSignature,
	)
}

func (n *File) SourceCallableABI(
	owner types.Object,
	signature *types.Signature,
) (api.CallableABIReference, error) {
	owner = api.GenericDeclarationOrigin(owner)
	if owner == nil ||
		owner.Pkg() == nil ||
		signature == nil ||
		signature.Recv() != nil {
		return api.CallableABIReference{}, &api.NameError{
			Reason: "source callable ABI identity is invalid",
		}
	}
	var sourceScope types.Object
	if api.ContainsGenericTypeParameter(signature) ||
		len(typeidentity.LocalComponents(signature)) != 0 {
		sourceScope = owner
	}
	return n.callableABI(signature, sourceScope)
}

func (n *File) callableABI(
	signature *types.Signature,
	sourceScope types.Object,
) (api.CallableABIReference, error) {
	localIdentity := semanticname.LocalNamedIdentity(
		n.generatedNamedObjectIdentity,
	)
	if sourceScope != nil {
		localIdentity = semanticname.LocalNamedIdentity(
			n.sourceGeneratedNamedObjectIdentity(sourceScope),
		)
	}
	identityName, err := semanticname.TypeWithLocalIdentity(
		signature,
		localIdentity,
	)
	if err != nil {
		return api.CallableABIReference{}, err
	}
	name, err := n.semanticGeneratedTypeName(
		"$goCallable$",
		signature,
	)
	if err != nil {
		return api.CallableABIReference{}, err
	}
	digest := sha256.Sum256([]byte("callable-abi|" + identityName))
	artifactKey := hex.EncodeToString(digest[:])
	binding, err := n.owner.registry.internCallableABI(
		artifactKey,
		signature,
		name,
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
	if sourceScope != nil {
		return api.NewSourceCallableABIReference(
			sourceScope,
			binding.owner,
			requests...,
		)
	}
	return api.NewCallableABIReference(binding.owner, requests...)
}

func (n *File) sourceGeneratedNamedObjectIdentity(
	owner types.Object,
) typeidentity.NamedObjectIdentity {
	artifactOwner := api.MustSourceArtifactOwner(owner)
	packageScope := owner.Pkg().Scope()
	return func(object *types.TypeName) (string, error) {
		if object == nil {
			return "", &api.NameError{
				Reason: "source-generated component is nil",
			}
		}
		if object.Pkg() == nil ||
			object.Parent() == object.Pkg().Scope() {
			if object.Pkg() != nil {
				if _, ok := n.owner.registry.byObject[object]; !ok {
					return "", &api.NameError{
						Name:   object.Name(),
						Reason: "source-generated component has no declaration identity",
					}
				}
			}
			return typeidentity.NamedObjectKey(object)
		}
		if object.Pkg() != owner.Pkg() {
			return "", &api.NameError{
				Name:   object.Name(),
				Reason: "source-generated component is foreign to its owner",
			}
		}
		return typeidentity.LexicalNamedObjectKey(
			object,
			artifactOwner,
			packageScope,
		)
	}
}

func (r *Registry) internCallableABI(
	artifactKey string,
	signature *types.Signature,
	name string,
) (callableABIBinding, error) {
	if r == nil ||
		len(artifactKey) != sha256.Size*2 ||
		signature == nil ||
		signature.Recv() != nil ||
		name == "" {
		return callableABIBinding{}, &api.NameError{
			Reason: "callable ABI canonicalization input is invalid",
		}
	}
	if existing, ok := r.callableABIs[artifactKey]; ok {
		_, valid := existing.owner.CallableABI()
		if !valid || existing.name != name {
			return callableABIBinding{}, &api.NameError{
				Name:   existing.name,
				Reason: "callable ABI key joined a different semantic contract",
			}
		}
		return existing, nil
	}
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
