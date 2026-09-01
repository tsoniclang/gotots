package environmentcontract

import (
	"fmt"
	"go/types"

	environmentidentity "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/emit/api"
)

func SelectDeclarationRequirements(
	object types.Object,
	requirements []api.DeclarationRequirement,
) ([]api.DeclarationRequirement, error) {
	selected := make([]api.DeclarationRequirement, 0, len(requirements))
	for _, requirement := range requirements {
		if owner, _, ok := requirement.NamedStructOperation(); ok {
			if owner != object {
				return nil, &api.ContextError{
					Reason: "environment named-struct operation requirement is foreign",
				}
			}
			continue
		}
		if owner, _, _, ok := requirement.GenericRepresentation(); ok {
			if owner != object {
				return nil, &api.ContextError{
					Reason: "environment generic representation requirement is foreign",
				}
			}
			selected = append(selected, requirement)
			continue
		}
		if owner, artifact, _, ok := requirement.TypeRepresentation(); ok {
			if owner != object || artifact != nil {
				return nil, &api.ContextError{
					Reason: "environment type representation requirement is foreign",
				}
			}
			selected = append(selected, requirement)
			continue
		}
		owner, enclosing, callable, control, ok := requirement.CallableControl()
		source, sourceOwned := owner.Source()
		if ok && sourceOwned && source == object && enclosing == nil &&
			callable == nil && control == api.CallableControlRecovery {
			selected = append(selected, requirement)
			continue
		}
		return nil, &api.ContextError{
			Reason: fmt.Sprintf(
				"environment declaration requirement kind %d is unsupported",
				requirement.Kind(),
			),
		}
	}
	return selected, nil
}

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
		api.DeclarationRequirementMapSpecialization,
		api.DeclarationRequirementValueReceiverCopy,
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
