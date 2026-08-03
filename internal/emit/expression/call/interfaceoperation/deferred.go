package interfaceoperation

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/deferredregistry"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func ApplyDeferred(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	sourceType types.Type,
	receiver api.ExpressionEmission,
	method *types.Func,
	signature *types.Signature,
	cooperative bool,
	arguments []tsgo.Expression,
	recovery tsgo.Expression,
) (api.ExpressionEmission, error) {
	if method == nil || signature == nil ||
		signature.Params().Len() != len(arguments) || recovery == nil {
		return api.ExpressionEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "deferred interface invocation identity is invalid",
		}
	}
	receiverContract, err := NonNilType(
		context,
		children,
		source,
		sourceType,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	nonNil, err := context.Names().Runtime(
		api.RuntimeInterfaceNonNil,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	member, err := context.Names().InterfaceMethodName(method)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	receiverName, err := context.Names().Temporary(
		api.TemporaryReceiverValue,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	deferredName, err := context.Names().Temporary(
		api.TemporaryDeferredCall,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	registrySignature := types.NewSignatureType(
		nil,
		nil,
		nil,
		signature.Params(),
		signature.Results(),
		signature.Variadic(),
	)
	registry, err := deferredregistry.Reference(
		context,
		source,
		registrySignature,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	token, err := context.Names().InterfaceMethodToken(method)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	resolveName := api.DeferredRegistryResolveMethodName
	guarded := context.Factory().CallExpression(
		context.Factory().Identifier(nonNil.Name()),
		nil,
		[]tsgo.TypeNode{receiverContract.Value()},
		[]tsgo.Expression{receiver.Value()},
		tsgo.NodeFlagsNone,
	)
	before := append(
		receiver.Before(),
		context.Factory().VariableStatement(
			nil,
			context.Factory().VariableDeclarationList(
				[]tsgo.VariableDeclaration{
					context.Factory().VariableDeclaration(
						context.Factory().Identifier(receiverName),
						nil,
						receiverContract.Value(),
						guarded,
					),
				},
				tsgo.NodeFlagsConst,
			),
		),
	)
	before = append(before, context.Factory().VariableStatement(
		nil,
		context.Factory().VariableDeclarationList(
			[]tsgo.VariableDeclaration{context.Factory().VariableDeclaration(
				context.Factory().Identifier(deferredName),
				nil,
				nil,
				context.Factory().CallExpression(
					context.Factory().PropertyAccessExpression(
						registry.Expression(context.Factory()),
						nil,
						context.Factory().Identifier(
							resolveName,
						),
						tsgo.NodeFlagsNone,
					),
					nil,
					nil,
					[]tsgo.Expression{
						context.Factory().Identifier(token.Name()),
						context.Factory().Identifier(receiverName),
					},
					tsgo.NodeFlagsNone,
				),
			)},
			tsgo.NodeFlagsConst,
		),
	))
	ordinary := context.Factory().CallExpression(
		context.Factory().PropertyAccessExpression(
			context.Factory().Identifier(receiverName),
			nil,
			context.Factory().Identifier(member),
			tsgo.NodeFlagsNone,
		),
		nil,
		nil,
		arguments,
		tsgo.NodeFlagsNone,
	)
	deferredArguments := []tsgo.Expression{
		recovery,
		context.Factory().Identifier(receiverName),
	}
	deferredArguments = append(deferredArguments, arguments...)
	deferred := context.Factory().CallExpression(
		context.Factory().Identifier(deferredName),
		nil,
		nil,
		deferredArguments,
		tsgo.NodeFlagsNone,
	)
	call := context.Factory().ConditionalExpression(
		context.Factory().BinaryExpression(
			nil,
			context.Factory().Identifier(deferredName),
			nil,
			context.Factory().BinaryOperatorToken(
				tsgo.BinaryOperatorEqualsEqualsEqualsToken,
			),
			context.Factory().Identifier("undefined"),
		),
		context.Factory().QuestionToken(),
		ordinary,
		context.Factory().ColonToken(),
		deferred,
	)
	return api.NewExpressionEmission(
		before,
		call,
		api.CombineRequests(
			receiver.Requests(),
			receiverContract.Requests(),
			nonNil.Requests(),
			registry.Requests(),
			token.Requests(),
		),
	)
}
