package interfaceadapter

import (
	"go/types"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/emit/api"
)

type demandedMethod struct {
	selection *types.Selection
	method    *types.Func
	contracts []*types.Func
}

func demandedMethods(
	sourceType types.Type,
	contracts []Contract,
) ([]demandedMethod, error) {
	methodSet := types.NewMethodSet(sourceType)
	required := make(map[*types.Func][]*types.Func)
	for _, selected := range contracts {
		contract := selected.methodSet
		if contract == nil ||
			!contract.Complete().IsMethodSet() ||
			!types.Implements(sourceType, contract) {
			return nil, &api.GeneratedArtifactShapeError{
				Reason: "adapter contract is not implemented by its source type",
			}
		}
		for index := range contract.NumMethods() {
			method := contract.Method(index)
			selected := methodSet.Lookup(method.Pkg(), method.Name())
			if selected == nil {
				return nil, &api.GeneratedArtifactShapeError{
					Reason: "adapter contract method has no concrete selection",
				}
			}
			concrete, ok := selected.Obj().(*types.Func)
			if !ok || !environmentcontract.EquivalentMethods(concrete, method) {
				return nil, &api.GeneratedArtifactShapeError{
					Reason: "adapter contract method selection is not exact",
				}
			}
			required[concrete] = append(required[concrete], method)
		}
	}
	demanded := make([]demandedMethod, 0, len(required))
	for index := range methodSet.Len() {
		selection := methodSet.At(index)
		method, ok := selection.Obj().(*types.Func)
		if !ok {
			return nil, &api.GeneratedArtifactShapeError{
				Reason: "adapter method set contains a non-method object",
			}
		}
		if targetContracts := required[method]; len(targetContracts) != 0 {
			demanded = append(demanded, demandedMethod{
				selection: selection,
				method:    method,
				contracts: targetContracts,
			})
		}
	}
	if len(demanded) != len(required) {
		return nil, &api.GeneratedArtifactShapeError{
			Reason: "adapter contract selection lost a required method",
		}
	}
	return demanded, nil
}
