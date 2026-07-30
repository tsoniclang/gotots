package definedtype

import (
	"go/ast"

	"github.com/tsoniclang/gotots/internal/emit/api"
	mapruntime "github.com/tsoniclang/gotots/internal/emit/runtime/map"
	panicruntime "github.com/tsoniclang/gotots/internal/emit/runtime/panic"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func emitMapMembers(
	context api.Context,
	_ *ast.TypeSpec,
	model definedtype.Model,
	underlyingType tsgo.TypeNode,
	typeParameters []tsgo.TypeParameterDeclaration,
	typeArguments []tsgo.TypeNode,
) ([]tsgo.ClassElement, []api.RootRequest, error) {
	className, err := context.Names().Declare(model.TypeName())
	if err != nil {
		return nil, nil, err
	}
	classType := context.Factory().TypeReferenceNode(
		context.Factory().Identifier(className),
		typeArguments,
	)
	optionalClassType := context.Factory().UnionTypeNode([]tsgo.TypeNode{
		classType,
		context.Factory().KeywordTypeNode(
			tsgo.KeywordTypeSyntaxKindUndefinedKeyword,
		),
	})
	panicReference, err := context.Names().Runtime(
		api.RuntimePanic,
		api.ImportPhaseValue,
	)
	if err != nil {
		return nil, nil, err
	}
	isNilName, err := mapruntime.Name(mapruntime.MemberIsNil)
	if err != nil {
		return nil, nil, err
	}
	return []tsgo.ClassElement{
			mapReadMember(
				context,
				optionalClassType,
				underlyingType,
				typeParameters,
			),
			mapStoreMember(
				context,
				optionalClassType,
				underlyingType,
				panicReference.Name(),
				typeParameters,
			),
			mapWrapMember(
				context,
				className,
				optionalClassType,
				underlyingType,
				isNilName,
				typeParameters,
			),
		}, api.CombineRequests(
			panicReference.Requests(),
		), nil
}

func mapReadMember(
	context api.Context,
	classType tsgo.TypeNode,
	underlyingType tsgo.TypeNode,
	typeParameters []tsgo.TypeParameterDeclaration,
) tsgo.MethodDeclaration {
	source := context.Factory().Identifier("$source")
	zero := context.Factory().Identifier("$zero")
	return mapStaticMethod(
		context,
		definedtype.MapReadMember,
		[]tsgo.ParameterDeclaration{
			mapParameter(context, source, classType),
			mapParameter(context, zero, underlyingType),
		},
		underlyingType,
		[]tsgo.Statement{
			context.Factory().IfStatement(
				mapUndefined(context, source),
				context.Factory().Block(
					[]tsgo.Statement{
						context.Factory().ReturnStatement(zero),
					},
					true,
				),
				nil,
			),
			context.Factory().ReturnStatement(
				mapStorage(context, source),
			),
		},
		typeParameters,
	)
}

func mapStoreMember(
	context api.Context,
	classType tsgo.TypeNode,
	underlyingType tsgo.TypeNode,
	panicName string,
	typeParameters []tsgo.TypeParameterDeclaration,
) tsgo.MethodDeclaration {
	source := context.Factory().Identifier("$source")
	return mapStaticMethod(
		context,
		definedtype.MapStoreMember,
		[]tsgo.ParameterDeclaration{
			mapParameter(context, source, classType),
		},
		underlyingType,
		[]tsgo.Statement{
			context.Factory().IfStatement(
				mapUndefined(context, source),
				context.Factory().Block(
					[]tsgo.Statement{
						context.Factory().ExpressionStatement(
							panicruntime.Call(
								context.Factory(),
								panicName,
								context.Factory().StringLiteral(
									"assignment to entry in nil map",
									tsgo.TokenFlagsNone,
								),
							),
						),
					},
					true,
				),
				nil,
			),
			context.Factory().ReturnStatement(
				mapStorage(context, source),
			),
		},
		typeParameters,
	)
}

func mapWrapMember(
	context api.Context,
	className string,
	classType tsgo.TypeNode,
	underlyingType tsgo.TypeNode,
	isNilName string,
	typeParameters []tsgo.TypeParameterDeclaration,
) tsgo.MethodDeclaration {
	source := context.Factory().Identifier("$source")
	isNil := context.Factory().CallExpression(
		context.Factory().PropertyAccessExpression(
			source,
			nil,
			context.Factory().Identifier(isNilName),
			tsgo.NodeFlagsNone,
		),
		nil,
		nil,
		nil,
		tsgo.NodeFlagsNone,
	)
	return mapStaticMethod(
		context,
		definedtype.MapWrapMember,
		[]tsgo.ParameterDeclaration{
			mapParameter(context, source, underlyingType),
		},
		classType,
		[]tsgo.Statement{
			context.Factory().ReturnStatement(
				context.Factory().ConditionalExpression(
					isNil,
					context.Factory().QuestionToken(),
					context.Factory().Identifier("undefined"),
					context.Factory().ColonToken(),
					context.Factory().NewExpression(
						context.Factory().Identifier(className),
						nil,
						[]tsgo.Expression{source},
					),
				),
			),
		},
		typeParameters,
	)
}

func mapStaticMethod(
	context api.Context,
	name string,
	parameters []tsgo.ParameterDeclaration,
	resultType tsgo.TypeNode,
	body []tsgo.Statement,
	typeParameters []tsgo.TypeParameterDeclaration,
) tsgo.MethodDeclaration {
	return context.Factory().MethodDeclaration(
		[]tsgo.ModifierLike{context.Factory().StaticKeyword()},
		nil,
		context.Factory().Identifier(name),
		nil,
		typeParameters,
		parameters,
		resultType,
		context.Factory().Block(body, true),
	)
}

func mapParameter(
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

func mapUndefined(
	context api.Context,
	source tsgo.Expression,
) tsgo.BinaryExpression {
	return context.Factory().BinaryExpression(
		nil,
		source,
		nil,
		context.Factory().BinaryOperatorToken(
			tsgo.BinaryOperatorEqualsEqualsEqualsToken,
		),
		context.Factory().Identifier("undefined"),
	)
}

func mapStorage(
	context api.Context,
	source tsgo.Expression,
) tsgo.PropertyAccessExpression {
	return context.Factory().PropertyAccessExpression(
		source,
		nil,
		context.Factory().Identifier(definedtype.ValueMember),
		tsgo.NodeFlagsNone,
	)
}
