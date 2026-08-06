package slice

import "github.com/tsoniclang/gotots/internal/target/tsgo"

type projectionBuilder struct {
	factory        tsgo.Factory
	projectionName string
	sliceName      string
	panicName      string
	pointerName    string
	pointerProject string
}

func BuildProjection(
	factory tsgo.Factory,
	projectionName string,
	sliceName string,
	panicName string,
	pointerName string,
	pointerProject string,
	capabilities Capabilities,
) tsgo.ClassDeclaration {
	builder := projectionBuilder{
		factory:        factory,
		projectionName: projectionName,
		sliceName:      sliceName,
		panicName:      panicName,
		pointerName:    pointerName,
		pointerProject: pointerProject,
	}
	members := []tsgo.ClassElement{
		builder.constructor(),
		builder.isNilMethod(),
		builder.getMethod(),
		builder.setMethod(),
		builder.sliceMethod(),
		builder.appendMethod(),
	}
	if capabilities.Storage {
		members = append(
			members,
			builder.initializeMethod(),
			builder.withLengthMethod(),
		)
	}
	if capabilities.AppendSlice {
		members = append(members, builder.appendSliceMethod())
	}
	if capabilities.Clear {
		members = append(members, builder.clearMethod())
	}
	if capabilities.Address {
		members = append(members, builder.addressMethod())
	}
	if capabilities.ArrayPointer || capabilities.Region {
		members = append(members, builder.arrayLocationMethod())
	}
	return factory.ClassDeclaration(
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		builder.id(projectionName),
		[]tsgo.TypeParameterDeclaration{
			builder.typeParameter("F"),
			builder.typeParameter("T"),
		},
		[]tsgo.HeritageClause{factory.HeritageClause(
			tsgo.HeritageClauseTokenKindExtendsKeyword,
			[]tsgo.ExpressionWithTypeArguments{factory.ExpressionWithTypeArguments(
				builder.id(sliceName),
				[]tsgo.TypeNode{builder.typeReference("T")},
			)},
		)},
		members,
	)
}

func (b projectionBuilder) constructor() tsgo.ConstructorDeclaration {
	readonly := []tsgo.ModifierLike{
		b.factory.PrivateKeyword(),
		b.factory.ReadonlyKeyword(),
	}
	return b.factory.ConstructorDeclaration(
		nil,
		nil,
		[]tsgo.ParameterDeclaration{
			b.parameter(readonly, "source", b.sliceType("F")),
			b.parameter(readonly, "fromSource", b.converterType("F", "T")),
			b.parameter(readonly, "toSource", b.converterType("T", "F")),
			b.parameter(readonly, "sourceZero", b.typeReference("F")),
			b.parameter(readonly, "targetZero", b.typeReference("T")),
		},
		nil,
		b.factory.Block([]tsgo.Statement{b.factory.ExpressionStatement(
			b.factory.CallExpression(
				b.factory.SuperExpression(),
				nil,
				nil,
				[]tsgo.Expression{
					b.factory.NullLiteral(),
					b.number("0"),
					b.property(b.id("source"), MemberName(MemberLength)),
					b.property(b.id("source"), MemberName(MemberCapacity)),
				},
				tsgo.NodeFlagsNone,
			),
		)}, true),
	)
}

func (b projectionBuilder) isNilMethod() tsgo.MethodDeclaration {
	return b.method(
		MemberName(MemberIsNil),
		nil,
		b.booleanType(),
		b.returnStatement(b.call(b.source(), MemberName(MemberIsNil))),
	)
}

func (b projectionBuilder) getMethod() tsgo.MethodDeclaration {
	return b.method(
		MemberName(MemberGet),
		[]tsgo.ParameterDeclaration{
			b.parameter(nil, "index", b.integerInputType()),
		},
		b.typeReference("T"),
		b.returnStatement(b.convert("fromSource", b.call(
			b.source(),
			MemberName(MemberGet),
			b.id("index"),
		))),
	)
}

func (b projectionBuilder) setMethod() tsgo.MethodDeclaration {
	value := b.id("value")
	return b.method(
		MemberName(MemberSet),
		[]tsgo.ParameterDeclaration{
			b.parameter(nil, "index", b.integerInputType()),
			b.parameter(nil, "value", b.typeReference("T")),
		},
		b.typeReference("T"),
		b.factory.ExpressionStatement(b.call(
			b.source(),
			MemberName(MemberSet),
			b.id("index"),
			b.convert("toSource", value),
		)),
		b.returnStatement(value),
	)
}

func (b projectionBuilder) sliceMethod() tsgo.MethodDeclaration {
	return b.method(
		MemberName(MemberSlice),
		[]tsgo.ParameterDeclaration{
			b.parameter(nil, "low", b.integerInputType()),
			b.parameter(nil, "high", b.optionalIntegerInputType()),
			b.parameter(nil, "max", b.optionalIntegerInputType()),
		},
		b.sliceType("T"),
		b.returnStatement(b.newProjection(b.call(
			b.source(),
			MemberName(MemberSlice),
			b.id("low"),
			b.id("high"),
			b.id("max"),
		))),
	)
}

