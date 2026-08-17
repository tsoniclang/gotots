package reflectiontype

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	basictype "github.com/tsoniclang/gotots/internal/emit/type/basic"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

// valueOperationCallback is one generated typed value callback: the object
// literal property name, the provider-facing result primitive, and whether
// the projected payload converts to the provider 64-bit carrier.
type valueOperationCallback struct {
	name      string
	result    api.PrimitiveAlias
	toBigInt  bool
	nilCheck  bool
	operation string
}

// valueOperationsStatement builds the generated value-operation
// registration for one reflection-visible type when its value facet is
// demanded. Every callback performs the exact generated adapter guard and
// projects the represented payload; message strings never select behavior.
func valueOperationsStatement(
	context api.Context,
	names api.ReflectionNames,
	operations api.NameReference,
	reflectionType *types.TypeName,
	targetType api.NameReference,
	sourceType types.Type,
	descriptorName string,
) (tsgo.Statement, []api.RootRequest, bool, error) {
	if structType, ok := types.Unalias(sourceType).Underlying().(*types.Struct); ok {
		return structValueOperationsStatement(
			context,
			names,
			operations,
			reflectionType,
			targetType,
			sourceType,
			descriptorName,
			structType,
		)
	}
	if pointerType, ok := types.Unalias(sourceType).Underlying().(*types.Pointer); ok {
		return pointerValueOperationsStatement(
			context,
			names,
			operations,
			reflectionType,
			targetType,
			sourceType,
			descriptorName,
			pointerType,
		)
	}
	scalarContext, err := scalarOperationContext(context, sourceType)
	if err != nil {
		return nil, nil, false, err
	}
	callbacks, _, err := selectValueCallbacks(scalarContext, sourceType)
	if err != nil {
		return nil, nil, false, err
	}
	factory := context.Factory()
	_, interfaceSource := types.Unalias(sourceType).Underlying().(*types.Interface)
	var adapter api.NameReference
	if !interfaceSource {
		adapter, err = context.Names().InterfaceAdapter(sourceType, nil)
		if err != nil {
			return nil, nil, false, err
		}
	}
	boxType, err := context.Names().Runtime(
		api.RuntimeInterfaceValue,
		api.ImportPhaseType,
	)
	if err != nil {
		return nil, nil, false, err
	}
	requests := boxType.Requests()
	var panicReference api.NameReference
	if !interfaceSource {
		panicReference, err = context.Names().Runtime(
			api.RuntimePanic,
			api.ImportPhaseValue,
		)
		if err != nil {
			return nil, nil, false, err
		}
		requests = api.CombineRequests(
			adapter.Requests(),
			boxType.Requests(),
			panicReference.Requests(),
		)
	}
	properties := make([]tsgo.ObjectLiteralElementLike, 0, len(callbacks))
	for _, callback := range callbacks {
		var resultType tsgo.TypeNode
		if callback.nilCheck {
			resultType = factory.KeywordTypeNode(
				tsgo.KeywordTypeSyntaxKindBooleanKeyword,
			)
		} else {
			result, resultErr := context.Names().ProviderPrimitive(
				callback.result,
			)
			if resultErr != nil {
				return nil, nil, false, resultErr
			}
			requests = append(requests, result.Requests()...)
			resultType = factory.TypeReferenceNode(
				result.EntityName(factory),
				nil,
			)
		}
		payload, payloadRequests, payloadErr := projectedScalarPayload(
			context,
			sourceType,
			factory.PropertyAccessExpression(
				factory.Identifier("box"),
				nil,
				factory.Identifier("$go$value"),
				tsgo.NodeFlagsNone,
			),
		)
		if payloadErr != nil {
			return nil, nil, false, payloadErr
		}
		requests = append(requests, payloadRequests...)
		var projected tsgo.Expression
		switch {
		case callback.nilCheck:
			projected = factory.BinaryExpression(
				nil,
				payload,
				nil,
				factory.BinaryOperatorToken(
					tsgo.BinaryOperatorEqualsEqualsEqualsToken,
				),
				factory.Identifier("undefined"),
			)
		case callback.toBigInt:
			projected = factory.CallExpression(
				factory.PropertyAccessExpression(
					factory.Identifier("globalThis"),
					nil,
					factory.Identifier("BigInt"),
					tsgo.NodeFlagsNone,
				),
				nil,
				nil,
				[]tsgo.Expression{payload},
				tsgo.NodeFlagsNone,
			)
		default:
			projected = payload
		}
		guarded := factory.ConditionalExpression(
			factory.CallExpression(
				factory.PropertyAccessExpression(
					adapter.Expression(factory),
					nil,
					factory.Identifier("$is"),
					tsgo.NodeFlagsNone,
				),
				nil,
				nil,
				[]tsgo.Expression{factory.Identifier("box")},
				tsgo.NodeFlagsNone,
			),
			factory.QuestionToken(),
			projected,
			factory.ColonToken(),
			factory.CallExpression(
				factory.PropertyAccessExpression(
					panicReference.Expression(factory),
					nil,
					factory.Identifier("raiseRuntime"),
					tsgo.NodeFlagsNone,
				),
				nil,
				nil,
				[]tsgo.Expression{factory.StringLiteral(
					"reflect: "+callback.operation+
						" received a foreign interface box",
					tsgo.TokenFlagsNone,
				)},
				tsgo.NodeFlagsNone,
			),
		)
		arrow := factory.ArrowFunction(
			nil,
			nil,
			[]tsgo.ParameterDeclaration{factory.ParameterDeclaration(
				nil,
				nil,
				factory.Identifier("box"),
				nil,
				factory.TypeReferenceNode(
					boxType.EntityName(factory),
					nil,
				),
				nil,
			)},
			resultType,
			factory.EqualsGreaterThanToken(),
			guarded,
		)
		properties = append(properties, factory.PropertyAssignment(
			nil,
			factory.Identifier(callback.name),
			nil,
			factory.FunctionTypeNode(
				nil,
				[]tsgo.ParameterDeclaration{factory.ParameterDeclaration(
					nil,
					nil,
					factory.Identifier("box"),
					nil,
					factory.TypeReferenceNode(
						boxType.EntityName(factory),
						nil,
					),
					nil,
				)},
				resultType,
			),
			arrow,
		))
	}
	scaffold := &locationScaffold{
		factory:    factory,
		adapter:    adapter,
		boxType:    boxType,
		panicRef:   panicReference,
		targetType: targetType,
	}
	extended, err := extendedValueProperties(
		context,
		names,
		reflectionType,
		sourceType,
		scaffold,
	)
	if err != nil {
		return nil, nil, false, err
	}
	properties = append(properties, extended...)
	requests = append(requests, scaffold.requests...)
	if len(properties) == 0 {
		return nil, nil, false, nil
	}
	statement := factory.ExpressionStatement(factory.CallExpression(
		factory.PropertyAccessExpression(
			operations.Expression(factory),
			nil,
			factory.Identifier("$registerValue"),
			tsgo.NodeFlagsNone,
		),
		nil,
		nil,
		[]tsgo.Expression{
			factory.Identifier(descriptorName),
			deferred(
				factory,
				factory.ObjectLiteralExpression(properties, true),
			),
		},
		tsgo.NodeFlagsNone,
	))
	return statement, requests, true, nil
}

