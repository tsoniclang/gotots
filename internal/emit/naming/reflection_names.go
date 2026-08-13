package naming

import (
	"encoding/base64"
	"encoding/hex"
	"go/types"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/type/typeidentity"
	"github.com/tsoniclang/gotots/internal/output"
)

const reflectionMethodIdentityHexLength = 20

// ReflectionMethodIdentity returns the compact collision-checked encoding of
// one canonical interface-method identity. The same token binding owner used
// by interface contracts reserves the truncated identity before it is
// encoded, so two distinct methods cannot silently share an identity.
func (n *File) ReflectionMethodIdentity(
	method *types.Func,
) (string, error) {
	runtime, _ := runtimeInterfaceMethodToken(method)
	binding, err := n.interfaceMethodTokenBinding(method, runtime)
	if err != nil {
		return "", err
	}
	key := binding.owner.ArtifactKey()
	if len(key) < reflectionMethodIdentityHexLength {
		return "", &api.NameError{
			Name:   method.Name(),
			Reason: "interface method identity is too short",
		}
	}
	decoded, err := hex.DecodeString(key[:reflectionMethodIdentityHexLength])
	if err != nil {
		return "", &api.NameError{
			Name:   method.Name(),
			Reason: "interface method identity is not hexadecimal",
		}
	}
	return base64.RawURLEncoding.EncodeToString(decoded), nil
}

