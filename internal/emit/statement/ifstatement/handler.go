package ifstatement

import (
	"go/ast"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.IfStmt,
) (api.StatementEmission, error) {
	if source.Body == nil {
		return api.StatementEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
	var initializer api.StatementEmission
	var err error
	if source.Init != nil {
		initializer, err = children.ScopedInitializer(
			context.WithRole(api.RoleIfInitializer),
			source.Init,
		)
		if err != nil {
			return api.StatementEmission{}, err
		}
	}
	condition, err := children.Condition(
		context.WithRole(api.RoleIfCondition),
		source.Cond,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	thenBlock, err := children.Block(
		context.WithRole(api.RoleIfThen),
		source.Body,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	var elseStatement tsgo.Statement
	var elseRequests []api.RootRequest
	switch alternate := source.Else.(type) {
	case nil:
	case *ast.BlockStmt:
		var alternateBlock api.BlockEmission
		alternateBlock, err = children.Block(
			context.WithRole(api.RoleIfElse),
			alternate,
		)
		if err == nil {
			elseStatement = alternateBlock.Value()
			elseRequests = alternateBlock.Requests()
		}
	case *ast.IfStmt:
		var alternateIf api.StatementEmission
		alternateIf, err = children.IfAlternate(
			context.WithRole(api.RoleIfElse),
			alternate,
		)
		if err == nil {
			alternateStatements := alternateIf.Statements()
			switch len(alternateStatements) {
			case 0:
				return api.StatementEmission{}, &api.InvariantError{
					Role:   api.RoleIfElse,
					Reason: "nested if emitted no target statement",
				}
			case 1:
				elseStatement = alternateStatements[0]
			default:
				elseStatement = context.Factory().Block(alternateStatements, true)
			}
			elseRequests = alternateIf.Requests()
		}
	default:
		return api.StatementEmission{},
			api.Unsupported(
				context.WithRole(api.RoleIfElse),
				api.CategoryStatement,
				alternate,
			)
	}
	if err != nil {
		return api.StatementEmission{}, err
	}
	target := context.Factory().IfStatement(
		condition.Value(),
		thenBlock.Value(),
		elseStatement,
	)
	requests := api.CombineRequests(
		initializer.Requests(),
		condition.Requests(),
		thenBlock.Requests(),
		elseRequests,
	)
	statements := condition.Before()
	statements = append(statements, target)
	if source.Init != nil {
		scoped := initializer.Statements()
		scoped = append(scoped, statements...)
		return api.DirectStatement(
			context.Factory().Block(scoped, true),
			requests...,
		), nil
	}
	return api.NewStatementEmission(
		statements,
		requests,
	)
}
