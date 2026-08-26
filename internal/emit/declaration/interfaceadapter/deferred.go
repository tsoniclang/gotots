package interfaceadapter

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	"github.com/tsoniclang/gotots/internal/emit/deferredregistry"
	panicruntime "github.com/tsoniclang/gotots/internal/emit/runtime/panic"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func methodDeferredSupport(
	context api.Context,
	adapterName string,
	dynamicTypeName string,
	memberName string,
	signature *types.Signature,
	target callable.SignatureEmission,
	resultType tsgo.TypeNode,
	contracts []*types.Func,
) ([]tsgo.Statement, []api.RootRequest, error) {
	if adapterName == "" || dynamicTypeName == "" || memberName == "" ||
		signature == nil || len(contracts) == 0 {
		return nil, nil, &api.GeneratedArtifactShapeError{
			Artifact: adapterName,
			Reason:   "deferred interface support identity is invalid",
		}
	}
	recovery, recoveryRequests, err :=
		callable.RecoveryAuthorityParameter(context)
	if err != nil {
		return nil, nil, err
	}
	runtimeValue, err := context.Names().Runtime(
		api.RuntimeInterfaceValue,
		api.ImportPhaseType,
	)
	if err != nil {
		return nil, nil, err
	}
	panicReference, err := context.Names().Runtime(
		api.RuntimePanic,
		api.ImportPhaseValue,
	)
	if err != nil {
		return nil, nil, err
	}
	registry, err := deferredregistry.Reference(context, nil, signature)
	if err != nil {
		return nil, nil, err
	}
	receiver := context.Factory().Identifier("receiver")
	dispatcherName := adapterName + "_" + memberName + api.DeferredEntrySuffix
	parameters := append(
		[]tsgo.ParameterDeclaration{
			recovery,
			context.Factory().ParameterDeclaration(
				nil,
				nil,
				receiver,
				nil,
				context.Factory().TypeReferenceNode(
					runtimeValue.EntityName(context.Factory()),
					nil,
				),
				nil,
			),
		},
		target.Parameters()...,
	)
	guard := context.Factory().CallExpression(
		context.Factory().PropertyAccessExpression(
			context.Factory().Identifier(adapterName),
			nil,
			context.Factory().Identifier(GuardMember),
			tsgo.NodeFlagsNone,
		),
		nil,
		nil,
		[]tsgo.Expression{receiver},
		tsgo.NodeFlagsNone,
	)
	arguments := append(
		[]tsgo.Expression{
			context.Factory().Identifier(callable.RecoveryAuthorityName),
		},
		target.ParameterReferences(context.Factory())...,
	)
	call := context.Factory().CallExpression(
		context.Factory().PropertyAccessExpression(
			receiver,
			nil,
			context.Factory().Identifier(
				memberName+api.DeferredEntrySuffix,
			),
			tsgo.NodeFlagsNone,
		),
		nil,
		nil,
		arguments,
		tsgo.NodeFlagsNone,
	)
	body := []tsgo.Statement{context.Factory().IfStatement(
		context.Factory().PrefixUnaryExpression(
			tsgo.PrefixUnaryExpressionOperatorKindExclamationToken,
			guard,
		),
		context.Factory().Block([]tsgo.Statement{
			context.Factory().ReturnStatement(panicruntime.Call(
				context.Factory(),
				panicReference.Name(),
				context.Factory().StringLiteral(
					"deferred interface adapter identity mismatch",
					tsgo.TokenFlagsNone,
				),
			)),
		}, true),
		nil,
	)}
	if signature.Results().Len() == 0 {
		body = append(body, context.Factory().ExpressionStatement(call))
	} else {
		body = append(body, context.Factory().ReturnStatement(call))
	}
	modifiers := []tsgo.ModifierLike{context.Factory().ExportKeyword()}
	statements := []tsgo.Statement{context.Factory().FunctionDeclaration(
		modifiers,
		nil,
		context.Factory().Identifier(dispatcherName),
		nil,
		parameters,
		resultType,
		context.Factory().Block(body, true),
	)}
	registerName := api.DeferredRegistryRegisterMethodName
	requests := api.CombineRequests(
		recoveryRequests,
		runtimeValue.Requests(),
		panicReference.Requests(),
		registry.Requests(),
	)
	for _, contract := range contracts {
		token, tokenErr := context.Names().InterfaceMethodToken(contract)
		if tokenErr != nil {
			return nil, nil, tokenErr
		}
		statements = append(statements, context.Factory().ExpressionStatement(
			context.Factory().CallExpression(
				context.Factory().PropertyAccessExpression(
					registry.Expression(context.Factory()),
					nil,
					context.Factory().Identifier(
						registerName,
					),
					tsgo.NodeFlagsNone,
				),
				nil,
				nil,
				[]tsgo.Expression{
					context.Factory().Identifier(token.Name()),
					context.Factory().Identifier(dynamicTypeName),
					context.Factory().Identifier(dispatcherName),
				},
				tsgo.NodeFlagsNone,
			),
		))
		requests = append(requests, token.Requests()...)
	}
	return statements, api.CombineRequests(requests), nil
}
