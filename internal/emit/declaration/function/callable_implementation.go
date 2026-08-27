package function

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func emitCallableImplementationBody(
	context api.Context,
	function *types.Func,
	signature *types.Signature,
	kernel bool,
	moduleFunction bool,
	moduleExport bool,
	valueReceiver bool,
	targetName string,
	parameters []tsgo.ParameterDeclaration,
) (api.BlockEmission, bool, error) {
	selection, selected, err := context.ResolveCallableImplementation(
		function,
		kernel,
	)
	if err != nil || !selected {
		return api.BlockEmission{}, selected, err
	}
	if valueReceiver && !moduleFunction {
		return api.BlockEmission{}, true, &api.InvariantError{
			Role:   context.Role(),
			Reason: "callable implementation cannot replace an instance method",
		}
	}
	if moduleFunction && !moduleExport {
		return api.BlockEmission{}, true, &api.InvariantError{
			Role:   context.Role(),
			Reason: "callable implementation requires an exported generated wrapper",
		}
	}
	names, ok := context.Names().(api.CallableImplementationNames)
	if !ok {
		return api.BlockEmission{}, true, &api.InvariantError{
			Role:   context.Role(),
			Reason: "callable implementation name owner is unavailable",
		}
	}
	reference, err := names.CallableImplementation(
		selection.OutputPath(),
		selection.Export(),
	)
	if err != nil {
		return api.BlockEmission{}, true, err
	}
	arguments := make([]tsgo.Expression, len(parameters))
	for index, parameter := range parameters {
		identifier, ok := parameter.Name().(tsgo.Identifier)
		if !ok {
			return api.BlockEmission{}, true, &api.InvariantError{
				Role:   context.Role(),
				Reason: "callable implementation parameter is not an identifier",
			}
		}
		arguments[index] = identifier
	}
	call := context.Factory().CallExpression(
		reference.Expression(context.Factory()),
		nil,
		nil,
		arguments,
		tsgo.NodeFlagsNone,
	)
	var statement tsgo.Statement
	if signature.Results().Len() == 0 {
		statement = context.Factory().ExpressionStatement(call)
	} else {
		statement = context.Factory().ReturnStatement(call)
	}
	var target api.CallableImplementationTarget
	if moduleFunction {
		target, err = api.NewCallableImplementationModuleTarget(
			names.TargetModulePath(),
			targetName,
		)
	} else {
		receiver := api.MethodReceiverTypeName(function)
		if receiver == nil || !receiver.Exported() {
			return api.BlockEmission{}, true, &api.InvariantError{
				Role:   context.Role(),
				Reason: "callable implementation static method has no exported receiver type",
			}
		}
		className, nameErr := context.Names().Declare(receiver)
		if nameErr != nil {
			return api.BlockEmission{}, true, nameErr
		}
		target, err = api.NewCallableImplementationStaticMethodTarget(
			names.TargetModulePath(),
			className,
			targetName,
		)
	}
	if err != nil {
		return api.BlockEmission{}, true, err
	}
	if err := context.AcceptCallableImplementation(selection, target); err != nil {
		return api.BlockEmission{}, true, err
	}
	return api.DirectBlock(
		context.Factory().Block([]tsgo.Statement{statement}, true),
		reference.Requests()...,
	), true, nil
}
