package call

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	cooperativecall "github.com/tsoniclang/gotots/internal/emit/concurrency/cooperative"
)

type genericConcretizationEffectSelection struct {
	effect                api.GenericConcretizationEffect
	synchronousParameters []int
	requests              []api.RootRequest
}

func selectGenericConcretizationEffect(
	context api.Context,
	source *ast.CallExpr,
	owner *types.Func,
	detached bool,
) (genericConcretizationEffectSelection, error) {
	canonical := genericConcretizationEffectSelection{
		effect: api.GenericConcretizationEffectCanonical,
	}
	if detached || source == nil {
		return canonical, nil
	}
	indexes, available, err :=
		context.GenericCallableSynchronousParameters(owner)
	if err != nil || !available {
		return canonical, err
	}
	if source.Ellipsis.IsValid() {
		return canonical, nil
	}
	var requests []api.RootRequest
	for _, index := range indexes {
		if index < 0 || index >= len(source.Args) {
			return canonical, nil
		}
		synchronous, selectedRequests, selectionErr :=
			cooperativecall.ExactSynchronousValue(
				context,
				source.Args[index],
			)
		if selectionErr != nil {
			return genericConcretizationEffectSelection{}, selectionErr
		}
		requests = append(requests, selectedRequests...)
		if !synchronous {
			canonical.requests = api.CombineRequests(requests)
			return canonical, nil
		}
	}
	return genericConcretizationEffectSelection{
		effect:                api.GenericConcretizationEffectSynchronous,
		synchronousParameters: indexes,
		requests:              api.CombineRequests(requests),
	}, nil
}
