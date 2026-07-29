package call

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	goruntimetype "github.com/tsoniclang/gotots/internal/emit/type/goruntime"
	interfacetype "github.com/tsoniclang/gotots/internal/emit/type/interfacevalue"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func ApplyInterfaceMethod(
	context api.Context,
	source ast.Node,
	sourceType types.Type,
	receiver api.ExpressionEmission,
	method *types.Func,
	arguments []tsgo.Expression,
	argumentBefore []tsgo.Statement,
	argumentRequests []api.RootRequest,
) (api.ExpressionEmission, error) {
	if method == nil {
		return api.ExpressionEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "interface method operation has no method identity",
		}
	}
	receiverContract, err := emitNonNilInterfaceType(
		context,
		source,
		sourceType,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	receiverName, err := context.Names().Temporary(
		api.TemporaryReceiverValue,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	before := append(
		receiver.Before(),
		context.Factory().VariableStatement(
			nil,
			context.Factory().VariableDeclarationList(
				[]tsgo.VariableDeclaration{
					context.Factory().VariableDeclaration(
						context.Factory().Identifier(receiverName),
						nil,
						nil,
						receiver.Value(),
					),
				},
				tsgo.NodeFlagsConst,
			),
		),
	)
	before = append(before, argumentBefore...)
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
	guarded := context.Factory().CallExpression(
		context.Factory().Identifier(nonNil.Name()),
		nil,
		[]tsgo.TypeNode{receiverContract.Value()},
		[]tsgo.Expression{
			context.Factory().Identifier(receiverName),
		},
		tsgo.NodeFlagsNone,
	)
	call := context.Factory().CallExpression(
		context.Factory().PropertyAccessExpression(
			guarded,
			nil,
			context.Factory().Identifier(member),
			tsgo.NodeFlagsNone,
		),
		nil,
		nil,
		arguments,
		tsgo.NodeFlagsNone,
	)
	return api.NewExpressionEmission(
		before,
		call,
		api.CombineRequests(
			receiver.Requests(),
			argumentRequests,
			nonNil.Requests(),
			receiverContract.Requests(),
		),
	)
}

func emitNonNilInterfaceType(
	context api.Context,
	source ast.Node,
	sourceType types.Type,
) (api.TypeEmission, error) {
	if target, handled, err := goruntimetype.EmitNonNil(
		context,
		source,
		sourceType,
	); handled {
		return target, err
	}
	return interfacetype.EmitNonNil(context, source, sourceType)
}
