package callable

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	panicruntime "github.com/tsoniclang/gotots/internal/emit/runtime/panic"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func emitDeferredBody(
	context api.Context,
	children api.ChildEmitter,
	source *ast.BlockStmt,
	signature *types.Signature,
	namedResults bool,
) (api.BlockEmission, error) {
	if defers, ok := staticDeferStatements(
		source,
		context.CallableControl(),
	); ok {
		return emitStaticDeferredBody(
			context,
			children,
			source,
			signature,
			namedResults,
			defers,
		)
	}
	stackName, err := context.Names().Temporary(api.TemporaryDeferStack)
	if err != nil {
		return api.BlockEmission{}, err
	}
	panicName, err := context.Names().Temporary(api.TemporaryActivePanic)
	if err != nil {
		return api.BlockEmission{}, err
	}
	returnLabel, err := context.Names().Temporary(api.TemporaryReturnLabel)
	if err != nil {
		return api.BlockEmission{}, err
	}
	bodyCatchName, err := context.Names().Temporary(
		api.TemporaryCaughtPanic,
	)
	if err != nil {
		return api.BlockEmission{}, err
	}
	deferredName, err := context.Names().Temporary(
		api.TemporaryDeferredCall,
	)
	if err != nil {
		return api.BlockEmission{}, err
	}
	recoveryName, err := context.Names().Temporary(
		api.TemporaryRecoveryAuthority,
	)
	if err != nil {
		return api.BlockEmission{}, err
	}
	deferCatchName, err := context.Names().Temporary(
		api.TemporaryCaughtPanic,
	)
	if err != nil {
		return api.BlockEmission{}, err
	}
	deferControl, err := api.NewDeferControl(stackName)
	if err != nil {
		return api.BlockEmission{}, err
	}
	returnControl, resultPrelude, resultRequests, resultName, err :=
		deferredReturnControl(
			context,
			children,
			source,
			signature.Results(),
			namedResults,
			returnLabel,
		)
	if err != nil {
		return api.BlockEmission{}, err
	}
	body, err := children.Block(
		context.
			WithDeferControl(deferControl).
			WithReturnControl(returnControl),
		source,
	)
	if err != nil {
		return api.BlockEmission{}, err
	}
	panicReference, err := context.Names().Runtime(
		api.RuntimePanic,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.BlockEmission{}, err
	}
	recoveryReference, err := context.Names().Runtime(
		api.RuntimeRecovery,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.BlockEmission{}, err
	}
	deferPopReference, err := context.Names().Runtime(
		api.RuntimeDeferPop,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.BlockEmission{}, err
	}
	deferStack, deferStackRequests, err := deferStackDeclaration(
		context,
		stackName,
		recoveryReference.Name(),
	)
	if err != nil {
		return api.BlockEmission{}, err
	}
	statements := []tsgo.Statement{
		deferStack,
		activePanicDeclaration(
			context,
			panicName,
			panicReference.Name(),
		),
	}
	statements = append(statements, resultPrelude...)
	statements = append(
		statements,
		deferEnvelope(
			context,
			body.Value(),
			stackName,
			panicName,
			returnLabel,
			panicReference.Name(),
			recoveryReference.Name(),
			deferPopReference.Name(),
			bodyCatchName,
			deferredName,
			recoveryName,
			deferCatchName,
		),
		finalPanicStatement(
			context,
			panicName,
		),
	)
	finalReturn, finalRequests, err := deferredFinalReturn(
		context,
		signature.Results(),
		namedResults,
		resultName,
	)
	if err != nil {
		return api.BlockEmission{}, err
	}
	statements = append(statements, finalReturn...)
	return api.DirectBlock(
		context.Factory().Block(statements, true),
		api.CombineRequests(
			body.Requests(),
			resultRequests,
			panicReference.Requests(),
			recoveryReference.Requests(),
			deferPopReference.Requests(),
			deferStackRequests,
			finalRequests,
		)...,
	), nil
}

