package naming

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func (n *File) sourceReferencePath(
	object types.Object,
	binding targetBinding,
) (string, bool, error) {
	if object == nil || binding.sourcePath == "" {
		return "", false, &api.NameError{
			Name:   objectName(object),
			Reason: "source reference identity is invalid",
		}
	}
	if n.packageScope == nil {
		return binding.sourcePath, object.Pkg() != nil, nil
	}
	crossPackage := n.packageScope != nil &&
		object.Pkg() != nil &&
		object.Pkg().Scope() != n.packageScope
	if !crossPackage {
		return binding.sourcePath, false, nil
	}
	referencePath := n.owner.registry.assemblyPathByPackage[object.Pkg()]
	if referencePath == "" {
		return "", false, &api.NameError{
			Name:   object.Name(),
			Reason: "cross-package declaration has no assembly path",
		}
	}
	return referencePath, true, nil
}
