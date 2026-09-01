package interfaceoperation

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	goruntimetype "github.com/tsoniclang/gotots/internal/emit/type/goruntime"
	interfacetype "github.com/tsoniclang/gotots/internal/emit/type/interfacevalue"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Apply(
	context api.Context,
	children api.ChildEmitter,
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
	receiverContract, err := NonNilType(
		context,
		children,
		source,
		sourceType,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	receiverValue := receiver.Value()
	before := receiver.Before()
	if len(argumentBefore) != 0 {
		receiverName, nameErr := context.Names().Temporary(
			api.TemporaryReceiverValue,
		)
		if nameErr != nil {
			return api.ExpressionEmission{}, nameErr
		}
		before = append(before, context.Factory().VariableStatement(
			nil,
			context.Factory().VariableDeclarationList(
				[]tsgo.VariableDeclaration{
					context.Factory().VariableDeclaration(
						context.Factory().Identifier(receiverName),
						nil,
						nil,
						receiverValue,
					),
				},
				tsgo.NodeFlagsConst,
			),
		))
		receiverValue = context.Factory().Identifier(receiverName)
	}
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
			receiverValue,
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

func NonNilType(
	context api.Context,
	children api.ChildEmitter,
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
	return interfacetype.EmitNonNil(context, children, source, sourceType)
}
