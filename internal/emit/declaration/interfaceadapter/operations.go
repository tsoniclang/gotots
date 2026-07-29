package interfaceadapter

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	interfaceruntime "github.com/tsoniclang/gotots/internal/emit/runtime/interfacevalue"
	mapruntime "github.com/tsoniclang/gotots/internal/emit/runtime/map"
	panicruntime "github.com/tsoniclang/gotots/internal/emit/runtime/panic"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func equalMethod(
	context api.Context,
	name string,
	runtimeValueName string,
	sourceType types.Type,
) (tsgo.MethodDeclaration, []api.RootRequest, error) {
	other := context.Factory().Identifier("other")
	body := []tsgo.Statement{
		context.Factory().IfStatement(
			context.Factory().PrefixUnaryExpression(
				tsgo.PrefixUnaryExpressionOperatorKindExclamationToken,
				context.Factory().CallExpression(
					context.Factory().PropertyAccessExpression(
						context.Factory().Identifier(name),
						nil,
						context.Factory().Identifier(GuardMember),
						tsgo.NodeFlagsNone,
					),
					nil,
					nil,
					[]tsgo.Expression{other},
					tsgo.NodeFlagsNone,
				),
			),
			context.Factory().Block(
				[]tsgo.Statement{
					context.Factory().ReturnStatement(
						context.Factory().FalseLiteral(),
					),
				},
				true,
			),
			nil,
		),
	}
	var requests []api.RootRequest
	if types.Comparable(sourceType) {
		equal, err := context.Values().Equal(
			context,
			nil,
			sourceType,
			payload(context.Factory(), context.Factory().ThisExpression()),
			payload(context.Factory(), other),
		)
		if err != nil {
			return nil, nil, err
		}
		body = append(body, equal.Before()...)
		body = append(
			body,
			context.Factory().ReturnStatement(equal.Value()),
		)
		requests = equal.Requests()
	} else {
		panicReference, err := context.Names().Runtime(
			api.RuntimePanic,
			api.ImportPhaseValue,
		)
		if err != nil {
			return nil, nil, err
		}
		body = append(
			body,
			context.Factory().ReturnStatement(
				panicruntime.Call(
					context.Factory(),
					panicReference.Name(),
					context.Factory().StringLiteral(
						"runtime error: comparing uncomparable dynamic interface value",
						tsgo.TokenFlagsNone,
					),
				),
			),
		)
		requests = panicReference.Requests()
	}
	return context.Factory().MethodDeclaration(
		nil,
		nil,
		context.Factory().Identifier(interfaceruntime.EqualMember),
		nil,
		nil,
		[]tsgo.ParameterDeclaration{
			parameter(
				context.Factory(),
				"other",
				context.Factory().TypeReferenceNode(
					context.Factory().Identifier(runtimeValueName),
					nil,
				),
			),
		},
		context.Factory().KeywordTypeNode(
			tsgo.KeywordTypeSyntaxKindBooleanKeyword,
		),
		context.Factory().Block(body, true),
	), requests, nil
}

func hashMethod(
	context api.Context,
	dynamicTypeName string,
	sourceType types.Type,
) (tsgo.MethodDeclaration, []api.RootRequest, error) {
	if !context.Values().SupportsHash(context, sourceType) {
		return panicHashMethod(context, sourceType)
	}
	hash, err := context.Values().Hash(
		context,
		nil,
		sourceType,
		payload(context.Factory(), context.Factory().ThisExpression()),
	)
	if err != nil {
		return nil, nil, err
	}
	mapHash, err := context.Names().Runtime(
		api.RuntimeMapHash,
		api.ImportPhaseValue,
	)
	if err != nil {
		return nil, nil, err
	}
	body := hash.Before()
	body = append(
		body,
		context.Factory().ReturnStatement(
			context.Factory().CallExpression(
				context.Factory().PropertyAccessExpression(
					context.Factory().Identifier(mapHash.Name()),
					nil,
					context.Factory().Identifier(mapruntime.HashMixMember),
					tsgo.NodeFlagsNone,
				),
				nil,
				nil,
				[]tsgo.Expression{
					context.Factory().CallExpression(
						context.Factory().PropertyAccessExpression(
							context.Factory().Identifier(mapHash.Name()),
							nil,
							context.Factory().Identifier(
								mapruntime.HashObjectMember,
							),
							tsgo.NodeFlagsNone,
						),
						nil,
						nil,
						[]tsgo.Expression{
							context.Factory().Identifier(
								dynamicTypeName,
							),
						},
						tsgo.NodeFlagsNone,
					),
					hash.Value(),
				},
				tsgo.NodeFlagsNone,
			),
		),
	)
	return context.Factory().MethodDeclaration(
			nil,
			nil,
			context.Factory().Identifier(interfaceruntime.HashMember),
			nil,
			nil,
			nil,
			context.Factory().KeywordTypeNode(
				tsgo.KeywordTypeSyntaxKindNumberKeyword,
			),
			context.Factory().Block(body, true),
		), api.CombineRequests(
			hash.Requests(),
			mapHash.Requests(),
		), nil
}

func panicHashMethod(
	context api.Context,
	sourceType types.Type,
) (tsgo.MethodDeclaration, []api.RootRequest, error) {
	panicReference, err := context.Names().Runtime(
		api.RuntimePanic,
		api.ImportPhaseValue,
	)
	if err != nil {
		return nil, nil, err
	}
	message := "runtime error: hash of unhashable dynamic interface value"
	if types.Comparable(sourceType) {
		message = "runtime error: dynamic interface hash is unavailable"
	}
	target := context.Factory().MethodDeclaration(
		nil,
		nil,
		context.Factory().Identifier(interfaceruntime.HashMember),
		nil,
		nil,
		nil,
		context.Factory().KeywordTypeNode(
			tsgo.KeywordTypeSyntaxKindNumberKeyword,
		),
		context.Factory().Block(
			[]tsgo.Statement{
				context.Factory().ReturnStatement(
					panicruntime.Call(
						context.Factory(),
						panicReference.Name(),
						context.Factory().StringLiteral(
							message,
							tsgo.TokenFlagsNone,
						),
					),
				),
			},
			true,
		),
	)
	return target, panicReference.Requests(), nil
}

func payload(
	factory tsgo.Factory,
	receiver tsgo.Expression,
) tsgo.PropertyAccessExpression {
	return factory.PropertyAccessExpression(
		receiver,
		nil,
		factory.Identifier(ValueMember),
		tsgo.NodeFlagsNone,
	)
}
