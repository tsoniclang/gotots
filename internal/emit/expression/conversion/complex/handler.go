package complex

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	runtimecomplex "github.com/tsoniclang/gotots/internal/emit/runtime/complex"
	complexvalue "github.com/tsoniclang/gotots/internal/emit/value/complex"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
	sourceType types.Type,
	targetType types.Type,
) (api.ExpressionEmission, error) {
	if source == nil || len(source.Args) != 1 {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	operand, err := children.Expression(
		context.
			WithRole(api.RoleConversionOperand).
			WithExpectedType(sourceType),
		source.Args[0],
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return Convert(
		context,
		source,
		sourceType,
		targetType,
		operand,
	)
}

func Convert(
	context api.Context,
	source ast.Node,
	sourceType types.Type,
	targetType types.Type,
	operand api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	sourceCarrier, sourceOK := complexvalue.Describe(sourceType)
	targetCarrier, targetOK := complexvalue.Describe(targetType)
	if !sourceOK || !targetOK {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if sourceCarrier.Bits() == targetCarrier.Bits() {
		return operand, nil
	}
	name, err := context.Names().Temporary(api.TemporaryConversionOperand)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	factory := context.Factory()
	before := append(
		operand.Before(),
		factory.VariableStatement(
			nil,
			factory.VariableDeclarationList(
				[]tsgo.VariableDeclaration{
					factory.VariableDeclaration(
						factory.Identifier(name),
						nil,
						nil,
						operand.Value(),
					),
				},
				tsgo.NodeFlagsConst,
			),
		),
	)
	temporary := factory.Identifier(name)
	component := func(member string) tsgo.Expression {
		return factory.PropertyAccessExpression(
			temporary,
			nil,
			factory.Identifier(member),
			tsgo.NodeFlagsNone,
		)
	}
	target, err := complexvalue.Construct(
		context,
		targetCarrier,
		component(runtimecomplex.RealMember),
		component(runtimecomplex.ImagMember),
		operand.Requests()...,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.NewExpressionEmission(
		before,
		target.Value(),
		target.Requests(),
	)
}
