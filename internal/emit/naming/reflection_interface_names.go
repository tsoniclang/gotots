package naming

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/type/typeidentity"
)

func (n *File) ReflectionInterfaceAdapter(
	sourceType types.Type,
) (api.NameReference, error) {
	reference, err := n.InterfaceAdapter(sourceType, nil)
	if err != nil {
		return api.NameReference{}, err
	}
	artifactKey, err := typeidentity.BuildKey(
		sourceType,
		n.generatedNamedObjectIdentity,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	binding, ok := n.owner.registry.interfaceAdapters[artifactKey]
	if !ok || binding.owner == nil {
		return api.NameReference{}, &api.NameError{
			Name:   types.TypeString(sourceType, nil),
			Reason: "reflection interface adapter was not canonicalized",
		}
	}
	empty, err := n.canonicalInterfaceContract(
		types.Universe.Lookup("any").Type(),
	)
	if err != nil {
		return api.NameReference{}, err
	}
	demands, err := n.owner.registry.recordReflectionInterfaceAdapter(
		empty,
		binding,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	return reference.WithRequests(api.CombineRequests(
		reference.Requests(),
		demands,
	)...)
}
