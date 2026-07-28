package definedtype

import (
	"go/ast"

	"github.com/tsoniclang/gotots/internal/emit/api"
	arraymember "github.com/tsoniclang/gotots/internal/emit/runtime/array/member"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func emitFamilyMembers(
	context api.Context,
	children api.ChildEmitter,
	source *ast.TypeSpec,
	model definedtype.Model,
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
