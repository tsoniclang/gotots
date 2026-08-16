package cooperative

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func ExactSynchronousValue(
	context api.Context,
	source ast.Expr,
) (bool, []api.RootRequest, error) {
	return exactSynchronousValue(
		context,
		source,
		make(map[*types.Var]struct{}),
	)
}

func ExactSynchronousField(
	context api.Context,
	field *types.Var,
) (bool, []api.RootRequest, error) {
	return exactSynchronousField(
		context,
		field,
		make(map[*types.Var]struct{}),
	)
}

func exactSynchronousValue(
	context api.Context,
	source ast.Expr,
	fields map[*types.Var]struct{},
) (bool, []api.RootRequest, error) {
	switch expression := source.(type) {
	case *ast.ParenExpr:
		return exactSynchronousValue(context, expression.X, fields)
	case *ast.FuncLit:
		if len(fields) != 0 {
			return false, nil, nil
		}
		facet, err := context.FunctionLiteralCallableFacet(expression)
		if err != nil {
			return false, nil, err
		}
		return observedSynchronous(context, facet)
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
		return observedSynchronous(context, facet)
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
				return observedSynchronous(context, facet)
			case *types.Var:
				return exactSynchronousField(context, object, fields)
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
		return observedSynchronous(context, facet)
	case *ast.IndexExpr:
		return exactSynchronousValue(context, expression.X, fields)
	case *ast.IndexListExpr:
		return exactSynchronousValue(context, expression.X, fields)
	default:
		return false, nil, nil
	}
}

func exactSynchronousField(
	context api.Context,
	field *types.Var,
	fields map[*types.Var]struct{},
) (bool, []api.RootRequest, error) {
	if field == nil {
		return false, nil, nil
	}
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
		synchronous, selected, err := exactSynchronousValue(
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

func observedSynchronous(
	context api.Context,
	facet api.CallableFacet,
) (bool, []api.RootRequest, error) {
	cooperative, err := context.ObserveCooperativeCallable(facet)
	if err != nil {
		return false, nil, err
	}
	return !cooperative.Cooperative(), cooperative.Requests(), nil
}
