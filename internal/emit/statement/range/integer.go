package rangestatement

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/statement/assignment"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	integervalue "github.com/tsoniclang/gotots/internal/emit/value/integer"
)

func emitInteger(
	context api.Context,
	children api.ChildEmitter,
	source *ast.RangeStmt,
	targetLabel string,
) (api.StatementEmission, error) {
	sourceType := context.TypesInfo().TypeOf(source.X)
	iterationType := integerIterationType(context, source, sourceType)
	if iterationType == nil {
		return api.StatementEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
	operand, err := children.Expression(
		context.
			WithRole(api.RoleRangeExpression).
			WithExpectedType(iterationType),
		source.X,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	operationContext := context
	operationType := iterationType
	if model, ok := definedtype.ResolveBasic(iterationType); ok {
		operationContext, err = model.OperationContext(context)
		if err != nil {
			return api.StatementEmission{}, err
		}
		operationType = model.Underlying()
		operand, err = model.Project(context, operand)
		if err != nil {
			return api.StatementEmission{}, err
		}
	}
	limit, before, requests, err := capture(
		context,
		api.TemporaryRangeOperand,
		operand,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	index, err := rangeIndex(context)
	if err != nil {
		return api.StatementEmission{}, err
	}
	var key assignment.RangeIterationValue
	if source.Key != nil && nonBlank(source.Key) {
		keyValue := api.DirectExpression(index)
		if model, ok := definedtype.ResolveBasic(iterationType); ok {
			keyValue, err = model.Wrap(context, keyValue)
			if err != nil {
				return api.StatementEmission{}, err
			}
		}
		key, err = iteration(iterationType, keyValue)
		if err != nil {
			return api.StatementEmission{}, err
		}
	}
	targetBody, err := body(
		context,
		children,
		source,
		key,
		assignment.RangeIterationValue{},
		targetLabel,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	return numericLoop(
		context,
		before,
		requests,
		index,
		limit,
		targetBody.Value(),
		targetBody.Requests(),
		integervalue.TypeUsesBigInt(operationContext, operationType),
		targetLabel,
	)
}

func integerIterationType(
	context api.Context,
	source *ast.RangeStmt,
	sourceType types.Type,
) types.Type {
	if basic, ok := types.Unalias(sourceType).(*types.Basic); ok &&
		basic.Info()&types.IsUntyped != 0 {
		if source.Key != nil {
			if source.Tok == token.DEFINE {
				identifier, ok := source.Key.(*ast.Ident)
				if !ok {
					return nil
				}
				if object, ok := context.TypesInfo().DefOf(identifier).(*types.Var); ok {
					return object.Type()
				}
			}
			if target := context.TypesInfo().TypeOf(source.Key); target != nil {
				return target
			}
		}
		return types.Default(sourceType)
	}
	return sourceType
}
