package rangestatement

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	pointerruntime "github.com/tsoniclang/gotots/internal/emit/runtime/pointer"
	"github.com/tsoniclang/gotots/internal/emit/statement/assignment"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	arrayvalue "github.com/tsoniclang/gotots/internal/emit/value/array"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func emitArray(
	context api.Context,
	children api.ChildEmitter,
	source *ast.RangeStmt,
	array arrayvalue.RuntimeArray,
	targetLabel string,
) (api.StatementEmission, error) {
	index, err := rangeIndex(context)
	if err != nil {
		return api.StatementEmission{}, err
	}
	var before []tsgo.Statement
	var requests []api.RootRequest
	var receiver tsgo.Identifier
	constantLength, err := constantLengthRangeExpression(context, source.X)
	if err != nil {
		return api.StatementEmission{}, err
	}
	evaluateOperand := source.Value != nil || !constantLength
	if evaluateOperand {
		operand, err := children.Expression(
			context.
				WithRole(api.RoleRangeExpression).
				WithExpectedType(array.SourceType()),
			source.X,
		)
		if err != nil {
			return api.StatementEmission{}, err
		}
		if source.Value != nil && nonBlank(source.Value) {
			operand, err = context.Values().Transfer(
				context.WithRole(api.RoleRangeExpression),
				source.X,
				array.SourceType(),
				array.SourceType(),
				api.ValueTransferCopy,
				operand,
			)
			if err != nil {
				return api.StatementEmission{}, err
			}
		}
		receiver, before, requests, err = capture(
			context,
			api.TemporaryRangeOperand,
			operand,
		)
		if err != nil {
			return api.StatementEmission{}, err
		}
	}
	key, err := arrayRangeKey(context, source, index)
	if err != nil {
		return api.StatementEmission{}, err
	}
	var value assignment.RangeIterationValue
	if source.Value != nil && nonBlank(source.Value) {
		element, elementErr := array.RangeElement(
			context,
			source,
			receiver,
			index,
		)
		if elementErr != nil {
			return api.StatementEmission{}, elementErr
		}
		value, err = iteration(
			array.ElementType(),
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
		context.Factory().NumericLiteral(
			arrayLength(array),
			tsgo.TokenFlagsNone,
		),
		targetBody.Value(),
		targetBody.Requests(),
		false,
		targetLabel,
	)
}

func emitPointerArray(
	context api.Context,
	children api.ChildEmitter,
	source *ast.RangeStmt,
	array arrayvalue.RuntimeArray,
	definedPointer bool,
	targetLabel string,
) (api.StatementEmission, error) {
	index, err := rangeIndex(context)
	if err != nil {
		return api.StatementEmission{}, err
	}
	var before []tsgo.Statement
	var requests []api.RootRequest
	var pointer tsgo.Identifier
	constantLength, err := constantLengthRangeExpression(context, source.X)
	if err != nil {
		return api.StatementEmission{}, err
	}
	evaluateOperand := source.Value != nil || !constantLength
	if evaluateOperand {
		operandType := context.TypesInfo().TypeOf(source.X)
		operand, err := children.Expression(
			context.
				WithRole(api.RoleRangeExpression).
				WithExpectedType(operandType),
			source.X,
		)
		if err != nil {
			return api.StatementEmission{}, err
		}
		if definedPointer {
			model, _ := definedtype.ResolvePointer(operandType)
			operand, err = model.Project(context, operand)
			if err != nil {
				return api.StatementEmission{}, err
			}
		}
		pointer, before, requests, err = capture(
			context,
			api.TemporaryRangeOperand,
			operand,
		)
		if err != nil {
			return api.StatementEmission{}, err
		}
	}
	key, err := arrayRangeKey(context, source, index)
	if err != nil {
		return api.StatementEmission{}, err
	}
	var value assignment.RangeIterationValue
	if source.Value != nil && nonBlank(source.Value) {
		element, elementRequests, err := pointerArrayElement(
			context,
			children,
			source,
			array,
			pointer,
			index,
		)
		if err != nil {
			return api.StatementEmission{}, err
		}
		requests = append(requests, elementRequests...)
		value, err = iteration(array.ElementType(), element)
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
		context.Factory().NumericLiteral(
			arrayLength(array),
			tsgo.TokenFlagsNone,
		),
		targetBody.Value(),
		targetBody.Requests(),
		false,
		targetLabel,
	)
}

func pointerArrayElement(
	context api.Context,
	children api.ChildEmitter,
	source *ast.RangeStmt,
	array arrayvalue.RuntimeArray,
	pointer tsgo.Expression,
	index tsgo.Expression,
) (api.ExpressionEmission, []api.RootRequest, error) {
	logicalType, err := children.RepresentedType(
		context.WithRole(api.RoleRangeExpression),
		source.X,
		array.SourceType(),
	)
	if err != nil {
		return api.ExpressionEmission{}, nil, err
	}
	storageType, err := context.Values().StorageType(
		context.WithRole(api.RoleStorageType),
		source.X,
		array.SourceType(),
	)
	if err != nil {
		return api.ExpressionEmission{}, nil, err
	}
	runtime, err := context.Names().Runtime(
		api.RuntimePointer,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.ExpressionEmission{}, nil, err
	}
	stored := api.DirectExpression(
		pointerruntime.CellValue(
			context.Factory(),
			runtime.Name(),
			logicalType.Value(),
			storageType.Value(),
			pointer,
		),
		api.CombineRequests(
			logicalType.Requests(),
			storageType.Requests(),
			runtime.Requests(),
		)...,
	)
	restored, err := context.Values().FromStorage(
		context.WithRole(api.RoleRangeValue),
		source,
		array.SourceType(),
		stored,
	)
	if err != nil {
		return api.ExpressionEmission{}, nil, err
	}
	element, err := array.RangeElement(
		context,
		source,
		restored.Value(),
		index,
	)
	if err != nil {
		return api.ExpressionEmission{}, nil, err
	}
	return element, api.CombineRequests(
		restored.Requests(),
		element.Requests(),
	), nil
}

func arrayRangeKey(
	context api.Context,
	source *ast.RangeStmt,
	index tsgo.Expression,
) (assignment.RangeIterationValue, error) {
	if source.Key == nil || !nonBlank(source.Key) {
		return assignment.RangeIterationValue{}, nil
	}
	return iteration(
		types.Typ[types.Int],
		api.DirectExpression(profileIndex(context, index)),
	)
}

func rangeIndex(context api.Context) (tsgo.Identifier, error) {
	name, err := context.Names().Temporary(api.TemporaryRangeIndex)
	if err != nil {
		return nil, err
	}
	return context.Factory().Identifier(name), nil
}

func profileIndex(
	context api.Context,
	index tsgo.Expression,
) tsgo.Expression {
	if context.IntegerRepresentation() != api.IntegerRepresentationBigInt {
		return index
	}
	return context.Factory().CallExpression(
		api.TargetIntrinsicBigInt.Expression(context.Factory()),
		nil,
		nil,
		[]tsgo.Expression{index},
		tsgo.NodeFlagsNone,
	)
}

func nonBlank(source ast.Expr) bool {
	identifier, ok := source.(*ast.Ident)
	return !ok || identifier.Name != "_"
}

func arrayLength(array arrayvalue.RuntimeArray) string {
	return fmtInt64(array.Length())
}
