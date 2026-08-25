package goroutine

import (
	"go/ast"

	"github.com/tsoniclang/gotots/internal/emit/api"
	concurrencyprofile "github.com/tsoniclang/gotots/internal/emit/concurrency/profile"
	runtimescheduler "github.com/tsoniclang/gotots/internal/emit/runtime/scheduler"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.GoStmt,
) (api.StatementEmission, error) {
	if source == nil || source.Call == nil {
		return api.StatementEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
	if err := concurrencyprofile.Admit(
		context,
		api.CategoryStatement,
		source,
	); err != nil {
		return api.StatementEmission{}, err
	}
	callContext := context.WithRole(api.RoleGoroutineCall)
	if context.ConcurrencySemantics() == api.ConcurrencySemanticsCooperative {
		callContext = callContext.WithDetachedInvocation()
	}
	call, err := children.DiscardedCall(callContext, source.Call)
	if err != nil {
		return api.StatementEmission{}, err
	}
	if context.ConcurrencySemantics() == api.ConcurrencySemanticsDisabled {
		statements := call.Before()
		statements = append(
			statements,
			context.Factory().ExpressionStatement(call.Value()),
		)
		return api.NewStatementEmission(statements, call.Requests())
	}
	scheduler, err := context.Names().Runtime(
		api.RuntimeScheduler,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	task := context.Factory().ArrowFunction(
		[]tsgo.ModifierLike{context.Factory().AsyncKeyword()},
		nil,
		nil,
		context.Factory().TypeReferenceNode(
			api.TargetIntrinsicPromise.TypeName(context.Factory()),
			[]tsgo.TypeNode{
				context.Factory().KeywordTypeNode(
					tsgo.KeywordTypeSyntaxKindVoidKeyword,
				),
			},
		),
		context.Factory().EqualsGreaterThanToken(),
		context.Factory().Block(
			[]tsgo.Statement{
				context.Factory().ExpressionStatement(call.Value()),
			},
			true,
		),
	)
	spawn := context.Factory().CallExpression(
		context.Factory().PropertyAccessExpression(
			context.Factory().Identifier(scheduler.Name()),
			nil,
			context.Factory().Identifier(runtimescheduler.SpawnMember),
			tsgo.NodeFlagsNone,
		),
		nil,
		nil,
		[]tsgo.Expression{task},
		tsgo.NodeFlagsNone,
	)
	statements := call.Before()
	statements = append(
		statements,
		context.Factory().ExpressionStatement(spawn),
	)
	return api.NewStatementEmission(
		statements,
		api.CombineRequests(
			call.Requests(),
			scheduler.Requests(),
		),
	)
}