func (b projectionBuilder) appendMethod() tsgo.MethodDeclaration {
	values := b.id("values")
	converted := b.id("converted")
	convert := b.factory.ArrowFunction(
		nil,
		nil,
		[]tsgo.ParameterDeclaration{
			b.parameter(nil, "value", b.typeReference("T")),
		},
		b.typeReference("F"),
		b.factory.EqualsGreaterThanToken(),
		b.convert("toSource", b.id("value")),
	)
	return b.method(
		MemberName(MemberAppend),
		[]tsgo.ParameterDeclaration{
			b.parameter(nil, "_zero", b.typeReference("T")),
			b.parameter(nil, "values", b.factory.ArrayTypeNode(b.typeReference("T"))),
		},
		b.sliceType("T"),
		b.variable(
			tsgo.NodeFlagsConst,
			"converted",
			b.call(values, "map", convert),
		),
		b.returnStatement(b.newProjection(b.call(
			b.source(),
			MemberName(MemberAppend),
			b.thisProperty("sourceZero"),
			converted,
		))),
	)
}

func (b projectionBuilder) initializeMethod() tsgo.MethodDeclaration {
	return b.method(
		StorageInitializeMember,
		[]tsgo.ParameterDeclaration{
			b.parameter(nil, "index", b.numberType()),
			b.parameter(nil, "value", b.typeReference("T")),
		},
		b.voidType(),
		b.factory.ExpressionStatement(b.call(
			b.source(),
			StorageInitializeMember,
			b.id("index"),
			b.convert("toSource", b.id("value")),
		)),
	)
}

func (b projectionBuilder) withLengthMethod() tsgo.MethodDeclaration {
	return b.method(
		StorageWithLengthMember,
		[]tsgo.ParameterDeclaration{
			b.parameter(nil, "length", b.numberType()),
		},
		b.sliceType("T"),
		b.returnStatement(b.newProjection(b.call(
			b.source(),
			StorageWithLengthMember,
			b.id("length"),
		))),
	)
}

func (b projectionBuilder) appendSliceMethod() tsgo.MethodDeclaration {
	values := b.id("values")
	source := b.id("next")
	return b.method(
		MemberName(MemberAppendSlice),
		[]tsgo.ParameterDeclaration{
			b.parameter(nil, "_zero", b.typeReference("T")),
			b.parameter(nil, "next", b.sliceType("T")),
		},
		b.sliceType("T"),
		b.variable(
			tsgo.NodeFlagsConst,
			"values",
			b.factory.NewExpression(
				b.id("Array"),
				[]tsgo.TypeNode{b.typeReference("T")},
				nil,
			),
		),
		b.loop(
			b.property(source, MemberName(MemberLength)),
			b.factory.ExpressionStatement(b.call(
				values,
				"push",
				b.call(source, MemberName(MemberGet), b.id("index")),
			)),
		),
		b.returnStatement(b.call(
			b.factory.ThisExpression(),
			MemberName(MemberAppend),
			b.thisProperty("targetZero"),
			values,
		)),
	)
}

func (b projectionBuilder) clearMethod() tsgo.MethodDeclaration {
	return b.method(
		MemberName(MemberClear),
		[]tsgo.ParameterDeclaration{
			b.parameter(nil, "zero", b.typeReference("T")),
		},
		b.voidType(),
		b.loop(
			b.property(b.source(), MemberName(MemberLength)),
			b.factory.ExpressionStatement(b.call(
				b.source(),
				MemberName(MemberSet),
				b.id("index"),
				b.convert("toSource", b.id("zero")),
			)),
		),
	)
}

func (b projectionBuilder) method(
	name string,
	parameters []tsgo.ParameterDeclaration,
	result tsgo.TypeNode,
	statements ...tsgo.Statement,
) tsgo.MethodDeclaration {
	return b.factory.MethodDeclaration(
		[]tsgo.ModifierLike{b.factory.OverrideKeyword()},
		nil,
		b.id(name),
		nil,
		nil,
		parameters,
		result,
		b.factory.Block(statements, true),
	)
}

func (b projectionBuilder) newProjection(source tsgo.Expression) tsgo.NewExpression {
	return b.factory.NewExpression(
		b.id(b.projectionName),
		[]tsgo.TypeNode{b.typeReference("F"), b.typeReference("T")},
		[]tsgo.Expression{
			source,
			b.thisProperty("fromSource"),
			b.thisProperty("toSource"),
			b.thisProperty("sourceZero"),
			b.thisProperty("targetZero"),
		},
	)
}

func (b projectionBuilder) converterType(
	from string,
	to string,
) tsgo.FunctionTypeNode {
	return b.factory.FunctionTypeNode(
		nil,
		[]tsgo.ParameterDeclaration{
			b.parameter(nil, "value", b.typeReference(from)),
		},
		b.typeReference(to),
	)
}

