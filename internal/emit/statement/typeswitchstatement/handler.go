package typeswitchstatement

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	interfacetype "github.com/tsoniclang/gotots/internal/emit/type/interfacevalue"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type guard struct {
	assertion  *ast.TypeAssertExpr
	binding    *ast.Ident
	sourceType types.Type
	source     *types.Interface
}

type caseType struct {
	sourceType types.Type
	nilCase    bool
	test       api.ExpressionEmission
}

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.TypeSwitchStmt,
) (api.StatementEmission, error) {
	context, targetLabel := context.TakeStatementLabel()
	selected, ok := resolveGuard(context, source)
	if !ok {
		return api.StatementEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
	initializer, err := emitInitializer(context, children, source)
	if err != nil {
		return api.StatementEmission{}, err
	}
	operand, err := children.Expression(
		context.
			WithRole(api.RoleTypeSwitchOperand).
			WithExpectedType(selected.sourceType),
		selected.assertion.X,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	represented, err := children.RepresentedType(
		context.WithRole(api.RoleTypeSwitchOperand),
		selected.assertion.X,
		selected.sourceType,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	valueName, err := context.Names().Temporary(
		api.TemporaryTypeSwitchValue,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	value := context.Factory().Identifier(valueName)
	clauses, requests, err := emitClauses(
		context,
		children,
		source,
		selected,
		value,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	target := tsgo.Statement(
		context.Factory().SwitchStatement(
			context.Factory().TrueLiteral(),
			context.Factory().CaseBlock(clauses),
		),
	)
	if targetLabel != "" {
		target = context.Factory().LabeledStatement(
			context.Factory().Identifier(targetLabel),
			target,
		)
	}
	statements := initializer.Statements()
	statements = append(statements, operand.Before()...)
	statements = append(
		statements,
		context.Factory().VariableStatement(
			nil,
			context.Factory().VariableDeclarationList(
				[]tsgo.VariableDeclaration{
					context.Factory().VariableDeclaration(
						value,
						nil,
						represented.Value(),
						operand.Value(),
					),
				},
				tsgo.NodeFlagsConst,
			),
		),
		target,
	)
	if source.Init != nil {
		statements = []tsgo.Statement{
			context.Factory().Block(statements, true),
		}
	}
	return api.NewStatementEmission(
		statements,
		api.CombineRequests(
			initializer.Requests(),
			operand.Requests(),
			represented.Requests(),
			requests,
		),
	)
}

func resolveGuard(
	context api.Context,
	source *ast.TypeSwitchStmt,
) (guard, bool) {
	if source == nil || source.Body == nil || source.Assign == nil {
		return guard{}, false
	}
	var assertion *ast.TypeAssertExpr
	var binding *ast.Ident
	switch assign := source.Assign.(type) {
	case *ast.ExprStmt:
		assertion, _ = assign.X.(*ast.TypeAssertExpr)
	case *ast.AssignStmt:
		if assign.Tok != token.DEFINE ||
			len(assign.Lhs) != 1 ||
			len(assign.Rhs) != 1 {
			return guard{}, false
		}
		binding, _ = assign.Lhs[0].(*ast.Ident)
		assertion, _ = assign.Rhs[0].(*ast.TypeAssertExpr)
		if binding == nil {
			return guard{}, false
		}
	default:
		return guard{}, false
	}
	if assertion == nil || assertion.X == nil || assertion.Type != nil {
		return guard{}, false
	}
	sourceType := context.TypesInfo().TypeOf(assertion.X)
	sourceInterface, ok := interfacetype.Resolve(sourceType)
	if !ok {
		return guard{}, false
	}
	return guard{
		assertion:  assertion,
		binding:    binding,
		sourceType: sourceType,
		source:     sourceInterface,
	}, true
}

func emitInitializer(
	context api.Context,
	children api.ChildEmitter,
	source *ast.TypeSwitchStmt,
) (api.StatementEmission, error) {
	if source.Init == nil {
		return api.NewStatementEmission(nil, nil)
	}
	return children.ScopedInitializer(
		context.WithRole(api.RoleTypeSwitchInitializer),
		source.Init,
	)
}
