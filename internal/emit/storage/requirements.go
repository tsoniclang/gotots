package storage

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func ApplyRequirements(
	context api.Context,
	source ast.Node,
	owner api.ArtifactOwner,
	requirements []api.DeclarationRequirement,
) (api.Context, error) {
	if source == nil ||
		!owner.Valid() ||
		context.ArtifactOwner() != owner {
		return api.Context{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "addressable-storage artifact identity is invalid",
		}
	}
	storageNames := make(map[*types.Var]string, len(requirements))
	for _, requirement := range requirements {
		if requirement.Kind() != api.DeclarationRequirementAddressableStorage {
			continue
		}
		requirementOwner, variable, ok := requirement.AddressableStorage()
		if !ok ||
			requirementOwner != owner ||
			variable.Pkg() != owner.Package() ||
			variable.Pos() < source.Pos() ||
			variable.Pos() > source.End() {
			return api.Context{}, &api.InvariantError{
				Role:   context.Role(),
				Reason: "artifact received foreign addressable-storage requirement",
			}
		}
		if _, duplicate := storageNames[variable]; duplicate {
			return api.Context{}, &api.InvariantError{
				Role:   context.Role(),
				Reason: "artifact received duplicate addressable-storage requirement",
			}
		}
		name, err := context.Names().Declare(variable)
		if err != nil {
			return api.Context{}, err
		}
		storageNames[variable] = name + "$storage"
	}
	return context.WithAddressableStorage(owner, storageNames)
}
