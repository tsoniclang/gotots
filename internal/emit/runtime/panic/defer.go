package panicruntime

import "github.com/tsoniclang/gotots/internal/target/tsgo"

func deferPop(
	factory tsgo.Factory,
	functionName string,
	panicName string,
) tsgo.FunctionDeclaration {
	typeName := factory.TypeReferenceNode(factory.Identifier("T"), nil)
	stack := factory.Identifier("stack")
	value := factory.Identifier("value")
	return factory.FunctionDeclaration(
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		nil,
		factory.Identifier(functionName),
		[]tsgo.TypeParameterDeclaration{
			factory.TypeParameterDeclaration(
				nil,
				factory.Identifier("T"),
				nil,
				nil,
				nil,
			),
		},
		[]tsgo.ParameterDeclaration{
			parameter(
				factory,
				"stack",
				factory.ArrayTypeNode(typeName),
			),
		},
		typeName,
		factory.Block(
			[]tsgo.Statement{
				factory.VariableStatement(
					nil,
					factory.VariableDeclarationList(
						[]tsgo.VariableDeclaration{
							factory.VariableDeclaration(
								value,
								nil,
								nil,
								factory.CallExpression(
									factory.PropertyAccessExpression(
										stack,
										nil,
										factory.Identifier("pop"),
										tsgo.NodeFlagsNone,
									),
									nil,
									nil,
									nil,
									tsgo.NodeFlagsNone,
								),
							),
						},
						tsgo.NodeFlagsConst,
					),
				),
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
								Call(
									factory,
									panicName,
									factory.StringLiteral(
										"defer stack underflow",
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
