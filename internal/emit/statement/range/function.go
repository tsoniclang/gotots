package rangestatement

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	panicruntime "github.com/tsoniclang/gotots/internal/emit/runtime/panic"
	"github.com/tsoniclang/gotots/internal/emit/statement/assignment"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

const (
	rangeDoneMessage         = "range function continued iteration after function for loop body returned false"
	rangePanicMessage        = "range function continued iteration after loop body panic"
	rangeExhaustedMessage    = "range function continued iteration after whole loop exit"
	rangeMissingPanicMessage = "range function recovered a loop body panic and did not resume panicking"
)

func emitIterator(
	context api.Context,
	children api.ChildEmitter,
	source *ast.RangeStmt,
	sourceType types.Type,
	signature *types.Signature,
	targetLabel string,
) (api.StatementEmission, error) {
	yield, ok := iteratorYield(signature)
	if !ok ||
		targetLabel != "" ||
		!iteratorClause(source, yield.Params().Len()) {
		return api.StatementEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
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
	iterator, before, requests, err := capture(
		context,
		api.TemporaryRangeOperand,
		operand,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	targetIterator := tsgo.Expression(iterator)
	if !callable.StaticallyNonNil(context.TypesInfo(), source.X) {
		guard, guardRequests, guardErr := callable.NilGuard(
			context.WithRole(api.RoleRangeExpression),
			targetIterator,
		)
		if guardErr != nil {
			return api.StatementEmission{}, guardErr
		}
		before = append(before, guard)
		requests = append(requests, guardRequests...)
	}
	if model, defined := definedtype.ResolveCallable(sourceType); defined {
		targetIterator = model.Unwrap(context.Factory(), targetIterator)
	}
	stateName, err := context.Names().Temporary(api.TemporaryRangeState)
	if err != nil {
		return api.StatementEmission{}, err
	}
	before = append(
		before,
		stateDeclaration(context, stateName, api.IteratorRangeStateReady),
	)
	panicReference, err := context.Names().Runtime(
		api.RuntimePanic,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	callback, callbackRequests, err := iteratorCallback(
		context,
		children,
		source,
		yield,
		stateName,
		panicReference.Name(),
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	before = append(
		before,
		context.Factory().ExpressionStatement(
			context.Factory().CallExpression(
				targetIterator,
				nil,
				nil,
				[]tsgo.Expression{callback},
				tsgo.NodeFlagsNone,
			),
		),
		statePanic(
			context,
			stateName,
			api.IteratorRangeStatePanicked,
			panicReference.Name(),
			rangeMissingPanicMessage,
		),
		stateAssignment(
			context,
			stateName,
			api.IteratorRangeStateExhausted,
		),
	)
	return api.NewStatementEmission(
		before,
		api.CombineRequests(
			requests,
			panicReference.Requests(),
			callbackRequests,
		),
	)
}

func iteratorYield(
	iterator *types.Signature,
) (*types.Signature, bool) {
	if iterator == nil ||
		iterator.Recv() != nil ||
		iterator.Variadic() ||
		iterator.Params().Len() != 1 ||
		iterator.Results().Len() != 0 {
		return nil, false
	}
	yield, ok := callable.Signature(iterator.Params().At(0).Type())
	if !ok ||
		yield.Recv() != nil ||
		yield.Variadic() ||
		yield.Params().Len() > 2 ||
		yield.Results().Len() != 1 ||
		!types.Identical(
			yield.Results().At(0).Type(),
			types.Typ[types.Bool],
		) {
		return nil, false
	}
	return yield, true
}

func iteratorClause(source *ast.RangeStmt, yielded int) bool {
	if source == nil || yielded < 0 || yielded > 2 {
		return false
	}
	switch {
	case source.Key == nil:
		return source.Value == nil && source.Tok == 0
	case source.Value == nil:
		return yielded >= 1 && validClause(source)
	default:
		return yielded == 2 && validClause(source)
	}
}

func iteratorCallback(
	context api.Context,
	children api.ChildEmitter,
	source *ast.RangeStmt,
	yield *types.Signature,
	stateName string,
	panicName string,
) (tsgo.ArrowFunction, []api.RootRequest, error) {
	targetSignature, err := callable.EmitAdapter(
		context.WithRole(api.RoleRangeValue),
		children,
		source,
		yield,
	)
	if err != nil {
		return nil, nil, err
	}
	parameters := targetSignature.ParameterReferences(context.Factory())
	key, value, err := iteratorValues(
		context,
		source,
		yield,
		parameters,
	)
	if err != nil {
		return nil, nil, err
	}
	bodyContext := context.
		WithRole(api.RoleRangeBody).
		EnterIteratorRange(stateName)
	var bindings api.StatementEmission
	if source.Key != nil {
		bindings, err = assignment.EmitRangeIteration(
			bodyContext,
			children,
			source,
			key,
			value,
		)
		if err != nil {
			return nil, nil, err
		}
	}
	sourceBody, err := children.Block(bodyContext, source.Body)
	if err != nil {
		return nil, nil, err
	}
	statements := iteratorEntryGuards(
		context,
		stateName,
		panicName,
	)
	statements = append(
		statements,
		stateAssignment(
			context,
			stateName,
			api.IteratorRangeStatePanicked,
		),
	)
	statements = append(statements, bindings.Statements()...)
	statements = append(statements, sourceBody.Value().Statements()...)
	statements = append(
		statements,
		stateAssignment(
			context,
			stateName,
			api.IteratorRangeStateReady,
		),
		context.Factory().ReturnStatement(
			context.Factory().TrueLiteral(),
		),
	)
	return context.Factory().ArrowFunction(
			nil,
			nil,
			targetSignature.Parameters(),
			targetSignature.Result(),
			context.Factory().EqualsGreaterThanToken(),
			context.Factory().Block(statements, true),
		), api.CombineRequests(
			targetSignature.Requests(),
			bindings.Requests(),
			sourceBody.Requests(),
		), nil
}

func iteratorValues(
	context api.Context,
	source *ast.RangeStmt,
	yield *types.Signature,
	parameters []tsgo.Expression,
) (
	assignment.RangeIterationValue,
	assignment.RangeIterationValue,
	error,
) {
	var key assignment.RangeIterationValue
	var value assignment.RangeIterationValue
	var err error
	if source.Key != nil && nonBlank(source.Key) {
		key, err = assignment.NewRangeIterationValue(
			yield.Params().At(0).Type(),
			api.DirectExpression(parameters[0]),
		)
		if err != nil {
			return key, value, err
		}
	}
	if source.Value != nil && nonBlank(source.Value) {
		value, err = assignment.NewRangeIterationValue(
			yield.Params().At(1).Type(),
			api.DirectExpression(parameters[1]),
		)
	}
	return key, value, err
}

func iteratorEntryGuards(
	context api.Context,
	stateName string,
	panicName string,
) []tsgo.Statement {
	return []tsgo.Statement{
		statePanic(
			context,
			stateName,
			api.IteratorRangeStateDone,
			panicName,
			rangeDoneMessage,
		),
		statePanic(
			context,
			stateName,
			api.IteratorRangeStatePanicked,
			panicName,
			rangePanicMessage,
		),
		statePanic(
			context,
			stateName,
			api.IteratorRangeStateExhausted,
			panicName,
			rangeExhaustedMessage,
		),
	}
}

func stateDeclaration(
	context api.Context,
	name string,
	state api.IteratorRangeState,
) tsgo.VariableStatement {
	return variable(
		context,
		tsgo.NodeFlagsLet,
		context.Factory().Identifier(name),
		stateLiteral(context, state),
	)
}

func stateAssignment(
	context api.Context,
	name string,
	state api.IteratorRangeState,
) tsgo.Statement {
	return context.Factory().ExpressionStatement(
		context.Factory().BinaryExpression(
			nil,
			context.Factory().Identifier(name),
			nil,
			context.Factory().BinaryOperatorToken(
				tsgo.BinaryOperatorEqualsToken,
			),
			stateLiteral(context, state),
		),
	)
}

func statePanic(
	context api.Context,
	name string,
	state api.IteratorRangeState,
	panicName string,
	message string,
) tsgo.IfStatement {
	return context.Factory().IfStatement(
		context.Factory().BinaryExpression(
			nil,
			context.Factory().Identifier(name),
			nil,
			context.Factory().BinaryOperatorToken(
				tsgo.BinaryOperatorEqualsEqualsEqualsToken,
			),
			stateLiteral(context, state),
		),
		context.Factory().Block(
			[]tsgo.Statement{
				context.Factory().ExpressionStatement(
					panicruntime.Call(
						context.Factory(),
						panicName,
						context.Factory().StringLiteral(
							message,
							tsgo.TokenFlagsNone,
						),
					),
				),
			},
			true,
		),
		nil,
	)
}

func stateLiteral(
	context api.Context,
	state api.IteratorRangeState,
) tsgo.NumericLiteral {
	literal := state.Literal()
	if literal == "" {
		panic("iterator-range state is invalid")
	}
	return context.Factory().NumericLiteral(
		literal,
		tsgo.TokenFlagsNone,
	)
}
