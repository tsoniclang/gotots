package emptystruct

import "github.com/tsoniclang/gotots/internal/target/tsgo"

const (
	makeMember        = "$make"
	zeroMember        = "$zero"
	copyMember        = "$copy"
	equalMember       = "$equal"
	hashMember        = "$hash"
	convertMember     = "$convert"
	storageOfMember   = "$storageOf"
	fromStorageMember = "$fromStorage"
)

func Build(factory tsgo.Factory, className string) tsgo.ClassDeclaration {
	target := builder{factory: factory, className: className}
	classType := target.classType()
	returnSource := func() tsgo.Expression {
		return target.factory.Identifier("$source")
	}
	returnNew := func() tsgo.Expression {
		return target.factory.NewExpression(
			target.factory.Identifier(className),
			nil,
			nil,
		)
	}
	returnTrue := func() tsgo.Expression {
		return target.factory.TrueLiteral()
	}
	returnHash := func() tsgo.Expression {
		return target.factory.NumericLiteral("2166136261", tsgo.TokenFlagsNone)
	}
	return factory.ClassDeclaration(
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		factory.Identifier(className),
		nil,
		nil,
		[]tsgo.ClassElement{
			target.brand(),
			target.constructor(),
			target.method(makeMember, nil, classType, returnNew()),
			target.method(zeroMember, nil, classType, returnNew()),
			target.method(
				copyMember,
				[]tsgo.ParameterDeclaration{target.parameter("$source", classType)},
				classType,
				returnSource(),
			),
			target.method(
				equalMember,
				[]tsgo.ParameterDeclaration{
					target.parameter("$left", classType),
					target.parameter("$right", classType),
				},
				target.booleanType(),
				returnTrue(),
			),
			target.method(
				hashMember,
				[]tsgo.ParameterDeclaration{target.parameter("$source", classType)},
				target.numberType(),
				returnHash(),
			),
			target.method(
				convertMember,
				[]tsgo.ParameterDeclaration{target.parameter("$source", target.objectType())},
				classType,
				returnNew(),
			),
			target.method(
				storageOfMember,
				[]tsgo.ParameterDeclaration{target.parameter("$source", classType)},
				classType,
				returnSource(),
			),
			target.method(
				fromStorageMember,
				[]tsgo.ParameterDeclaration{target.parameter("$source", classType)},
				classType,
				returnSource(),
			),
		},
	)
}

type builder struct {
	factory   tsgo.Factory
	className string
}

func (b builder) brand() tsgo.PropertyDeclaration {
	return b.factory.PropertyDeclaration(
		[]tsgo.ModifierLike{
			b.factory.DeclareKeyword(),
			b.factory.PrivateKeyword(),
			b.factory.ReadonlyKeyword(),
		},
		b.factory.Identifier("$go$emptyStruct"),
		nil,
		b.factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindVoidKeyword),
		nil,
	)
}

func (b builder) constructor() tsgo.ConstructorDeclaration {
	return b.factory.ConstructorDeclaration(
		[]tsgo.ModifierLike{b.factory.PrivateKeyword()},
		nil,
		nil,
		nil,
		b.factory.Block(nil, true),
	)
}

func (b builder) method(
	name string,
	parameters []tsgo.ParameterDeclaration,
	result tsgo.TypeNode,
	value tsgo.Expression,
) tsgo.MethodDeclaration {
	return b.factory.MethodDeclaration(
		[]tsgo.ModifierLike{b.factory.StaticKeyword()},
		nil,
		b.factory.Identifier(name),
		nil,
		nil,
		parameters,
		result,
		b.factory.Block(
			[]tsgo.Statement{b.factory.ReturnStatement(value)},
			true,
		),
	)
}

func (b builder) parameter(
	name string,
	targetType tsgo.TypeNode,
) tsgo.ParameterDeclaration {
	return b.factory.ParameterDeclaration(
		nil,
		nil,
		b.factory.Identifier(name),
		nil,
		targetType,
		nil,
	)
}

func (b builder) classType() tsgo.TypeNode {
	return b.factory.TypeReferenceNode(
		b.factory.Identifier(b.className),
		nil,
	)
}

func (b builder) booleanType() tsgo.TypeNode {
	return b.factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindBooleanKeyword)
}

func (b builder) numberType() tsgo.TypeNode {
	return b.factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindNumberKeyword)
}

func (b builder) objectType() tsgo.TypeNode {
	return b.factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindObjectKeyword)
}
