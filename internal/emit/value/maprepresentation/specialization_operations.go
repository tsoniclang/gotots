package maprepresentation

import "github.com/tsoniclang/gotots/internal/target/tsgo"

const (
	specializationZeroOperation       = "$zeroValue"
	specializationHashOperation       = "$hash"
	specializationEqualOperation      = "$equal"
	specializationCopyOperation       = "$copyKey"
	specializationCopyValueOperation  = "$copyValue"
	specializationProjectKeyOperation = "$projectKey"
	specializationReifyKeyOperation   = "$reifyKey"
	specializationFindOperation       = "$find"
)

func (b specializationBuilder) buildHashed() []tsgo.ClassElement {
	members := []tsgo.ClassElement{
		b.constructor(),
		b.operationMethod(
			specializationZeroOperation,
			nil,
			b.valueType,
			b.zero,
		),
		b.operationMethod(
			specializationHashOperation,
			[]tsgo.ParameterDeclaration{
				b.parameter("$key", b.storageKeyType),
			},
			b.numberType(),
			b.hash,
		),
		b.operationMethod(
			specializationEqualOperation,
			[]tsgo.ParameterDeclaration{
				b.parameter("$left", b.storageKeyType),
				b.parameter("$right", b.storageKeyType),
			},
			b.booleanType(),
			b.equal,
		),
		b.operationMethod(
			specializationCopyOperation,
			[]tsgo.ParameterDeclaration{
				b.parameter("$key", b.storageKeyType),
			},
			b.storageKeyType,
			b.copyKey,
		),
		b.operationMethod(
			specializationCopyValueOperation,
			[]tsgo.ParameterDeclaration{
				b.parameter("$value", b.valueType),
			},
			b.valueType,
			b.copyValue,
		),
	}
	if b.keyProjection {
		members = append(
			members,
			b.operationMethod(
				specializationProjectKeyOperation,
				[]tsgo.ParameterDeclaration{
					b.parameter("$key", b.keyType),
				},
				b.storageKeyType,
				b.projectKey,
			),
			b.operationMethod(
				specializationReifyKeyOperation,
				[]tsgo.ParameterDeclaration{
					b.parameter("$storageKey", b.storageKeyType),
				},
				b.keyType,
				b.reifyKey,
			),
		)
	}
	members = append(
		members,
		b.nilMethod(),
		b.makeMethod(),
		b.findMethod(),
		b.lookupMethod(),
		b.lookupOKMethod(),
		b.storeMethod(),
		b.deleteMethod(),
		b.lengthMethod(),
		b.isNilMethod(),
	)
	members = append(members, b.clearMethod(), b.keysMethod())
	return members
}

func (b specializationBuilder) constructor() tsgo.ConstructorDeclaration {
	return b.factory.ConstructorDeclaration(
		[]tsgo.ModifierLike{b.factory.PrivateKeyword()},
		nil,
		[]tsgo.ParameterDeclaration{
			b.parameterProperty("zeroValue", b.valueType, true),
			b.parameterProperty("buckets", b.storageType(), true),
			b.parameterProperty("count", b.numberType(), false),
		},
		nil,
		b.factory.Block([]tsgo.Statement{b.superCall()}, true),
	)
}

func (b specializationBuilder) operationMethod(
	name string,
	parameters []tsgo.ParameterDeclaration,
	result tsgo.TypeNode,
	body operationBody,
) tsgo.MethodDeclaration {
	statements := append(
		body.before,
		b.factory.ReturnStatement(body.value),
	)
	return b.method(
		[]tsgo.ModifierLike{
			b.factory.PrivateKeyword(),
			b.factory.StaticKeyword(),
		},
		name,
		parameters,
		result,
		statements...,
	)
}

func (b specializationBuilder) nilMethod() tsgo.MethodDeclaration {
	return b.method(
		[]tsgo.ModifierLike{b.factory.StaticKeyword()},
		b.members.nilMember,
		nil,
		b.classType(),
		b.factory.ReturnStatement(
			b.factory.NewExpression(
				b.id(b.className),
				nil,
				[]tsgo.Expression{
					b.staticCall(specializationZeroOperation),
					b.id("undefined"),
					b.number("0"),
				},
			),
		),
	)
}

func (b specializationBuilder) makeMethod() tsgo.MethodDeclaration {
	result := b.id("result")
	entry := b.id("entry")
	return b.method(
		[]tsgo.ModifierLike{b.factory.StaticKeyword()},
		b.members.makeMember,
		[]tsgo.ParameterDeclaration{
			b.parameter(
				"size",
				b.factory.UnionTypeNode([]tsgo.TypeNode{
					b.numberType(),
					b.factory.KeywordTypeNode(
						tsgo.KeywordTypeSyntaxKindBigIntKeyword,
					),
				}),
			),
			b.parameter("entries", b.factory.ArrayTypeNode(b.entryType())),
		},
		b.classType(),
		b.variable(
			tsgo.NodeFlagsConst,
			"result",
			b.classType(),
			b.factory.NewExpression(
				b.id(b.className),
				nil,
				[]tsgo.Expression{
					b.staticCall(specializationZeroOperation),
					b.factory.NewExpression(
						b.id("Map"),
						[]tsgo.TypeNode{
							b.numberType(),
							b.bucketType(),
						},
						nil,
					),
					b.number("0"),
				},
			),
		),
		b.forEntries(
			b.id("entries"),
			b.factory.ExpressionStatement(
				b.call(
					result,
					b.members.store,
					b.element(entry, b.number("0")),
					b.element(entry, b.number("1")),
				),
			),
		),
		b.factory.ReturnStatement(result),
	)
}
