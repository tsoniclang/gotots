package sequence

import (
	"go/ast"
	"go/token"
	"go/types"
	"strconv"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type gotoLocal struct {
	source *ast.Ident
	object *types.Var
}

func emitStateGoto(
	context api.Context,
	children api.ChildEmitter,
	owner ast.Node,
	statements []ast.Stmt,
	labels []blockLabel,
) (api.StatementEmission, error) {
	stateName, err := context.Names().Temporary(api.TemporaryGotoState)
	if err != nil {
		return api.StatementEmission{}, err
	}
	dispatchName, err := context.Names().Temporary(
		api.TemporaryGotoDispatch,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	locals, err := stateGotoLocals(context, statements)
	if err != nil {
		return api.StatementEmission{}, err
	}
	variables := make([]*types.Var, 0, len(locals))
	for _, local := range locals {
		variables = append(variables, local.object)
	}
	selected := context.WithGotoLocals(variables)
	for index, label := range labels {
		target, err := api.NewStateGotoTarget(
			dispatchName,
			stateName,
			index+1,
		)
		if err != nil {
			return api.StatementEmission{}, err
		}
		selected = selected.WithGotoTarget(label.object, target)
	}
	prelude, requests, err := stateGotoPrelude(selected, children, locals)
	if err != nil {
		return api.StatementEmission{}, err
	}
	segments := stateGotoSegments(statements, labels)
	clauses := make([]tsgo.CaseOrDefaultClause, 0, len(segments)+1)
	var staticPrelude []tsgo.Statement
	for index, segment := range segments {
		var statements []tsgo.Statement
		for _, statement := range segment {
			if declaration, labeled, ok := stateStaticDeclaration(
				statement,
			); ok {
				emission, err := children.Statement(
					selected,
					declaration,
				)
				if err != nil {
					return api.StatementEmission{}, err
				}
				staticPrelude = append(
					staticPrelude,
					emission.Statements()...,
				)
				requests = append(requests, emission.Requests()...)
				if labeled {
					statements = append(
						statements,
						context.Factory().EmptyStatement(),
					)
				}
				continue
			}
			emission, err := children.Statement(
				selected,
				statement,
			)
			if err != nil {
				return api.StatementEmission{}, err
			}
			statements = append(statements, emission.Statements()...)
			requests = append(requests, emission.Requests()...)
		}
		if FallsThrough(statements) {
			statements = append(
				statements,
				stateTransition(
					context,
					stateName,
					dispatchName,
					index+1,
				)...,
			)
		}
		clauses = append(
			clauses,
			context.Factory().CaseClause(
				context.Factory().NumericLiteral(
					strconv.Itoa(index),
					tsgo.TokenFlagsNone,
				),
				statements,
			),
		)
	}
	defaultBranch := tsgo.Statement(
		context.Factory().BreakStatement(
			context.Factory().Identifier(dispatchName),
		),
	)
	if results := context.FunctionResults(); results != nil &&
		results.Len() != 0 &&
		isCurrentCallableBody(context, owner) &&
		!context.CallableControl().Defer() {
		defaultBranch = context.Factory().ContinueStatement(
			context.Factory().Identifier(dispatchName),
		)
	}
	clauses = append(
		clauses,
		context.Factory().DefaultClause(
			nil,
			[]tsgo.Statement{defaultBranch},
		),
	)
	prelude = append(staticPrelude, prelude...)
	target := append(
		prelude,
		gotoVariableStatement(
			context,
			tsgo.NodeFlagsLet,
			stateName,
			nil,
			context.Factory().NumericLiteral("0", tsgo.TokenFlagsNone),
		),
		context.Factory().LabeledStatement(
			context.Factory().Identifier(dispatchName),
			context.Factory().WhileStatement(
				context.Factory().TrueLiteral(),
				context.Factory().Block(
					[]tsgo.Statement{
						context.Factory().SwitchStatement(
							context.Factory().Identifier(stateName),
							context.Factory().CaseBlock(clauses),
						),
					},
					true,
				),
			),
		),
	)
	return api.NewStatementEmission(target, requests)
}

func isCurrentCallableBody(
	context api.Context,
	owner ast.Node,
) bool {
	source, ok := owner.(*ast.BlockStmt)
	return ok && context.IsCurrentCallableBody(source)
}

func stateStaticDeclaration(
	statement ast.Stmt,
) (*ast.DeclStmt, bool, bool) {
	labeled := false
	for {
		sourceLabel, ok := statement.(*ast.LabeledStmt)
		if !ok {
			break
		}
		labeled = true
		statement = sourceLabel.Stmt
	}
	declaration, ok := statement.(*ast.DeclStmt)
	if !ok {
		return nil, false, false
	}
	generic, ok := declaration.Decl.(*ast.GenDecl)
	if !ok || generic.Tok != token.CONST && generic.Tok != token.TYPE {
		return nil, false, false
	}
	return declaration, labeled, true
}

func stateGotoSegments(
	statements []ast.Stmt,
	labels []blockLabel,
) [][]ast.Stmt {
	selected := make(map[*ast.LabeledStmt]struct{}, len(labels))
	for _, label := range labels {
		selected[label.source] = struct{}{}
	}
	segments := make([][]ast.Stmt, len(labels)+1)
	segment := 0
	for _, statement := range statements {
		labeled, ok := statement.(*ast.LabeledStmt)
		if _, target := selected[labeled]; ok && target {
			segment++
		}
		segments[segment] = append(segments[segment], statement)
	}
	return segments
}

func stateGotoLocals(
	context api.Context,
	statements []ast.Stmt,
) ([]gotoLocal, error) {
	var locals []gotoLocal
	for _, statement := range statements {
		for {
			labeled, ok := statement.(*ast.LabeledStmt)
			if !ok {
				break
			}
			statement = labeled.Stmt
		}
		switch statement := statement.(type) {
		case *ast.DeclStmt:
			declaration, ok := statement.Decl.(*ast.GenDecl)
			if !ok || declaration.Tok != token.VAR {
				continue
			}
			for _, sourceSpec := range declaration.Specs {
				spec, ok := sourceSpec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for _, name := range spec.Names {
					local, err := stateGotoLocal(context, name)
					if err != nil {
						return nil, err
					}
					if local.object != nil {
						locals = append(locals, local)
					}
				}
			}
		case *ast.AssignStmt:
			if statement.Tok != token.DEFINE {
				continue
			}
			for _, left := range statement.Lhs {
				name, ok := left.(*ast.Ident)
				if !ok {
					continue
				}
				local, err := stateGotoLocal(context, name)
				if err != nil {
					return nil, err
				}
				if local.object != nil {
					locals = append(locals, local)
				}
			}
		}
	}
	return locals, nil
}

func stateGotoLocal(
	context api.Context,
	name *ast.Ident,
) (gotoLocal, error) {
	if name == nil || name.Name == "_" {
		return gotoLocal{}, nil
	}
	object, ok := context.TypesInfo().DefOf(name).(*types.Var)
	if !ok {
		return gotoLocal{}, api.Unsupported(
			context.WithRole(api.RoleLocalDeclaration),
			api.CategoryStatement,
			name,
		)
	}
	return gotoLocal{source: name, object: object}, nil
}

func stateGotoPrelude(
	context api.Context,
	children api.ChildEmitter,
	locals []gotoLocal,
) ([]tsgo.Statement, []api.RootRequest, error) {
	var statements []tsgo.Statement
	var requests []api.RootRequest
	for _, local := range locals {
		zero, err := context.Values().Zero(
			context.WithRole(api.RoleLocalValue),
			local.source,
			local.object.Type(),
		)
		if err != nil {
			return nil, nil, err
		}
		name, err := context.Names().Declare(local.object)
		var targetType tsgo.TypeNode
		if err == nil {
			target, targetErr := children.RepresentedType(
				context.WithRole(api.RoleLocalType),
				local.source,
				local.object.Type(),
			)
			err = targetErr
			if targetErr == nil {
				targetType = target.Value()
				requests = append(requests, target.Requests()...)
			}
		}
		if err != nil {
			return nil, nil, err
		}
		statements = append(statements, zero.Before()...)
		statements = append(
			statements,
			gotoVariableStatement(
				context,
				tsgo.NodeFlagsLet,
				name,
				targetType,
				zero.Value(),
			),
		)
		requests = append(requests, zero.Requests()...)
	}
	return statements, requests, nil
}

func gotoVariableStatement(
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

func stateTransition(
	context api.Context,
	stateName string,
	dispatchName string,
	next int,
) []tsgo.Statement {
	return []tsgo.Statement{
		context.Factory().ExpressionStatement(
			context.Factory().BinaryExpression(
				nil,
				context.Factory().Identifier(stateName),
				nil,
				context.Factory().BinaryOperatorToken(
					tsgo.BinaryOperatorEqualsToken,
				),
				context.Factory().NumericLiteral(
					strconv.Itoa(next),
					tsgo.TokenFlagsNone,
				),
			),
		),
		context.Factory().ContinueStatement(
			context.Factory().Identifier(dispatchName),
		),
	}
}
