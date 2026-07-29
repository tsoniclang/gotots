package maprepresentation

import (
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type specializationBuilder struct {
	factory   tsgo.Factory
	className string
	keyType   tsgo.TypeNode
	valueType tsgo.TypeNode
	panicName string
	zero      operationBody
	hash      operationBody
	equal     operationBody
	copyKey   operationBody
	copyValue operationBody
	members   specializationMemberNames
}

type operationBody struct {
	before []tsgo.Statement
	value  tsgo.Expression
}

type specializationMemberNames struct {
	nilMember    string
	makeMember   string
	lookup       string
	lookupOK     string
	store        string
	deleteMember string
	length       string
	isNil        string
	clear        string
	keys         string
}

func (b specializationBuilder) id(name string) tsgo.Identifier {
	return b.factory.Identifier(name)
}

func (b specializationBuilder) number(value string) tsgo.NumericLiteral {
	return b.factory.NumericLiteral(value, tsgo.TokenFlagsNone)
}

func (b specializationBuilder) numberType() tsgo.TypeNode {
	return b.factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindNumberKeyword)
}

func (b specializationBuilder) booleanType() tsgo.TypeNode {
	return b.factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindBooleanKeyword)
}

func (b specializationBuilder) voidType() tsgo.TypeNode {
	return b.factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindVoidKeyword)
}

func (b specializationBuilder) undefinedType() tsgo.TypeNode {
	return b.factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindUndefinedKeyword)
}

func (b specializationBuilder) classType() tsgo.TypeNode {
	return b.factory.TypeReferenceNode(b.id(b.className), nil)
}

func (b specializationBuilder) entryType() tsgo.TypeNode {
	return b.factory.TupleTypeNode(
		[]tsgo.TypeNode{b.keyType, b.valueType},
	)
}

func (b specializationBuilder) bucketType() tsgo.TypeNode {
	return b.factory.ArrayTypeNode(b.entryType())
}

func (b specializationBuilder) storageType() tsgo.TypeNode {
	return b.factory.UnionTypeNode([]tsgo.TypeNode{
		b.factory.TypeReferenceNode(
			b.id("Map"),
			[]tsgo.TypeNode{b.numberType(), b.bucketType()},
		),
		b.undefinedType(),
	})
}

func (b specializationBuilder) parameter(
	name string,
	targetType tsgo.TypeNode,
) tsgo.ParameterDeclaration {
	return b.factory.ParameterDeclaration(
		nil,
		nil,
		b.id(name),
		nil,
		targetType,
		nil,
	)
}

func (b specializationBuilder) parameterProperty(
	name string,
	targetType tsgo.TypeNode,
	readonly bool,
) tsgo.ParameterDeclaration {
	modifiers := []tsgo.ModifierLike{b.factory.PrivateKeyword()}
	if readonly {
		modifiers = append(modifiers, b.factory.ReadonlyKeyword())
	}
	return b.factory.ParameterDeclaration(
		modifiers,
		nil,
		b.id(name),
		nil,
		targetType,
		nil,
	)
}

func (b specializationBuilder) property(
	receiver tsgo.Expression,
	name string,
) tsgo.PropertyAccessExpression {
	return b.factory.PropertyAccessExpression(
		receiver,
		nil,
		b.id(name),
		tsgo.NodeFlagsNone,
	)
}

func (b specializationBuilder) element(
	receiver tsgo.Expression,
	index tsgo.Expression,
) tsgo.ElementAccessExpression {
	return b.factory.ElementAccessExpression(
		receiver,
		nil,
		index,
		tsgo.NodeFlagsNone,
	)
}

func (b specializationBuilder) call(
	receiver tsgo.Expression,
	name string,
	arguments ...tsgo.Expression,
) tsgo.CallExpression {
	return b.factory.CallExpression(
		b.property(receiver, name),
		nil,
		nil,
		arguments,
		tsgo.NodeFlagsNone,
	)
}

func (b specializationBuilder) staticCall(
	name string,
	arguments ...tsgo.Expression,
) tsgo.CallExpression {
	return b.call(b.id(b.className), name, arguments...)
}

func (b specializationBuilder) binary(
	left tsgo.Expression,
	operator tsgo.BinaryOperator,
	right tsgo.Expression,
) tsgo.BinaryExpression {
	return b.factory.BinaryExpression(
		nil,
		left,
		nil,
		b.factory.BinaryOperatorToken(operator),
		right,
	)
}

func (b specializationBuilder) assign(
	left tsgo.Expression,
	right tsgo.Expression,
) tsgo.BinaryExpression {
	return b.binary(left, tsgo.BinaryOperatorEqualsToken, right)
}

func (b specializationBuilder) undefined(
	value tsgo.Expression,
) tsgo.BinaryExpression {
	return b.binary(
		value,
		tsgo.BinaryOperatorEqualsEqualsEqualsToken,
		b.id("undefined"),
	)
}

func (b specializationBuilder) variable(
	flags tsgo.NodeFlags,
	name string,
	targetType tsgo.TypeNode,
	value tsgo.Expression,
) tsgo.VariableStatement {
	return b.factory.VariableStatement(
		nil,
		b.factory.VariableDeclarationList(
			[]tsgo.VariableDeclaration{
				b.factory.VariableDeclaration(
					b.id(name),
					nil,
					targetType,
					value,
				),
			},
			flags,
		),
	)
}

func (b specializationBuilder) method(
	modifiers []tsgo.ModifierLike,
	name string,
	parameters []tsgo.ParameterDeclaration,
	result tsgo.TypeNode,
	statements ...tsgo.Statement,
) tsgo.MethodDeclaration {
	return b.factory.MethodDeclaration(
		modifiers,
		nil,
		b.id(name),
		nil,
		nil,
		parameters,
		result,
		b.factory.Block(statements, true),
	)
}

func (b specializationBuilder) forEntries(
	bucket tsgo.Expression,
	statements ...tsgo.Statement,
) tsgo.ForOfStatement {
	return b.factory.ForOfStatement(
		nil,
		b.factory.VariableDeclarationList(
			[]tsgo.VariableDeclaration{
				b.factory.VariableDeclaration(
					b.id("entry"),
					nil,
					nil,
					nil,
				),
			},
			tsgo.NodeFlagsConst,
		),
		bucket,
		b.factory.Block(statements, true),
	)
}

func (b specializationBuilder) returnBlock(
	value tsgo.Expression,
) tsgo.Block {
	return b.factory.Block(
		[]tsgo.Statement{b.factory.ReturnStatement(value)},
		true,
	)
}
