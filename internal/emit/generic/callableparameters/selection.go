package callableparameters

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	providerboundary "github.com/tsoniclang/gotots/internal/emit/value/providerboundary"
)

type Selection struct {
	parameters     []int
	providerKernel bool
}

func ForCallable(
	context api.Context,
	owner *types.Func,
) (Selection, error) {
	parameters, available, err := context.GenericCallableParameters(owner)
	if err != nil {
		return Selection{}, err
	}
	if !available {
		if err := providerboundary.RequireProviderCallable(
			context,
			owner,
		); err != nil {
			return Selection{}, err
		}
		return Selection{}, nil
	}
	return Selection{
		parameters:     parameters,
		providerKernel: true,
	}, nil
}

func (s Selection) Parameters() []int {
	return append([]int(nil), s.parameters...)
}

func (s Selection) ProviderKernel() bool {
	return s.providerKernel
}
