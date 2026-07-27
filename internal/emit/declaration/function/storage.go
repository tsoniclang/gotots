package function

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func applyAddressableStorage(
	context api.Context,
	source *ast.FuncDecl,
	owner *types.Func,
	requirements []api.DeclarationRequirement,
) (api.Context, error) {
	if source == nil || owner == nil {
		return api.Context{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "addressable-storage artifact identity is nil",
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
			variable.Pkg() != owner.Pkg() ||
			variable.Pos() < source.Pos() ||
			variable.Pos() > source.End() {
			return api.Context{}, &api.InvariantError{
				Role:   context.Role(),
				Reason: "function received foreign addressable-storage requirement",
			}
		}
		if _, duplicate := storageNames[variable]; duplicate {
			return api.Context{}, &api.InvariantError{
				Role:   context.Role(),
				Reason: "function received duplicate addressable-storage requirement",
			}
		}
		name, err := context.Names().Declare(variable)
		if err != nil {
			return api.Context{}, err
		}
		storageNames[variable] = name + "$storage"
	}
	return context.WithAddressableStorage(owner, storageNames), nil
}
