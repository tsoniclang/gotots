package effect

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	providerboundary "github.com/tsoniclang/gotots/internal/emit/value/providerboundary"
)

type Selection struct {
	effect     api.GenericConcretizationEffect
	parameters []int
}

func ForExecutionProfile(
	context api.Context,
	owner *types.Func,
) (Selection, error) {
	canonical := Selection{effect: api.GenericConcretizationEffectCanonical}
	if context.ConcurrencySemantics() != api.ConcurrencySemanticsDisabled {
		return canonical, nil
	}
	parameters, available, err :=
		context.GenericCallableSynchronousParameters(owner)
	if err != nil {
		return Selection{}, err
	}
	if !available {
		if err := providerboundary.RequireSynchronousCallable(
			context,
			owner,
		); err != nil {
			return Selection{}, err
		}
		return canonical, nil
	}
	return Selection{
		effect:     api.GenericConcretizationEffectSynchronous,
		parameters: parameters,
	}, nil
}

func (s Selection) Effect() api.GenericConcretizationEffect {
	return s.effect
}

func (s Selection) SynchronousParameters() []int {
	return append([]int(nil), s.parameters...)
}
