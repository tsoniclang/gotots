package reflectiontype

import (
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

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
		scaffoldPayload(scaffold),
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

// runtimeNilCallback projects the represented container's own nil
// evidence: runtime slices and maps both carry an exact isNil method.
func runtimeNilCallback(
	scaffold *locationScaffold,
) tsgo.ObjectLiteralElementLike {
	factory := scaffold.factory
	return expressionProperty(factory, "isNil", factory.ArrowFunction(
		nil,
		nil,
		[]tsgo.ParameterDeclaration{boxParameter(scaffold)},
		factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindBooleanKeyword),
		factory.EqualsGreaterThanToken(),
		factory.ParenthesizedExpression(guardedProjection(
			scaffold,
			"Value.IsNil",
			factory.CallExpression(
				factory.PropertyAccessExpression(
					scaffoldPayload(scaffold),
					nil,
					factory.Identifier("isNil"),
					tsgo.NodeFlagsNone,
				),
				nil,
				nil,
				nil,
				tsgo.NodeFlagsNone,
			),
		)),
	))
}

// reducedSliceProperties is the exact evidence subset of one slice whose
// element sits outside the location model: nil, length, and capacity.
func reducedSliceProperties(
	scaffold *locationScaffold,
	carrier api.IntegerCarrier,
	indexType api.NameReference,
) []tsgo.ObjectLiteralElementLike {
	factory := scaffold.factory
	return []tsgo.ObjectLiteralElementLike{
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
}
