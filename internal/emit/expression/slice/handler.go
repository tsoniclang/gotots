package slice

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	expressionoperands "github.com/tsoniclang/gotots/internal/emit/expression/operands"
	slicingexpression "github.com/tsoniclang/gotots/internal/emit/expression/slicing"
	basictype "github.com/tsoniclang/gotots/internal/emit/type/basic"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	slicevalue "github.com/tsoniclang/gotots/internal/emit/value/slice"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.SliceExpr,
) (api.ExpressionEmission, error) {
	operandType := context.TypesInfo().TypeOf(source.X)
	if _, _, ok := slicevalue.Resolve(operandType); ok {
		return slicingexpression.Emit(context, children, source)
	}
	if _, ok := definedtype.ResolveSlice(operandType); ok {
		return slicingexpression.Emit(context, children, source)
	}
	resultType := context.TypesInfo().TypeOf(source)
	if source.Slice3 ||
		source.Max != nil ||
		!basictype.SupportsString(operandType) ||
		!basictype.SupportsString(resultType) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if expected := context.ExpectedType(); expected != nil &&
		!types.AssignableTo(resultType, expected) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	operand, err := children.Expression(
		context.
			WithRole(api.RoleSliceOperand).
			WithExpectedType(types.Typ[types.String]),
		source.X,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	low, err := bound(
		context.WithRole(api.RoleSliceLow),
		children,
		source,
		source.Low,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	operands := []expressionoperands.Item{
		expressionoperands.Present(operand),
		expressionoperands.Present(low),
	}
	var high api.ExpressionEmission
	if source.High != nil {
		high, err = bound(
			context.WithRole(api.RoleSliceHigh),
			children,
			source,
			source.High,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		operands = append(operands, expressionoperands.Present(high))
	}
	ordered, err := expressionoperands.Preserve(
		context,
		api.TemporarySliceOperand,
		operands...,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	reference, err := context.Names().Runtime(
		api.RuntimeStringSlice,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.NewExpressionEmission(
		ordered.Before(),
		context.Factory().CallExpression(
			context.Factory().Identifier(reference.Name()),
			nil,
			nil,
			ordered.Values(),
			tsgo.NodeFlagsNone,
		),
		api.CombineRequests(
			ordered.Requests(),
			reference.Requests(),
		),
	)
}

func bound(
	context api.Context,
	children api.ChildEmitter,
	source *ast.SliceExpr,
	value ast.Expr,
) (api.ExpressionEmission, error) {
	if value == nil {
		zero, err := api.IntegerLiteral(
			context.Factory(),
			context.IntegerRepresentation(),
			"0",
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		return api.DirectExpression(zero), nil
	}
	sourceType := context.TypesInfo().TypeOf(value)
	if !basictype.SupportsStringIndex(context.TypesSizes(), sourceType) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	expectedType := sourceType
	if basic, ok := types.Unalias(sourceType).(*types.Basic); ok &&
		basic.Info()&types.IsUntyped != 0 {
		expectedType = types.Typ[types.Int]
	}
	return children.Expression(context.WithExpectedType(expectedType), value)
}
