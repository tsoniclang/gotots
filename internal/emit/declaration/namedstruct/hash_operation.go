package namedstruct

import (
	"go/ast"

	"github.com/tsoniclang/gotots/internal/emit/api"
	mapruntime "github.com/tsoniclang/gotots/internal/emit/runtime/map"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func hashMethod(
	context api.Context,
	source ast.Node,
	memberName string,
	classType tsgo.TypeNode,
	fields []field,
	capabilities []tsgo.ParameterDeclaration,
	typeParameters []tsgo.TypeParameterDeclaration,
	canonicalStorage bool,
) (tsgo.MethodDeclaration, []api.RootRequest, error) {
	runtime, err := context.Names().Runtime(
		api.RuntimeMapHash,
		api.ImportPhaseValue,
	)
	if err != nil {
		return nil, nil, err
	}
	hashName := "$hash"
	hash := context.Factory().Identifier(hashName)
	body := []tsgo.Statement{
		context.Factory().VariableStatement(
			nil,
			context.Factory().VariableDeclarationList(
				[]tsgo.VariableDeclaration{
					context.Factory().VariableDeclaration(
						hash,
						nil,
						nil,
						context.Factory().NumericLiteral(
							"2166136261",
							tsgo.TokenFlagsNone,
						),
					),
				},
				tsgo.NodeFlagsLet,
			),
		),
	}
	requests := runtime.Requests()
	for _, field := range fields {
		if field.blank {
			continue
		}
		value, err := operationFieldValue(
			context.WithRole(api.RoleStructHashField),
			field.source,
			"$source",
			field,
			canonicalStorage,
		)
		if err != nil {
			return nil, nil, err
		}
		fieldHash, err := context.Values().Hash(
			context.WithRole(api.RoleStructHashField),
			field.source,
			field.object.Type(),
			value.Value(),
		)
		if err != nil {
			return nil, nil, err
		}
		fieldHash, err = api.NewExpressionEmission(
			append(value.Before(), fieldHash.Before()...),
			fieldHash.Value(),
			api.CombineRequests(value.Requests(), fieldHash.Requests()),
		)
		if err != nil {
			return nil, nil, err
		}
		body = append(body, fieldHash.Before()...)
		body = append(
			body,
			context.Factory().ExpressionStatement(
				context.Factory().BinaryExpression(
					nil,
					hash,
					nil,
					context.Factory().BinaryOperatorToken(
						tsgo.BinaryOperatorEqualsToken,
					),
					context.Factory().CallExpression(
						context.Factory().PropertyAccessExpression(
							context.Factory().Identifier(runtime.Name()),
							nil,
							context.Factory().Identifier(
								mapruntime.HashMixMember,
							),
							tsgo.NodeFlagsNone,
						),
						nil,
						nil,
						[]tsgo.Expression{hash, fieldHash.Value()},
						tsgo.NodeFlagsNone,
					),
				),
			),
		)
		requests = append(requests, fieldHash.Requests()...)
	}
	body = append(body, context.Factory().ReturnStatement(hash))
	return operationMethod(
		context,
		memberName,
		[]tsgo.ParameterDeclaration{
			parameter(context, "$source", classType),
		},
		context.Factory().KeywordTypeNode(
			tsgo.KeywordTypeSyntaxKindNumberKeyword,
		),
		body,
		capabilities,
		typeParameters,
	), requests, nil
}
