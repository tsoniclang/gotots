package call

import (
	"go/ast"
	"go/types"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/emit/api"
	assertionoperation "github.com/tsoniclang/gotots/internal/emit/expression/typeassertion/operation"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

const reflectTypeAssertIdentity = "reflect|kind=4|receiver=|name=TypeAssert"

// emitReflectionTypeAssert intercepts reflect.TypeAssert with one concrete
// type argument: the boxed value unboxes through the provider accessor and
// the ordinary interface type-assertion machinery selects the exact
// adapters, contracts, and comma-ok tuple — reflect.TypeAssert is exactly
// v.Interface().(T). The provider binding never participates, so the call
// routes as a generated facet.
func emitReflectionTypeAssert(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
	discarded bool,
	detached bool,
) (api.ExpressionEmission, bool, error) {
	owner, instance, ok := genericFunctionInstance(context, source)
	if !ok {
		return api.ExpressionEmission{}, false, nil
	}
	contract, err := environmentcontract.Describe(owner)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	if contract.Identity() != reflectTypeAssertIdentity {
		return api.ExpressionEmission{}, false, nil
	}
	if discarded || detached || len(source.Args) != 1 ||
		instance.TypeArgs.Len() != 1 ||
		instance.TypeArgs.ContainsGenericTypeParameter() {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	targetType := instance.TypeArgs.At(0)
	observer, ok := context.Names().(environmentcontract.ImplementationObserver)
	if !ok {
		return api.ExpressionEmission{}, true, &api.ContextError{
			Reason: "environment implementation observer is unavailable",
		}
	}
	if err := observer.ObserveEnvironmentImplementation(
		owner,
		environmentcontract.UseDemandCallable,
		environmentcontract.RouteGeneratedFacet,
	); err != nil {
		return api.ExpressionEmission{}, true, err
	}
	operandType := context.TypesInfo().TypeOf(source.Args[0])
	if operandType == nil {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	operand, err := children.Expression(
		context.
			WithRole(api.RoleConversionOperand).
			WithExpectedType(operandType),
		source.Args[0],
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	unboxed := context.Factory().CallExpression(
		context.Factory().PropertyAccessExpression(
			operand.Value(),
			nil,
			context.Factory().Identifier("$unbox"),
			tsgo.NodeFlagsNone,
		),
		nil,
		nil,
		nil,
		tsgo.NodeFlagsNone,
	)
	receiver, err := api.NewExpressionEmission(
		operand.Before(),
		unboxed,
		operand.Requests(),
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	anyType := types.Universe.Lookup("any").Type()
	emission, err := assertionoperation.Apply(
		context,
		children,
		source,
		anyType,
		targetType,
		true,
		receiver,
	)
	return emission, true, err
}
