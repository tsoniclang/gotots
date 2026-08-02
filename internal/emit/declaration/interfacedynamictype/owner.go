package interfacedynamictype

import (
	interfacecontract "github.com/tsoniclang/gotots/internal/emit/runtime/interfacevalue/contract"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Build(
	factory tsgo.Factory,
	name string,
	modifiers []tsgo.ModifierLike,
	comparable bool,
) tsgo.VariableStatement {
	comparableValue := tsgo.Expression(factory.FalseLiteral())
	if comparable {
		comparableValue = factory.TrueLiteral()
	}
	return factory.VariableStatement(
		modifiers,
		factory.VariableDeclarationList(
			[]tsgo.VariableDeclaration{
				factory.VariableDeclaration(
					factory.Identifier(name),
					nil,
					interfacecontract.DynamicType(factory),
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
							factory.ObjectLiteralExpression(
								[]tsgo.ObjectLiteralElementLike{
									factory.PropertyAssignment(
										nil,
										factory.Identifier(interfacecontract.DynamicTypeComparable),
										nil,
										factory.KeywordTypeNode(
											tsgo.KeywordTypeSyntaxKindBooleanKeyword,
										),
										comparableValue,
									),
								},
								false,
							),
						},
						tsgo.NodeFlagsNone,
					),
				),
			},
			tsgo.NodeFlagsConst,
		),
	)
}
