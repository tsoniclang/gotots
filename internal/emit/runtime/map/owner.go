package mapruntime

import (
	"github.com/tsoniclang/gotots/internal/emit/api"
	panicruntime "github.com/tsoniclang/gotots/internal/emit/runtime/panic"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

const (
	keyTypeName   = "K"
	valueTypeName = "V"
	zeroName      = "zeroValue"
	valuesName    = "values"
	keyName       = "key"
	valueName     = "value"
	sizeName      = "size"
	entriesName   = "entries"
)

type memberNames struct {
	nilMember  string
	makeMember string
	lookup     string
	lookupOK   string
	store      string
	delete     string
	length     string
	isNil      string
}

func resolveMemberNames() (memberNames, error) {
	resolved := make([]string, 0, MemberIsNil)
	for member := MemberNil; member <= MemberIsNil; member++ {
		name, err := Name(member)
		if err != nil {
			return memberNames{}, err
		}
		resolved = append(resolved, name)
	}
	return memberNames{
		nilMember:  resolved[0],
		makeMember: resolved[1],
		lookup:     resolved[2],
		lookupOK:   resolved[3],
		store:      resolved[4],
		delete:     resolved[5],
		length:     resolved[6],
		isNil:      resolved[7],
	}, nil
}

func Build(
	factory tsgo.Factory,
	symbol api.RuntimeSymbol,
	panicName string,
) (tsgo.Statement, error) {
	contract, err := api.RuntimeContract(symbol)
	if err != nil {
		return nil, err
	}
	if symbol != api.RuntimeMap ||
		contract.Module() != api.RuntimeModuleMap {
		return nil, &api.RuntimeSymbolError{Symbol: symbol}
	}
	members, err := resolveMemberNames()
	if err != nil {
		return nil, err
	}
	className := contract.ExportedName()
	keyType := typeName(factory, keyTypeName)
	valueType := typeName(factory, valueTypeName)

	return factory.ClassDeclaration(
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		factory.Identifier(className),
		[]tsgo.TypeParameterDeclaration{
			typeParameter(factory, keyTypeName),
			typeParameter(factory, valueTypeName),
		},
		nil,
		[]tsgo.ClassElement{
			constructor(factory, keyType, valueType),
			nilMethod(factory, className, members.nilMember),
			makeMethod(factory, className, members.makeMember),
			lookupMethod(factory, valueType, members.lookup),
			lookupOKMethod(factory, valueType, members.lookupOK),
			storeMethod(factory, valueType, members.store, panicName),
			deleteMethod(factory, members.delete),
			lengthMethod(factory, members.length),
			nilStateMethod(factory, members.isNil),
		},
	), nil
}

func constructor(
	factory tsgo.Factory,
	keyType tsgo.TypeNode,
	valueType tsgo.TypeNode,
) tsgo.ConstructorDeclaration {
	return factory.ConstructorDeclaration(
		[]tsgo.ModifierLike{factory.PrivateKeyword()},
		nil,
		[]tsgo.ParameterDeclaration{
			parameterProperty(factory, zeroName, valueType),
			parameterProperty(
				factory,
				valuesName,
				factory.UnionTypeNode([]tsgo.TypeNode{
					nativeMapType(factory, keyType, valueType),
					factory.KeywordTypeNode(
						tsgo.KeywordTypeSyntaxKindUndefinedKeyword,
					),
				}),
			),
		},
		nil,
		factory.Block(nil, true),
	)
}

func nilMethod(
	factory tsgo.Factory,
	className string,
	memberName string,
) tsgo.MethodDeclaration {
	keyType := typeName(factory, keyTypeName)
	valueType := typeName(factory, valueTypeName)
	return factory.MethodDeclaration(
		[]tsgo.ModifierLike{
			factory.StaticKeyword(),
		},
		nil,
		factory.Identifier(memberName),
		nil,
		[]tsgo.TypeParameterDeclaration{
			typeParameter(factory, keyTypeName),
			typeParameter(factory, valueTypeName),
		},
		[]tsgo.ParameterDeclaration{
			parameter(factory, zeroName, valueType),
		},
		runtimeMapType(factory, className, keyType, valueType),
		factory.Block([]tsgo.Statement{
			factory.ReturnStatement(
				factory.NewExpression(
					factory.Identifier(className),
					[]tsgo.TypeNode{keyType, valueType},
					[]tsgo.Expression{
						factory.Identifier(zeroName),
						factory.Identifier("undefined"),
					},
				),
			),
		}, true),
	)
}

func makeMethod(
	factory tsgo.Factory,
	className string,
	memberName string,
) tsgo.MethodDeclaration {
	keyType := typeName(factory, keyTypeName)
	valueType := typeName(factory, valueTypeName)
	return factory.MethodDeclaration(
		[]tsgo.ModifierLike{
			factory.StaticKeyword(),
		},
		nil,
		factory.Identifier(memberName),
		nil,
		[]tsgo.TypeParameterDeclaration{
			typeParameter(factory, keyTypeName),
			typeParameter(factory, valueTypeName),
		},
		[]tsgo.ParameterDeclaration{
			parameter(factory, zeroName, valueType),
			parameter(
				factory,
				sizeName,
				factory.UnionTypeNode([]tsgo.TypeNode{
					factory.KeywordTypeNode(
						tsgo.KeywordTypeSyntaxKindNumberKeyword,
					),
					factory.KeywordTypeNode(
						tsgo.KeywordTypeSyntaxKindBigIntKeyword,
					),
				}),
			),
			parameter(
				factory,
				entriesName,
				factory.ArrayTypeNode(
					factory.TupleTypeNode(
						[]tsgo.TypeNode{keyType, valueType},
					),
				),
			),
		},
		runtimeMapType(factory, className, keyType, valueType),
		factory.Block([]tsgo.Statement{
			factory.ReturnStatement(
				factory.NewExpression(
					factory.Identifier(className),
					[]tsgo.TypeNode{keyType, valueType},
					[]tsgo.Expression{
						factory.Identifier(zeroName),
						factory.NewExpression(
							factory.Identifier("Map"),
							[]tsgo.TypeNode{keyType, valueType},
							[]tsgo.Expression{
								factory.Identifier(entriesName),
							},
						),
					},
				),
			),
		}, true),
	)
}

func lookupMethod(
	factory tsgo.Factory,
	valueType tsgo.TypeNode,
	memberName string,
) tsgo.MethodDeclaration {
	return factory.MethodDeclaration(
		nil,
		nil,
		factory.Identifier(memberName),
		nil,
		nil,
		[]tsgo.ParameterDeclaration{
			parameter(factory, keyName, typeName(factory, keyTypeName)),
		},
		valueType,
		factory.Block(lookupStatements(factory, false), true),
	)
}

func lookupOKMethod(
	factory tsgo.Factory,
	valueType tsgo.TypeNode,
	memberName string,
) tsgo.MethodDeclaration {
	tupleType := factory.TupleTypeNode([]tsgo.TypeNode{
		valueType,
		factory.KeywordTypeNode(
			tsgo.KeywordTypeSyntaxKindBooleanKeyword,
		),
	})
	return factory.MethodDeclaration(
		nil,
		nil,
		factory.Identifier(memberName),
		nil,
		nil,
		[]tsgo.ParameterDeclaration{
			parameter(factory, keyName, typeName(factory, keyTypeName)),
		},
		tupleType,
		factory.Block(lookupStatements(factory, true), true),
	)
}

func storeMethod(
	factory tsgo.Factory,
	valueType tsgo.TypeNode,
	memberName string,
	panicName string,
) tsgo.MethodDeclaration {
	return factory.MethodDeclaration(
		nil,
		nil,
		factory.Identifier(memberName),
		nil,
		nil,
		[]tsgo.ParameterDeclaration{
			parameter(factory, keyName, typeName(factory, keyTypeName)),
			parameter(factory, valueName, valueType),
		},
		factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindVoidKeyword),
		factory.Block([]tsgo.Statement{
			nilWriteGuard(factory, panicName),
			factory.ExpressionStatement(
				methodCall(
					factory,
					field(factory, valuesName),
					"set",
					factory.Identifier(keyName),
					factory.Identifier(valueName),
				),
			),
		}, true),
	)
}

