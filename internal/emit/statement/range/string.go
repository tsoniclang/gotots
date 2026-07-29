package rangestatement

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/statement/assignment"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func emitString(
	context api.Context,
	children api.ChildEmitter,
	source *ast.RangeStmt,
	targetLabel string,
) (api.StatementEmission, error) {
	sourceType := context.TypesInfo().TypeOf(source.X)
	operand, err := children.Expression(
		context.
			WithRole(api.RoleRangeExpression).
			WithExpectedType(sourceType),
		source.X,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	if model, ok := definedtype.ResolveBasic(sourceType); ok {
		operand, err = model.Project(context, operand)
		if err != nil {
			return api.StatementEmission{}, err
		}
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
	decodedName, err := context.Names().Temporary(api.TemporaryRangeDecode)
	if err != nil {
		return api.StatementEmission{}, err
	}
	decoded := context.Factory().Identifier(decodedName)
	decoder, err := context.Names().Runtime(
		api.RuntimeStringDecodeRune,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	requests = append(requests, decoder.Requests()...)
	decodedRune := tupleElement(context, decoded, "0")
	decodedWidth := tupleElement(context, decoded, "1")
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
		value, err = iteration(
			types.Typ[types.Rune],
			api.DirectExpression(profileRune(context, decodedRune)),
		)
		if err != nil {
			return api.StatementEmission{}, err
		}
	}
	bindings, err := rangeBindings(
		context,
		children,
		source,
		key,
		value,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	sourceBody, err := children.Block(
		context.WithRole(api.RoleRangeBody).EnterLoop(),
		source.Body,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	loopBody := []tsgo.Statement{
		variable(
			context,
			tsgo.NodeFlagsConst,
			decoded,
			context.Factory().CallExpression(
				context.Factory().Identifier(decoder.Name()),
				nil,
				nil,
				[]tsgo.Expression{receiver, index},
				tsgo.NodeFlagsNone,
			),
		),
	}
	loopBody = append(loopBody, bindings.Statements()...)
	loopBody = append(
		loopBody,
		context.Factory().ExpressionStatement(
			context.Factory().BinaryExpression(
				nil,
				index,
				nil,
				context.Factory().BinaryOperatorToken(
					tsgo.BinaryOperatorPlusEqualsToken,
				),
				decodedWidth,
			),
		),
	)
	loopBody = append(loopBody, sourceBody.Value().Statements()...)
	loop := context.Factory().ForStatement(
		context.Factory().VariableDeclarationList(
			[]tsgo.VariableDeclaration{
				context.Factory().VariableDeclaration(
					index,
					nil,
					nil,
					context.Factory().NumericLiteral(
						"0",
						tsgo.TokenFlagsNone,
					),
				),
			},
			tsgo.NodeFlagsLet,
		),
		context.Factory().BinaryExpression(
			nil,
			index,
			nil,
			context.Factory().BinaryOperatorToken(
				tsgo.BinaryOperatorLessThanToken,
			),
			property(context, receiver, "length"),
		),
		nil,
		context.Factory().Block(loopBody, true),
	)
	before = append(before, labelTarget(context, targetLabel, loop))
	return api.NewStatementEmission(
		before,
		api.CombineRequests(
			requests,
			bindings.Requests(),
			sourceBody.Requests(),
		),
	)
}

func rangeBindings(
	context api.Context,
	children api.ChildEmitter,
	source *ast.RangeStmt,
	key assignment.RangeIterationValue,
	value assignment.RangeIterationValue,
) (api.StatementEmission, error) {
	if source.Key == nil {
		return api.NewStatementEmission(nil, nil)
	}
	return assignment.EmitRangeIteration(
		context.WithRole(api.RoleRangeBody),
		children,
		source,
		key,
		value,
	)
}

func tupleElement(
	context api.Context,
	value tsgo.Expression,
	index string,
) tsgo.ElementAccessExpression {
	return context.Factory().ElementAccessExpression(
		value,
		nil,
		context.Factory().NumericLiteral(index, tsgo.TokenFlagsNone),
		tsgo.NodeFlagsNone,
	)
}

func profileRune(
	context api.Context,
	value tsgo.Expression,
) tsgo.Expression {
	if context.IntegerRepresentation() != api.IntegerRepresentationBigInt {
		return value
	}
	return context.Factory().CallExpression(
		context.Factory().Identifier("BigInt"),
		nil,
		nil,
		[]tsgo.Expression{value},
		tsgo.NodeFlagsNone,
	)
}
