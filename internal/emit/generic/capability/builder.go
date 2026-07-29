package capability

import (
	"fmt"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	basicbinary "github.com/tsoniclang/gotots/internal/emit/expression/binary/basic"
	floatbinary "github.com/tsoniclang/gotots/internal/emit/expression/binary/float"
	integerbinary "github.com/tsoniclang/gotots/internal/emit/expression/binary/integer"
	builtinoperation "github.com/tsoniclang/gotots/internal/emit/expression/builtin"
	calloperation "github.com/tsoniclang/gotots/internal/emit/expression/call"
	conversionoperation "github.com/tsoniclang/gotots/internal/emit/expression/conversion"
	indexoperation "github.com/tsoniclang/gotots/internal/emit/expression/index"
	unaryoperation "github.com/tsoniclang/gotots/internal/emit/expression/unary/operation"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	floatvalue "github.com/tsoniclang/gotots/internal/emit/value/float"
	integervalue "github.com/tsoniclang/gotots/internal/emit/value/integer"
	"github.com/tsoniclang/gotots/internal/emit/value/interfacevalue"
	"github.com/tsoniclang/gotots/internal/emit/value/maprepresentation"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Build(
	context api.Context,
	children api.ChildEmitter,
	artifact *api.GeneratedArtifact,
) (tsgo.Statement, []api.RootRequest, error) {
	signature, selection, ok := artifact.GenericCapability()
	if !ok {
		return nil, nil, invariant(
			context,
			"generated artifact is not a generic capability",
		)
	}
	target, err := callable.EmitAdapter(
		context.WithRole(api.RoleFileDeclaration),
		children,
		nil,
		signature,
	)
	if err != nil {
		return nil, nil, err
	}
	value, err := emitValue(
		context.WithRole(api.RoleFunctionBody),
		children,
		selection,
		signature,
		target.ParameterReferences(context.Factory()),
	)
	if err != nil {
		return nil, nil, err
	}
	body := append(
		value.Before(),
		context.Factory().ReturnStatement(value.Value()),
	)
	statement := tsgo.Statement(context.Factory().FunctionDeclaration(
		[]tsgo.ModifierLike{context.Factory().ExportKeyword()},
		nil,
		context.Factory().Identifier(artifact.TargetName()),
		nil,
		target.Parameters(),
		target.Result(),
		context.Factory().Block(body, true),
	))
	return statement, api.CombineRequests(
		target.Requests(),
		value.Requests(),
	), nil
}

func emitValue(
	context api.Context,
	children api.ChildEmitter,
	selection api.GenericOperationSelection,
	signature *types.Signature,
	arguments []tsgo.Expression,
) (api.ExpressionEmission, error) {
	operation := selection.Operation()
	if signature == nil || signature.Results().Len() != 1 {
		return api.ExpressionEmission{}, shapeError(context, operation)
	}
	switch operation {
	case api.GenericOperationZero:
		if len(arguments) != 0 {
			return api.ExpressionEmission{}, shapeError(context, operation)
		}
		return context.Values().Zero(context, nil, signature.Results().At(0).Type())
	case api.GenericOperationCopy:
		if len(arguments) != 1 {
			return api.ExpressionEmission{}, shapeError(context, operation)
		}
		return context.Values().Copy(
			context,
			nil,
			signature.Params().At(0).Type(),
			api.DirectExpression(arguments[0]),
		)
	case api.GenericOperationEqual:
		return emitEquality(context, operation, signature, arguments, false)
	case api.GenericOperationHash:
		if len(arguments) != 1 {
			return api.ExpressionEmission{}, shapeError(context, operation)
		}
		return context.Values().Hash(
			context,
			nil,
			signature.Params().At(0).Type(),
			arguments[0],
		)
	case api.GenericOperationUnaryPlus,
		api.GenericOperationUnaryMinus,
		api.GenericOperationUnaryNot,
		api.GenericOperationUnaryXor:
		return emitUnary(context, operation, signature, arguments)
	case api.GenericOperationConvert:
		if len(arguments) != 1 || signature.Params().Len() != 1 {
			return api.ExpressionEmission{}, shapeError(context, operation)
		}
		target, handled, err := conversionoperation.Apply(
			context,
			children,
			nil,
			signature.Params().At(0).Type(),
			signature.Results().At(0).Type(),
			api.DirectExpression(arguments[0]),
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		if !handled {
			return api.ExpressionEmission{}, invariant(
				context,
				"generic conversion capability has no concrete operation",
			)
		}
		return target, nil
	case api.GenericOperationIndex:
		if len(arguments) != 2 ||
			signature.Params().Len() != 2 {
			return api.ExpressionEmission{}, shapeError(context, operation)
		}
		target, handled, err := indexoperation.Apply(
			context,
			nil,
			signature.Params().At(0).Type(),
			signature.Params().At(1).Type(),
			signature.Results().At(0).Type(),
			api.DirectExpression(arguments[0]),
			api.DirectExpression(arguments[1]),
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		if !handled {
			return api.ExpressionEmission{}, invariant(
				context,
				"generic index capability has no concrete operation",
			)
		}
		return target, nil
	case api.GenericOperationMapConstruct:
		return emitMapConstruct(
			context,
			operation,
			signature,
			arguments,
		)
	case api.GenericOperationInterfaceAdapt:
		if len(arguments) != 1 ||
			signature.Params().Len() != 1 {
			return api.ExpressionEmission{}, shapeError(context, operation)
		}
		target, handled, err := interfacevalue.Convert(
			context,
			nil,
			signature.Params().At(0).Type(),
			signature.Results().At(0).Type(),
			api.DirectExpression(arguments[0]),
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		if !handled {
			return api.ExpressionEmission{}, invariant(
				context,
				fmt.Sprintf(
					"generic interface adaptation %s -> %s has no concrete operation",
					types.TypeString(
						signature.Params().At(0).Type(),
						nil,
					),
					types.TypeString(
						signature.Results().At(0).Type(),
						nil,
					),
				),
			)
		}
		return target, nil
	case api.GenericOperationLength, api.GenericOperationCapacity:
		if len(arguments) != 1 ||
			signature.Params().Len() != 1 {
			return api.ExpressionEmission{}, shapeError(context, operation)
		}
		target, handled, err := builtinoperation.ApplyMeasure(
			context,
			children,
			nil,
			operation,
			signature.Params().At(0).Type(),
			signature.Results().At(0).Type(),
			api.DirectExpression(arguments[0]),
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		if !handled {
			return api.ExpressionEmission{}, invariant(
				context,
				"generic measure capability has no concrete operation",
			)
		}
		return target, nil
	case api.GenericOperationConstraintMethod:
		return emitConstraintMethod(
			context,
			operation,
			selection,
			signature,
			arguments,
		)
	case api.GenericOperationBinaryEqual:
		return emitEquality(context, operation, signature, arguments, false)
	case api.GenericOperationBinaryNotEqual:
		return emitEquality(context, operation, signature, arguments, true)
	default:
		if sourceToken, ok := operation.BinaryToken(); ok {
			return emitBinary(
				context,
				operation,
				sourceToken,
				signature,
				arguments,
			)
		}
		return api.ExpressionEmission{}, invariant(
			context,
			"generic capability operation is not implemented: "+
				operation.String(),
		)
	}
}

func emitMapConstruct(
	context api.Context,
	operation api.GenericOperation,
	signature *types.Signature,
	arguments []tsgo.Expression,
) (api.ExpressionEmission, error) {
	if signature.Params().Len() != len(arguments) ||
		signature.Results().Len() != 1 ||
		len(arguments) == 0 {
		return api.ExpressionEmission{}, shapeError(context, operation)
	}
	mapType, ok := maprepresentation.Source(
		context,
		signature.Results().At(0).Type(),
	)
	if !ok ||
		!types.Identical(
			signature.Params().At(0).Type(),
			mapType.Element(),
		) {
		return api.ExpressionEmission{}, shapeError(context, operation)
	}
	size := tsgo.Expression(
		context.Factory().NumericLiteral("0", tsgo.TokenFlagsNone),
	)
	var entries []tsgo.Expression
	switch {
	case len(arguments) == 1:
	case len(arguments) == 2:
		if !integerType(signature.Params().At(1).Type()) {
			return api.ExpressionEmission{}, shapeError(context, operation)
		}
		size = arguments[1]
	default:
		if (len(arguments)-1)%2 != 0 {
			return api.ExpressionEmission{}, shapeError(context, operation)
		}
		size = context.Factory().NumericLiteral(
			fmt.Sprintf("%d", (len(arguments)-1)/2),
			tsgo.TokenFlagsNone,
		)
		for index := 1; index < len(arguments); index += 2 {
			if !types.Identical(
				signature.Params().At(index).Type(),
				mapType.Key(),
			) ||
				!types.Identical(
					signature.Params().At(index+1).Type(),
					mapType.Element(),
				) {
				return api.ExpressionEmission{},
					shapeError(context, operation)
			}
			entries = append(
				entries,
				context.Factory().ArrayLiteralExpression(
					[]tsgo.Expression{arguments[index], arguments[index+1]},
					false,
				),
			)
		}
	}
	return maprepresentation.Make(
		context,
		nil,
		mapType.Type(),
		arguments[0],
		size,
		entries,
	)
}

func integerType(sourceType types.Type) bool {
	basic, ok := types.Unalias(sourceType).(*types.Basic)
	return ok && basic.Info()&types.IsInteger != 0
}

func emitConstraintMethod(
	context api.Context,
	operation api.GenericOperation,
	selection api.GenericOperationSelection,
	signature *types.Signature,
	arguments []tsgo.Expression,
) (api.ExpressionEmission, error) {
	method, ok := selection.Method()
	if !ok ||
		len(arguments) == 0 ||
		signature.Params().Len() != len(arguments) {
		return api.ExpressionEmission{}, shapeError(context, operation)
	}
	methodSignature, ok := method.Type().(*types.Signature)
	if !ok || methodSignature.Recv() == nil {
		return api.ExpressionEmission{}, shapeError(context, operation)
	}
	receiverType := signature.Params().At(0).Type()
	interfaceType := methodSignature.Recv().Type()
	receiver, handled, err := interfacevalue.Convert(
		context,
		nil,
		receiverType,
		interfaceType,
		api.DirectExpression(arguments[0]),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if !handled {
		return api.ExpressionEmission{}, invariant(
			context,
			"generic constraint-method receiver has no concrete interface adaptation",
		)
	}
	return calloperation.ApplyInterfaceMethod(
		context,
		receiver,
		method,
		arguments[1:],
		nil,
		nil,
	)
}

func emitEquality(
	context api.Context,
	operation api.GenericOperation,
	signature *types.Signature,
	arguments []tsgo.Expression,
	negated bool,
) (api.ExpressionEmission, error) {
	if len(arguments) != 2 ||
		!types.Identical(
			signature.Params().At(0).Type(),
			signature.Params().At(1).Type(),
		) {
		return api.ExpressionEmission{}, shapeError(context, operation)
	}
	result, err := context.Values().Equal(
		context,
		nil,
		signature.Params().At(0).Type(),
		arguments[0],
		arguments[1],
	)
	if err != nil || !negated {
		return result, err
	}
	return api.NewExpressionEmission(
		result.Before(),
		context.Factory().PrefixUnaryExpression(
			tsgo.PrefixUnaryExpressionOperatorKindExclamationToken,
			result.Value(),
		),
		result.Requests(),
	)
}

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
		left = api.DirectExpression(
			model.Unwrap(context.Factory(), arguments[0]),
		)
		right = api.DirectExpression(
			model.Unwrap(context.Factory(), arguments[1]),
		)
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

func shapeError(
	context api.Context,
	operation api.GenericOperation,
) error {
	return invariant(
		context,
		"generic capability signature has invalid operation shape: "+
			operation.String(),
	)
}

func invariant(context api.Context, reason string) error {
	return &api.InvariantError{
		Role:   context.Role(),
		Reason: reason,
	}
}
