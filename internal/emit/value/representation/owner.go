package representation

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	genericoperation "github.com/tsoniclang/gotots/internal/emit/generic/operation"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	interfacetype "github.com/tsoniclang/gotots/internal/emit/type/interfacevalue"
	pointertype "github.com/tsoniclang/gotots/internal/emit/type/pointer"
	arrayvalue "github.com/tsoniclang/gotots/internal/emit/value/array"
	complexvalue "github.com/tsoniclang/gotots/internal/emit/value/complex"
	integervalue "github.com/tsoniclang/gotots/internal/emit/value/integer"
	interfacevalue "github.com/tsoniclang/gotots/internal/emit/value/interfacevalue"
	"github.com/tsoniclang/gotots/internal/emit/value/maprepresentation"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type Owner struct {
	children api.ChildEmitter
}

func NewOwner(children api.ChildEmitter) Owner {
	if children == nil {
		panic("value representation owner has no child emitter")
	}
	return Owner{children: children}
}

func (Owner) RequiresCustomEquality(
	context api.Context,
	sourceType types.Type,
) bool {
	if _, ok := api.GenericTypeParameter(sourceType); ok {
		return true
	}
	if _, ok := interfacetype.Resolve(sourceType); ok {
		return true
	}
	if panicNilRuntimeValue(context, sourceType) {
		return true
	}
	if _, ok := definedtype.Resolve(sourceType); ok {
		return true
	}
	if _, ok := complexvalue.Describe(sourceType); ok {
		return true
	}
	if _, ok := arrayvalue.Resolve(context, sourceType); ok {
		return true
	}
	if _, ok := isAnonymousStruct(sourceType); ok {
		return true
	}
	if pointerValue(sourceType) {
		return true
	}
	if channelValue(sourceType) {
		return true
	}
	if callableValue(sourceType) {
		return true
	}
	_, _, ok := namedStruct(sourceType)
	return ok
}

func (Owner) RequiresExplicitType(
	context api.Context,
	sourceType types.Type,
) bool {
	if _, ok := api.GenericTypeParameter(sourceType); ok {
		return true
	}
	if _, ok := interfacetype.Resolve(sourceType); ok {
		return true
	}
	if panicNilRuntimeValue(context, sourceType) {
		return true
	}
	if model, ok := maprepresentation.Source(
		context,
		sourceType,
	); ok && !model.Nominal() {
		return true
	}
	if defined, ok := definedtype.Resolve(sourceType); ok {
		return defined.NilCapable()
	}
	return pointerValue(sourceType) ||
		callableValue(sourceType) ||
		channelValue(sourceType)
}

func (Owner) RequiresStructuralCopy(
	context api.Context,
	sourceType types.Type,
) bool {
	if _, ok := api.GenericTypeParameter(sourceType); ok {
		return true
	}
	if panicNilRuntimeValue(context, sourceType) {
		return true
	}
	if defined, ok := definedtype.Resolve(sourceType); ok {
		return defined.Family() == definedtype.FamilyArray
	}
	if _, ok := arrayvalue.Resolve(context, sourceType); ok {
		return true
	}
	if _, ok := isAnonymousStruct(sourceType); ok {
		return true
	}
	_, _, ok := namedStruct(sourceType)
	return ok
}

