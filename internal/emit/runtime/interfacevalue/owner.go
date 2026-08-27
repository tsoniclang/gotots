package interfacevalue

import (
	"github.com/tsoniclang/gotots/internal/emit/api"
	interfacecontract "github.com/tsoniclang/gotots/internal/emit/runtime/interfacevalue/contract"
	panicruntime "github.com/tsoniclang/gotots/internal/emit/runtime/panic"
	"github.com/tsoniclang/gotots/internal/emit/typescriptclass"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Build(
	factory tsgo.Factory,
	symbol api.RuntimeSymbol,
	valueName string,
	panicName string,
) (tsgo.Statement, error) {
	switch symbol {
	case api.RuntimeInterfaceNonNil:
		return nonNil(factory, valueName, panicName), nil
	case api.RuntimeInterfaceEqual:
		return equal(factory, valueName), nil
	case api.RuntimeInterfaceFormat:
		return formatClass(factory, panicName), nil
	default:
		return nil, &api.RuntimeSymbolError{Symbol: symbol}
	}
}

func BuildValue(
	factory tsgo.Factory,
	symbol api.RuntimeSymbol,
	valueName string,
) (tsgo.Statement, error) {
	switch symbol {
	case api.RuntimeInterfaceValue:
		return valueContract(factory, valueName), nil
	case api.RuntimeInterfaceAdapterFactory:
		name, err := runtimeName(api.RuntimeInterfaceAdapterFactory)
		if err != nil {
			return nil, err
		}
		return adapterFactoryFunction(factory, name, valueName), nil
	case api.RuntimeProviderInterfaceBridge:
		return providerBridgeContract(factory, valueName), nil
	case api.RuntimeErrorMethodToken:
		return methodToken(factory, "GoErrorMethodToken"), nil
	case api.RuntimeRuntimeErrorToken:
		return methodToken(factory, "GoRuntimeErrorMethodToken"), nil
	case api.RuntimeBuiltinErrorType:
		return errorTypeDefinition(factory, false)
	case api.RuntimeBuiltinErrorContract:
		return errorContractDefinition(factory, false)
	case api.RuntimeBuiltinErrorGuard:
		return errorGuardDefinition(factory, false)
	case api.RuntimeErrorType:
		return errorTypeDefinition(factory, true)
	case api.RuntimeErrorContract:
		return errorContractDefinition(factory, true)
	case api.RuntimeErrorGuard:
		return errorGuardDefinition(factory, true)
	default:
		return nil, &api.RuntimeSymbolError{Symbol: symbol}
	}
}

func errorTypeDefinition(
	factory tsgo.Factory,
	runtimeError bool,
) (tsgo.Statement, error) {
	symbol := api.RuntimeBuiltinErrorType
	if runtimeError {
		symbol = api.RuntimeErrorType
	}
	name, err := runtimeName(symbol)
	if err != nil {
		return nil, err
	}
	valueName, err := runtimeName(api.RuntimeInterfaceValue)
	if err != nil {
		return nil, err
	}
	return errorInterface(
		factory,
		name,
		valueName,
		runtimeError,
	), nil
}

func errorContractDefinition(
	factory tsgo.Factory,
	runtimeError bool,
) (tsgo.Statement, error) {
	symbol := api.RuntimeBuiltinErrorContract
	if runtimeError {
		symbol = api.RuntimeErrorContract
	}
	name, err := runtimeName(symbol)
	if err != nil {
		return nil, err
	}
	errorToken, err := runtimeName(api.RuntimeErrorMethodToken)
	if err != nil {
		return nil, err
	}
	runtimeToken := ""
	if runtimeError {
		runtimeToken, err = runtimeName(api.RuntimeRuntimeErrorToken)
		if err != nil {
			return nil, err
		}
	}
	return errorContract(factory, name, errorToken, runtimeToken), nil
}

func errorGuardDefinition(
	factory tsgo.Factory,
	runtimeError bool,
) (tsgo.Statement, error) {
	guardSymbol := api.RuntimeBuiltinErrorGuard
	typeSymbol := api.RuntimeBuiltinErrorType
	contractSymbol := api.RuntimeBuiltinErrorContract
	if runtimeError {
		guardSymbol = api.RuntimeErrorGuard
		typeSymbol = api.RuntimeErrorType
		contractSymbol = api.RuntimeErrorContract
	}
	name, err := runtimeName(guardSymbol)
	if err != nil {
		return nil, err
	}
	typeName, err := runtimeName(typeSymbol)
	if err != nil {
		return nil, err
	}
	valueName, err := runtimeName(api.RuntimeInterfaceValue)
	if err != nil {
		return nil, err
	}
	contractName, err := runtimeName(contractSymbol)
	if err != nil {
		return nil, err
	}
	return errorGuard(factory, name, typeName, valueName, contractName), nil
}

func methodToken(
	factory tsgo.Factory,
	name string,
) tsgo.VariableStatement {
	return factory.VariableStatement(
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		factory.VariableDeclarationList(
			[]tsgo.VariableDeclaration{
				factory.VariableDeclaration(
					factory.Identifier(name),
					nil,
					factory.KeywordTypeNode(
						tsgo.KeywordTypeSyntaxKindObjectKeyword,
					),
					factory.CallExpression(
						factory.PropertyAccessExpression(
							factory.Identifier("Object"),
							nil,
							factory.Identifier("freeze"),
							tsgo.NodeFlagsNone,
						),
						nil,
						nil,
						[]tsgo.Expression{
							factory.ObjectLiteralExpression(nil, false),
						},
						tsgo.NodeFlagsNone,
					),
				),
			},
			tsgo.NodeFlagsConst,
		),
	)
}

func equal(
	factory tsgo.Factory,
	valueName string,
) tsgo.FunctionDeclaration {
	valueType := factory.UnionTypeNode([]tsgo.TypeNode{
		factory.TypeReferenceNode(
			factory.Identifier(valueName),
			nil,
		),
		factory.KeywordTypeNode(
			tsgo.KeywordTypeSyntaxKindUndefinedKeyword,
		),
	})
	left := factory.Identifier("left")
	right := factory.Identifier("right")
	return factory.FunctionDeclaration(
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		nil,
		factory.Identifier("goInterfaceEqual"),
		nil,
		[]tsgo.ParameterDeclaration{
			parameter(factory, "left", valueType),
			parameter(factory, "right", valueType),
		},
		factory.KeywordTypeNode(
			tsgo.KeywordTypeSyntaxKindBooleanKeyword,
		),
		factory.Block(
			[]tsgo.Statement{
				factory.ReturnStatement(
					factory.ConditionalExpression(
						strictUndefined(factory, left),
						factory.QuestionToken(),
						strictUndefined(factory, right),
						factory.ColonToken(),
						factory.BinaryExpression(
							nil,
							strictDefined(factory, right),
							nil,
							factory.BinaryOperatorToken(
								tsgo.BinaryOperatorAmpersandAmpersandToken,
							),
							factory.CallExpression(
								factory.PropertyAccessExpression(
									left,
									nil,
									factory.Identifier(interfacecontract.EqualMember),
									tsgo.NodeFlagsNone,
								),
								nil,
								nil,
								[]tsgo.Expression{right},
								tsgo.NodeFlagsNone,
							),
						),
					),
				),
			},
			true,
		),
	)
}

func strictUndefined(
	factory tsgo.Factory,
	value tsgo.Expression,
) tsgo.BinaryExpression {
	return factory.BinaryExpression(
		nil,
		value,
		nil,
		factory.BinaryOperatorToken(
			tsgo.BinaryOperatorEqualsEqualsEqualsToken,
		),
		factory.Identifier("undefined"),
	)
}

func strictDefined(
	factory tsgo.Factory,
	value tsgo.Expression,
) tsgo.BinaryExpression {
	return factory.BinaryExpression(
		nil,
		value,
		nil,
		factory.BinaryOperatorToken(
			tsgo.BinaryOperatorExclamationEqualsEqualsToken,
		),
		factory.Identifier("undefined"),
	)
}

func valueContract(
	factory tsgo.Factory,
	name string,
) tsgo.ClassDeclaration {
	return typescriptclass.Declaration(factory,
		[]tsgo.ModifierLike{
			factory.ExportKeyword(),
			factory.AbstractKeyword(),
		},
		factory.Identifier(name),
		nil,
		nil,
		[]tsgo.ClassElement{
			factory.PropertyDeclaration(
				[]tsgo.ModifierLike{
					factory.AbstractKeyword(),
					factory.ReadonlyKeyword(),
				},
				factory.Identifier(interfacecontract.DynamicTypeMember),
				nil,
				interfacecontract.DynamicType(factory),
				nil,
			),
			factory.PropertyDeclaration(
				[]tsgo.ModifierLike{
					factory.AbstractKeyword(),
					factory.ReadonlyKeyword(),
				},
				factory.Identifier(interfacecontract.MethodsMember),
				nil,
				readonlySetType(factory),
				nil,
			),
			factory.MethodDeclaration(
				[]tsgo.ModifierLike{factory.AbstractKeyword()},
				nil,
				factory.Identifier(interfacecontract.ImplementsMember),
				nil,
				nil,
				[]tsgo.ParameterDeclaration{
					parameter(
						factory,
						"contract",
						readonlyObjectArray(factory),
					),
				},
				factory.KeywordTypeNode(
					tsgo.KeywordTypeSyntaxKindBooleanKeyword,
				),
				nil,
			),
			factory.MethodDeclaration(
				[]tsgo.ModifierLike{factory.AbstractKeyword()},
				nil,
				factory.Identifier(interfacecontract.EqualMember),
				nil,
				nil,
				[]tsgo.ParameterDeclaration{
					parameter(
						factory,
						"other",
						factory.TypeReferenceNode(
							factory.Identifier(name),
							nil,
						),
					),
				},
				factory.KeywordTypeNode(
					tsgo.KeywordTypeSyntaxKindBooleanKeyword,
				),
				nil,
			),
			factory.MethodDeclaration(
				[]tsgo.ModifierLike{factory.AbstractKeyword()},
				nil,
				factory.Identifier(interfacecontract.HashMember),
				nil,
				nil,
				nil,
				factory.KeywordTypeNode(
					tsgo.KeywordTypeSyntaxKindNumberKeyword,
				),
				nil,
			),
			factory.PropertyDeclaration(
				[]tsgo.ModifierLike{
					factory.AbstractKeyword(),
					factory.ReadonlyKeyword(),
				},
				factory.Identifier(interfacecontract.FormatStringMember),
				nil,
				factory.KeywordTypeNode(
					tsgo.KeywordTypeSyntaxKindBooleanKeyword,
				),
				nil,
			),
			factory.MethodDeclaration(
				[]tsgo.ModifierLike{factory.AbstractKeyword()},
				nil,
				factory.Identifier(interfacecontract.FormatMember),
				nil,
				nil,
				[]tsgo.ParameterDeclaration{
					parameter(
						factory,
						"verb",
						factory.KeywordTypeNode(
							tsgo.KeywordTypeSyntaxKindStringKeyword,
						),
					),
					parameter(
						factory,
						"flags",
						factory.KeywordTypeNode(
							tsgo.KeywordTypeSyntaxKindStringKeyword,
						),
					),
					parameter(
						factory,
						"precision",
						factory.UnionTypeNode([]tsgo.TypeNode{
							factory.KeywordTypeNode(
								tsgo.KeywordTypeSyntaxKindNumberKeyword,
							),
							factory.KeywordTypeNode(
								tsgo.KeywordTypeSyntaxKindUndefinedKeyword,
							),
						}),
					),
				},
				factory.KeywordTypeNode(
					tsgo.KeywordTypeSyntaxKindStringKeyword,
				),
				nil,
			),
		},
	)
}

func nonNil(
	factory tsgo.Factory,
	valueName string,
	panicName string,
) tsgo.FunctionDeclaration {
	typeName := factory.TypeReferenceNode(factory.Identifier("T"), nil)
	value := factory.Identifier("value")
	return factory.FunctionDeclaration(
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		nil,
		factory.Identifier("goInterfaceNonNil"),
		[]tsgo.TypeParameterDeclaration{
			factory.TypeParameterDeclaration(
				nil,
				factory.Identifier("T"),
				factory.TypeReferenceNode(
					factory.Identifier(valueName),
					nil,
				),
				nil,
				nil,
			),
		},
		[]tsgo.ParameterDeclaration{
			parameter(
				factory,
				"value",
				factory.UnionTypeNode([]tsgo.TypeNode{
					typeName,
					factory.KeywordTypeNode(
						tsgo.KeywordTypeSyntaxKindUndefinedKeyword,
					),
				}),
			),
		},
		typeName,
		factory.Block(
			[]tsgo.Statement{
				factory.IfStatement(
					factory.BinaryExpression(
						nil,
						value,
						nil,
						factory.BinaryOperatorToken(
							tsgo.BinaryOperatorEqualsEqualsEqualsToken,
						),
						factory.Identifier("undefined"),
					),
					factory.Block(
						[]tsgo.Statement{
							factory.ExpressionStatement(
								panicruntime.Call(
									factory,
									panicName,
									factory.StringLiteral(
										"runtime error: invalid memory address or nil pointer dereference",
										tsgo.TokenFlagsNone,
									),
								),
							),
						},
						true,
					),
					nil,
				),
				factory.ReturnStatement(value),
			},
			true,
		),
	)
}

func readonlySetType(factory tsgo.Factory) tsgo.TypeNode {
	return factory.TypeReferenceNode(
		factory.Identifier("ReadonlySet"),
		[]tsgo.TypeNode{
			factory.KeywordTypeNode(
				tsgo.KeywordTypeSyntaxKindObjectKeyword,
			),
		},
	)
}

func readonlyObjectArray(factory tsgo.Factory) tsgo.TypeNode {
	return factory.TypeOperatorNode(
		tsgo.TypeOperatorNodeOperatorKindReadonlyKeyword,
		factory.ArrayTypeNode(
			factory.KeywordTypeNode(
				tsgo.KeywordTypeSyntaxKindObjectKeyword,
			),
		),
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
