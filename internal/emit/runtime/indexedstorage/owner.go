package indexedstorage

import (
	panicruntime "github.com/tsoniclang/gotots/internal/emit/runtime/panic"
	"github.com/tsoniclang/gotots/internal/emit/typescriptclass"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

const (
	getMember = "get"
)

func Build(
	factory tsgo.Factory,
	className string,
	panicName string,
) tsgo.ClassDeclaration {
	return typescriptclass.Declaration(factory,
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		factory.Identifier(className),
		nil,
		nil,
		[]tsgo.ClassElement{
			getMethod(factory, panicName),
		},
	)
}

func Element(
	factory tsgo.Factory,
	className string,
	values tsgo.Expression,
	index tsgo.Expression,
) tsgo.CallExpression {
	return factory.CallExpression(
		factory.PropertyAccessExpression(
			factory.Identifier(className),
			nil,
			factory.Identifier(getMember),
			tsgo.NodeFlagsNone,
		),
		nil,
		nil,
		[]tsgo.Expression{values, index},
		tsgo.NodeFlagsNone,
	)
}

func getMethod(
	factory tsgo.Factory,
	panicName string,
) tsgo.MethodDeclaration {
	typeT := factory.TypeReferenceNode(factory.Identifier("T"), nil)
	value := factory.Identifier("value")
	present := factory.BinaryExpression(
		nil,
		factory.Identifier("index"),
		nil,
		factory.BinaryOperatorToken(tsgo.BinaryOperatorInKeyword),
		factory.Identifier("values"),
	)
	return factory.MethodDeclaration(
		[]tsgo.ModifierLike{
			factory.PublicKeyword(),
			factory.StaticKeyword(),
		},
		nil,
		factory.Identifier(getMember),
		nil,
		[]tsgo.TypeParameterDeclaration{typeParameter(factory)},
		[]tsgo.ParameterDeclaration{
			parameter(factory, "values", readonlyArray(factory, typeT)),
			parameter(
				factory,
				"index",
				factory.KeywordTypeNode(
					tsgo.KeywordTypeSyntaxKindNumberKeyword,
				),
			),
		},
		typeT,
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
								factory.ElementAccessExpression(
									factory.Identifier("values"),
									nil,
									factory.Identifier("index"),
									tsgo.NodeFlagsNone,
								),
							),
						},
						tsgo.NodeFlagsConst,
					),
				),
				factory.IfStatement(
					factory.PrefixUnaryExpression(
						tsgo.PrefixUnaryExpressionOperatorKindExclamationToken,
						present,
					),
					factory.ExpressionStatement(panicruntime.Call(
						factory,
						panicName,
						factory.StringLiteral(
							"dense storage index is absent",
							tsgo.TokenFlagsNone,
						),
					)),
					nil,
				),
				factory.ReturnStatement(factory.AsExpression(value, typeT)),
			},
			true,
		),
	)
}

func typeParameter(factory tsgo.Factory) tsgo.TypeParameterDeclaration {
	return factory.TypeParameterDeclaration(
		nil,
		factory.Identifier("T"),
		nil,
		nil,
		nil,
	)
}

func readonlyArray(
	factory tsgo.Factory,
	element tsgo.TypeNode,
) tsgo.TypeNode {
	return factory.TypeOperatorNode(
		tsgo.TypeOperatorNodeOperatorKindReadonlyKeyword,
		factory.ArrayTypeNode(element),
	)
}

func parameter(
	factory tsgo.Factory,
	name string,
	targetType tsgo.TypeNode,
) tsgo.ParameterDeclaration {
	return factory.ParameterDeclaration(
		nil,
		nil,
		factory.Identifier(name),
		nil,
		targetType,
		nil,
	)
}
