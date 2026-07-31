package naming

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/emit/api"
)

func (n *File) DefinedValueRepresentation(
	typeName *types.TypeName,
) (api.DefinedValueRepresentation, error) {
	if typeName == nil {
		return api.DefinedValueRepresentation{}, &api.NameError{
			Reason: "defined-value owner is nil",
		}
	}
	binding, ok := n.owner.byObject[typeName]
	if !ok && n.owner.registry != nil {
		binding, ok = n.owner.registry.byObject[typeName]
	}
	if !ok || binding.kind != targetBindingProvider {
		return api.NewDefinedValueRepresentation(
			api.DefinedValueRepresentationGeneratedWrapper,
			api.NameReference{},
		)
	}
	switch binding.providerDefinedValue {
	case gostdlib.DefinedValueRepresentationIdentity:
		if !binding.providerEffect.Valid() {
			return api.DefinedValueRepresentation{}, &api.NameError{
				Name:   typeName.Name(),
				Reason: "provider callable effect is uncertified",
			}
		}
		return api.NewProviderIdentityDefinedValueRepresentation(
			binding.providerEffect == gostdlib.EffectAsynchronous,
		)
	case gostdlib.DefinedValueRepresentationOperations:
		reference, providerOwned, err := n.providerFacetReference(
			typeName,
			gostdlib.FacetDefinedValueOperations,
			gostdlib.FacetCapabilityProject,
			api.ImportPhaseValue,
		)
		if err != nil {
			return api.DefinedValueRepresentation{}, err
		}
		if !providerOwned {
			return api.DefinedValueRepresentation{}, &api.NameError{
				Name:   typeName.Name(),
				Reason: "provider defined-value operations lost provider ownership",
			}
		}
		return api.NewDefinedValueRepresentation(
			api.DefinedValueRepresentationProviderOperations,
			reference,
		)
	default:
		return api.DefinedValueRepresentation{}, &api.NameError{
			Name:   typeName.Name(),
			Reason: "provider defined-value representation is uncertified",
		}
	}
}
