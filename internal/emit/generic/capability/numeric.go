package capability

import (
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	basicbinary "github.com/tsoniclang/gotots/internal/emit/expression/binary/basic"
	floatbinary "github.com/tsoniclang/gotots/internal/emit/expression/binary/float"
	integerbinary "github.com/tsoniclang/gotots/internal/emit/expression/binary/integer"
	unaryoperation "github.com/tsoniclang/gotots/internal/emit/expression/unary/operation"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	floatvalue "github.com/tsoniclang/gotots/internal/emit/value/float"
	integervalue "github.com/tsoniclang/gotots/internal/emit/value/integer"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func emitUnary(
	context api.Context,
	operation api.GenericOperation,
	signature *types.Signature,
	arguments []tsgo.Expression,
) (api.ExpressionEmission, error) {
	if len(arguments) != 1 {
		return api.ExpressionEmission{}, shapeError(context, operation)
	}
	var sourceToken token.Token
	switch operation {
	case api.GenericOperationUnaryPlus:
		sourceToken = token.ADD
	case api.GenericOperationUnaryMinus:
		sourceToken = token.SUB
	case api.GenericOperationUnaryNot:
		sourceToken = token.NOT
	case api.GenericOperationUnaryXor:
		sourceToken = token.XOR
	default:
		return api.ExpressionEmission{}, shapeError(context, operation)
	}
	result, handled, err := unaryoperation.Apply(
		context,
		sourceToken,
		signature.Params().At(0).Type(),
		api.DirectExpression(arguments[0]),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if !handled {
		return api.ExpressionEmission{}, invariant(
			context,
			"generic unary capability has no concrete operation: "+
				operation.String(),
		)
	}
	return result, nil
}

func emitBinary(
	context api.Context,
	operation api.GenericOperation,
	sourceToken token.Token,
	signature *types.Signature,
	arguments []tsgo.Expression,
) (api.ExpressionEmission, error) {
	if len(arguments) != 2 {
		return api.ExpressionEmission{}, shapeError(context, operation)
	}
	if orderedComparison(sourceToken) {
		return emitOrderedComparison(
			context,
			operation,
			sourceToken,
			signature,
			arguments,
		)
	}
	result, handled, err := context.Values().BinaryUpdate(
		context,
		nil,
		nil,
		signature.Params().At(0).Type(),
		signature.Params().At(1).Type(),
		sourceToken,
		arguments[0],
		api.DirectExpression(arguments[1]),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if !handled {
		return api.ExpressionEmission{}, invariant(
			context,
			"generic binary capability has no concrete operation: "+
				operation.String(),
		)
	}
	return result, nil
}

func emitOrderedComparison(
	context api.Context,
	operation api.GenericOperation,
	sourceToken token.Token,
	signature *types.Signature,
	arguments []tsgo.Expression,
) (api.ExpressionEmission, error) {
	leftType := signature.Params().At(0).Type()
	rightType := signature.Params().At(1).Type()
	if !types.AssignableTo(rightType, leftType) {
		return api.ExpressionEmission{}, shapeError(context, operation)
	}
	left := api.DirectExpression(arguments[0])
	right := api.DirectExpression(arguments[1])
	if model, ok := definedtype.ResolveBasic(leftType); ok {
		basic, valid := model.Basic()
		if !valid {
			return api.ExpressionEmission{}, shapeError(context, operation)
		}
		var err error
		left, err = model.Project(context, left)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		right, err = model.Project(context, right)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		leftType = basic
	}
	var (
		result  api.ExpressionEmission
		handled bool
		err     error
	)
	if carrier, ok := integervalue.Describe(
		context.TypesSizes(),
		leftType,
	); ok {
		result, handled, err = integerbinary.Apply(
			context,
			sourceToken,
			carrier,
			left,
			right,
		)
	} else if carrier, ok := floatvalue.Describe(leftType); ok {
		result, handled, err = floatbinary.Apply(
			context,
			sourceToken,
			carrier,
			left,
			right,
		)
	} else {
		result, handled = basicbinary.Apply(
			context,
			leftType,
			sourceToken,
			left,
			right,
		)
	}
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if !handled {
		return api.ExpressionEmission{}, invariant(
			context,
			"generic ordered comparison has no concrete operation: "+
				operation.String(),
		)
	}
	return result, nil
}

func orderedComparison(sourceToken token.Token) bool {
	switch sourceToken {
	case token.LSS, token.LEQ, token.GTR, token.GEQ:
		return true
	default:
		return false
	}
}
