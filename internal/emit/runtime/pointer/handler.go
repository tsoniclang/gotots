package pointer

import (
	panicruntime "github.com/tsoniclang/gotots/internal/emit/runtime/panic"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

const (
	CellValueName   = "value"
	DereferenceName = "dereference"
)

func Build(
	factory tsgo.Factory,
	className string,
	panicName string,
) tsgo.Statement {
	typeParameter := targetTypeParameter(factory)
	cell := factory.ParameterDeclaration(
		[]tsgo.ModifierLike{factory.PublicKeyword()},
		nil,
		factory.Identifier(CellValueName),
		nil,
		factory.TypeReferenceNode(factory.Identifier("T"), nil),
		nil,
	)
	constructor := factory.ConstructorDeclaration(
		nil,
		nil,
		[]tsgo.ParameterDeclaration{cell},
		nil,
		factory.Block(nil, true),
	)
	pointerType := targetPointerType(factory, className)
	pointer := factory.ParameterDeclaration(
		nil,
		nil,
		factory.Identifier("pointer"),
		nil,
		factory.UnionTypeNode([]tsgo.TypeNode{
			pointerType,
			factory.KeywordTypeNode(
				tsgo.KeywordTypeSyntaxKindUndefinedKeyword,
			),
		}),
		nil,
	)
	nilCondition := factory.BinaryExpression(
		nil,
		factory.Identifier("pointer"),
		nil,
		factory.BinaryOperatorToken(
			tsgo.BinaryOperatorEqualsEqualsEqualsToken,
		),
		factory.VoidExpression(
			factory.NumericLiteral("0", tsgo.TokenFlagsNone),
		),
	)
	nilFailure := factory.ExpressionStatement(
		panicruntime.Call(
			factory,
			panicName,
			factory.StringLiteral(
				"nil pointer dereference",
				tsgo.TokenFlagsNone,
			),
		),
	)
	dereference := factory.MethodDeclaration(
		[]tsgo.ModifierLike{factory.StaticKeyword()},
		nil,
		factory.Identifier(DereferenceName),
		nil,
		[]tsgo.TypeParameterDeclaration{targetTypeParameter(factory)},
		[]tsgo.ParameterDeclaration{pointer},
		pointerType,
		factory.Block(
			[]tsgo.Statement{
				factory.IfStatement(
					nilCondition,
					factory.Block(
						[]tsgo.Statement{nilFailure},
						true,
					),
					nil,
				),
				factory.ReturnStatement(factory.Identifier("pointer")),
			},
			true,
		),
	)
	return factory.ClassDeclaration(
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		factory.Identifier(className),
		[]tsgo.TypeParameterDeclaration{typeParameter},
		nil,
		[]tsgo.ClassElement{constructor, dereference},
	)
}

func CellValue(
	factory tsgo.Factory,
	runtimeName string,
	elementType tsgo.TypeNode,
	pointer tsgo.Expression,
) tsgo.PropertyAccessExpression {
	dereference := factory.CallExpression(
		factory.PropertyAccessExpression(
			factory.Identifier(runtimeName),
			nil,
			factory.Identifier(DereferenceName),
			tsgo.NodeFlagsNone,
		),
		nil,
		[]tsgo.TypeNode{elementType},
		[]tsgo.Expression{pointer},
		tsgo.NodeFlagsNone,
	)
	return factory.PropertyAccessExpression(
		dereference,
		nil,
		factory.Identifier(CellValueName),
		tsgo.NodeFlagsNone,
	)
}

func targetTypeParameter(
	factory tsgo.Factory,
) tsgo.TypeParameterDeclaration {
	return factory.TypeParameterDeclaration(
		nil,
		factory.Identifier("T"),
		nil,
		nil,
		nil,
	)
}

func targetPointerType(
	factory tsgo.Factory,
	className string,
) tsgo.TypeReferenceNode {
	return factory.TypeReferenceNode(
		factory.Identifier(className),
		[]tsgo.TypeNode{
			factory.TypeReferenceNode(factory.Identifier("T"), nil),
		},
	)
}
