package unsafecodec

import "github.com/tsoniclang/gotots/internal/target/tsgo"

func (b builder) memoryType() tsgo.TypeNode {
	storage := b.typeReference("S")
	read := b.factory.FunctionTypeNode(nil, nil, storage)
	write := b.factory.FunctionTypeNode(
		nil,
		[]tsgo.ParameterDeclaration{b.parameter("value", storage, nil)},
		b.voidType(),
	)
	region := b.factory.TypeOperatorNode(
		tsgo.TypeOperatorNodeOperatorKindReadonlyKeyword,
		b.factory.TupleTypeNode([]tsgo.TypeNode{
			b.factory.ArrayTypeNode(storage),
			b.numberType(),
		}),
	)
	optionalRegion := b.factory.UnionTypeNode([]tsgo.TypeNode{
		region,
		b.factory.KeywordTypeNode(
			tsgo.KeywordTypeSyntaxKindUndefinedKeyword,
		),
	})
	return b.factory.TypeOperatorNode(
		tsgo.TypeOperatorNodeOperatorKindReadonlyKeyword,
		b.factory.TupleTypeNode([]tsgo.TypeNode{
			b.factory.KeywordTypeNode(
				tsgo.KeywordTypeSyntaxKindObjectKeyword,
			),
			read,
			write,
			optionalRegion,
		}),
	)
}

func (b builder) bindingsProperty() tsgo.PropertyDeclaration {
	target := b.factory.TypeReferenceNode(
		b.id("WeakMap"),
		[]tsgo.TypeNode{
			b.factory.KeywordTypeNode(
				tsgo.KeywordTypeSyntaxKindObjectKeyword,
			),
			b.memoryType(),
		},
	)
	return b.factory.PropertyDeclaration(
		[]tsgo.ModifierLike{
			b.factory.PrivateKeyword(),
			b.factory.ReadonlyKeyword(),
		},
		b.id("bindings"),
		nil,
		target,
		b.factory.NewExpression(b.id("WeakMap"), nil, nil),
	)
}

func (b builder) bind() tsgo.MethodDeclaration {
	return b.method(
		BindName,
		[]tsgo.ParameterDeclaration{
			b.parameter(
				"pointer",
				b.factory.KeywordTypeNode(
					tsgo.KeywordTypeSyntaxKindObjectKeyword,
				),
				nil,
			),
			b.parameter("memory", b.memoryType(), nil),
		},
		b.voidType(),
		b.factory.ExpressionStatement(b.call(
			b.property(b.factory.ThisExpression(), "bindings"),
			"set",
			b.id("pointer"),
			b.id("memory"),
		)),
	)
}

func (b builder) bound() tsgo.MethodDeclaration {
	return b.method(
		BoundName,
		[]tsgo.ParameterDeclaration{b.parameter(
			"pointer",
			b.factory.KeywordTypeNode(
				tsgo.KeywordTypeSyntaxKindObjectKeyword,
			),
			nil,
		)},
		b.factory.UnionTypeNode([]tsgo.TypeNode{
			b.memoryType(),
			b.factory.KeywordTypeNode(
				tsgo.KeywordTypeSyntaxKindUndefinedKeyword,
			),
		}),
		b.factory.ReturnStatement(b.call(
			b.property(b.factory.ThisExpression(), "bindings"),
			"get",
			b.id("pointer"),
		)),
	)
}