// selectValueCallbacks derives the exact callback set admitted by one
// type's kind. Only plain basic scalars and pointers participate in the
// scalar slice; later slices add aggregate, slice, and map callbacks.
func selectValueCallbacks(
	context api.Context,
	sourceType types.Type,
) ([]valueOperationCallback, bool, error) {
	switch selected := types.Unalias(sourceType).Underlying().(type) {
	case *types.Basic:
		provider, ok := context.ProviderScalarABI()
		if !ok {
			return nil, false, &api.GeneratedArtifactShapeError{
				Artifact: selected.String(),
				Reason:   "reflection value provider scalar ABI is absent",
			}
		}
		info := selected.Info()
		switch {
		case info&types.IsBoolean != 0:
			return []valueOperationCallback{{
				name:      "bool",
				result:    api.PrimitiveBool,
				operation: "Value.Bool",
			}}, true, nil
		case info&types.IsString != 0:
			return []valueOperationCallback{{
				name:      "string",
				result:    api.PrimitiveString,
				operation: "Value.String",
			}}, true, nil
		case info&types.IsUnsigned != 0:
			toBigInt, err := carrierWidens(
				context,
				provider,
				sourceType,
				api.PrimitiveUint64,
			)
			if err != nil {
				return nil, false, err
			}
			return []valueOperationCallback{{
				name:      "uint",
				result:    api.PrimitiveUint64,
				toBigInt:  toBigInt,
				operation: "Value.Uint",
			}}, true, nil
		case info&types.IsInteger != 0:
			toBigInt, err := carrierWidens(
				context,
				provider,
				sourceType,
				api.PrimitiveInt64,
			)
			if err != nil {
				return nil, false, err
			}
			return []valueOperationCallback{{
				name:      "int",
				result:    api.PrimitiveInt64,
				toBigInt:  toBigInt,
				operation: "Value.Int",
			}}, true, nil
		case info&types.IsFloat != 0:
			return []valueOperationCallback{{
				name:      "float",
				result:    api.PrimitiveFloat64,
				operation: "Value.Float",
			}}, true, nil
		default:
			return nil, false, nil
		}
	case *types.Pointer:
		return []valueOperationCallback{{
			name:      "isNil",
			nilCheck:  true,
			operation: "Value.IsNil",
		}}, true, nil
	default:
		return nil, false, nil
	}
}

