package environmentcontract

import (
	"go/types"

	environmentidentity "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/emit/api"
)

// RequirementUseDemand classifies the closed environment use demand created
// by scheduling one declaration requirement against its owner.
func RequirementUseDemand(
	requirement api.DeclarationRequirement,
) environmentidentity.UseDemand {
	switch requirement.Kind() {
	case api.DeclarationRequirementConstantProjection,
		api.DeclarationRequirementLocalConstantProjection:
		return environmentidentity.UseDemandValue
	case api.DeclarationRequirementCallableControl,
		api.DeclarationRequirementCallableABI,
		api.DeclarationRequirementCooperativeCallable,
		api.DeclarationRequirementDeferredCallableRegistry,
		api.DeclarationRequirementClassMethod,
		api.DeclarationRequirementInterfaceMethodCallable,
		api.DeclarationRequirementGenericConcretization,
		api.DeclarationRequirementGenericOperation,
		api.DeclarationRequirementGenericCapability:
		return environmentidentity.UseDemandCallable
	case api.DeclarationRequirementTypeRepresentation,
		api.DeclarationRequirementGenericRepresentation,
		api.DeclarationRequirementAnonymousStruct,
		api.DeclarationRequirementAnonymousInterface,
		api.DeclarationRequirementPointerRepresentation,
		api.DeclarationRequirementMapSpecialization,
		api.DeclarationRequirementValueReceiverCopy,
		api.DeclarationRequirementUnsafeCodec,
		api.DeclarationRequirementNamedStructOperation,
		api.DeclarationRequirementProviderStatefulRepresentation:
		return environmentidentity.UseDemandTypeContract
	case api.DeclarationRequirementInterfaceAdapter,
		api.DeclarationRequirementInterfaceDynamicTypeToken,
		api.DeclarationRequirementInterfaceMethodToken,
		api.DeclarationRequirementReflectionType,
		api.DeclarationRequirementReflectionValueOperations:
		return environmentidentity.UseDemandRuntimeFacet
	case api.DeclarationRequirementProviderInterfaceBridge,
		api.DeclarationRequirementProviderInterfaceCapability,
		api.DeclarationRequirementProviderProfileInterfaceCapability:
		return environmentidentity.UseDemandInterfaceCapability
	default:
		return environmentidentity.UseDemandInvalid
	}
}

// ArtifactFacetUseDemand classifies the closed environment use demand of
// one artifact-facet dependency on a declaration owner.
func ArtifactFacetUseDemand(
	facet api.ArtifactFacet,
	object types.Object,
) environmentidentity.UseDemand {
	switch facet {
	case api.ArtifactFacetCallableSignature,
		api.ArtifactFacetCallableRecovery:
		return environmentidentity.UseDemandCallable
	case api.ArtifactFacetConstructorSurface,
		api.ArtifactFacetInstanceTypeSurface,
		api.ArtifactFacetStaticSurface:
		return environmentidentity.UseDemandTypeContract
	case api.ArtifactFacetValueSurface,
		api.ArtifactFacetExportSurface,
		api.ArtifactFacetImplementation:
		if _, ok := object.(*types.Func); ok {
			return environmentidentity.UseDemandCallable
		}
		return environmentidentity.UseDemandValue
	default:
		return environmentidentity.UseDemandInvalid
	}
}
