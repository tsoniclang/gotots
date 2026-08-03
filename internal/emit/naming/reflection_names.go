package naming

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/type/typeidentity"
	"github.com/tsoniclang/gotots/internal/output"
)

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
	binding, err := n.owner.registry.internReflectionType(
		artifactKey,
		sourceType,
		reflectionType,
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

func (r *Registry) internReflectionType(
	artifactKey string,
	sourceType types.Type,
	reflectionType *types.TypeName,
) (reflectionTypeBinding, error) {
	if r == nil || artifactKey == "" || sourceType == nil ||
		reflectionType == nil {
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
	name, err := interfaceTargetName("$goReflectType_", artifactKey)
	if err != nil {
		return reflectionTypeBinding{}, err
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