// carrierWidens reports whether projecting one scalar payload to the
// provider 64-bit carrier requires the BigInt widening conversion.
func carrierWidens(
	context api.Context,
	provider api.ScalarABI,
	sourceType types.Type,
	target api.PrimitiveAlias,
) (bool, error) {
	targetCarrier, err := provider.Carrier(target)
	if err != nil {
		return false, err
	}
	basic, basicOK := types.Unalias(sourceType).Underlying().(*types.Basic)
	if !basicOK {
		return false, &api.GeneratedArtifactShapeError{
			Artifact: sourceType.String(),
			Reason:   "reflection value scalar has no basic underlying type",
		}
	}
	sourceAlias, ok := basictype.PrimitiveAlias(
		context.TypesSizes(),
		basic,
	)
	if !ok {
		return false, &api.GeneratedArtifactShapeError{
			Artifact: sourceType.String(),
			Reason:   "reflection value scalar has no primitive alias",
		}
	}
	sourceCarrier, err := context.ScalarABI().Carrier(sourceAlias)
	if err != nil {
		return false, err
	}
	return targetCarrier != sourceCarrier, nil
}

// projectedScalarPayload routes one boxed payload through the defined-type
// projection when the registered type carries a branded representation, so
// scalar callbacks always operate on the raw carrier.
func projectedScalarPayload(
	context api.Context,
	sourceType types.Type,
	payload tsgo.Expression,
) (tsgo.Expression, []api.RootRequest, error) {
	model, defined := definedtype.Resolve(sourceType)
	if !defined {
		return payload, nil, nil
	}
	projected, err := model.Project(
		context,
		api.DirectExpression(payload),
	)
	if err != nil {
		return nil, nil, err
	}
	return projected.Value(), projected.Requests(), nil
}

// constructedScalarValue wraps one raw carrier value back into the
// registered type's branded representation when one exists.
func constructedScalarValue(
	context api.Context,
	sourceType types.Type,
	value tsgo.Expression,
) (tsgo.Expression, []api.RootRequest, error) {
	model, defined := definedtype.Resolve(sourceType)
	if !defined {
		return value, nil, nil
	}
	constructed, err := model.Construct(context, value)
	if err != nil {
		return nil, nil, err
	}
	return constructed.Value(), constructed.Requests(), nil
}
