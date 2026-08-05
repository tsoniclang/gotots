package reflectiontype

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	basictype "github.com/tsoniclang/gotots/internal/emit/type/basic"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

// sliceValueProperties adds the len, cap, index, append, and makeSlice
// callbacks of one slice whose element is a plain basic scalar: element
// locations alias the represented backing array through the runtime slice
// accessors, append delegates to the runtime append with the exact element
// zero, and makeSlice constructs the runtime slice for the canonical
// descriptor.
func sliceValueProperties(
	context api.Context,
	names api.ReflectionNames,
	reflectionType *types.TypeName,
	sliceType *types.Slice,
	scaffold *locationScaffold,
) ([]tsgo.ObjectLiteralElementLike, error) {
	factory := scaffold.factory
	elementType := sliceType.Elem()
	elementBasic, ok := types.Unalias(elementType).(*types.Basic)
	if !ok || elementBasic.Info()&(types.IsBoolean|types.IsString|
		types.IsInteger|types.IsFloat) == 0 {
		return nil, &api.GeneratedArtifactShapeError{
			Artifact: sliceType.String(),
			Reason: "reflection value slice element type " +
				elementType.String() +
				" is outside the supported scalar location model",
		}
	}
	provider, providerOK := context.ProviderScalarABI()
	if !providerOK {
		return nil, &api.GeneratedArtifactShapeError{
			Artifact: sliceType.String(),
			Reason:   "reflection value provider scalar ABI is absent",
		}
	}
	carrier, err := api.IntegerCarrierRepresentation(
		api.PrimitiveInt64,
		provider,
	)
	if err != nil {
		return nil, err
	}
	indexType, err := context.Names().ProviderPrimitive(api.PrimitiveInt64)
	if err != nil {
		return nil, err
	}
	elementAdapter, err := context.Names().InterfaceAdapter(elementType, nil)
	if err != nil {
		return nil, err
	}
	descriptor, err := names.ReflectionValueType(elementType, reflectionType)
	if err != nil {
		return nil, err
	}
	runtimeSlice, err := context.Names().Runtime(
		api.RuntimeSlice,
		api.ImportPhaseValue,
	)
	if err != nil {
		return nil, err
	}
	elementZero, err := scalarZeroExpression(context, factory, elementBasic)
	if err != nil {
		return nil, err
	}
	if elementZero == nil {
		return nil, &api.GeneratedArtifactShapeError{
			Artifact: sliceType.String(),
			Reason:   "reflection value slice element has no exact zero",
		}
	}
	scaffold.requests = append(scaffold.requests, indexType.Requests()...)
	scaffold.requests = append(
		scaffold.requests,
		elementAdapter.Requests()...,
	)
	scaffold.requests = append(scaffold.requests, descriptor.Requests()...)
	scaffold.requests = append(scaffold.requests, runtimeSlice.Requests()...)
	properties := []tsgo.ObjectLiteralElementLike{
		expressionProperty(factory, "len", sliceExtentCallback(
			scaffold,
			"length",
			carrier,
			indexType,
			"Value.Len",
		)),
		expressionProperty(factory, "cap", sliceExtentCallback(
			scaffold,
			"capacity",
			carrier,
			indexType,
			"Value.Cap",
		)),
	}
	elementGet := factory.CallExpression(
		memberAccess(factory, "instance", "get"),
		nil,
		nil,
		[]tsgo.Expression{factory.Identifier("index")},
		tsgo.NodeFlagsNone,
	)
	location := locationLiteral(scaffold, locationCallbacks{
		descriptor: descriptor,
		settable:   true,
		get: factory.NewExpression(
			elementAdapter.Expression(factory),
			nil,
			[]tsgo.Expression{elementGet},
		),
		set: factory.Block([]tsgo.Statement{
			factory.ExpressionStatement(factory.CallExpression(
				memberAccess(factory, "instance", "set"),
				nil,
				nil,
				[]tsgo.Expression{
					factory.Identifier("index"),
					guardedForeignPayload(
						scaffold,
						elementAdapter,
						"Value.Set",
					),
				},
				tsgo.NodeFlagsNone,
			)),
		}, true),
	})
	indexBody := factory.Block([]tsgo.Statement{
		foreignBoxGuardStatement(scaffold, "Value.Index"),
		constStatement(factory, "instance", boxPayload(factory)),
		factory.ReturnStatement(location),
	}, true)
	properties = append(properties, expressionProperty(
		factory,
		"index",
		factory.ArrowFunction(
			nil,
			nil,
			[]tsgo.ParameterDeclaration{
				boxParameter(scaffold),
				factory.ParameterDeclaration(
					nil,
					nil,
					factory.Identifier("index"),
					nil,
					factory.TypeReferenceNode(
						indexType.EntityName(factory),
						nil,
					),
					nil,
				),
			},
			nil,
			factory.EqualsGreaterThanToken(),
			indexBody,
		),
	))
	appendBody := factory.Block([]tsgo.Statement{
		foreignBoxGuardStatement(scaffold, "Value.Append"),
		constStatement(factory, "instance", boxPayload(factory)),
		factory.ReturnStatement(factory.NewExpression(
			scaffold.adapter.Expression(factory),
			nil,
			[]tsgo.Expression{factory.CallExpression(
				memberAccess(factory, "instance", "append"),
				nil,
				nil,
				[]tsgo.Expression{
					elementZero,
					factory.CallExpression(
						memberAccess(factory, "values", "map"),
						nil,
						nil,
						[]tsgo.Expression{factory.ArrowFunction(
							nil,
							nil,
							[]tsgo.ParameterDeclaration{
								factory.ParameterDeclaration(
									nil,
									nil,
									factory.Identifier("value"),
									nil,
									factory.TypeReferenceNode(
										scaffold.boxType.EntityName(
											factory,
										),
										nil,
									),
									nil,
								),
							},
							nil,
							factory.EqualsGreaterThanToken(),
							factory.ParenthesizedExpression(
								guardedForeignPayload(
									scaffold,
									elementAdapter,
									"Value.Append",
								),
							),
						)},
						tsgo.NodeFlagsNone,
					),
				},
				tsgo.NodeFlagsNone,
			)},
		)),
	}, true)
	properties = append(properties, expressionProperty(
		factory,
		"append",
		factory.ArrowFunction(
			nil,
			nil,
			[]tsgo.ParameterDeclaration{
				boxParameter(scaffold),
				factory.ParameterDeclaration(
					nil,
					nil,
					factory.Identifier("values"),
					nil,
					factory.TypeOperatorNode(
						tsgo.TypeOperatorNodeOperatorKindReadonlyKeyword,
						factory.ArrayTypeNode(factory.TypeReferenceNode(
							scaffold.boxType.EntityName(factory),
							nil,
						)),
					),
					nil,
				),
			},
			nil,
			factory.EqualsGreaterThanToken(),
			appendBody,
		),
	))
	properties = append(properties, expressionProperty(
		factory,
		"makeSlice",
		factory.ArrowFunction(
			nil,
			nil,
			[]tsgo.ParameterDeclaration{
				factory.ParameterDeclaration(
					nil,
					nil,
					factory.Identifier("length"),
					nil,
					factory.TypeReferenceNode(
						indexType.EntityName(factory),
						nil,
					),
					nil,
				),
				factory.ParameterDeclaration(
					nil,
					nil,
					factory.Identifier("capacity"),
					nil,
					factory.TypeReferenceNode(
						indexType.EntityName(factory),
						nil,
					),
					nil,
				),
			},
			nil,
			factory.EqualsGreaterThanToken(),
			factory.ParenthesizedExpression(factory.NewExpression(
				scaffold.adapter.Expression(factory),
				nil,
				[]tsgo.Expression{factory.CallExpression(
					factory.PropertyAccessExpression(
						runtimeSlice.Expression(factory),
						nil,
						factory.Identifier("make"),
						tsgo.NodeFlagsNone,
					),
					nil,
					nil,
					[]tsgo.Expression{
						factory.Identifier("length"),
						factory.Identifier("capacity"),
						elementZero,
					},
					tsgo.NodeFlagsNone,
				)},
			)),
		),
	))
	resliced := factory.ArrowFunction(
		nil,
		nil,
		[]tsgo.ParameterDeclaration{
			boxParameter(scaffold),
			factory.ParameterDeclaration(
				nil,
				nil,
				factory.Identifier("length"),
				nil,
				factory.TypeReferenceNode(
					indexType.EntityName(factory),
					nil,
				),
				nil,
			),
		},
		nil,
		factory.EqualsGreaterThanToken(),
		factory.ParenthesizedExpression(guardedProjection(
			scaffold,
			"Value.SetLen",
			factory.NewExpression(
				scaffold.adapter.Expression(factory),
				nil,
				[]tsgo.Expression{factory.CallExpression(
					factory.PropertyAccessExpression(
						boxPayload(factory),
						nil,
						factory.Identifier("slice"),
						tsgo.NodeFlagsNone,
					),
					nil,
					nil,
					[]tsgo.Expression{
						factory.NumericLiteral("0", tsgo.TokenFlagsNone),
						factory.Identifier("length"),
						factory.NullLiteral(),
					},
					tsgo.NodeFlagsNone,
				)},
			),
		)),
	)
	properties = append(
		properties,
		expressionProperty(factory, "resliced", resliced),
	)
	elementAlias, aliasOK := basictype.PrimitiveAlias(
		context.TypesSizes(),
		elementBasic,
	)
	if !aliasOK {
		return nil, &api.GeneratedArtifactShapeError{
			Artifact: sliceType.String(),
			Reason:   "reflection value slice element has no primitive alias",
		}
	}
	elementProduct, err := context.Names().Primitive(elementAlias)
	if err != nil {
		return nil, err
	}
	scaffold.requests = append(scaffold.requests, elementProduct.Requests()...)
	filler := factory.CallExpression(
		factory.PropertyAccessExpression(
			factory.NewExpression(
				factory.PropertyAccessExpression(
					factory.Identifier("globalThis"),
					nil,
					factory.Identifier("Array"),
					tsgo.NodeFlagsNone,
				),
				[]tsgo.TypeNode{factory.TypeReferenceNode(
					elementProduct.EntityName(factory),
					nil,
				)},
				[]tsgo.Expression{factory.CallExpression(
					factory.PropertyAccessExpression(
						factory.Identifier("globalThis"),
						nil,
						factory.Identifier("Number"),
						tsgo.NodeFlagsNone,
					),
					nil,
					nil,
					[]tsgo.Expression{factory.Identifier("count")},
					tsgo.NodeFlagsNone,
				)},
			),
			nil,
			factory.Identifier("fill"),
			tsgo.NodeFlagsNone,
		),
		nil,
		nil,
		[]tsgo.Expression{elementZero},
		tsgo.NodeFlagsNone,
	)
	grownSlice := factory.CallExpression(
		factory.PropertyAccessExpression(
			factory.CallExpression(
				factory.PropertyAccessExpression(
					boxPayload(factory),
					nil,
					factory.Identifier("append"),
					tsgo.NodeFlagsNone,
				),
				nil,
				nil,
				[]tsgo.Expression{elementZero, filler},
				tsgo.NodeFlagsNone,
			),
			nil,
			factory.Identifier("slice"),
			tsgo.NodeFlagsNone,
		),
		nil,
		nil,
		[]tsgo.Expression{
			factory.NumericLiteral("0", tsgo.TokenFlagsNone),
			factory.PropertyAccessExpression(
				boxPayload(factory),
				nil,
				factory.Identifier("length"),
				tsgo.NodeFlagsNone,
			),
			factory.NullLiteral(),
		},
		tsgo.NodeFlagsNone,
	)
	grown := factory.ArrowFunction(
		nil,
		nil,
		[]tsgo.ParameterDeclaration{
			boxParameter(scaffold),
			factory.ParameterDeclaration(
				nil,
				nil,
				factory.Identifier("count"),
				nil,
				factory.TypeReferenceNode(
					indexType.EntityName(factory),
					nil,
				),
				nil,
			),
		},
		nil,
		factory.EqualsGreaterThanToken(),
		factory.ParenthesizedExpression(guardedProjection(
			scaffold,
			"Value.Grow",
			factory.NewExpression(
				scaffold.adapter.Expression(factory),
				nil,
				[]tsgo.Expression{grownSlice},
			),
		)),
	)
	properties = append(
		properties,
		expressionProperty(factory, "grown", grown),
	)
	if elementBasic.Kind() == types.Uint8 {
		providerByte, byteErr := context.Names().ProviderPrimitive(
			api.PrimitiveUint8,
		)
		if byteErr != nil {
			return nil, byteErr
		}
		scaffold.requests = append(
			scaffold.requests,
			providerByte.Requests()...,
		)
		byteSliceType := factory.TypeReferenceNode(
			runtimeSlice.EntityName(factory),
			[]tsgo.TypeNode{factory.TypeReferenceNode(
				providerByte.EntityName(factory),
				nil,
			)},
		)
		bytes := factory.ArrowFunction(
			nil,
			nil,
			[]tsgo.ParameterDeclaration{boxParameter(scaffold)},
			byteSliceType,
			factory.EqualsGreaterThanToken(),
			factory.ParenthesizedExpression(guardedProjection(
				scaffold,
				"Value.Bytes",
				boxPayload(factory),
			)),
		)
		boxBytes := factory.ArrowFunction(
			nil,
			nil,
			[]tsgo.ParameterDeclaration{factory.ParameterDeclaration(
				nil,
				nil,
				factory.Identifier("value"),
				nil,
				byteSliceType,
				nil,
			)},
			factory.TypeReferenceNode(
				scaffold.boxType.EntityName(factory),
				nil,
			),
			factory.EqualsGreaterThanToken(),
			factory.NewExpression(
				scaffold.adapter.Expression(factory),
				nil,
				[]tsgo.Expression{factory.Identifier("value")},
			),
		)
		properties = append(
			properties,
			expressionProperty(factory, "bytes", bytes),
			expressionProperty(factory, "boxBytes", boxBytes),
		)
	}
	return properties, nil
}

