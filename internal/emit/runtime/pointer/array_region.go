package pointer

import "github.com/tsoniclang/gotots/internal/target/tsgo"

func (b builder) arrayRegionMethod() tsgo.MethodDeclaration {
	typeL := b.typeReference("L")
	typeT := b.typeReference("T")
	typeS := b.typeReference("S")
	indexType := b.factory.UnionTypeNode([]tsgo.TypeNode{
		b.numberType(),
		b.factory.KeywordTypeNode(
			tsgo.KeywordTypeSyntaxKindBigIntKeyword,
		),
	})
	storageConstraint := b.factory.TypeLiteralNode([]tsgo.TypeElement{
		b.factory.PropertySignatureDeclaration(
			nil,
			b.id("length"),
			nil,
			b.numberType(),
			b.factory.OmittedExpression(),
		),
		b.factory.MethodSignatureDeclaration(
			nil,
			b.id("get"),
			nil,
			nil,
			[]tsgo.ParameterDeclaration{
				b.parameter("index", indexType),
			},
			typeT,
		),
		b.factory.MethodSignatureDeclaration(
			nil,
			b.id("set"),
			nil,
			nil,
			[]tsgo.ParameterDeclaration{
				b.parameter("index", indexType),
				b.parameter("value", typeT),
			},
			b.factory.KeywordTypeNode(
				tsgo.KeywordTypeSyntaxKindVoidKeyword,
			),
		),
	})
	locationType := b.factory.TypeOperatorNode(
		tsgo.TypeOperatorNodeOperatorKindReadonlyKeyword,
		b.factory.TupleTypeNode([]tsgo.TypeNode{
			b.factory.ArrayTypeNode(typeT),
			b.numberType(),
		}),
	)
	backing := b.id("backing")
	offset := b.id("offset")
	index := b.id("index")
	view := b.id("view")
	next := b.id("next")
	return b.method(
		[]tsgo.ModifierLike{b.factory.StaticKeyword()},
		ArrayRegionName,
		[]tsgo.TypeParameterDeclaration{
			b.typeParameter("L", nil),
			b.typeParameter("T", nil),
			b.typeParameter("S", storageConstraint),
		},
		[]tsgo.ParameterDeclaration{
			b.parameter("location", locationType),
			b.parameter("view", typeS),
		},
		b.pointerType(typeL, typeS),
		b.variable(
			tsgo.NodeFlagsConst,
			"backing",
			nil,
			b.factory.ElementAccessExpression(
				b.id("location"),
				nil,
				b.factory.NumericLiteral("0", tsgo.TokenFlagsNone),
				tsgo.NodeFlagsNone,
			),
		),
		b.variable(
			tsgo.NodeFlagsConst,
			"offset",
			nil,
			b.factory.ElementAccessExpression(
				b.id("location"),
				nil,
				b.factory.NumericLiteral("1", tsgo.TokenFlagsNone),
				tsgo.NodeFlagsNone,
			),
		),
		b.factory.ReturnStatement(
			b.newPointerWithWriteBody(
				typeL,
				typeS,
				b.call(
					b.id(b.className),
					"child",
					b.call(b.id(b.className), "root", backing),
					offset,
				),
				view,
				[]tsgo.Statement{b.factory.ForStatement(
					b.factory.VariableDeclarationList(
						[]tsgo.VariableDeclaration{
							b.factory.VariableDeclaration(
								index,
								nil,
								nil,
								b.factory.NumericLiteral(
									"0",
									tsgo.TokenFlagsNone,
								),
							),
						},
						tsgo.NodeFlagsLet,
					),
					b.binary(
						index,
						tsgo.BinaryOperatorLessThanToken,
						b.property(view, "length"),
					),
					b.factory.PostfixUnaryExpression(
						index,
						tsgo.PostfixUnaryExpressionOperatorKindPlusPlusToken,
					),
					b.factory.Block([]tsgo.Statement{
						b.factory.ExpressionStatement(
							b.call(
								view,
								"set",
								index,
								b.call(next, "get", index),
							),
						),
					}, true),
				)},
			),
		),
	)
}
