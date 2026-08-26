package reflectiontype

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/value/maprepresentation"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

// mapValueProperties adds the complete map callback family through the
// canonical map representation. The representation owns key and value copy
// semantics; reflection owns only guarded boxing and operation topology.
func mapValueProperties(
	context api.Context,
	sourceType types.Type,
	mapType *types.Map,
	scaffold *locationScaffold,
) ([]tsgo.ObjectLiteralElementLike, error) {
	factory := scaffold.factory
	payload, payloadRequests, err := projectedScalarPayload(
		context,
		sourceType,
		boxPayload(scaffold.factory),
	)
	if err != nil {
		return nil, err
	}
	scaffold.requests = append(scaffold.requests, payloadRequests...)
	scaffold.payload = payload
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
			scaffoldPayload(scaffold),
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
	keyMember, err := newReflectionMemberBox(context, mapType.Key())
	if err != nil {
		return nil, err
	}
	elementMember, err := newReflectionMemberBox(context, mapType.Elem())
	if err != nil {
		return nil, err
	}
	elementZero, err := context.Values().Zero(
		context.WithRole(api.RoleMapValue),
		nil,
		mapType.Elem(),
	)
	if err != nil {
		return nil, err
	}
	scaffold.requests = append(scaffold.requests, extentType.Requests()...)
	scaffold.requests = append(scaffold.requests, keyMember.requests()...)
	scaffold.requests = append(scaffold.requests, elementMember.requests()...)
	scaffold.requests = append(scaffold.requests, elementZero.Requests()...)
	keyParameter := factory.ParameterDeclaration(
		nil,
		nil,
		factory.Identifier("key"),
		nil,
		optionalInterfaceBoxType(factory, scaffold.boxType),
		nil,
	)
	indexKey, err := keyMember.admittedValue(
		context,
		"key",
		"Value.MapIndex",
		scaffold,
	)
	if err != nil {
		return nil, err
	}
	lookupCall := factory.CallExpression(
		factory.PropertyAccessExpression(
			factory.Identifier("instance"),
			nil,
			factory.Identifier("lookupOk"),
			tsgo.NodeFlagsNone,
		),
		nil,
		nil,
		[]tsgo.Expression{indexKey.Value()},
		tsgo.NodeFlagsNone,
	)
	entryValue := elementMember.boxedValue(
		factory,
		factory.ElementAccessExpression(
			factory.Identifier("entry"),
			nil,
			factory.NumericLiteral("0", tsgo.TokenFlagsNone),
			tsgo.NodeFlagsNone,
		),
	)
	mapIndexType := factory.TupleTypeNode([]tsgo.TypeNode{
		optionalInterfaceBoxType(factory, scaffold.boxType),
		factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindBooleanKeyword),
	})
	mapIndexBody := factory.Block([]tsgo.Statement{
		foreignBoxGuardStatement(scaffold, "Value.MapIndex"),
		constStatement(factory, "instance", scaffoldPayload(scaffold)),
		constStatement(factory, "entry", lookupCall),
		factory.ReturnStatement(factory.ConditionalExpression(
			factory.ElementAccessExpression(
				factory.Identifier("entry"),
				nil,
				factory.NumericLiteral("1", tsgo.TokenFlagsNone),
				tsgo.NodeFlagsNone,
			),
			factory.QuestionToken(),
			factory.ArrayLiteralExpression(
				[]tsgo.Expression{entryValue, factory.TrueLiteral()},
				false,
			),
			factory.ColonToken(),
			factory.ArrayLiteralExpression(
				[]tsgo.Expression{
					factory.Identifier("undefined"),
					factory.FalseLiteral(),
				},
				false,
			),
		)),
	}, true)
	mapIndex := factory.ArrowFunction(
		nil,
		nil,
		[]tsgo.ParameterDeclaration{boxParameter(scaffold), keyParameter},
		mapIndexType,
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
	deleteParameter := factory.ParameterDeclaration(
		nil,
		nil,
		factory.Identifier("deleteEntry"),
		nil,
		factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindBooleanKeyword),
		nil,
	)
	storeKey, err := keyMember.admittedValue(
		context,
		"key",
		"Value.SetMapIndex",
		scaffold,
	)
	if err != nil {
		return nil, err
	}
	storedElement, err := elementMember.admittedValue(
		context,
		"value",
		"Value.SetMapIndex",
		scaffold,
	)
	if err != nil {
		return nil, err
	}
	mapStoreBody := factory.Block([]tsgo.Statement{
		foreignBoxGuardStatement(scaffold, "Value.SetMapIndex"),
		constStatement(factory, "instance", scaffoldPayload(scaffold)),
		constStatement(factory, "entry", storeKey.Value()),
		factory.IfStatement(
			factory.Identifier("deleteEntry"),
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
				storedElement.Value(),
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
			deleteParameter,
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
							scaffoldPayload(scaffold),
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
						keyMember.boxedValue(
							factory,
							factory.Identifier("key"),
						),
					),
				)},
				tsgo.NodeFlagsNone,
			),
		)),
	)
	made, err := maprepresentation.Make(
		context.WithRole(api.RoleDefinedValue),
		nil,
		sourceType,
		elementZero.Value(),
		factory.NumericLiteral("0", tsgo.TokenFlagsNone),
		nil,
		elementZero.Requests(),
	)
	if err != nil {
		return nil, err
	}
	zeroValue, err := context.Values().Zero(
		context.WithRole(api.RoleDefinedValue),
		nil,
		sourceType,
	)
	if err != nil {
		return nil, err
	}
	scaffold.requests = append(scaffold.requests, made.Requests()...)
	scaffold.requests = append(scaffold.requests, zeroValue.Requests()...)
	scaffold.requests = append(scaffold.requests, indexKey.Requests()...)
	scaffold.requests = append(scaffold.requests, storeKey.Requests()...)
	scaffold.requests = append(scaffold.requests, storedElement.Requests()...)
	makeMap := reflectionMapBoxArrow(made, scaffold)
	zero := reflectionMapBoxArrow(zeroValue, scaffold)
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

func reflectionMapBoxArrow(
	value api.ExpressionEmission,
	scaffold *locationScaffold,
) tsgo.ArrowFunction {
	factory := scaffold.factory
	statements := append([]tsgo.Statement(nil), value.Before()...)
	statements = append(statements, factory.ReturnStatement(factory.NewExpression(
		scaffold.adapter.Expression(factory),
		nil,
		[]tsgo.Expression{value.Value()},
	)))
	return factory.ArrowFunction(
		nil,
		nil,
		nil,
		factory.TypeReferenceNode(scaffold.boxType.EntityName(factory), nil),
		factory.EqualsGreaterThanToken(),
		factory.Block(statements, true),
	)
}