func deferStackDeclaration(
	context api.Context,
	stackName string,
	recoveryName string,
) (tsgo.VariableStatement, []api.RootRequest, error) {
	recoveryType := context.Factory().TypeReferenceNode(
		context.Factory().Identifier(recoveryName),
		nil,
	)
	resultType := api.DirectType(context.Factory().KeywordTypeNode(
		tsgo.KeywordTypeSyntaxKindVoidKeyword,
	))
	if context.IsCooperative() {
		var err error
		resultType, err = IndirectResult(context, resultType.Value())
		if err != nil {
			return nil, nil, err
		}
	}
	callableType := context.Factory().FunctionTypeNode(
		nil,
		[]tsgo.ParameterDeclaration{
			context.Factory().ParameterDeclaration(
				nil,
				nil,
				context.Factory().Identifier(RecoveryAuthorityName),
				nil,
				recoveryType,
				nil,
			),
		},
		resultType.Value(),
	)
	return variableStatement(
		context,
		tsgo.NodeFlagsConst,
		stackName,
		context.Factory().ArrayTypeNode(callableType),
		context.Factory().ArrayLiteralExpression(nil, false),
	), resultType.Requests(), nil
}

func activePanicDeclaration(
	context api.Context,
	name string,
	panicName string,
) tsgo.VariableStatement {
	return variableStatement(
		context,
		tsgo.NodeFlagsLet,
		name,
		context.Factory().UnionTypeNode([]tsgo.TypeNode{
			context.Factory().TypeReferenceNode(
				context.Factory().Identifier(panicName),
				nil,
			),
			context.Factory().KeywordTypeNode(
				tsgo.KeywordTypeSyntaxKindUndefinedKeyword,
			),
		}),
		context.Factory().Identifier("undefined"),
	)
}

func deferEnvelope(
	context api.Context,
	body tsgo.Block,
	stackName string,
	panicName string,
	returnLabel string,
	panicTypeName string,
	recoveryTypeName string,
	deferPopName string,
	bodyCatchName string,
	deferredName string,
	recoveryName string,
	deferCatchName string,
) tsgo.TryStatement {
	protected := context.Factory().TryStatement(
		context.Factory().Block(
			[]tsgo.Statement{
				context.Factory().LabeledStatement(
					context.Factory().Identifier(returnLabel),
					body,
				),
			},
			true,
		),
		panicCatch(
			context,
			panicName,
			panicTypeName,
			bodyCatchName,
		),
		nil,
	)
	return context.Factory().TryStatement(
		context.Factory().Block(
			[]tsgo.Statement{protected},
			true,
		),
		nil,
		context.Factory().Block(
			[]tsgo.Statement{
				drainDefers(
					context,
					stackName,
					panicName,
					panicTypeName,
					recoveryTypeName,
					deferPopName,
					deferredName,
					recoveryName,
					deferCatchName,
				),
			},
			true,
		),
	)
}

func panicCatch(
	context api.Context,
	panicName string,
	panicTypeName string,
	caughtName string,
) tsgo.CatchClause {
	return context.Factory().CatchClause(
		context.Factory().VariableDeclaration(
			context.Factory().Identifier(caughtName),
			nil,
			nil,
			nil,
		),
		context.Factory().Block(
			panicCatchStatements(
				context,
				caughtName,
				panicName,
				panicTypeName,
			),
			true,
		),
	)
}

func panicCatchStatements(
	context api.Context,
	caughtName string,
	panicName string,
	panicTypeName string,
) []tsgo.Statement {
	caught := context.Factory().Identifier(caughtName)
	isPanic := context.Factory().BinaryExpression(
		nil,
		caught,
		nil,
		context.Factory().BinaryOperatorToken(
			tsgo.BinaryOperatorInstanceOfKeyword,
		),
		context.Factory().Identifier(panicTypeName),
	)
	return []tsgo.Statement{
		context.Factory().IfStatement(
			context.Factory().PrefixUnaryExpression(
				tsgo.PrefixUnaryExpressionOperatorKindExclamationToken,
				isPanic,
			),
			context.Factory().Block(
				[]tsgo.Statement{
					panicruntime.Rethrow(context.Factory(), caught),
				},
				true,
			),
			nil,
		),
		context.Factory().ExpressionStatement(
			context.Factory().BinaryExpression(
				nil,
				context.Factory().Identifier(panicName),
				nil,
				context.Factory().BinaryOperatorToken(
					tsgo.BinaryOperatorEqualsToken,
				),
				caught,
			),
		),
	}
}

