package capability

import (
	"fmt"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	builtinoperation "github.com/tsoniclang/gotots/internal/emit/expression/builtin"
	clearoperation "github.com/tsoniclang/gotots/internal/emit/expression/builtin/clear"
	conversionoperation "github.com/tsoniclang/gotots/internal/emit/expression/conversion"
	indexoperation "github.com/tsoniclang/gotots/internal/emit/expression/index"
	slicingoperation "github.com/tsoniclang/gotots/internal/emit/expression/slicing"
	assertionoperation "github.com/tsoniclang/gotots/internal/emit/expression/typeassertion/operation"
	"github.com/tsoniclang/gotots/internal/emit/value/interfacevalue"
	"github.com/tsoniclang/gotots/internal/emit/value/maprepresentation"
	"github.com/tsoniclang/gotots/internal/emit/value/nilcomparison"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Build(
	context api.Context,
	children api.ChildEmitter,
	artifact *api.GeneratedArtifact,
	modifiers []tsgo.ModifierLike,
) (tsgo.Statement, []api.RootRequest, error) {
	signature, selection, ok := artifact.GenericCapability()
	if !ok {
		return nil, nil, invariant(
			context,
			"generated artifact is not a generic capability",
		)
	}
	if sourceType, facet, direction, storage :=
		api.GenericStorageOperationType(selection, signature); storage {
		return buildStorage(
			context,
			children,
			artifact.TargetName(),
			modifiers,
			sourceType,
			facet,
			direction,
		)
	}
	target, modifiers, resultType, err := buildCallableSurface(
		context,
		children,
		signature,
		artifact.Placement(),
		modifiers,
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
	body := value.Before()
	if signature.Results().Len() == 0 {
		body = append(
			body,
			context.Factory().ExpressionStatement(value.Value()),
		)
	} else {
		body = append(
			body,
			context.Factory().ReturnStatement(value.Value()),
		)
	}
	statement := tsgo.Statement(context.Factory().FunctionDeclaration(
		modifiers,
		nil,
		context.Factory().Identifier(artifact.TargetName()),
		nil,
		target.Parameters(),
		resultType,
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
	if signature == nil {
		return api.ExpressionEmission{}, shapeError(context, operation)
	}
	switch operation {
	case api.GenericOperationInterfaceAssert:
		return emitInterfaceAssertion(
			context,
			children,
			operation,
			signature,
			arguments,
			false,
		)
	case api.GenericOperationInterfaceAssertOK:
		return emitInterfaceAssertion(
			context,
			children,
			operation,
			signature,
			arguments,
			true,
		)
	case api.GenericOperationClear:
		if len(arguments) != 1 ||
			signature.Params().Len() != 1 ||
			signature.Results().Len() != 0 {
			return api.ExpressionEmission{}, shapeError(context, operation)
		}
		target, handled, err := clearoperation.Apply(
			context,
			nil,
			signature.Params().At(0).Type(),
			api.DirectExpression(arguments[0]),
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		if !handled {
			return api.ExpressionEmission{}, invariant(
				context,
				"generic clear capability has no concrete operation",
			)
		}
		return target, nil
	}
	if signature.Results().Len() != 1 {
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
		valueType := signature.Params().At(0).Type()
		return context.Values().Transfer(
			context,
			nil,
			valueType,
			valueType,
			api.ValueTransferCopy,
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
	case api.GenericOperationNilEqual:
		if len(arguments) != 1 ||
			signature.Params().Len() != 1 ||
			!types.Identical(
				signature.Results().At(0).Type(),
				types.Typ[types.Bool],
			) {
			return api.ExpressionEmission{}, shapeError(context, operation)
		}
		target, handled, err := nilcomparison.Apply(
			context,
			nil,
			signature.Params().At(0).Type(),
			api.DirectExpression(arguments[0]),
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		if !handled {
			return api.ExpressionEmission{}, invariant(
				context,
				"generic nil equality has no concrete operation",
			)
		}
		return target, nil
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
	case api.GenericOperationSlice, api.GenericOperationSliceFull:
		full := operation == api.GenericOperationSliceFull
		wantParameters := 3
		if full {
			wantParameters = 4
		}
		if len(arguments) != wantParameters ||
			signature.Params().Len() != wantParameters {
			return api.ExpressionEmission{}, shapeError(context, operation)
		}
		bounds := make([]api.ExpressionEmission, 0, wantParameters-1)
		for _, argument := range arguments[1:] {
			bounds = append(bounds, api.DirectExpression(argument))
		}
		target, handled, err := slicingoperation.Apply(
			context,
			nil,
			signature.Params().At(0).Type(),
			signature.Results().At(0).Type(),
			api.DirectExpression(arguments[0]),
			bounds,
			full,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		if !handled {
			return api.ExpressionEmission{}, invariant(
				context,
				"generic slicing capability has no concrete operation",
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
	case api.GenericOperationAppendSpread:
		if len(arguments) != 2 ||
			signature.Params().Len() != 2 {
			return api.ExpressionEmission{}, shapeError(context, operation)
		}
		target, handled, err := builtinoperation.ApplyAppendSpread(
			context,
			nil,
			signature.Results().At(0).Type(),
			signature.Params().At(0).Type(),
			signature.Params().At(1).Type(),
			api.DirectExpression(arguments[0]),
			api.DirectExpression(arguments[1]),
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		if !handled {
			return api.ExpressionEmission{}, invariant(
				context,
				"generic append-spread capability has no concrete operation",
			)
		}
		return target, nil
	case api.GenericOperationReflectionType:
		return emitReflectionType(context, operation, signature, arguments)
	case api.GenericOperationReflectionValue:
		return emitReflectionValue(context, operation, signature, arguments)
	case api.GenericOperationIndexAddress:
		receiver, _, element, selected :=
			api.GenericIndexAddressOperation(selection, signature)
		if !selected {
			return api.ExpressionEmission{}, shapeError(context, operation)
		}
		value, handled, err := emitIndexAddressValue(
			context,
			children,
			receiver,
			element,
			arguments,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		if !handled {
			return api.ExpressionEmission{}, shapeError(context, operation)
		}
		return value, nil
	case api.GenericOperationConstraintMethod:
		return emitConstraintMethod(
			context,
			children,
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

func emitInterfaceAssertion(
	context api.Context,
	children api.ChildEmitter,
	operation api.GenericOperation,
	signature *types.Signature,
	arguments []tsgo.Expression,
	commaOK bool,
) (api.ExpressionEmission, error) {
	expectedResults := 1
	if commaOK {
		expectedResults = 2
	}
	if len(arguments) != 1 ||
		signature.Params().Len() != 1 ||
		signature.Results().Len() != expectedResults ||
		(commaOK &&
			!types.Identical(
				signature.Results().At(1).Type(),
				types.Typ[types.Bool],
			)) {
		return api.ExpressionEmission{}, shapeError(context, operation)
	}
	return assertionoperation.Apply(
		context,
		children,
		nil,
		signature.Params().At(0).Type(),
		signature.Results().At(0).Type(),
		commaOK,
		api.DirectExpression(arguments[0]),
	)
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
