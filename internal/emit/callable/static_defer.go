package callable

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	panicruntime "github.com/tsoniclang/gotots/internal/emit/runtime/panic"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func staticDeferStatements(
	body *ast.BlockStmt,
	control api.CallableControlDemand,
) ([]*ast.DeferStmt, bool) {
	if body == nil || !control.Defer() || control.Goto() {
		return nil, false
	}
	var direct []*ast.DeferStmt
	for _, statement := range body.List {
		if deferred, ok := statement.(*ast.DeferStmt); ok {
			direct = append(direct, deferred)
		}
	}
	if len(direct) == 0 {
		return nil, false
	}
	return control.ExactDefers(direct)
}

func emitStaticDeferredBody(
	context api.Context,
	children api.ChildEmitter,
	source *ast.BlockStmt,
	signature *types.Signature,
	namedResults bool,
	deferredStatements []*ast.DeferStmt,
) (api.BlockEmission, error) {
	bindings := make(map[*ast.DeferStmt]string, len(deferredStatements))
	slots := make([]string, 0, len(deferredStatements))
	recoveryNames := make([]string, 0, len(deferredStatements))
	catchNames := make([]string, 0, len(deferredStatements))
	for _, statement := range deferredStatements {
		slot, err := context.Names().Temporary(api.TemporaryDeferredCall)
		if err != nil {
			return api.BlockEmission{}, err
		}
		recovery, err := context.Names().Temporary(
			api.TemporaryRecoveryAuthority,
		)
		if err != nil {
			return api.BlockEmission{}, err
		}
		caught, err := context.Names().Temporary(api.TemporaryCaughtPanic)
		if err != nil {
			return api.BlockEmission{}, err
		}
		bindings[statement] = slot
		slots = append(slots, slot)
		recoveryNames = append(recoveryNames, recovery)
		catchNames = append(catchNames, caught)
	}
	deferControl, err := api.NewStaticDeferControl(bindings)
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
	bodyCatchName, err := context.Names().Temporary(api.TemporaryCaughtPanic)
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
	callableType, callableRequests, err := deferredCallableType(
		context,
		recoveryReference.Name(),
	)
	if err != nil {
		return api.BlockEmission{}, err
	}
	statements := make([]tsgo.Statement, 0, len(slots)+4+len(resultPrelude))
	for _, slot := range slots {
		statements = append(
			statements,
			variableStatement(
				context,
				tsgo.NodeFlagsLet,
				slot,
				context.Factory().UnionTypeNode([]tsgo.TypeNode{
					callableType,
					context.Factory().KeywordTypeNode(
						tsgo.KeywordTypeSyntaxKindUndefinedKeyword,
					),
				}),
				context.Factory().Identifier("undefined"),
			),
		)
	}
	statements = append(
		statements,
		activePanicDeclaration(context, panicName, panicReference.Name()),
	)
	statements = append(statements, resultPrelude...)
	statements = append(
		statements,
		staticDeferEnvelope(
			context,
			body.Value(),
			slots,
			recoveryNames,
			catchNames,
			panicName,
			returnLabel,
			panicReference.Name(),
			recoveryReference.Name(),
			bodyCatchName,
		),
		finalPanicStatement(context, panicName),
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
			callableRequests,
			finalRequests,
		)...,
	), nil
}

func deferredCallableType(
	context api.Context,
	recoveryName string,
) (tsgo.TypeNode, []api.RootRequest, error) {
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
	return context.Factory().FunctionTypeNode(
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
	), resultType.Requests(), nil
}

func staticDeferEnvelope(
	context api.Context,
	body tsgo.Block,
	slots []string,
	recoveryNames []string,
	catchNames []string,
	panicName string,
	returnLabel string,
	panicTypeName string,
	recoveryTypeName string,
	bodyCatchName string,
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
		panicCatch(context, panicName, panicTypeName, bodyCatchName),
		nil,
	)
	finalizers := make([]tsgo.Statement, 0, len(slots))
	for index := len(slots) - 1; index >= 0; index-- {
		finalizers = append(
			finalizers,
			staticDeferredInvocation(
				context,
				slots[index],
				recoveryNames[index],
				catchNames[index],
				panicName,
				panicTypeName,
				recoveryTypeName,
			),
		)
	}
	return context.Factory().TryStatement(
		context.Factory().Block([]tsgo.Statement{protected}, true),
		nil,
		context.Factory().Block(finalizers, true),
	)
}

func staticDeferredInvocation(
	context api.Context,
	slotName string,
	recoveryName string,
	catchName string,
	panicName string,
	panicTypeName string,
	recoveryTypeName string,
) tsgo.IfStatement {
	slot := context.Factory().Identifier(slotName)
	recovery := context.Factory().Identifier(recoveryName)
	body := []tsgo.Statement{
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
					deferredCallStatement(context, slot, recovery),
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
										context.Factory().Identifier(panicName),
										nil,
										context.Factory().BinaryOperatorToken(
											tsgo.BinaryOperatorEqualsToken,
										),
										context.Factory().Identifier("undefined"),
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
					context.Factory().Identifier(catchName),
					nil,
					nil,
					nil,
				),
				context.Factory().Block(
					panicCatchStatements(
						context,
						catchName,
						panicName,
						panicTypeName,
					),
					true,
				),
			),
			nil,
		),
	}
	return context.Factory().IfStatement(
		context.Factory().BinaryExpression(
			nil,
			slot,
			nil,
			context.Factory().BinaryOperatorToken(
				tsgo.BinaryOperatorExclamationEqualsEqualsToken,
			),
			context.Factory().Identifier("undefined"),
		),
		context.Factory().Block(body, true),
		nil,
	)
}
