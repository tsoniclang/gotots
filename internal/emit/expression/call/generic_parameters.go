package call

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	genericparameters "github.com/tsoniclang/gotots/internal/emit/generic/callableparameters"
)

type genericCallableParameterSelection struct {
	parameters     []int
	providerKernel bool
}

func selectGenericCallableParameters(
	context api.Context,
	_ *ast.CallExpr,
	owner *types.Func,
	_ bool,
) (genericCallableParameterSelection, error) {
	selection, err := genericparameters.ForCallable(context, owner)
	if err != nil {
		return genericCallableParameterSelection{}, err
	}
	return genericCallableParameterSelection{
		parameters:     selection.Parameters(),
		providerKernel: selection.ProviderKernel(),
	}, nil
}
