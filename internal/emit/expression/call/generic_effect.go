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
	return exactSynchronousCallableValue(
		context,
		source,
		make(map[*types.Var]struct{}),
	)
}

func exactSynchronousCallableValue(
	context api.Context,
	source ast.Expr,
	fields map[*types.Var]struct{},
) (bool, []api.RootRequest, error) {
	switch expression := source.(type) {
	case *ast.ParenExpr:
		return exactSynchronousCallableValue(context, expression.X, fields)
	case *ast.FuncLit:
		if len(fields) != 0 {
			return false, nil, nil
		}
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
		selection := context.TypesInfo().SelectionOf(expression)
		if selection != nil {
			switch object := selection.Obj().(type) {
			case *types.Func:
				if !concreteMethodSelection(selection) {
					return false, nil, nil
				}
				facet, err := api.NewSourceCallableFacet(object.Origin())
				if err != nil {
					return false, nil, err
				}
				return observedSynchronousCallable(context, facet)
			case *types.Var:
				return exactSynchronousCallableField(context, object, fields)
			default:
				return false, nil, nil
			}
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
		return exactSynchronousCallableValue(context, expression.X, fields)
	case *ast.IndexListExpr:
		return exactSynchronousCallableValue(context, expression.X, fields)
	default:
		return false, nil, nil
	}
}

func exactSynchronousCallableField(
	context api.Context,
	field *types.Var,
	fields map[*types.Var]struct{},
) (bool, []api.RootRequest, error) {
	if _, resolving := fields[field]; resolving {
		return false, nil, nil
	}
	assignments, exact := context.ExactCallableFieldAssignments(field)
	if !exact {
		return false, nil, nil
	}
	fields[field] = struct{}{}
	defer delete(fields, field)
	var requests []api.RootRequest
	for _, assignment := range assignments {
		synchronous, selected, err := exactSynchronousCallableValue(
			context,
			assignment,
			fields,
		)
		if err != nil {
			return false, nil, err
		}
		requests = append(requests, selected...)
		if !synchronous {
			return false, api.CombineRequests(requests), nil
		}
	}
	return true, api.CombineRequests(requests), nil
}

func concreteMethodSelection(selection *types.Selection) bool {
	if selection == nil ||
		(selection.Kind() != types.MethodVal &&
			selection.Kind() != types.MethodExpr) {
		return false
	}
	receiver := selection.Recv()
	if pointer, ok := receiver.(*types.Pointer); ok {
		receiver = pointer.Elem()
	}
	if receiver == nil {
		return false
	}
	if _, dynamic := types.Unalias(receiver).Underlying().(*types.Interface); dynamic {
		return false
	}
	method, _ := selection.Obj().(*types.Func)
	if method == nil || method.Signature().Recv() == nil {
		return false
	}
	declaredReceiver := method.Signature().Recv().Type()
	if pointer, ok := declaredReceiver.(*types.Pointer); ok {
		declaredReceiver = pointer.Elem()
	}
	_, dynamic := types.Unalias(declaredReceiver).Underlying().(*types.Interface)
	return !dynamic
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
