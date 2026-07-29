package definedtype

import (
	"go/ast"

	"github.com/tsoniclang/gotots/internal/emit/api"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func emitReferenceMembers(
	context api.Context,
	source *ast.TypeSpec,
	model definedtype.Model,
	className string,
	underlying tsgo.TypeNode,
) ([]tsgo.ClassElement, []api.RootRequest, error) {
	classType := context.Factory().TypeReferenceNode(
		context.Factory().Identifier(className),
		nil,
	)
	optionalClassType := context.Factory().UnionTypeNode([]tsgo.TypeNode{
		classType,
		context.Factory().KeywordTypeNode(
			tsgo.KeywordTypeSyntaxKindUndefinedKeyword,
		),
	})
	value := context.Factory().Identifier("value")
	isNil, err := referenceNilCondition(context, model, value)
	if err != nil {
		return nil, nil, err
	}
	from := context.Factory().MethodDeclaration(
		[]tsgo.ModifierLike{
			context.Factory().PublicKeyword(),
			context.Factory().StaticKeyword(),
		},
		nil,
		context.Factory().Identifier(definedtype.FromMember),
		nil,
		nil,
		[]tsgo.ParameterDeclaration{referenceParameter(
			context,
			value,
			underlying,
		)},
		optionalClassType,
		context.Factory().Block(
			[]tsgo.Statement{context.Factory().ReturnStatement(
				context.Factory().ConditionalExpression(
					isNil,
					context.Factory().QuestionToken(),
					context.Factory().Identifier("undefined"),
					context.Factory().ColonToken(),
					context.Factory().NewExpression(
						context.Factory().Identifier(className),
						nil,
						[]tsgo.Expression{value},
					),
				),
			)},
			true,
		),
	)
	zero, err := context.Values().Zero(
		context.WithRole(api.RoleDefinedUnderlyingType),
		source,
		model.Underlying(),
	)
	if err != nil {
		return nil, nil, err
	}
	valueOf := context.Factory().MethodDeclaration(
		[]tsgo.ModifierLike{
			context.Factory().PublicKeyword(),
			context.Factory().StaticKeyword(),
		},
		nil,
		context.Factory().Identifier(definedtype.ValueOfMember),
		nil,
		nil,
		[]tsgo.ParameterDeclaration{referenceParameter(
			context,
			value,
			optionalClassType,
		)},
		underlying,
		context.Factory().Block(
			append(
				zero.Before(),
				context.Factory().ReturnStatement(
					context.Factory().ConditionalExpression(
						strictUndefined(context, value),
						context.Factory().QuestionToken(),
						zero.Value(),
						context.Factory().ColonToken(),
						context.Factory().PropertyAccessExpression(
							value,
							nil,
							context.Factory().Identifier(
								definedtype.ValueMember,
							),
							tsgo.NodeFlagsNone,
						),
					),
				),
			),
			true,
		),
	)
	return []tsgo.ClassElement{from, valueOf}, zero.Requests(), nil
}

func referenceNilCondition(
	context api.Context,
	model definedtype.Model,
	value tsgo.Expression,
) (tsgo.Expression, error) {
	switch model.Family() {
	case definedtype.FamilyPointer, definedtype.FamilyChannel:
		return strictUndefined(context, value), nil
	case definedtype.FamilySlice:
		return context.Factory().CallExpression(
			context.Factory().PropertyAccessExpression(
				value,
				nil,
				context.Factory().Identifier("isNil"),
				tsgo.NodeFlagsNone,
			),
			nil,
			nil,
			nil,
			tsgo.NodeFlagsNone,
		), nil
	default:
		return nil, &api.InvariantError{
			Role:   context.Role(),
			Reason: "defined reference family has no nil condition",
		}
	}
}

func strictUndefined(
	context api.Context,
	value tsgo.Expression,
) tsgo.Expression {
	return context.Factory().BinaryExpression(
		nil,
		value,
		nil,
		context.Factory().BinaryOperatorToken(
			tsgo.BinaryOperatorEqualsEqualsEqualsToken,
		),
		context.Factory().Identifier("undefined"),
	)
}

func referenceParameter(
	context api.Context,
	name tsgo.Identifier,
	targetType tsgo.TypeNode,
) tsgo.ParameterDeclaration {
	return context.Factory().ParameterDeclaration(
		nil,
		nil,
		name,
		nil,
		targetType,
		nil,
	)
}

func emitFamilyMembers(
	context api.Context,
	children api.ChildEmitter,
	source *ast.TypeSpec,
	model definedtype.Model,
	className string,
	underlying tsgo.TypeNode,
	requirements []api.DeclarationRequirement,
) ([]tsgo.ClassElement, []api.RootRequest, error) {
	switch model.Family() {
	case definedtype.FamilyBasic:
		if len(requirements) != 0 {
			return nil, nil, &api.InvariantError{
				Role:   context.Role(),
				Reason: "defined basic type received declaration requirements",
			}
		}
		return nil, nil, nil
	case definedtype.FamilySlice,
		definedtype.FamilyPointer,
		definedtype.FamilyChannel:
		if len(requirements) != 0 {
			return nil, nil, &api.InvariantError{
				Role:   context.Role(),
				Reason: "defined reference type received declaration requirements",
			}
		}
		return emitReferenceMembers(
			context,
			source,
			model,
			className,
			underlying,
		)
	case definedtype.FamilyMap:
		if len(requirements) != 0 {
			return nil, nil, &api.InvariantError{
				Role:   context.Role(),
				Reason: "defined map received declaration requirements",
			}
		}
		return emitMapMembers(
			context,
			source,
			model,
			underlying,
		)
	case definedtype.FamilyArray:
		if len(requirements) != 0 {
			return nil, nil, &api.InvariantError{
				Role:   context.Role(),
				Reason: "defined array received declaration requirements",
			}
		}
		return nil, nil, nil
	case definedtype.FamilyCallable:
		if len(requirements) != 0 {
			return nil, nil, &api.InvariantError{
				Role:   context.Role(),
				Reason: "defined callable received declaration requirements",
			}
		}
		return nil, nil, nil
	default:
		return nil, nil, &api.InvariantError{
			Role:   context.Role(),
			Reason: "defined type has no target family",
		}
	}
}
