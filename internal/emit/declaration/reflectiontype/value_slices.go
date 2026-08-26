package reflectiontype

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	runtimeslice "github.com/tsoniclang/gotots/internal/emit/runtime/slice"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func sliceValueProperties(
	context api.Context,
	names api.ReflectionNames,
	reflectionType *types.TypeName,
	sourceType types.Type,
	sliceType *types.Slice,
	scaffold *locationScaffold,
) ([]tsgo.ObjectLiteralElementLike, error) {
	factory := scaffold.factory
	payload, payloadRequests, err := projectedScalarPayload(
		context,
		sourceType,
		boxPayload(factory),
	)
	if err != nil {
		return nil, err
	}
	scaffold.requests = append(scaffold.requests, payloadRequests...)
	scaffold.payload = payload
	var wrapFailure error
	wrapSlice := func(raw tsgo.Expression) tsgo.Expression {
		wrapped, requests, wrapErr := constructedScalarValue(
			context,
			sourceType,
			raw,
		)
		if wrapErr != nil {
			if wrapFailure == nil {
				wrapFailure = wrapErr
			}
			return raw
		}
		scaffold.requests = append(scaffold.requests, requests...)
		return wrapped
	}
	provider, ok := context.ProviderScalarABI()
	if !ok {
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
	runtimeSlice, err := context.Names().Runtime(
		api.RuntimeSlice,
		api.ImportPhaseValue,
	)
	if err != nil {
		return nil, err
	}
	element, err := newReflectionSliceElement(
		context,
		names,
		reflectionType,
		sliceType.Elem(),
		scaffold,
	)
	if err != nil {
		return nil, err
	}
	scaffold.requests = append(scaffold.requests, indexType.Requests()...)
	scaffold.requests = append(scaffold.requests, runtimeSlice.Requests()...)

	properties := []tsgo.ObjectLiteralElementLike{
		runtimeNilCallback(scaffold),
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
	index, err := element.indexOperation(context, indexType, scaffold)
	if err != nil {
		return nil, err
	}
	properties = append(properties, expressionProperty(factory, "index", index))
	appendOperation, err := element.appendOperation(
		context,
		runtimeSlice,
		wrapSlice,
		scaffold,
	)
	if err != nil {
		return nil, err
	}
	properties = append(
		properties,
		expressionProperty(factory, "append", appendOperation),
	)
	makeSlice, err := element.makeOperation(
		context,
		indexType,
		runtimeSlice,
		wrapSlice,
		scaffold,
	)
	if err != nil {
		return nil, err
	}
	properties = append(
		properties,
		expressionProperty(factory, "makeSlice", makeSlice),
	)
	properties = append(properties, expressionProperty(
		factory,
		"resliced",
		sliceResliceOperation(indexType, wrapSlice, scaffold),
	))
	zero, err := sliceZeroOperation(context, sourceType, scaffold)
	if err != nil {
		return nil, err
	}
	properties = append(properties, expressionProperty(factory, "zero", zero))
	grown, err := element.growOperation(
		context,
		indexType,
		runtimeSlice,
		wrapSlice,
		scaffold,
	)
	if err != nil {
		return nil, err
	}
	properties = append(properties, expressionProperty(factory, "grown", grown))
	if basic, basicOK := types.Unalias(sliceType.Elem()).(*types.Basic); basicOK && basic.Kind() == types.Uint8 {
		byteProperties, byteErr := sliceByteOperations(
			context,
			runtimeSlice,
			wrapSlice,
			scaffold,
		)
		if byteErr != nil {
			return nil, byteErr
		}
		properties = append(properties, byteProperties...)
	}
	if wrapFailure != nil {
		return nil, wrapFailure
	}
	return properties, nil
}

func sliceResliceOperation(
	indexType api.NameReference,
	wrapSlice func(tsgo.Expression) tsgo.Expression,
	scaffold *locationScaffold,
) tsgo.ArrowFunction {
	factory := scaffold.factory
	resliced := factory.CallExpression(
		factory.PropertyAccessExpression(
			scaffoldPayload(scaffold),
			nil,
			factory.Identifier(runtimeslice.MemberName(runtimeslice.MemberSlice)),
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
	)
	return factory.ArrowFunction(
		nil,
		nil,
		[]tsgo.ParameterDeclaration{
			boxParameter(scaffold),
			factory.ParameterDeclaration(
				nil,
				nil,
				factory.Identifier("length"),
				nil,
				factory.TypeReferenceNode(indexType.EntityName(factory), nil),
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
				[]tsgo.Expression{wrapSlice(resliced)},
			),
		)),
	)
}

func sliceZeroOperation(
	context api.Context,
	sourceType types.Type,
	scaffold *locationScaffold,
) (tsgo.ArrowFunction, error) {
	zero, err := context.Values().Zero(
		context.WithRole(api.RoleDefinedValue),
		nil,
		sourceType,
	)
	if err != nil {
		return nil, err
	}
	scaffold.requests = append(scaffold.requests, zero.Requests()...)
	statements := append([]tsgo.Statement(nil), zero.Before()...)
	statements = append(statements, scaffold.factory.ReturnStatement(
		scaffold.factory.NewExpression(
			scaffold.adapter.Expression(scaffold.factory),
			nil,
			[]tsgo.Expression{zero.Value()},
		),
	))
	return scaffold.factory.ArrowFunction(
		nil,
		nil,
		nil,
		scaffold.factory.TypeReferenceNode(
			scaffold.boxType.EntityName(scaffold.factory),
			nil,
		),
		scaffold.factory.EqualsGreaterThanToken(),
		scaffold.factory.Block(statements, true),
	), nil
}

func sliceByteOperations(
	context api.Context,
	runtimeSlice api.NameReference,
	wrapSlice func(tsgo.Expression) tsgo.Expression,
	scaffold *locationScaffold,
) ([]tsgo.ObjectLiteralElementLike, error) {
	factory := scaffold.factory
	providerByte, err := context.Names().ProviderPrimitive(api.PrimitiveUint8)
	if err != nil {
		return nil, err
	}
	scaffold.requests = append(scaffold.requests, providerByte.Requests()...)
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
			scaffoldPayload(scaffold),
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
		factory.TypeReferenceNode(scaffold.boxType.EntityName(factory), nil),
		factory.EqualsGreaterThanToken(),
		factory.NewExpression(
			scaffold.adapter.Expression(factory),
			nil,
			[]tsgo.Expression{wrapSlice(factory.Identifier("value"))},
		),
	)
	return []tsgo.ObjectLiteralElementLike{
		expressionProperty(factory, "bytes", bytes),
		expressionProperty(factory, "boxBytes", boxBytes),
	}, nil
}