func (n *File) ReflectionType(
	sourceType types.Type,
	reflectionType *types.TypeName,
) (api.NameReference, error) {
	if sourceType == nil || reflectionType == nil {
		return api.NameReference{}, &api.NameError{
			Reason: "reflection-type identity is invalid",
		}
	}
	artifactKey, err := typeidentity.BuildKey(
		sourceType,
		n.generatedNamedObjectIdentity,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	name, err := n.semanticGeneratedTypeName("$goReflectType$", sourceType)
	if err != nil {
		return api.NameReference{}, err
	}
	binding, err := n.owner.registry.internReflectionType(
		artifactKey,
		sourceType,
		reflectionType,
		name,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	request, err := api.NewReflectionTypeRequest(binding.owner)
	if err != nil {
		return api.NameReference{}, err
	}
	return n.generatedValueReference(
		binding.owner,
		binding.name,
		request,
		api.ArtifactFacetValueSurface,
	)
}

func (n *File) ReflectionOperations(
	reflectionType *types.TypeName,
) (api.NameReference, error) {
	reference, providerOwned, err := n.providerFacetReference(
		reflectionType,
		gostdlib.FacetReflectionTypeOperations,
		gostdlib.FacetCapabilityMetadata,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	if !providerOwned {
		return api.NameReference{}, &api.NameError{
			Name:   reflectionType.Name(),
			Reason: "reflection type has no certified metadata operations",
		}
	}
	return reference, nil
}

func (n *File) ReflectionTypeOf(
	argumentType types.Type,
	reflectionType *types.TypeName,
) (api.NameReference, error) {
	if argumentType == nil || reflectionType == nil || reflectionType.IsAlias() {
		return api.NameReference{}, &api.NameError{
			Reason: "reflection TypeOf contract is invalid",
		}
	}
	registry := n.owner.registry
	operations, err := n.ReflectionOperations(reflectionType)
	if err != nil {
		return api.NameReference{}, err
	}
	staticType, err := n.ReflectionType(argumentType, reflectionType)
	if err != nil {
		return api.NameReference{}, err
	}
	readiness := staticType.Requests()
	if _, isInterface := types.Unalias(argumentType).Underlying().(*types.Interface); isInterface {
		contract, key, contractErr := n.canonicalInterfaceContract(argumentType)
		if contractErr != nil {
			return api.NameReference{}, contractErr
		}
		dynamicReadiness, demandErr := registry.recordInterfaceReflectionDemand(
			key,
			contract,
			reflectionType,
		)
		if demandErr != nil {
			return api.NameReference{}, demandErr
		}
		readiness = api.CombineRequests(readiness, dynamicReadiness)
	}
	modulePath, err := output.ModuleSpecifier(
		n.targetPath,
		output.ReflectionTypeSupportPath,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	initialize, err := api.NewSideEffectImportRequest(n.factory, modulePath)
	if err != nil {
		return api.NameReference{}, err
	}
	requests := api.CombineRequests(
		operations.Requests(),
		readiness,
		[]api.RootRequest{initialize},
	)
	return operations.WithRequests(api.CombineRequests(requests)...)
}

// ReflectionValueOf demands the canonical descriptor plus the generated
// value-operation facet for one reflected operand type. Interface operands
// subscribe the canonical contract so every concrete adapter reaching the
// contract, before or after this observation, demands its value facet.
func (n *File) ReflectionValueOf(
	argumentType types.Type,
	reflectionType *types.TypeName,
) (api.NameReference, error) {
	if argumentType == nil || reflectionType == nil || reflectionType.IsAlias() {
		return api.NameReference{}, &api.NameError{
			Reason: "reflection ValueOf contract is invalid",
		}
	}
	registry := n.owner.registry
	operations, err := n.ReflectionOperations(reflectionType)
	if err != nil {
		return api.NameReference{}, err
	}
	requests := operations.Requests()
	if _, isInterface := types.Unalias(argumentType).Underlying().(*types.Interface); isInterface {
		contract, key, contractErr := n.canonicalInterfaceContract(argumentType)
		if contractErr != nil {
			return api.NameReference{}, contractErr
		}
		dynamicReadiness, demandErr := registry.recordReflectionValueContract(
			key,
			contract,
			reflectionType,
		)
		if demandErr != nil {
			return api.NameReference{}, demandErr
		}
		requests = api.CombineRequests(requests, dynamicReadiness)
	} else {
		artifactKey, keyErr := typeidentity.BuildKey(
			argumentType,
			n.generatedNamedObjectIdentity,
		)
		if keyErr != nil {
			return api.NameReference{}, keyErr
		}
		registry.reflectionValueDemands[artifactKey] = struct{}{}
		staticType, typeErr := n.ReflectionType(argumentType, reflectionType)
		if typeErr != nil {
			return api.NameReference{}, typeErr
		}
		facet, facetErr := registry.reflectionValueOperationsRequest(artifactKey)
		if facetErr != nil {
			return api.NameReference{}, facetErr
		}
		requests = api.CombineRequests(
			requests,
			staticType.Requests(),
			[]api.RootRequest{facet},
		)
	}
	modulePath, err := output.ModuleSpecifier(
		n.targetPath,
		output.ReflectionTypeSupportPath,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	initialize, err := api.NewSideEffectImportRequest(n.factory, modulePath)
	if err != nil {
		return api.NameReference{}, err
	}
	requests = api.CombineRequests(requests, []api.RootRequest{initialize})
	return operations.WithRequests(requests...)
}

// ReflectionValueOperationsDemanded reports whether the value-operation
// facet was demanded for one canonical reflection artifact.
func (n *File) ReflectionValueOperationsDemanded(artifactKey string) bool {
	_, demanded := n.owner.registry.reflectionValueDemands[artifactKey]
	return demanded
}

// ReflectionValueType returns the canonical descriptor reference for one
// type while joining its value-operation facet demand, closing the value
// metadata over navigable child types (fields and pointees). The distinct
// value-operation requirement requeues a descriptor that was already
// constructed before this demand arrived.
func (n *File) ReflectionValueType(
	sourceType types.Type,
	reflectionType *types.TypeName,
) (api.NameReference, error) {
	var dynamicReadiness []api.RootRequest
	if _, isInterface := types.Unalias(sourceType).Underlying().(*types.Interface); isInterface {
		contract, key, contractErr := n.canonicalInterfaceContract(sourceType)
		if contractErr != nil {
			return api.NameReference{}, contractErr
		}
		var demandErr error
		dynamicReadiness, demandErr =
			n.owner.registry.recordReflectionValueContract(
				key,
				contract,
				reflectionType,
			)
		if demandErr != nil {
			return api.NameReference{}, demandErr
		}
	}
	artifactKey, err := typeidentity.BuildKey(
		sourceType,
		n.generatedNamedObjectIdentity,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	n.owner.registry.reflectionValueDemands[artifactKey] = struct{}{}
	reference, err := n.ReflectionType(sourceType, reflectionType)
	if err != nil {
		return api.NameReference{}, err
	}
	facet, err := n.owner.registry.reflectionValueOperationsRequest(
		artifactKey,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	return reference.WithRequests(api.CombineRequests(
		reference.Requests(),
		dynamicReadiness,
		[]api.RootRequest{facet},
	)...)
}

// ProviderOwnedDeclaration reports whether one declaration's truth is a
// certified provider facet, which excludes its class internals from the
// generated location model.
func (n *File) ProviderOwnedDeclaration(
	object types.Object,
) (bool, error) {
	_, owned, err := n.providerFacetOwner(object)
	return owned, err
}

// EnvironmentOwnedDeclaration reports whether a declaration is represented
// by the bounded environment contract rather than generated package syntax.
func (n *File) EnvironmentOwnedDeclaration(
	object types.Object,
) (bool, error) {
	if object == nil || n == nil || n.owner == nil || n.owner.registry == nil {
		return false, &api.NameError{
			Reason: "environment declaration ownership query is invalid",
		}
	}
	binding, ok := n.owner.registry.byObject[object]
	if !ok {
		return false, &api.NameError{
			Name:   object.Name(),
			Reason: "environment declaration has no target binding",
		}
	}
	return binding.kind == targetBindingEnvironment ||
		binding.kind == targetBindingProvider ||
		binding.kind == targetBindingMissingProvider, nil
}

// reflectionValueOperationsRequest builds the value-operation facet
// requirement of one interned canonical descriptor.
func (r *Registry) reflectionValueOperationsRequest(
	artifactKey string,
) (api.RootRequest, error) {
	binding, ok := r.reflectionTypes[artifactKey]
	if !ok || binding.owner == nil {
		return api.RootRequest{}, &api.NameError{
			Reason: "reflection value demand has no interned descriptor",
		}
	}
	return api.NewReflectionValueOperationsRequest(binding.owner)
}

func (r *Registry) internReflectionType(
	artifactKey string,
	sourceType types.Type,
	reflectionType *types.TypeName,
	name string,
) (reflectionTypeBinding, error) {
	if r == nil || artifactKey == "" || sourceType == nil ||
		reflectionType == nil || name == "" {
		return reflectionTypeBinding{}, &api.NameError{
			Reason: "reflection-type canonicalization input is invalid",
		}
	}
	if existing, ok := r.reflectionTypes[artifactKey]; ok {
		bound, contract, valid := existing.owner.ReflectionType()
		if !valid || !types.Identical(bound, sourceType) ||
			contract != reflectionType {
			return reflectionTypeBinding{}, &api.NameError{
				Name:   existing.name,
				Reason: "reflection-type key joined non-identical Go types",
			}
		}
		return existing, nil
	}
	if err := reserveGeneratedName(
		r.reflectionTypeNames,
		name,
		artifactKey,
		"reflection type",
	); err != nil {
		return reflectionTypeBinding{}, err
	}
	owner, err := api.NewCompilationReflectionTypeArtifact(
		sourceType,
		reflectionType,
		artifactKey,
		name,
		output.ReflectionTypeSupportPath,
	)
	if err != nil {
		return reflectionTypeBinding{}, err
	}
	binding := reflectionTypeBinding{owner: owner, name: name}
	r.reflectionTypes[artifactKey] = binding
	return binding, nil
}
