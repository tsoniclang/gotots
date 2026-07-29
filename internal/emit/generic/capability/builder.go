package capability

import (
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	unaryoperation "github.com/tsoniclang/gotots/internal/emit/expression/unary/operation"
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