func (b projectionBuilder) sliceType(element string) tsgo.TypeReferenceNode {
	return b.factory.TypeReferenceNode(
		b.id(b.sliceName),
		[]tsgo.TypeNode{b.typeReference(element)},
	)
}

func (b projectionBuilder) pointerType(
	logical tsgo.TypeNode,
	storage tsgo.TypeNode,
) tsgo.TypeReferenceNode {
	return b.factory.TypeReferenceNode(
		b.id(b.pointerName),
		[]tsgo.TypeNode{logical, storage},
	)
}

func (b projectionBuilder) integerInputType() tsgo.UnionTypeNode {
	return b.factory.UnionTypeNode([]tsgo.TypeNode{
		b.numberType(),
		b.factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindBigIntKeyword),
	})
}

func (b projectionBuilder) optionalIntegerInputType() tsgo.UnionTypeNode {
	return b.factory.UnionTypeNode([]tsgo.TypeNode{
		b.integerInputType(),
		b.factory.LiteralTypeNode(b.factory.NullLiteral()),
	})
}

func (b projectionBuilder) parameter(
	modifiers []tsgo.ModifierLike,
	name string,
	typeNode tsgo.TypeNode,
) tsgo.ParameterDeclaration {
	return b.factory.ParameterDeclaration(
		modifiers,
		nil,
		b.id(name),
		nil,
		typeNode,
		nil,
	)
}

func (b projectionBuilder) typeParameter(name string) tsgo.TypeParameterDeclaration {
	return b.factory.TypeParameterDeclaration(nil, b.id(name), nil, nil, nil)
}

func (b projectionBuilder) typeReference(name string) tsgo.TypeReferenceNode {
	return b.factory.TypeReferenceNode(b.id(name), nil)
}

func (b projectionBuilder) numberType() tsgo.TypeNode {
	return b.factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindNumberKeyword)
}

func (b projectionBuilder) booleanType() tsgo.TypeNode {
	return b.factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindBooleanKeyword)
}

func (b projectionBuilder) voidType() tsgo.TypeNode {
	return b.factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindVoidKeyword)
}

func (b projectionBuilder) id(name string) tsgo.Identifier {
	return b.factory.Identifier(name)
}

func (b projectionBuilder) number(value string) tsgo.NumericLiteral {
	return b.factory.NumericLiteral(value, tsgo.TokenFlagsNone)
}

func (b projectionBuilder) source() tsgo.Expression {
	return b.thisProperty("source")
}

func (b projectionBuilder) thisProperty(name string) tsgo.PropertyAccessExpression {
	return b.property(b.factory.ThisExpression(), name)
}

func (b projectionBuilder) property(
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

func (b projectionBuilder) call(
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

func (b projectionBuilder) convert(
	member string,
	value tsgo.Expression,
) tsgo.CallExpression {
	return b.factory.CallExpression(
		b.thisProperty(member),
		nil,
		nil,
		[]tsgo.Expression{value},
		tsgo.NodeFlagsNone,
	)
}

func (b projectionBuilder) element(
	value tsgo.Expression,
	index tsgo.Expression,
) tsgo.ElementAccessExpression {
	return b.factory.ElementAccessExpression(
		value,
		nil,
		index,
		tsgo.NodeFlagsNone,
	)
}

func (b projectionBuilder) assign(
	left tsgo.Expression,
	right tsgo.Expression,
) tsgo.BinaryExpression {
	return b.factory.BinaryExpression(
		nil,
		left,
		nil,
		b.factory.BinaryOperatorToken(tsgo.BinaryOperatorEqualsToken),
		right,
	)
}

func (b projectionBuilder) variable(
	flags tsgo.NodeFlags,
	name string,
	value tsgo.Expression,
) tsgo.VariableStatement {
	return b.factory.VariableStatement(
		nil,
		b.factory.VariableDeclarationList(
			[]tsgo.VariableDeclaration{
				b.factory.VariableDeclaration(b.id(name), nil, nil, value),
			},
			flags,
		),
	)
}

func (b projectionBuilder) loop(
	limit tsgo.Expression,
	statements ...tsgo.Statement,
) tsgo.ForStatement {
	return b.factory.ForStatement(
		b.factory.VariableDeclarationList(
			[]tsgo.VariableDeclaration{b.factory.VariableDeclaration(
				b.id("index"),
				nil,
				nil,
				b.number("0"),
			)},
			tsgo.NodeFlagsLet,
		),
		b.factory.BinaryExpression(
			nil,
			b.id("index"),
			nil,
			b.factory.BinaryOperatorToken(tsgo.BinaryOperatorLessThanToken),
			limit,
		),
		b.factory.PostfixUnaryExpression(
			b.id("index"),
			tsgo.PostfixUnaryExpressionOperatorKindPlusPlusToken,
		),
		b.factory.Block(statements, true),
	)
}

func (b projectionBuilder) returnStatement(value tsgo.Expression) tsgo.ReturnStatement {
	return b.factory.ReturnStatement(value)
}
