package pointer

import "github.com/tsoniclang/gotots/internal/target/tsgo"

func (b builder) fieldsMethod() tsgo.MethodDeclaration {
	typeL := b.typeReference("L")
	typeS := b.typeReference("S")
	typePL := b.typeReference("PL")
	typePS := b.typeReference("PS")
	parent := b.id("parent")
	parentStorage := b.factory.CallExpression(
		b.property(parent, "read"),
		nil,
		nil,
		nil,
		tsgo.NodeFlagsNone,
	)
	address := b.factory.CallExpression(
		b.id("address"),
		nil,
		nil,
		[]tsgo.Expression{b.property(parent, AddressName)},
		tsgo.NodeFlagsNone,
	)
	read := b.factory.CallExpression(
		b.id("read"),
		nil,
		nil,
		[]tsgo.Expression{parentStorage},
		tsgo.NodeFlagsNone,
	)
	write := b.factory.CallExpression(
		b.id("write"),
		nil,
		nil,
		[]tsgo.Expression{parentStorage, b.id("next")},
		tsgo.NodeFlagsNone,
	)
	addressProjection := b.factory.FunctionTypeNode(
		nil,
		[]tsgo.ParameterDeclaration{b.parameter("source", b.objectType())},
		b.objectType(),
	)
	readProjection := b.factory.FunctionTypeNode(
		nil,
		[]tsgo.ParameterDeclaration{b.parameter("source", typePS)},
		typeS,
	)
	writeProjection := b.factory.FunctionTypeNode(
		nil,
		[]tsgo.ParameterDeclaration{
			b.parameter("source", typePS),
			b.parameter("value", typeS),
		},
		b.factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindVoidKeyword),
	)
	return b.method(
		[]tsgo.ModifierLike{b.factory.StaticKeyword()},
		FieldsName,
		[]tsgo.TypeParameterDeclaration{
			b.typeParameter("L", nil),
			b.typeParameter("S", nil),
			b.typeParameter("PL", nil),
			b.typeParameter("PS", b.objectType()),
		},
		[]tsgo.ParameterDeclaration{
			b.parameter("parent", b.pointerType(typePL, typePS)),
			b.parameter("address", addressProjection),
			b.parameter("read", readProjection),
			b.parameter("write", writeProjection),
		},
		b.pointerType(typeL, typeS),
		b.factory.ReturnStatement(
			b.newPointerWithWrite(typeL, typeS, address, read, write),
		),
	)
}

func Fields(
	factory tsgo.Factory,
	runtimeName string,
	logicalType tsgo.TypeNode,
	storageType tsgo.TypeNode,
	parentLogicalType tsgo.TypeNode,
	parentStorageType tsgo.TypeNode,
	parent tsgo.Expression,
	members []string,
) tsgo.CallExpression {
	addressParameter := factory.Identifier("$go$address")
	storageParameter := factory.Identifier("$go$storage")
	nextParameter := factory.Identifier("$go$next")
	address := childAddress(factory, runtimeName, addressParameter, members[0])
	for _, member := range members[1:] {
		address = factory.CallExpression(
			factory.PropertyAccessExpression(
				factory.Identifier(runtimeName),
				nil,
				factory.Identifier(ChildName),
				tsgo.NodeFlagsNone,
			),
			nil,
			nil,
			[]tsgo.Expression{
				address,
				factory.StringLiteral(member, tsgo.TokenFlagsNone),
			},
			tsgo.NodeFlagsNone,
		)
	}
	read := fieldPath(factory, storageParameter, members)
	write := fieldPath(factory, storageParameter, members)
	assignment := factory.BinaryExpression(
		nil,
		write,
		nil,
		factory.BinaryOperatorToken(tsgo.BinaryOperatorEqualsToken),
		nextParameter,
	)
	return factory.CallExpression(
		factory.PropertyAccessExpression(
			factory.Identifier(runtimeName),
			nil,
			factory.Identifier(FieldsName),
			tsgo.NodeFlagsNone,
		),
		nil,
		[]tsgo.TypeNode{
			logicalType,
			storageType,
			parentLogicalType,
			parentStorageType,
		},
		[]tsgo.Expression{
			parent,
			factory.ArrowFunction(
				nil,
				nil,
				[]tsgo.ParameterDeclaration{factory.ParameterDeclaration(
					nil, nil, addressParameter, nil, nil, nil,
				)},
				nil,
				factory.EqualsGreaterThanToken(),
				address,
			),
			factory.ArrowFunction(
				nil,
				nil,
				[]tsgo.ParameterDeclaration{factory.ParameterDeclaration(
					nil, nil, storageParameter, nil, nil, nil,
				)},
				nil,
				factory.EqualsGreaterThanToken(),
				read,
			),
			factory.ArrowFunction(
				nil,
				nil,
				[]tsgo.ParameterDeclaration{
					factory.ParameterDeclaration(
						nil, nil, storageParameter, nil, nil, nil,
					),
					factory.ParameterDeclaration(
						nil, nil, nextParameter, nil, nil, nil,
					),
				},
				nil,
				factory.EqualsGreaterThanToken(),
				assignment,
			),
		},
		tsgo.NodeFlagsNone,
	)
}

func childAddress(
	factory tsgo.Factory,
	runtimeName string,
	parent tsgo.Expression,
	member string,
) tsgo.CallExpression {
	return factory.CallExpression(
		factory.PropertyAccessExpression(
			factory.Identifier(runtimeName),
			nil,
			factory.Identifier(ChildName),
			tsgo.NodeFlagsNone,
		),
		nil,
		nil,
		[]tsgo.Expression{
			parent,
			factory.StringLiteral(member, tsgo.TokenFlagsNone),
		},
		tsgo.NodeFlagsNone,
	)
}

func fieldPath(
	factory tsgo.Factory,
	root tsgo.Expression,
	members []string,
) tsgo.PropertyAccessExpression {
	result := factory.PropertyAccessExpression(
		root,
		nil,
		factory.Identifier(members[0]),
		tsgo.NodeFlagsNone,
	)
	for _, member := range members[1:] {
		result = factory.PropertyAccessExpression(
			result,
			nil,
			factory.Identifier(member),
			tsgo.NodeFlagsNone,
		)
	}
	return result
}
