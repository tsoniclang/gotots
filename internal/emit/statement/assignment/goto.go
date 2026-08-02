package assignment

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func emitGotoDefinition(
	context api.Context,
	children api.ChildEmitter,
	source *ast.AssignStmt,
) (api.StatementEmission, bool, error) {
	if len(source.Lhs) != 1 || len(source.Rhs) != 1 {
		return api.StatementEmission{}, false, nil
	}
	name, ok := source.Lhs[0].(*ast.Ident)
	if !ok {
		return api.StatementEmission{}, false, nil
	}
	object, ok := context.TypesInfo().DefOf(name).(*types.Var)
	if !ok || !context.IsGotoLocal(object) {
		return api.StatementEmission{}, false, nil
	}
	targetType := context.TypesInfo().TypeOfObject(object)
	sourceType := context.TypesInfo().TypeOf(source.Rhs[0])
	if sourceType == nil || targetType == nil ||
		!types.AssignableTo(sourceType, targetType) {
		return api.StatementEmission{}, true,
			api.Unsupported(context, api.CategoryStatement, source)
	}
	value, err := children.Expression(
		context.
			WithRole(api.RoleLocalValue).
			WithExpectedType(targetType),
		source.Rhs[0],
	)
	if err != nil {
		return api.StatementEmission{}, true, err
	}
	value, err = context.Values().Transfer(
		context.WithRole(api.RoleLocalValue),
		source.Rhs[0],
		sourceType,
		targetType,
		api.ValueTransferCopy,
		value,
	)
	if err != nil {
		return api.StatementEmission{}, true, err
	}
	targetName, storage := context.AddressableStorage().Name(context, object)
	if storage {
		value, err = context.AddressableStorage().Cell(
			context,
			children,
			name,
			targetType,
			value,
		)
	} else {
		targetName, err = context.Names().Declare(object)
	}
	if err != nil {
		return api.StatementEmission{}, true, err
	}
	statements := append(
		value.Before(),
		assignmentStatement(context, targetName, value.Value()),
	)
	target, err := api.NewStatementEmission(statements, value.Requests())
	return target, true, err
}