func deleteMethod(
	factory tsgo.Factory,
	memberName string,
) tsgo.MethodDeclaration {
	return factory.MethodDeclaration(
		nil,
		nil,
		factory.Identifier(memberName),
		nil,
		nil,
		[]tsgo.ParameterDeclaration{
			parameter(factory, keyName, typeName(factory, keyTypeName)),
		},
		factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindVoidKeyword),
		factory.Block([]tsgo.Statement{
			factory.IfStatement(
				definedValues(factory),
				factory.Block([]tsgo.Statement{
					factory.ExpressionStatement(
						methodCall(
							factory,
							field(factory, valuesName),
							"delete",
							factory.Identifier(keyName),
						),
					),
				}, true),
				nil,
			),
		}, true),
	)
}

func lengthMethod(
	factory tsgo.Factory,
	memberName string,
) tsgo.MethodDeclaration {
	return factory.MethodDeclaration(
		nil,
		nil,
		factory.Identifier(memberName),
		nil,
		nil,
		nil,
		factory.KeywordTypeNode(
			tsgo.KeywordTypeSyntaxKindNumberKeyword,
		),
		factory.Block([]tsgo.Statement{
			factory.ReturnStatement(
				factory.ConditionalExpression(
					definedValues(factory),
					factory.QuestionToken(),
					factory.PropertyAccessExpression(
						field(factory, valuesName),
						nil,
						factory.Identifier("size"),
						tsgo.NodeFlagsNone,
					),
					factory.ColonToken(),
					factory.NumericLiteral("0", tsgo.TokenFlagsNone),
				),
			),
		}, true),
	)
}

