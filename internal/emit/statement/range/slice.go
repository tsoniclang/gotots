package rangestatement

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/statement/assignment"
	slicevalue "github.com/tsoniclang/gotots/internal/emit/value/slice"
)

func emitSlice(
	context api.Context,
	children api.ChildEmitter,
	source *ast.RangeStmt,
	targetLabel string,
) (api.StatementEmission, error) {
	sourceType := context.TypesInfo().TypeOf(source.X)
	_, elementType, ok := slicevalue.Source(sourceType)
	if !ok {
		return api.StatementEmission{},
			api.Unsupported(context, api.CategoryExpression, source.X)
	}
	operand, err := children.Expression(
		context.
			WithRole(api.RoleRangeExpression).
			WithExpectedType(sourceType),
		source.X,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	operand, err = slicevalue.Project(context, sourceType, operand)
	if err != nil {
		return api.StatementEmission{}, err
	}
	receiver, before, requests, err := capture(
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
		key, err = iteration(
			types.Typ[types.Int],
			api.DirectExpression(profileIndex(context, index)),
		)
		if err != nil {
			return api.StatementEmission{}, err
		}
	}
	var value assignment.RangeIterationValue
	if source.Value != nil && nonBlank(source.Value) {
		element, elementErr := slicevalue.RangeElement(
			context,
			source,
			elementType,
			receiver,
			index,
		)
		if elementErr != nil {
			return api.StatementEmission{}, elementErr
		}
		value, err = iteration(
			elementType,
			element,
		)
		if err != nil {
			return api.StatementEmission{}, err
		}
	}
	targetBody, err := body(
		context,
		children,
		source,
		key,
		value,
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
		slicevalue.RangeLength(context, receiver),
		targetBody.Value(),
		targetBody.Requests(),
		false,
		targetLabel,
	)
}
