package slice

import "github.com/tsoniclang/gotots/internal/target/tsgo"

const ArrayViewMember = "$view"

func (b builder) arrayViewMethod() tsgo.MethodDeclaration {
	return b.method(
		[]tsgo.ModifierLike{b.factory.StaticKeyword()},
		ArrayViewMember,
		[]tsgo.TypeParameterDeclaration{b.typeParameter()},
		[]tsgo.ParameterDeclaration{
			b.parameter("backing", b.factory.ArrayTypeNode(b.typeT())),
			b.parameter("offset", b.numberType()),
			b.parameter("length", b.numberType()),
			b.parameter("capacity", b.numberType()),
		},
		b.sliceType(),
		b.returnStatement(b.newSlice(
			b.id("backing"),
			b.id("offset"),
			b.id("length"),
			b.id("capacity"),
		)),
	)
}

func BuildArraySlice(
	factory tsgo.Factory,
	functionName string,
	sliceName string,
	arrayName string,
	locationName string,
) tsgo.FunctionDeclaration {
	typeT := typeReference(factory, "T")
	typeN := typeReference(factory, "N")
	arrayType := factory.TypeReferenceNode(
		factory.Identifier(arrayName),
		[]tsgo.TypeNode{typeT, typeN},
	)
	sliceType := factory.TypeReferenceNode(
		factory.Identifier(sliceName),
		[]tsgo.TypeNode{typeT},
	)
	integerType := factory.UnionTypeNode([]tsgo.TypeNode{
		factory.KeywordTypeNode(
			tsgo.KeywordTypeSyntaxKindNumberKeyword,
		),
		factory.KeywordTypeNode(
			tsgo.KeywordTypeSyntaxKindBigIntKeyword,
		),
	})
	optionalIntegerType := factory.UnionTypeNode([]tsgo.TypeNode{
		integerType,
		factory.LiteralTypeNode(factory.NullLiteral()),
	})
	value := factory.Identifier("value")
	location := factory.Identifier("location")
	locationElement := func(index string) tsgo.ElementAccessExpression {
		return factory.ElementAccessExpression(
			location,
			nil,
			factory.NumericLiteral(index, tsgo.TokenFlagsNone),
			tsgo.NodeFlagsNone,
		)
	}
	length := factory.PropertyAccessExpression(
		value,
		nil,
		factory.Identifier("length"),
		tsgo.NodeFlagsNone,
	)
	view := factory.CallExpression(
		factory.PropertyAccessExpression(
			factory.Identifier(sliceName),
			nil,
			factory.Identifier(ArrayViewMember),
			tsgo.NodeFlagsNone,
		),
		nil,
		[]tsgo.TypeNode{typeT},
		[]tsgo.Expression{
			locationElement("0"),
			locationElement("1"),
			length,
			length,
		},
		tsgo.NodeFlagsNone,
	)
	result := factory.CallExpression(
		factory.PropertyAccessExpression(
			view,
			nil,
			factory.Identifier(MemberName(MemberSlice)),
			tsgo.NodeFlagsNone,
		),
		nil,
		nil,
		[]tsgo.Expression{
			factory.Identifier("low"),
			factory.Identifier("high"),
			factory.Identifier("max"),
		},
		tsgo.NodeFlagsNone,
	)
	return factory.FunctionDeclaration(
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		nil,
		factory.Identifier(functionName),
		[]tsgo.TypeParameterDeclaration{
			typeParameter(factory, "T"),
			factory.TypeParameterDeclaration(
				nil,
				factory.Identifier("N"),
				factory.KeywordTypeNode(
					tsgo.KeywordTypeSyntaxKindNumberKeyword,
				),
				nil,
				nil,
			),
		},
		[]tsgo.ParameterDeclaration{
			parameter(factory, "value", arrayType),
			parameter(factory, "low", integerType),
			parameter(factory, "high", optionalIntegerType),
			parameter(factory, "max", optionalIntegerType),
		},
		sliceType,
		factory.Block([]tsgo.Statement{
			arraySliceVariable(
				factory,
				"location",
				factory.CallExpression(
					factory.Identifier(locationName),
					nil,
					[]tsgo.TypeNode{typeT, typeN},
					[]tsgo.Expression{value},
					tsgo.NodeFlagsNone,
				),
			),
			factory.ReturnStatement(result),
		}, true),
	)
}

func arraySliceVariable(
	factory tsgo.Factory,
	name string,
	value tsgo.Expression,
) tsgo.VariableStatement {
	return factory.VariableStatement(
		nil,
		factory.VariableDeclarationList(
			[]tsgo.VariableDeclaration{
				factory.VariableDeclaration(
					factory.Identifier(name),
					nil,
					nil,
					value,
				),
			},
			tsgo.NodeFlagsConst,
		),
	)
}
