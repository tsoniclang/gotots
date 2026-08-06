package float

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	floatvalue "github.com/tsoniclang/gotots/internal/emit/value/float"
	integervalue "github.com/tsoniclang/gotots/internal/emit/value/integer"
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
		context.ScalarABI(),
		source,
		sourceType,
		targetType,
		operand,
	)
}

func Convert(
	context api.Context,
	sourceABI api.ScalarABI,
	source ast.Node,
	sourceType types.Type,
	targetType types.Type,
	operand api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	targetCarrier, ok := floatvalue.Describe(targetType)
	if !ok {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	sourceCarrier, floatSource := floatvalue.Describe(sourceType)
	integerCarrier, integerSource := integervalue.Describe(
		context.TypesSizes(),
		sourceType,
	)
	if !floatSource && !integerSource {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	value := operand.Value()
	requests := operand.Requests()
	sourceRepresentation := api.IntegerCarrierInvalid
	if integerSource {
		var err error
		sourceRepresentation, err = sourceABI.Carrier(integerCarrier.Alias())
		if err != nil {
			return api.ExpressionEmission{}, err
		}
	}
	if sourceRepresentation == api.IntegerCarrierBigInt {
		value = context.Factory().CallExpression(
			api.TargetIntrinsicNumber.Expression(context.Factory()),
			nil,
			nil,
			[]tsgo.Expression{value},
			tsgo.NodeFlagsNone,
		)
	}
	if targetCarrier.Bits() == 32 &&
		(!floatSource || sourceCarrier.Bits() != 32) {
		reference, err := context.Names().Runtime(
			api.RuntimeFloat32Round,
			api.ImportPhaseValue,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		value = context.Factory().CallExpression(
			reference.Expression(context.Factory()),
			nil,
			nil,
			[]tsgo.Expression{value},
			tsgo.NodeFlagsNone,
		)
		requests = api.CombineRequests(requests, reference.Requests())
	}
	return api.NewExpressionEmission(
		operand.Before(),
		value,
		requests,
	)
}