// sliceExtentCallback projects one runtime slice extent field to the
// provider 64-bit carrier with the exact widening the provider scalar
// representation requires.
func sliceExtentCallback(
	scaffold *locationScaffold,
	member string,
	carrier api.IntegerCarrier,
	resultType api.NameReference,
	operation string,
) tsgo.Expression {
	factory := scaffold.factory
	var projected tsgo.Expression = factory.PropertyAccessExpression(
		boxPayload(factory),
		nil,
		factory.Identifier(member),
		tsgo.NodeFlagsNone,
	)
	if carrier == api.IntegerCarrierBigInt {
		projected = factory.CallExpression(
			factory.PropertyAccessExpression(
				factory.Identifier("globalThis"),
				nil,
				factory.Identifier("BigInt"),
				tsgo.NodeFlagsNone,
			),
			nil,
			nil,
			[]tsgo.Expression{projected},
			tsgo.NodeFlagsNone,
		)
	}
	return factory.ArrowFunction(
		nil,
		nil,
		[]tsgo.ParameterDeclaration{boxParameter(scaffold)},
		factory.TypeReferenceNode(resultType.EntityName(factory), nil),
		factory.EqualsGreaterThanToken(),
		factory.ParenthesizedExpression(guardedProjection(
			scaffold,
			operation,
			projected,
		)),
	)
}
