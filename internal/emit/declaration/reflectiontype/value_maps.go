package reflectiontype

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

// mapValueProperties adds the len, mapIndex, mapStore, mapKeys, and
// makeMap callbacks of one map whose key and element are plain basic
// scalars: lookups project through the runtime map with exact key and
// element adapter guards, an absent key yields no box (the Go zero
// Value), storing an absent box deletes the key, and construction binds
// the runtime map factory to the canonical descriptor.
func mapValueProperties(
	context api.Context,
	names api.ReflectionNames,
	reflectionType *types.TypeName,
	mapType *types.Map,
	scaffold *locationScaffold,
) ([]tsgo.ObjectLiteralElementLike, error) {
	factory := scaffold.factory
	keyBasic, keyOK := types.Unalias(mapType.Key()).(*types.Basic)
	elementBasic, elementOK := types.Unalias(mapType.Elem()).(*types.Basic)
	supported := keyOK && elementOK &&
		keyBasic.Info()&(types.IsBoolean|types.IsString|
			types.IsInteger|types.IsFloat) != 0 &&
		elementBasic.Info()&(types.IsBoolean|types.IsString|
			types.IsInteger|types.IsFloat) != 0
	provider, providerOK := context.ProviderScalarABI()
	if !providerOK {
		return nil, &api.GeneratedArtifactShapeError{
			Artifact: mapType.String(),
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
	extentType, err := context.Names().ProviderPrimitive(api.PrimitiveInt64)
	if err != nil {
		return nil, err
	}
	lengthCall := factory.CallExpression(
		factory.PropertyAccessExpression(
			boxPayload(factory),
			nil,
			factory.Identifier("length"),
			tsgo.NodeFlagsNone,
		),
		nil,
		nil,
		nil,
		tsgo.NodeFlagsNone,
	)
	var lengthProjected tsgo.Expression = lengthCall
	if carrier == api.IntegerCarrierBigInt {
		lengthProjected = factory.CallExpression(
			factory.PropertyAccessExpression(
				factory.Identifier("globalThis"),
				nil,
				factory.Identifier("BigInt"),
				tsgo.NodeFlagsNone,
			),
			nil,
			nil,
			[]tsgo.Expression{lengthCall},
			tsgo.NodeFlagsNone,
		)
	}
	length := factory.ArrowFunction(
		nil,
		nil,
		[]tsgo.ParameterDeclaration{boxParameter(scaffold)},
		factory.TypeReferenceNode(extentType.EntityName(factory), nil),
		factory.EqualsGreaterThanToken(),
		factory.ParenthesizedExpression(guardedProjection(
			scaffold,
			"Value.Len",
			lengthProjected,
		)),
	)
	if !supported {
		// Key or element kinds sit outside the location model: the map
		// keeps exact nil and length evidence while entry navigation
		// stays a loud typed boundary through operation absence.
		scaffold.requests = append(scaffold.requests, extentType.Requests()...)
		return []tsgo.ObjectLiteralElementLike{
			runtimeNilCallback(scaffold),
			expressionProperty(factory, "len", length),
		}, nil
	}
	keyAdapter, err := context.Names().InterfaceAdapter(mapType.Key(), nil)
	if err != nil {
		return nil, err
	}
	elementAdapter, err := context.Names().InterfaceAdapter(
		mapType.Elem(),
		nil,
	)
	if err != nil {
		return nil, err
	}
	keyDescriptor, err := names.ReflectionValueType(
		mapType.Key(),
		reflectionType,
	)
	if err != nil {
		return nil, err
	}
	elementDescriptor, err := names.ReflectionValueType(
		mapType.Elem(),
		reflectionType,
	)
	if err != nil {
		return nil, err
	}
	runtimeMap, err := context.Names().Runtime(
		api.RuntimeMap,
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
			Artifact: mapType.String(),
			Reason:   "reflection value map element has no exact zero",
		}
	}
	scaffold.requests = append(scaffold.requests, extentType.Requests()...)
	scaffold.requests = append(scaffold.requests, keyAdapter.Requests()...)
	scaffold.requests = append(
		scaffold.requests,
		elementAdapter.Requests()...,
	)
	scaffold.requests = append(
		scaffold.requests,
		keyDescriptor.Requests()...,
	)
	scaffold.requests = append(
		scaffold.requests,
		elementDescriptor.Requests()...,
	)
	scaffold.requests = append(scaffold.requests, runtimeMap.Requests()...)
	keyParameter := factory.ParameterDeclaration(
		nil,
		nil,
		factory.Identifier("key"),
		nil,
		factory.TypeReferenceNode(
			scaffold.boxType.EntityName(factory),
			nil,
		),
		nil,
	)
	lookupCall := factory.CallExpression(
		factory.PropertyAccessExpression(
			factory.Identifier("instance"),
			nil,
			factory.Identifier("lookupOk"),
			tsgo.NodeFlagsNone,
		),
		nil,
		nil,
		[]tsgo.Expression{guardedForeignOperand(
			scaffold,
			keyAdapter,
			"key",
			"Value.MapIndex",
		)},
		tsgo.NodeFlagsNone,
	)
	mapIndexBody := factory.Block([]tsgo.Statement{
		foreignBoxGuardStatement(scaffold, "Value.MapIndex"),
		constStatement(factory, "instance", boxPayload(factory)),
		constStatement(factory, "entry", lookupCall),
		factory.ReturnStatement(factory.ConditionalExpression(
			factory.ElementAccessExpression(
				factory.Identifier("entry"),
				nil,
				factory.NumericLiteral("1", tsgo.TokenFlagsNone),
				tsgo.NodeFlagsNone,
			),
			factory.QuestionToken(),
			factory.NewExpression(
				elementAdapter.Expression(factory),
				nil,
				[]tsgo.Expression{factory.ElementAccessExpression(
					factory.Identifier("entry"),
					nil,
					factory.NumericLiteral("0", tsgo.TokenFlagsNone),
					tsgo.NodeFlagsNone,
				)},
			),
			factory.ColonToken(),
			factory.Identifier("undefined"),
		)),
	}, true)
	mapIndex := factory.ArrowFunction(
		nil,
		nil,
		[]tsgo.ParameterDeclaration{boxParameter(scaffold), keyParameter},
		nil,
		factory.EqualsGreaterThanToken(),
		mapIndexBody,
	)
	valueParameter := factory.ParameterDeclaration(
		nil,
		nil,
		factory.Identifier("value"),
		nil,
		factory.UnionTypeNode([]tsgo.TypeNode{
			factory.TypeReferenceNode(
				scaffold.boxType.EntityName(factory),
				nil,
			),
			factory.KeywordTypeNode(
				tsgo.KeywordTypeSyntaxKindUndefinedKeyword,
			),
		}),
		nil,
	)
	mapStoreBody := factory.Block([]tsgo.Statement{
		foreignBoxGuardStatement(scaffold, "Value.SetMapIndex"),
		constStatement(factory, "instance", boxPayload(factory)),
		constStatement(factory, "entry", guardedForeignOperand(
			scaffold,
			keyAdapter,
			"key",
			"Value.SetMapIndex",
		)),
		factory.IfStatement(
			factory.BinaryExpression(
				nil,
				factory.Identifier("value"),
				nil,
				factory.BinaryOperatorToken(
					tsgo.BinaryOperatorEqualsEqualsEqualsToken,
				),
				factory.Identifier("undefined"),
			),
			factory.Block([]tsgo.Statement{
				factory.ExpressionStatement(factory.CallExpression(
					factory.PropertyAccessExpression(
						factory.Identifier("instance"),
						nil,
						factory.Identifier("delete"),
						tsgo.NodeFlagsNone,
					),
					nil,
					nil,
					[]tsgo.Expression{factory.Identifier("entry")},
					tsgo.NodeFlagsNone,
				)),
				factory.ReturnStatement(nil),
			}, true),
			nil,
		),
		factory.ExpressionStatement(factory.CallExpression(
			factory.PropertyAccessExpression(
				factory.Identifier("instance"),
				nil,
				factory.Identifier("store"),
				tsgo.NodeFlagsNone,
			),
			nil,
			nil,
			[]tsgo.Expression{
				factory.Identifier("entry"),
				guardedForeignPayload(
					scaffold,
					elementAdapter,
					"Value.SetMapIndex",
				),
			},
			tsgo.NodeFlagsNone,
		)),
	}, true)
	mapStore := factory.ArrowFunction(
		nil,
		nil,
		[]tsgo.ParameterDeclaration{
			boxParameter(scaffold),
			keyParameter,
			valueParameter,
		},
		factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindVoidKeyword),
		factory.EqualsGreaterThanToken(),
		mapStoreBody,
	)
	mapKeys := factory.ArrowFunction(
		nil,
		nil,
		[]tsgo.ParameterDeclaration{boxParameter(scaffold)},
		nil,
		factory.EqualsGreaterThanToken(),
		factory.ParenthesizedExpression(guardedProjection(
			scaffold,
			"Value.MapRange",
			factory.CallExpression(
				factory.PropertyAccessExpression(
					factory.CallExpression(
						factory.PropertyAccessExpression(
							boxPayload(factory),
							nil,
							factory.Identifier("keys"),
							tsgo.NodeFlagsNone,
						),
						nil,
						nil,
						nil,
						tsgo.NodeFlagsNone,
					),
					nil,
					factory.Identifier("map"),
					tsgo.NodeFlagsNone,
				),
				nil,
				nil,
				[]tsgo.Expression{factory.ArrowFunction(
					nil,
					nil,
					[]tsgo.ParameterDeclaration{
						factory.ParameterDeclaration(
							nil,
							nil,
							factory.Identifier("key"),
							nil,
							nil,
							nil,
						),
					},
					nil,
					factory.EqualsGreaterThanToken(),
					factory.ParenthesizedExpression(
						factory.NewExpression(
							keyAdapter.Expression(factory),
							nil,
							[]tsgo.Expression{
								factory.Identifier("key"),
							},
						),
					),
				)},
				tsgo.NodeFlagsNone,
			),
		)),
	)
	makeMap := factory.ArrowFunction(
		nil,
		nil,
		nil,
		nil,
		factory.EqualsGreaterThanToken(),
		factory.ParenthesizedExpression(factory.NewExpression(
			scaffold.adapter.Expression(factory),
			nil,
			[]tsgo.Expression{factory.CallExpression(
				factory.PropertyAccessExpression(
					runtimeMap.Expression(factory),
					nil,
					factory.Identifier("make"),
					tsgo.NodeFlagsNone,
				),
				nil,
				nil,
				[]tsgo.Expression{
					elementZero,
					factory.NumericLiteral("0", tsgo.TokenFlagsNone),
					factory.ArrayLiteralExpression(nil, false),
				},
				tsgo.NodeFlagsNone,
			)},
		)),
	)
	zero := factory.ArrowFunction(
		nil,
		nil,
		nil,
		factory.TypeReferenceNode(
			scaffold.boxType.EntityName(factory),
			nil,
		),
		factory.EqualsGreaterThanToken(),
		factory.ParenthesizedExpression(factory.NewExpression(
			scaffold.adapter.Expression(factory),
			nil,
			[]tsgo.Expression{factory.CallExpression(
				factory.PropertyAccessExpression(
					runtimeMap.Expression(factory),
					nil,
					factory.Identifier("nil"),
					tsgo.NodeFlagsNone,
				),
				nil,
				nil,
				[]tsgo.Expression{elementZero},
				tsgo.NodeFlagsNone,
			)},
		)),
	)
	return []tsgo.ObjectLiteralElementLike{
		runtimeNilCallback(scaffold),
		expressionProperty(factory, "zero", zero),
		expressionProperty(factory, "len", length),
		expressionProperty(factory, "mapIndex", mapIndex),
		expressionProperty(factory, "mapStore", mapStore),
		expressionProperty(factory, "mapKeys", mapKeys),
		expressionProperty(factory, "makeMap", makeMap),
	}, nil
}
