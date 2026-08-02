package function

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	genericabi "github.com/tsoniclang/gotots/internal/emit/generic/abi"
	genericdeclaration "github.com/tsoniclang/gotots/internal/emit/generic/declaration"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func deferredRegistrationStatement(
	context api.Context,
	registry tsgo.Expression,
	source tsgo.Expression,
	deferred tsgo.Expression,
) tsgo.Statement {
	return context.Factory().ExpressionStatement(
		context.Factory().CallExpression(
			context.Factory().PropertyAccessExpression(
				registry,
				nil,
				context.Factory().Identifier(
					api.DeferredRegistryRegisterName,
				),
				tsgo.NodeFlagsNone,
			),
			nil,
			nil,
			[]tsgo.Expression{source, deferred},
			tsgo.NodeFlagsNone,
		),
	)
}

func modifiersWithoutExport(
	modifiers []tsgo.ModifierLike,
) []tsgo.ModifierLike {
	result := make([]tsgo.ModifierLike, 0, len(modifiers))
	for _, modifier := range modifiers {
		if modifier.Kind() != tsgo.SyntaxKindExportKeyword {
			result = append(result, modifier)
		}
	}
	return result
}

func emitVariantParameters(
	context api.Context,
	children api.ChildEmitter,
	source *ast.FuncDecl,
	function *types.Func,
	signature *types.Signature,
	genericParameters genericdeclaration.Parameters,
	targetSignature callable.SignatureEmission,
	valueReceiver bool,
) (
	[]tsgo.ParameterDeclaration,
	[]api.RootRequest,
	error,
) {
	sourceParameters := targetSignature.Parameters()
	var capabilityParameters []tsgo.ParameterDeclaration
	var err error
	if len(api.GenericDeclarationParameters(function)) != 0 {
		capabilityParameters, err = genericabi.JoinCapabilities(
			function,
			genericParameters.Operations(),
			genericParameters.Capabilities(),
		)
		if err != nil {
			return nil, nil, err
		}
	}
	parameters := append(capabilityParameters, sourceParameters...)
	requests := api.CombineRequests(
		genericParameters.Requests(),
		targetSignature.Requests(),
	)
	if signature.RecvTypeParams().Len() != 0 {
		sourceBindings, bindErr := genericabi.SourceParameters(
			function,
			sourceParameters,
		)
		if bindErr != nil {
			return nil, nil, bindErr
		}
		parameters, err = genericabi.JoinClassMethod(
			function,
			genericParameters.Operations(),
			genericabi.Combine(
				genericParameters.Capabilities(),
				sourceBindings,
			),
		)
		if err != nil {
			return nil, nil, err
		}
	}
	if signature.Recv() != nil && !valueReceiver {
		receiver, receiverRequests, err := emitReceiver(
			context,
			children,
			source,
			signature,
		)
		if err != nil {
			return nil, nil, err
		}
		parameters = append(
			[]tsgo.ParameterDeclaration{receiver},
			parameters...,
		)
		requests = append(receiverRequests, requests...)
	}
	return parameters, requests, nil
}

func valueReceiverCopySelected(
	context api.Context,
	function *types.Func,
	requirements []api.DeclarationRequirement,
) (bool, error) {
	selected := false
	for _, requirement := range requirements {
		if requirement.Kind() !=
			api.DeclarationRequirementValueReceiverCopy {
			continue
		}
		method, ok := requirement.ValueReceiverCopy()
		if !ok || method != function || selected {
			return false, &api.InvariantError{
				Role:   context.Role(),
				Reason: "function received an invalid value-receiver copy requirement",
			}
		}
		selected = true
	}
	return selected, nil
}

func prependValueReceiverCopy(
	context api.Context,
	children api.ChildEmitter,
	source *ast.FuncDecl,
	signature *types.Signature,
	body api.BlockEmission,
) (api.BlockEmission, error) {
	receiver := signature.Recv()
	if source.Recv == nil || len(source.Recv.List) != 1 {
		return api.BlockEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "value-receiver copy prologue has no receiver syntax",
		}
	}
	binding, ok := context.ValueReceiver(receiver)
	if !ok || !binding.CopySelected() {
		return api.BlockEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "value-receiver copy prologue has no selected receiver",
		}
	}
	if _, addressable := context.AddressableStorage().Name(
		context,
		receiver,
	); addressable {
		return body, nil
	}
	copied, err := context.Values().Transfer(
		context.WithRole(api.RoleReceiverValue),
		source,
		receiver.Type(),
		receiver.Type(),
		api.ValueTransferCopy,
		api.DirectExpression(binding.OriginalValue()),
	)
	if err != nil {
		return api.BlockEmission{}, err
	}
	receiverType, err := children.RepresentedType(
		context.WithRole(api.RoleReceiverType),
		source.Recv.List[0].Type,
		receiver.Type(),
	)
	if err != nil {
		return api.BlockEmission{}, err
	}
	statements := append([]tsgo.Statement(nil), copied.Before()...)
	statements = append(
		statements,
		context.Factory().VariableStatement(
			nil,
			context.Factory().VariableDeclarationList(
				[]tsgo.VariableDeclaration{
					context.Factory().VariableDeclaration(
						context.Factory().Identifier(
							binding.CopyName(),
						),
						nil,
						receiverType.Value(),
						copied.Value(),
					),
				},
				tsgo.NodeFlagsLet,
			),
		),
	)
	statements = append(statements, body.Value().Statements()...)
	return api.DirectBlock(
		context.Factory().Block(statements, true),
		api.CombineRequests(
			copied.Requests(),
			receiverType.Requests(),
			body.Requests(),
		)...,
	), nil
}
