package interfacevalue

import (
	"github.com/tsoniclang/gotots/internal/emit/api"
	interfacecontract "github.com/tsoniclang/gotots/internal/emit/runtime/interfacevalue/contract"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func errorInterface(
	factory tsgo.Factory,
	name string,
	valueName string,
	runtimeError bool,
) tsgo.InterfaceDeclaration {
	members := []tsgo.TypeElement{
		factory.MethodSignatureDeclaration(
			nil,
			factory.Identifier("Error"),
			nil,
			nil,
			nil,
			factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindStringKeyword),
		),
	}
	if runtimeError {
		members = append(
			members,
			factory.MethodSignatureDeclaration(
				nil,
				factory.Identifier("RuntimeError"),
				nil,
				nil,
				nil,
				factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindVoidKeyword),
			),
		)
	}
	return factory.InterfaceDeclaration(
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		factory.Identifier(name),
		nil,
		[]tsgo.HeritageClause{
			factory.HeritageClause(
				tsgo.HeritageClauseTokenKindExtendsKeyword,
				[]tsgo.ExpressionWithTypeArguments{
					factory.ExpressionWithTypeArguments(
						factory.Identifier(valueName),
						nil,
					),
				},
			),
		},
		members,
	)
}

func errorContract(
	factory tsgo.Factory,
	name string,
	errorTokenName string,
	runtimeErrorTokenName string,
) tsgo.VariableStatement {
	tokens := []tsgo.Expression{factory.Identifier(errorTokenName)}
	if runtimeErrorTokenName != "" {
		tokens = append(tokens, factory.Identifier(runtimeErrorTokenName))
	}
	return factory.VariableStatement(
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		factory.VariableDeclarationList(
			[]tsgo.VariableDeclaration{
				factory.VariableDeclaration(
					factory.Identifier(name),
					nil,
					readonlyObjectArray(factory),
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
							factory.ArrayLiteralExpression(tokens, false),
						},
						tsgo.NodeFlagsNone,
					),
				),
			},
			tsgo.NodeFlagsConst,
		),
	)
}

func errorGuard(
	factory tsgo.Factory,
	name string,
	interfaceName string,
	valueName string,
	contractName string,
) tsgo.FunctionDeclaration {
	value := factory.Identifier("value")
	return factory.FunctionDeclaration(
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		nil,
		factory.Identifier(name),
		nil,
		[]tsgo.ParameterDeclaration{
			parameter(
				factory,
				"value",
				factory.UnionTypeNode([]tsgo.TypeNode{
					factory.TypeReferenceNode(
						factory.Identifier(valueName),
						nil,
					),
					factory.KeywordTypeNode(
						tsgo.KeywordTypeSyntaxKindUndefinedKeyword,
					),
				}),
			),
		},
		factory.TypePredicateNode(
			nil,
			value,
			factory.TypeReferenceNode(
				factory.Identifier(interfaceName),
				nil,
			),
		),
		factory.Block(
			[]tsgo.Statement{
				factory.ReturnStatement(
					factory.BinaryExpression(
						nil,
						strictDefined(factory, value),
						nil,
						factory.BinaryOperatorToken(
							tsgo.BinaryOperatorAmpersandAmpersandToken,
						),
						factory.CallExpression(
							factory.PropertyAccessExpression(
								value,
								nil,
								factory.Identifier(
									interfacecontract.ImplementsMember,
								),
								tsgo.NodeFlagsNone,
							),
							nil,
							nil,
							[]tsgo.Expression{
								factory.Identifier(contractName),
							},
							tsgo.NodeFlagsNone,
						),
					),
				),
			},
			true,
		),
	)
}

func runtimeName(symbol api.RuntimeSymbol) (string, error) {
	contract, err := api.RuntimeContract(symbol)
	if err != nil {
		return "", err
	}
	return contract.ExportedName(), nil
}
