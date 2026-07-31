package slice

import (
	panicruntime "github.com/tsoniclang/gotots/internal/emit/runtime/panic"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func BuildRegion(
	factory tsgo.Factory,
	functionName string,
	sliceName string,
	panicName string,
) tsgo.FunctionDeclaration {
	b := builder{factory: factory, className: sliceName, panicName: panicName}
	locationType := b.factory.TypeOperatorNode(
		tsgo.TypeOperatorNodeOperatorKindReadonlyKeyword,
		b.factory.TupleTypeNode([]tsgo.TypeNode{
			b.factory.ArrayTypeNode(b.typeT()),
			b.numberType(),
		}),
	)
	optionalLocation := b.factory.UnionTypeNode([]tsgo.TypeNode{
		locationType,
		b.factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindUndefinedKeyword),
	})
	location := b.id("location")
	length := b.id("numericLength")
	backing := b.factory.ElementAccessExpression(location, nil, b.number("0"), tsgo.NodeFlagsNone)
	offset := b.factory.ElementAccessExpression(location, nil, b.number("1"), tsgo.NodeFlagsNone)
	panicStatement := func(message string) tsgo.ExpressionStatement {
		return b.factory.ExpressionStatement(panicruntime.Call(
			b.factory,
			panicName,
			b.factory.StringLiteral(message, tsgo.TokenFlagsNone),
		))
	}
	return b.factory.FunctionDeclaration(
		[]tsgo.ModifierLike{b.factory.ExportKeyword()},
		nil,
		b.id(functionName),
		[]tsgo.TypeParameterDeclaration{b.typeParameter()},
		[]tsgo.ParameterDeclaration{
			b.parameter("location", optionalLocation),
			b.parameter("length", b.integerInputType()),
		},
		b.sliceType(),
		b.factory.Block([]tsgo.Statement{
			b.variable(tsgo.NodeFlagsConst, "numericLength", b.toNumber(b.id("length"))),
			b.factory.IfStatement(
				b.binary(length, tsgo.BinaryOperatorLessThanToken, b.number("0")),
				panicStatement("unsafe slice length is negative"),
				nil,
			),
			b.factory.IfStatement(
				b.binary(location, tsgo.BinaryOperatorEqualsEqualsEqualsToken, b.factory.VoidExpression(b.number("0"))),
				b.factory.Block([]tsgo.Statement{
					b.factory.IfStatement(
						b.binary(length, tsgo.BinaryOperatorEqualsEqualsEqualsToken, b.number("0")),
						b.factory.Block([]tsgo.Statement{b.returnStatement(
							b.factory.CallExpression(
								b.property(b.id(sliceName), MemberName(MemberNil)),
								nil,
								[]tsgo.TypeNode{b.typeT()},
								nil,
								tsgo.NodeFlagsNone,
							),
						)}, true),
						nil,
					),
					panicStatement("unsafe slice on nil pointer"),
				}, true),
				nil,
			),
			b.factory.IfStatement(
				b.binary(
					length,
					tsgo.BinaryOperatorGreaterThanToken,
					b.binary(b.property(backing, "length"), tsgo.BinaryOperatorMinusToken, offset),
				),
				panicStatement("unsafe slice exceeds pointer region"),
				nil,
			),
			b.returnStatement(b.factory.CallExpression(
				b.property(b.id(sliceName), ArrayViewMember),
				nil,
				[]tsgo.TypeNode{b.typeT()},
				[]tsgo.Expression{backing, offset, length, length},
				tsgo.NodeFlagsNone,
			)),
		}, true),
	)
}