func nilStateMethod(
	factory tsgo.Factory,
	memberName string,
) tsgo.MethodDeclaration {
	return factory.MethodDeclaration(
		nil,
		nil,
		factory.Identifier(memberName),
		nil,
		nil,
		nil,
		factory.KeywordTypeNode(
			tsgo.KeywordTypeSyntaxKindBooleanKeyword,
		),
		factory.Block([]tsgo.Statement{
			factory.ReturnStatement(
				factory.BinaryExpression(
					nil,
					field(factory, valuesName),
					nil,
					factory.BinaryOperatorToken(
						tsgo.BinaryOperatorEqualsEqualsEqualsToken,
					),
					factory.Identifier("undefined"),
				),
			),
		}, true),
	)
}

func nilWriteGuard(
	factory tsgo.Factory,
	panicName string,
) tsgo.IfStatement {
	return factory.IfStatement(
		factory.BinaryExpression(
			nil,
			field(factory, valuesName),
			nil,
			factory.BinaryOperatorToken(
				tsgo.BinaryOperatorEqualsEqualsEqualsToken,
			),
			factory.Identifier("undefined"),
		),
		factory.Block([]tsgo.Statement{
			factory.ExpressionStatement(
				panicruntime.Call(
					factory,
					panicName,
					factory.StringLiteral(
						"assignment to entry in nil map",
						tsgo.TokenFlagsNone,
					),
				),
			),
		}, true),
		nil,
	)
}

func definedValues(factory tsgo.Factory) tsgo.Expression {
	return factory.BinaryExpression(
		nil,
		field(factory, valuesName),
		nil,
		factory.BinaryOperatorToken(
			tsgo.BinaryOperatorExclamationEqualsEqualsToken,
		),
		factory.Identifier("undefined"),
	)
}

func parameterProperty(
	factory tsgo.Factory,
	name string,
	typeNode tsgo.TypeNode,
) tsgo.ParameterDeclaration {
	return factory.ParameterDeclaration(
		[]tsgo.ModifierLike{
			factory.PrivateKeyword(),
			factory.ReadonlyKeyword(),
		},
		nil,
		factory.Identifier(name),
		nil,
		typeNode,
		nil,
	)
}

func parameter(
	factory tsgo.Factory,
	name string,
	typeNode tsgo.TypeNode,
) tsgo.ParameterDeclaration {
	return factory.ParameterDeclaration(
		nil,
		nil,
		factory.Identifier(name),
		nil,
		typeNode,
		nil,
	)
}

func typeParameter(
	factory tsgo.Factory,
	name string,
) tsgo.TypeParameterDeclaration {
	return factory.TypeParameterDeclaration(
		nil,
		factory.Identifier(name),
		scalarConstraint(factory),
		nil,
		nil,
	)
}

func scalarConstraint(factory tsgo.Factory) tsgo.TypeNode {
	return factory.UnionTypeNode([]tsgo.TypeNode{
		factory.KeywordTypeNode(
			tsgo.KeywordTypeSyntaxKindBooleanKeyword,
		),
		factory.KeywordTypeNode(
			tsgo.KeywordTypeSyntaxKindNumberKeyword,
		),
		factory.KeywordTypeNode(
			tsgo.KeywordTypeSyntaxKindBigIntKeyword,
		),
		factory.KeywordTypeNode(
			tsgo.KeywordTypeSyntaxKindStringKeyword,
		),
	})
}

func typeName(factory tsgo.Factory, name string) tsgo.TypeNode {
	return factory.TypeReferenceNode(factory.Identifier(name), nil)
}

func runtimeMapType(
	factory tsgo.Factory,
	className string,
	keyType tsgo.TypeNode,
	valueType tsgo.TypeNode,
) tsgo.TypeNode {
	return factory.TypeReferenceNode(
		factory.Identifier(className),
		[]tsgo.TypeNode{keyType, valueType},
	)
}

func nativeMapType(
	factory tsgo.Factory,
	keyType tsgo.TypeNode,
	valueType tsgo.TypeNode,
) tsgo.TypeNode {
	return factory.TypeReferenceNode(
		factory.Identifier("Map"),
		[]tsgo.TypeNode{keyType, valueType},
	)
}

func methodCall(
	factory tsgo.Factory,
	receiver tsgo.Expression,
	name string,
	arguments ...tsgo.Expression,
) tsgo.CallExpression {
	return factory.CallExpression(
		factory.PropertyAccessExpression(
			receiver,
			nil,
			factory.Identifier(name),
			tsgo.NodeFlagsNone,
		),
		nil,
		nil,
		arguments,
		tsgo.NodeFlagsNone,
	)
}

func field(factory tsgo.Factory, name string) tsgo.PropertyAccessExpression {
	return factory.PropertyAccessExpression(
		factory.ThisExpression(),
		nil,
		factory.Identifier(name),
		tsgo.NodeFlagsNone,
	)
}
