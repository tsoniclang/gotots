package pointer

import (
	panicruntime "github.com/tsoniclang/gotots/internal/emit/runtime/panic"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func (b builder) regionType(storage tsgo.TypeNode) tsgo.TypeNode {
	return b.factory.TypeOperatorNode(
		tsgo.TypeOperatorNodeOperatorKindReadonlyKeyword,
		b.factory.TupleTypeNode([]tsgo.TypeNode{
			b.factory.ArrayTypeNode(storage),
			b.numberType(),
		}),
	)
}

func (b builder) optionalRegionType(storage tsgo.TypeNode) tsgo.TypeNode {
	return b.factory.UnionTypeNode([]tsgo.TypeNode{
		b.regionType(storage),
		b.undefinedType(),
	})
}

func (b builder) regionMethod() tsgo.MethodDeclaration {
	pointer := b.id("pointer")
	requested := b.id("requested")
	region := b.id("region")
	return b.method(
		[]tsgo.ModifierLike{b.factory.StaticKeyword()},
		RegionMethod,
		[]tsgo.TypeParameterDeclaration{
			b.typeParameter("L", nil),
			b.typeParameter("S", nil),
		},
		[]tsgo.ParameterDeclaration{
			b.parameter("pointer", b.factory.UnionTypeNode([]tsgo.TypeNode{
				b.pointerType(b.typeL(), b.typeS()),
				b.undefinedType(),
			})),
			b.parameter("length", b.factory.UnionTypeNode([]tsgo.TypeNode{
				b.numberType(),
				b.factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindBigIntKeyword),
			})),
		},
		b.optionalRegionType(b.typeS()),
		b.variable(tsgo.NodeFlagsConst, "requested", nil, b.factory.CallExpression(
			b.factory.PropertyAccessExpression(
				b.id("globalThis"), nil, b.id("Number"), tsgo.NodeFlagsNone,
			), nil, nil, []tsgo.Expression{b.id("length")}, tsgo.NodeFlagsNone,
		)),
		b.factory.IfStatement(
			b.binary(requested, tsgo.BinaryOperatorLessThanToken, b.factory.NumericLiteral("0", tsgo.TokenFlagsNone)),
			b.factory.ExpressionStatement(panicruntime.Call(b.factory, b.panicName, b.factory.StringLiteral("unsafe length is negative", tsgo.TokenFlagsNone))),
			nil,
		),
		b.factory.IfStatement(
			b.binary(pointer, tsgo.BinaryOperatorEqualsEqualsEqualsToken, b.undefined()),
			b.factory.Block([]tsgo.Statement{
				b.factory.IfStatement(
					b.binary(requested, tsgo.BinaryOperatorEqualsEqualsEqualsToken, b.factory.NumericLiteral("0", tsgo.TokenFlagsNone)),
					b.factory.Block([]tsgo.Statement{b.factory.ReturnStatement(b.undefined())}, true),
					nil,
				),
				b.factory.ExpressionStatement(panicruntime.Call(
					b.factory, b.panicName, b.factory.StringLiteral("unsafe operation on nil pointer", tsgo.TokenFlagsNone),
				)),
			}, true),
			nil,
		),
		b.variable(tsgo.NodeFlagsConst, "region", nil, b.property(pointer, RegionName)),
		b.factory.IfStatement(
			b.binary(
				b.binary(region, tsgo.BinaryOperatorEqualsEqualsEqualsToken, b.undefined()),
				tsgo.BinaryOperatorBarBarToken,
				b.binary(
					requested,
					tsgo.BinaryOperatorGreaterThanToken,
					b.binary(
						b.property(b.factory.ElementAccessExpression(region, nil, b.factory.NumericLiteral("0", tsgo.TokenFlagsNone), tsgo.NodeFlagsNone), "length"),
						tsgo.BinaryOperatorMinusToken,
						b.factory.ElementAccessExpression(region, nil, b.factory.NumericLiteral("1", tsgo.TokenFlagsNone), tsgo.NodeFlagsNone),
					),
				),
			),
			b.factory.ExpressionStatement(panicruntime.Call(b.factory, b.panicName, b.factory.StringLiteral("unsafe operation requires a contiguous pointer region", tsgo.TokenFlagsNone))),
			nil,
		),
		b.factory.ReturnStatement(region),
	)
}

func Region(
	factory tsgo.Factory,
	functionName string,
	pointerName string,
) tsgo.FunctionDeclaration {
	b := builder{factory: factory, className: pointerName}
	return factory.FunctionDeclaration(
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		nil,
		factory.Identifier(functionName),
		[]tsgo.TypeParameterDeclaration{
			b.typeParameter("L", nil),
			b.typeParameter("S", nil),
		},
		[]tsgo.ParameterDeclaration{
			b.parameter("pointer", factory.UnionTypeNode([]tsgo.TypeNode{
				b.pointerType(b.typeL(), b.typeS()),
				b.undefinedType(),
			})),
			b.parameter("length", factory.UnionTypeNode([]tsgo.TypeNode{
				b.numberType(),
				factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindBigIntKeyword),
			})),
		},
		b.optionalRegionType(b.typeS()),
		factory.Block([]tsgo.Statement{factory.ReturnStatement(
			factory.CallExpression(
				factory.PropertyAccessExpression(factory.Identifier(pointerName), nil, factory.Identifier(RegionMethod), tsgo.NodeFlagsNone),
				nil,
				[]tsgo.TypeNode{b.typeL(), b.typeS()},
				[]tsgo.Expression{factory.Identifier("pointer"), factory.Identifier("length")},
				tsgo.NodeFlagsNone,
			),
		)}, true),
	)
}
