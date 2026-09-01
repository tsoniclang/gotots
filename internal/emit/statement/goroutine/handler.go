package goroutine

import (
	"go/ast"

	"github.com/tsoniclang/gotots/internal/emit/api"
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
	call, err := children.DiscardedCall(
		context.WithRole(api.RoleGoroutineCall),
		source.Call,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	spawn, err := context.Names().Runtime(
		api.RuntimeGoSpawn,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	operation := context.Factory().ArrowFunction(
		nil,
		nil,
		nil,
		context.Factory().KeywordTypeNode(
			tsgo.KeywordTypeSyntaxKindVoidKeyword,
		),
		context.Factory().EqualsGreaterThanToken(),
		context.Factory().Block(
			[]tsgo.Statement{context.Factory().ExpressionStatement(call.Value())},
			true,
		),
	)
	statements := call.Before()
	statements = append(
		statements,
		context.Factory().ExpressionStatement(
			context.Factory().CallExpression(
				spawn.Expression(context.Factory()),
				nil,
				nil,
				[]tsgo.Expression{operation},
				tsgo.NodeFlagsNone,
			),
		),
	)
	return api.NewStatementEmission(
		statements,
		api.CombineRequests(call.Requests(), spawn.Requests()),
	)
}