func drainDefers(
	context api.Context,
	stackName string,
	panicName string,
	panicTypeName string,
	recoveryTypeName string,
	deferPopName string,
	deferredName string,
	recoveryName string,
	caughtName string,
) tsgo.WhileStatement {
	stack := context.Factory().Identifier(stackName)
	deferred := context.Factory().Identifier(deferredName)
	recovery := context.Factory().Identifier(recoveryName)
	body := []tsgo.Statement{
		variableStatement(
			context,
			tsgo.NodeFlagsConst,
			deferredName,
			nil,
			context.Factory().CallExpression(
				context.Factory().Identifier(deferPopName),
				nil,
				nil,
				[]tsgo.Expression{stack},
				tsgo.NodeFlagsNone,
			),
		),
		variableStatement(
			context,
			tsgo.NodeFlagsConst,
			recoveryName,
			nil,
			context.Factory().NewExpression(
				context.Factory().Identifier(recoveryTypeName),
				nil,
				[]tsgo.Expression{
					context.Factory().Identifier(panicName),
				},
			),
		),
		context.Factory().TryStatement(
			context.Factory().Block(
				[]tsgo.Statement{
					deferredCallStatement(context, deferred, recovery),
					context.Factory().IfStatement(
						context.Factory().CallExpression(
							context.Factory().PropertyAccessExpression(
								recovery,
								nil,
								context.Factory().Identifier(
									panicruntime.RecoveredName,
								),
								tsgo.NodeFlagsNone,
							),
							nil,
							nil,
							nil,
							tsgo.NodeFlagsNone,
						),
						context.Factory().Block(
							[]tsgo.Statement{
								context.Factory().ExpressionStatement(
									context.Factory().BinaryExpression(
										nil,
										context.Factory().Identifier(
											panicName,
										),
										nil,
										context.Factory().BinaryOperatorToken(
											tsgo.BinaryOperatorEqualsToken,
										),
										context.Factory().Identifier(
											"undefined",
										),
									),
								),
							},
							true,
						),
						nil,
					),
				},
				true,
			),
			context.Factory().CatchClause(
				context.Factory().VariableDeclaration(
					context.Factory().Identifier(caughtName),
					nil,
					nil,
					nil,
				),
				context.Factory().Block(
					panicCatchStatements(
						context,
						caughtName,
						panicName,
						panicTypeName,
					),
					true,
				),
			),
			nil,
		),
	}
	return context.Factory().WhileStatement(
		context.Factory().BinaryExpression(
			nil,
			context.Factory().PropertyAccessExpression(
				stack,
				nil,
				context.Factory().Identifier("length"),
				tsgo.NodeFlagsNone,
			),
			nil,
			context.Factory().BinaryOperatorToken(
				tsgo.BinaryOperatorExclamationEqualsEqualsToken,
			),
			context.Factory().NumericLiteral("0", tsgo.TokenFlagsNone),
		),
		context.Factory().Block(body, true),
	)
}

func deferredCallStatement(
	context api.Context,
	deferred tsgo.Expression,
	recovery tsgo.Expression,
) tsgo.Statement {
	var call tsgo.Expression = context.Factory().CallExpression(
		deferred,
		nil,
		nil,
		[]tsgo.Expression{recovery},
		tsgo.NodeFlagsNone,
	)
	if context.IsCooperative() {
		call = context.Factory().AwaitExpression(call)
	}
	return context.Factory().ExpressionStatement(call)
}

func finalPanicStatement(
	context api.Context,
	panicName string,
) tsgo.IfStatement {
	return context.Factory().IfStatement(
		context.Factory().BinaryExpression(
			nil,
			context.Factory().Identifier(panicName),
			nil,
			context.Factory().BinaryOperatorToken(
				tsgo.BinaryOperatorExclamationEqualsEqualsToken,
			),
			context.Factory().Identifier("undefined"),
		),
		context.Factory().Block(
			[]tsgo.Statement{
				panicruntime.Rethrow(
					context.Factory(),
					context.Factory().Identifier(panicName),
				),
			},
			true,
		),
		nil,
	)
}

func variableStatement(
	context api.Context,
	flags tsgo.NodeFlags,
	name string,
	targetType tsgo.TypeNode,
	value tsgo.Expression,
) tsgo.VariableStatement {
	return context.Factory().VariableStatement(
		nil,
		context.Factory().VariableDeclarationList(
			[]tsgo.VariableDeclaration{
				context.Factory().VariableDeclaration(
					context.Factory().Identifier(name),
					nil,
					targetType,
					value,
				),
			},
			flags,
		),
	)
}
