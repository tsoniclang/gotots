package call

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
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
			exactSynchronousCallable(context, source.Args[index])
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

func exactSynchronousCallable(
	context api.Context,
	source ast.Expr,
) (bool, []api.RootRequest, error) {
	switch expression := source.(type) {
	case *ast.ParenExpr:
		return exactSynchronousCallable(context, expression.X)
	case *ast.FuncLit:
		facet, err := context.FunctionLiteralCallableFacet(expression)
		if err != nil {
			return false, nil, err
		}
		return observedSynchronousCallable(context, facet)
	case *ast.Ident:
		object := context.TypesInfo().UseOf(expression)
		if object == types.Universe.Lookup("nil") {
			return true, nil, nil
		}
		function, ok := object.(*types.Func)
		if !ok {
			return false, nil, nil
		}
		facet, err := api.NewSourceCallableFacet(function.Origin())
		if err != nil {
			return false, nil, err
		}
		return observedSynchronousCallable(context, facet)
	case *ast.SelectorExpr:
		if context.TypesInfo().SelectionOf(expression) != nil {
			return false, nil, nil
		}
		function, _ := context.TypesInfo().UseOf(expression.Sel).(*types.Func)
		if function == nil {
			return false, nil, nil
		}
		facet, err := api.NewSourceCallableFacet(function.Origin())
		if err != nil {
			return false, nil, err
		}
		return observedSynchronousCallable(context, facet)
	case *ast.IndexExpr:
		return exactSynchronousCallable(context, expression.X)
	case *ast.IndexListExpr:
		return exactSynchronousCallable(context, expression.X)
	default:
		return false, nil, nil
	}
}

func observedSynchronousCallable(
	context api.Context,
	facet api.CallableFacet,
) (bool, []api.RootRequest, error) {
	cooperative, err := context.ObserveCooperativeCallable(facet)
	if err != nil {
		return false, nil, err
	}
	return !cooperative.Cooperative(), cooperative.Requests(), nil
}