func (owner Owner) Zero(
	context api.Context,
	source ast.Node,
	sourceType types.Type,
) (api.ExpressionEmission, error) {
	if parameter, ok := api.GenericTypeParameter(sourceType); ok {
		return genericoperation.Call(
			context,
			source,
			api.GenericOperationZero,
			nil,
			[]types.Type{parameter},
			nil,
		)
	}
	if _, ok := interfacetype.Resolve(sourceType); ok {
		return api.DirectExpression(
			context.Factory().VoidExpression(
				context.Factory().NumericLiteral(
					"0",
					tsgo.TokenFlagsNone,
				),
			),
		), nil
	}
	if panicNilRuntimeValue(context, sourceType) {
		return panicNilZero(context)
	}
	if defined, ok := definedtype.Resolve(sourceType); ok {
		operationContext, err := defined.OperationContext(context)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		zero, err := owner.Zero(
			operationContext.WithRole(api.RoleDefinedValue),
			source,
			defined.Underlying(),
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		return defined.Wrap(context, zero)
	}
	if carrier, ok := complexvalue.Describe(sourceType); ok {
		return complexvalue.Construct(
			context,
			carrier,
			context.Factory().NumericLiteral("0", tsgo.TokenFlagsNone),
			context.Factory().NumericLiteral("0", tsgo.TokenFlagsNone),
		)
	}
	if _, ok := maprepresentation.Source(context, sourceType); ok {
		if api.ContainsGenericTypeParameter(sourceType) {
			return genericoperation.Call(
				context,
				source,
				api.GenericOperationZero,
				nil,
				[]types.Type{sourceType},
				nil,
			)
		}
		return maprepresentation.Nil(
			context,
			owner.children,
			source,
			sourceType,
		)
	}
	if array, ok := arrayvalue.Resolve(context, sourceType); ok {
		return array.Zero(context, owner.children, source)
	}
	if structType, ok := isAnonymousStruct(sourceType); ok {
		return anonymousStructZero(context, source, structType)
	}
	if alias, ok := primitive(context, sourceType); ok {
		var literal tsgo.Expression
		switch alias {
		case api.PrimitiveBool:
			literal = context.Factory().FalseLiteral()
		case api.PrimitiveString:
			literal = context.Factory().StringLiteral("", tsgo.TokenFlagsNone)
		case api.PrimitiveFloat32, api.PrimitiveFloat64:
			literal = context.Factory().NumericLiteral("0", tsgo.TokenFlagsNone)
		default:
			var err error
			literal, err = integervalue.Literal(context, sourceType, "0")
			if err != nil {
				return api.ExpressionEmission{}, err
			}
		}
		return api.DirectExpression(literal), nil
	}
	if _, _, ok := pointertype.Resolve(sourceType); ok {
		return api.DirectExpression(
			context.Factory().VoidExpression(
				context.Factory().NumericLiteral("0", tsgo.TokenFlagsNone),
			),
		), nil
	}
	if channelValue(sourceType) {
		return api.DirectExpression(
			context.Factory().Identifier("undefined"),
		), nil
	}
	if callableValue(sourceType) {
		return api.DirectExpression(
			context.Factory().VoidExpression(
				context.Factory().NumericLiteral("0", tsgo.TokenFlagsNone),
			),
		), nil
	}
	if _, _, ok := scalarSlice(context, sourceType); ok {
		return sliceZero(context, owner.children, source, sourceType)
	}
	_, _, ok := namedStruct(sourceType)
	if !ok {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	return owner.namedStructOperation(
		context,
		source,
		sourceType,
		api.NamedStructOperationZero,
		nil,
	)
}

func (owner Owner) Transfer(
	context api.Context,
	source ast.Node,
	actualType types.Type,
	destinationType types.Type,
	mode api.ValueTransferMode,
	value api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	if actualType == nil ||
		destinationType == nil ||
		!mode.Valid() ||
		!types.AssignableTo(actualType, destinationType) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if _, destinationInterface := interfacetype.Resolve(
		destinationType,
	); destinationInterface {
		adapted, handled, err := interfacevalue.Assign(
			context,
			source,
			actualType,
			destinationType,
			value,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		if !handled {
			return api.ExpressionEmission{}, &api.InvariantError{
				Role:   context.Role(),
				Reason: "interface transfer was not handled",
			}
		}
		if mode == api.ValueTransferRepresentation {
			return adapted, nil
		}
		return owner.copyExact(
			context,
			source,
			destinationType,
			adapted,
		)
	}
	if representedAtDestination(actualType, destinationType) {
		actualType = destinationType
	}
	if _, generic := api.GenericTypeParameter(actualType); generic {
		if _, mapDestination :=
			maprepresentation.Source(context, destinationType); mapDestination &&
			!types.Identical(actualType, destinationType) &&
			types.ConvertibleTo(actualType, destinationType) {
			return genericoperation.Call(
				context,
				source,
				api.GenericOperationConvert,
				[]types.Type{actualType},
				[]types.Type{destinationType},
				[]api.ExpressionEmission{value},
			)
		}
	}
	if !types.Identical(actualType, destinationType) {
		actualOperationContext := context
		if actual, ok := definedtype.Resolve(actualType); ok {
			var err error
			actualOperationContext, err = actual.OperationContext(context)
			if err != nil {
				return api.ExpressionEmission{}, err
			}
			value, err = actual.Project(context, value)
			if err != nil {
				return api.ExpressionEmission{}, err
			}
			actualType = actual.Underlying()
		}
		if destination, ok := definedtype.Resolve(destinationType); ok {
			destinationOperationContext, err := destination.OperationContext(context)
			if err != nil {
				return api.ExpressionEmission{}, err
			}
			value, err = transferIntegerRepresentation(
				context,
				actualType,
				destination.Underlying(),
				actualOperationContext.ScalarABI(),
				destinationOperationContext.ScalarABI(),
				value,
			)
			if err != nil {
				return api.ExpressionEmission{}, err
			}
			if mode == api.ValueTransferCopy {
				value, err = owner.copyExact(
					destinationOperationContext.WithRole(
						api.RoleDefinedValue,
					),
					source,
					destination.Underlying(),
					value,
				)
				if err != nil {
					return api.ExpressionEmission{}, err
				}
			}
			return destination.Wrap(context, value)
		}
		var err error
		value, err = transferIntegerRepresentation(
			context,
			actualType,
			destinationType,
			actualOperationContext.ScalarABI(),
			context.ScalarABI(),
			value,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
	}
	if mode == api.ValueTransferRepresentation {
		return value, nil
	}
	return owner.copyExact(context, source, destinationType, value)
}

func (owner Owner) copyExact(
	context api.Context,
	source ast.Node,
	sourceType types.Type,
	value api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	if parameter, ok := api.GenericTypeParameter(sourceType); ok {
		target, err := genericoperation.Call(
			context,
			source,
			api.GenericOperationCopy,
			[]types.Type{parameter},
			[]types.Type{parameter},
			[]api.ExpressionEmission{value},
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		return target, nil
	}
	if _, ok := interfacetype.Resolve(sourceType); ok {
		return api.NewExpressionEmission(
			value.Before(),
			value.Value(),
			value.Requests(),
		)
	}
	if panicNilRuntimeValue(context, sourceType) {
		return panicNilCopy(context, value)
	}
	if defined, ok := definedtype.Resolve(sourceType); ok &&
		defined.Family() != definedtype.FamilyArray {
		return api.NewExpressionEmission(
			value.Before(),
			value.Value(),
			value.Requests(),
		)
	}
	if array, ok := arrayvalue.Resolve(context, sourceType); ok {
		return array.Copy(
			context,
			owner.children,
			source,
			ownsFreshValue(context, source),
			value,
		)
	}
	if structType, ok := isAnonymousStruct(sourceType); ok {
		if ownsFreshValue(context, source) {
			return value, nil
		}
		return anonymousStructCopy(context, source, structType, value)
	}
	_, complexOK := complexvalue.Describe(sourceType)
	_, primitiveOK := primitive(context, sourceType)
	if complexOK ||
		primitiveOK ||
		callableValue(sourceType) ||
		pointerValue(sourceType) ||
		channelValue(sourceType) ||
		isScalarSlice(context, sourceType) ||
		mapValue(context, sourceType) {
		return api.NewExpressionEmission(
			value.Before(),
			value.Value(),
			value.Requests(),
		)
	}
	_, _, ok := namedStruct(sourceType)
	if !ok {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if ownsFreshValue(context, source) {
		return api.NewExpressionEmission(
			value.Before(),
			value.Value(),
			value.Requests(),
		)
	}
	target, err := owner.namedStructOperation(
		context,
		source,
		sourceType,
		api.NamedStructOperationCopy,
		[]tsgo.Expression{value.Value()},
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.NewExpressionEmission(
		value.Before(),
		target.Value(),
		api.CombineRequests(value.Requests(), target.Requests()),
	)
}

func (Owner) Assign(
	context api.Context,
	source ast.Node,
	sourceType types.Type,
	target tsgo.Expression,
	value api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	if _, ok := api.GenericTypeParameter(sourceType); ok {
		return api.NewExpressionEmission(
			value.Before(),
			context.Factory().BinaryExpression(
				nil,
				target,
				nil,
				context.Factory().BinaryOperatorToken(
					tsgo.BinaryOperatorEqualsToken,
				),
				value.Value(),
			),
			value.Requests(),
		)
	}
	_, definedOK := definedtype.Resolve(sourceType)
	_, arrayOK := arrayvalue.Resolve(context, sourceType)
	_, complexOK := complexvalue.Describe(sourceType)
	_, primitiveOK := primitive(context, sourceType)
	_, _, structOK := namedStruct(sourceType)
	_, anonymousStructOK := isAnonymousStruct(sourceType)
	_, interfaceOK := interfacetype.Resolve(sourceType)
	if !definedOK &&
		!arrayOK &&
		!complexOK &&
		!primitiveOK &&
		!callableValue(sourceType) &&
		!pointerValue(sourceType) &&
		!channelValue(sourceType) &&
		!isScalarSlice(context, sourceType) &&
		!mapValue(context, sourceType) &&
		!interfaceOK &&
		!anonymousStructOK &&
		!structOK {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	return api.NewExpressionEmission(
		value.Before(),
		context.Factory().BinaryExpression(
			nil,
			target,
			nil,
			context.Factory().BinaryOperatorToken(
				tsgo.BinaryOperatorEqualsToken,
			),
			value.Value(),
		),
		value.Requests(),
	)
}

func (owner Owner) AssignStable(
	context api.Context,
	source ast.Node,
	sourceType types.Type,
	target tsgo.Expression,
	value api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	if _, _, ok := namedStruct(sourceType); !ok {
		return api.ExpressionEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "stable-identity assignment requires a named struct",
		}
	}
	assigned, err := owner.namedStructOperation(
		context,
		source,
		sourceType,
		api.NamedStructOperationAssign,
		[]tsgo.Expression{target, value.Value()},
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.NewExpressionEmission(
		value.Before(),
		assigned.Value(),
		api.CombineRequests(value.Requests(), assigned.Requests()),
	)
}
