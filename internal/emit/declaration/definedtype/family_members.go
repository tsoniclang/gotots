package definedtype

import (
	"go/ast"

	"github.com/tsoniclang/gotots/internal/emit/api"
	arraymember "github.com/tsoniclang/gotots/internal/emit/runtime/array/member"
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
	case definedtype.FamilyPointer:
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
			Reason: "defined reference family is neither slice nor pointer",
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
	case definedtype.FamilySlice, definedtype.FamilyPointer:
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
	case definedtype.FamilyArray:
		addressIndex, err := arrayRequirements(
			context,
			model,
			requirements,
		)
		if err != nil || !addressIndex {
			return nil, nil, err
		}
		return emitArrayMembers(context, children, source, model)
	default:
		return nil, nil, &api.InvariantError{
			Role:   context.Role(),
			Reason: "defined type has no target family",
		}
	}
}

func arrayRequirements(
	context api.Context,
	model definedtype.Model,
	requirements []api.DeclarationRequirement,
) (bool, error) {
	addressIndex := false
	for _, requirement := range requirements {
		owner, operation, ok := requirement.DefinedArrayOperation()
		if !ok || owner != model.TypeName() {
			return false, &api.InvariantError{
				Role:   context.Role(),
				Reason: "defined array received a foreign declaration requirement",
			}
		}
		switch operation {
		case api.DefinedArrayOperationAddressIndex:
			addressIndex = true
		default:
			return false, &api.InvariantError{
				Role:   context.Role(),
				Reason: "defined array operation is invalid",
			}
		}
	}
	return addressIndex, nil
}

func emitArrayMembers(
	context api.Context,
	children api.ChildEmitter,
	source *ast.TypeSpec,
	model definedtype.Model,
) ([]tsgo.ClassElement, []api.RootRequest, error) {
	array, ok := model.Array()
	if !ok {
		return nil, nil, &api.InvariantError{
			Role:   context.Role(),
			Reason: "defined array family has no array underlying type",
		}
	}
	element, err := children.RepresentedType(
		context.WithRole(api.RoleArrayElement),
		source.Type,
		array.Elem(),
	)
	if err != nil {
		return nil, nil, err
	}
	indexType := context.Factory().UnionTypeNode([]tsgo.TypeNode{
		context.Factory().KeywordTypeNode(
			tsgo.KeywordTypeSyntaxKindNumberKeyword,
		),
		context.Factory().KeywordTypeNode(
			tsgo.KeywordTypeSyntaxKindBigIntKeyword,
		),
	})
	index := context.Factory().Identifier("index")
	value := context.Factory().Identifier("value")
	get := context.Factory().MethodDeclaration(
		[]tsgo.ModifierLike{context.Factory().PublicKeyword()},
		nil,
		context.Factory().Identifier(arraymember.Get.Name()),
		nil,
		nil,
		[]tsgo.ParameterDeclaration{
			arrayMethodParameter(context, index, indexType),
		},
		element.Value(),
		context.Factory().Block(
			[]tsgo.Statement{context.Factory().ReturnStatement(
				arrayMemberCall(
					context,
					definedStorage(context),
					arraymember.Get.Name(),
					index,
				),
			)},
			true,
		),
	)
	set := context.Factory().MethodDeclaration(
		[]tsgo.ModifierLike{context.Factory().PublicKeyword()},
		nil,
		context.Factory().Identifier(arraymember.Set.Name()),
		nil,
		nil,
		[]tsgo.ParameterDeclaration{
			arrayMethodParameter(context, index, indexType),
			arrayMethodParameter(context, value, element.Value()),
		},
		context.Factory().KeywordTypeNode(
			tsgo.KeywordTypeSyntaxKindVoidKeyword,
		),
		context.Factory().Block(
			[]tsgo.Statement{context.Factory().ExpressionStatement(
				arrayMemberCall(
					context,
					definedStorage(context),
					arraymember.Set.Name(),
					index,
					value,
				),
			)},
			true,
		),
	)
	return []tsgo.ClassElement{get, set}, element.Requests(), nil
}

func definedStorage(context api.Context) tsgo.PropertyAccessExpression {
	return context.Factory().PropertyAccessExpression(
		context.Factory().ThisExpression(),
		nil,
		context.Factory().Identifier(definedtype.ValueMember),
		tsgo.NodeFlagsNone,
	)
}

func arrayMethodParameter(
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

func arrayMemberCall(
	context api.Context,
	receiver tsgo.Expression,
	member string,
	arguments ...tsgo.Expression,
) tsgo.CallExpression {
	return context.Factory().CallExpression(
		context.Factory().PropertyAccessExpression(
			receiver,
			nil,
			context.Factory().Identifier(member),
			tsgo.NodeFlagsNone,
		),
		nil,
		nil,
		arguments,
		tsgo.NodeFlagsNone,
	)
}
