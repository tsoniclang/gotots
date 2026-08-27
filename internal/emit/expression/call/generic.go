package call

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	genericabi "github.com/tsoniclang/gotots/internal/emit/generic/abi"
	genericinstance "github.com/tsoniclang/gotots/internal/emit/generic/instance"
	providerboundary "github.com/tsoniclang/gotots/internal/emit/value/providerboundary"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func emitGeneric(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
	discarded bool,
	detached bool,
) (api.ExpressionEmission, bool, error) {
	owner, instance, ok := genericFunctionInstance(
		context,
		source,
	)
	if !ok {
		return api.ExpressionEmission{}, false, nil
	}
	signature, ok := instance.Type.(*types.Signature)
	if !ok || signature.Recv() != nil {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	operationSet, ok, err := context.ResolveGenericCallable(owner)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	if !ok ||
		instance.TypeArgs.Len() != len(operationSet.Parameters()) {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if err := validateResults(context, source, signature, discarded); err != nil {
		return api.ExpressionEmission{}, true, err
	}
	requiresConcretization, err :=
		context.GenericCallableRequiresConcretization(owner)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	openConcretization := requiresConcretization &&
		instance.TypeArgs.ContainsGenericTypeParameter()
	parameterSelection, err := selectGenericCallableParameters(
		context,
		source,
		owner,
		detached,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	arguments, before, argumentRequests, err :=
		emitArgumentsWithProviderCallableParameters(
			context,
			children,
			source,
			signature,
			detached,
			parameterSelection.parameters,
		)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	var (
		reference     api.NameReference
		typeArguments []tsgo.TypeNode
		typeRequests  []api.RootRequest
		mechanicArgs  []tsgo.Expression
		mechanicReqs  []api.RootRequest
	)
	if requiresConcretization && !openConcretization {
		concrete, concreteErr := context.ResolveGenericConcretization(
			owner,
			instance.TypeArgs,
			signature,
		)
		if concreteErr != nil {
			return api.ExpressionEmission{}, true, concreteErr
		}
		reference, err = api.NewNameReference(
			concrete.Name(),
			concrete.Requests()...,
		)
		if err != nil {
			return api.ExpressionEmission{}, true, err
		}
	} else if openConcretization {
		kernelNames, available := context.Names().(api.GenericKernelNames)
		if !available {
			return api.ExpressionEmission{}, true, &api.ContextError{
				Reason: "generic kernel names are unavailable",
			}
		}
		reference, err = kernelNames.GenericKernel(owner)
		if err == nil {
			typeArguments, typeRequests, err =
				genericinstance.EmitFunctionTypeArguments(
					context,
					children,
					source,
					owner,
					instance.TypeArgs,
				)
		}
		if err == nil {
			capabilities, capabilityRequests, capabilityErr :=
				genericinstance.EmitCapabilities(
					context,
					children,
					source,
					operationSet,
					instance.TypeArgs,
				)
			if capabilityErr != nil {
				return api.ExpressionEmission{}, true, capabilityErr
			}
			mechanicArgs, capabilityErr = genericabi.JoinCapabilities(
				owner,
				operationSet.Operations(),
				capabilities,
			)
			if capabilityErr != nil {
				return api.ExpressionEmission{}, true, capabilityErr
			}
			mechanicReqs = capabilityRequests
		}
	} else {
		if len(operationSet.Operations()) != 0 {
			return api.ExpressionEmission{}, true, &api.InvariantError{
				Role:   context.Role(),
				Reason: "generic mechanics reached a source-facing callable",
			}
		}
		if parameterSelection.providerKernel {
			kernelNames, available := context.Names().(api.GenericKernelNames)
			if !available {
				return api.ExpressionEmission{}, true, &api.ContextError{
					Reason: "generic kernel names are unavailable",
				}
			}
			reference, err = kernelNames.GenericKernel(owner)
		} else {
			reference, err = context.Names().Reference(owner.Origin())
		}
		if err == nil {
			typeArguments, typeRequests, err =
				genericinstance.EmitFunctionTypeArguments(
					context,
					children,
					source,
					owner,
					instance.TypeArgs,
				)
		}
	}
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	contract, ok := owner.Type().(*types.Signature)
	if !ok {
		return api.ExpressionEmission{}, true, &api.InvariantError{
			Role:   context.Role(),
			Reason: "generic callable owner has no signature",
		}
	}
	if reference.ProviderBoundary() {
		var providerBefore []tsgo.Statement
		var providerRequests []api.RootRequest
		arguments, providerBefore, providerRequests, err =
			providerboundary.ToProviderGenericArgumentsWithCallableParameters(
				context,
				children,
				contract.Params(),
				signature.Params(),
				arguments,
				parameterSelection.parameters,
			)
		if err != nil {
			return api.ExpressionEmission{}, true, err
		}
		before = append(before, providerBefore...)
		argumentRequests = api.CombineRequests(
			argumentRequests,
			providerRequests,
		)
	}
	arguments = append(mechanicArgs, arguments...)
	result, err := api.NewExpressionEmission(
		before,
		context.Factory().CallExpression(
			reference.Expression(context.Factory()),
			nil,
			typeArguments,
			arguments,
			tsgo.NodeFlagsNone,
		),
		api.CombineRequests(
			reference.Requests(),
			typeRequests,
			mechanicReqs,
			argumentRequests,
		),
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	if reference.ProviderBoundary() && !detached && !discarded {
		result, err = providerboundary.FromProviderGenericResults(
			context,
			children,
			contract.Results(),
			signature.Results(),
			result,
		)
	}
	return result, true, err
}

func genericFunctionInstance(
	context api.Context,
	source *ast.CallExpr,
) (*types.Func, api.TypeInstance, bool) {
	if source == nil {
		return nil, api.TypeInstance{}, false
	}
	return genericinstance.FunctionEvidence(context.TypesInfo(), source.Fun)
}
