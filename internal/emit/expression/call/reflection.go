package call

import (
	"go/ast"
	"go/types"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/emit/api"
	genericoperation "github.com/tsoniclang/gotots/internal/emit/generic/operation"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

const reflectTypeForIdentity = "reflect|kind=4|receiver=|name=TypeFor"
const reflectTypeOfIdentity = "reflect|kind=4|receiver=|name=TypeOf"

func emitReflectionTypeOf(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
	discarded bool,
	detached bool,
) (api.ExpressionEmission, bool, error) {
	owner, ok := calleeObject(context.TypesInfo(), source.Fun)
	if !ok {
		return api.ExpressionEmission{}, false, nil
	}
	contract, err := environmentcontract.Describe(owner)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	if contract.Identity() != reflectTypeOfIdentity {
		return api.ExpressionEmission{}, false, nil
	}
	if detached {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryStatement, source)
	}
	signature, ok := owner.Type().(*types.Signature)
	if !ok || signature.Results() == nil || signature.Results().Len() != 1 ||
		len(source.Args) != 1 {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if err := validateResults(context, source, signature, discarded); err != nil {
		return api.ExpressionEmission{}, true, err
	}
	reflectionType, ok := types.Unalias(
		signature.Results().At(0).Type(),
	).(*types.Named)
	if !ok || reflectionType.Obj() == nil {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	names, ok := context.Names().(api.ReflectionNames)
	if !ok {
		return api.ExpressionEmission{}, true, &api.ContextError{
			Reason: "reflection names are unavailable",
		}
	}
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
	if signature.Params() == nil || signature.Params().Len() != 1 {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	argumentType := context.TypesInfo().TypeOf(source.Args[0])
	if argumentType == nil {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if basic, ok := types.Unalias(argumentType).(*types.Basic); ok &&
		basic.Kind() == types.UntypedNil {
		zero, zeroErr := context.Values().Zero(
			context,
			source,
			signature.Results().At(0).Type(),
		)
		return zero, true, zeroErr
	}
	if api.ContainsGenericTypeParameter(argumentType) {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	operations, err := names.ReflectionTypeOf(
		argumentType,
		reflectionType.Obj(),
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	arguments, before, argumentRequests, err := emitArguments(
		context,
		children,
		source,
		signature,
		false,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	typeOf, err := operations.MemberExpression(context.Factory(), "$typeOf")
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	emission, err := api.NewExpressionEmission(
		before,
		context.Factory().CallExpression(
			typeOf,
			nil,
			nil,
			arguments,
			tsgo.NodeFlagsNone,
		),
		api.CombineRequests(
			operations.Requests(),
			argumentRequests,
		),
	)
	return emission, true, err
}

func emitReflectionTypeFor(
	context api.Context,
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
	if contract.Identity() != reflectTypeForIdentity {
		return api.ExpressionEmission{}, false, nil
	}
	if discarded || detached || len(source.Args) != 0 ||
		instance.TypeArgs.Len() != 1 {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	signature, ok := instance.Type.(*types.Signature)
	if !ok || signature.Results() == nil || signature.Results().Len() != 1 {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	reflectionType, ok := types.Unalias(
		signature.Results().At(0).Type(),
	).(*types.Named)
	if !ok || reflectionType.Obj() == nil {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	names, ok := context.Names().(api.ReflectionNames)
	if !ok {
		return api.ExpressionEmission{}, true, &api.ContextError{
			Reason: "reflection names are unavailable",
		}
	}
	observer, ok := context.Names().(environmentcontract.ImplementationObserver)
	if !ok {
		return api.ExpressionEmission{}, true, &api.ContextError{
			Reason: "environment implementation observer is unavailable",
		}
	}
	if instance.TypeArgs.ContainsGenericTypeParameter() {
		if err := observer.ObserveEnvironmentImplementation(
			owner,
			environmentcontract.UseDemandCallable,
			environmentcontract.RouteGeneratedFacet,
		); err != nil {
			return api.ExpressionEmission{}, true, err
		}
		witnessType := types.NewPointer(instance.TypeArgs.At(0))
		witness, err := context.Values().Zero(context, source, witnessType)
		if err != nil {
			return api.ExpressionEmission{}, true, err
		}
		emission, err := genericoperation.Call(
			context,
			source,
			api.GenericOperationReflectionType,
			[]types.Type{witnessType},
			[]types.Type{signature.Results().At(0).Type()},
			[]api.ExpressionEmission{witness},
		)
		return emission, true, err
	}
	if err := observer.ObserveEnvironmentImplementation(
		owner,
		environmentcontract.UseDemandCallable,
		environmentcontract.RouteGeneratedFacet,
	); err != nil {
		return api.ExpressionEmission{}, true, err
	}
	reference, err := names.ReflectionType(
		instance.TypeArgs.At(0),
		reflectionType.Obj(),
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	return api.DirectExpression(
		reference.Expression(context.Factory()),
		reference.Requests()...,
	), true, nil
}
