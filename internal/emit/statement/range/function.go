package rangestatement

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	cooperativecall "github.com/tsoniclang/gotots/internal/emit/concurrency/cooperative"
	panicruntime "github.com/tsoniclang/gotots/internal/emit/runtime/panic"
	"github.com/tsoniclang/gotots/internal/emit/statement/assignment"
	"github.com/tsoniclang/gotots/internal/emit/statement/returnstatement"
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
	if model, defined := definedtype.ResolveCallable(sourceType); defined {
		projected, projectErr := model.Project(
			context.WithRole(api.RoleRangeExpression),
			api.DirectExpression(targetIterator),
		)
		if projectErr != nil {
			return api.StatementEmission{}, projectErr
		}
		before = append(before, projected.Before()...)
		requests = append(requests, projected.Requests()...)
		targetIterator = projected.Value()
	}
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
	stateName, err := context.Names().Temporary(api.TemporaryRangeState)
	if err != nil {
		return api.StatementEmission{}, err
	}
	before = append(
		before,
		stateDeclaration(context, stateName, api.IteratorRangeStateReady),
	)
	returnSelected := context.CallableControl().IteratorReturn(source)
	resultName, resultPrelude, resultRequests, err := iteratorReturnStorage(
		context,
		children,
		source,
		returnSelected,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	before = append(before, resultPrelude...)
	requests = append(requests, resultRequests...)
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
		resultName,
		returnSelected,
		panicReference.Name(),
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	invocation, err := cooperativecall.ValueCall(
		context.WithRole(api.RoleRangeExpression),
		source,
		signature,
		api.DirectExpression(
			context.Factory().CallExpression(
				targetIterator,
				nil,
				nil,
				[]tsgo.Expression{callback},
				tsgo.NodeFlagsNone,
			),
			callbackRequests...,
		),
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	after := append(
		invocation.Before(),
		context.Factory().ExpressionStatement(
			invocation.Value(),
		),
	)
	after = append(after,
		statePanic(
			context,
			stateName,
			api.IteratorRangeStatePanicked,
			panicReference.Name(),
			rangeMissingPanicMessage,
		),
	)
	requests = append(requests, invocation.Requests()...)
	if returnSelected {
		propagation, propagationErr := iteratorReturnPropagation(
			context,
			children,
			source,
			stateName,
			resultName,
		)
		if propagationErr != nil {
			return api.StatementEmission{}, propagationErr
		}
		after = append(after, propagation.Statements()...)
		requests = append(requests, propagation.Requests()...)
	}
	after = append(
		after,
		stateAssignment(
			context,
			stateName,
			api.IteratorRangeStateExhausted,
		),
	)
	before = append(before, after...)
	return api.NewStatementEmission(
		before,
		api.CombineRequests(
			requests,
			panicReference.Requests(),
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
	resultName string,
	returnSelected bool,
	panicName string,
) (tsgo.ArrowFunction, []api.RootRequest, error) {
	reference, err := callable.ABIReference(context, yield)
	if err != nil {
		return nil, nil, err
	}
	facet, err := context.CallableABIFacet(reference)
	if err != nil {
		return nil, nil, err
	}
	observation, err := context.ObserveCooperativeCallable(facet)
	if err != nil {
		return nil, nil, err
	}
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
	control, err := api.NewIteratorRangeControl(
		source,
		stateName,
		resultName,
		returnSelected,
	)
	if err != nil {
		return nil, nil, err
	}
	bodyContext := context.
		WithRole(api.RoleRangeBody).
		EnterIteratorRange(control).
		WithCooperativeCallableABI(facet, observation.Cooperative())
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
	var modifiers []tsgo.ModifierLike
	resultType := targetSignature.Result()
	if observation.Cooperative() {
		modifiers = []tsgo.ModifierLike{context.Factory().AsyncKeyword()}
		resultType = callable.PromiseResult(context.Factory(), resultType)
	}
	return context.Factory().ArrowFunction(
			modifiers,
			nil,
			targetSignature.Parameters(),
			resultType,
			context.Factory().EqualsGreaterThanToken(),
			context.Factory().Block(statements, true),
		), api.CombineRequests(
			reference.Requests(),
			observation.Requests(),
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
		statePanic(
			context,
			stateName,
			api.IteratorRangeStateReturned,
			panicName,
			rangeDoneMessage,
		),
	}
}

func iteratorReturnStorage(
	context api.Context,
	children api.ChildEmitter,
	source *ast.RangeStmt,
	selected bool,
) (string, []tsgo.Statement, []api.RootRequest, error) {
	results := context.FunctionResults()
	if !selected || results == nil || results.Len() == 0 {
		return "", nil, nil, nil
	}
	name, err := context.Names().Temporary(api.TemporaryRangeReturn)
	if err != nil {
		return "", nil, nil, err
	}
	targetType, typeRequests, err := callable.EmitResultType(
		context.WithRole(api.RoleResultType),
		children,
		source,
		results,
	)
	if err != nil {
		return "", nil, nil, err
	}
	zero, err := callable.ZeroResult(context, source, results)
	if err != nil {
		return "", nil, nil, err
	}
	statements := zero.Before()
	statements = append(
		statements,
		context.Factory().VariableStatement(
			nil,
			context.Factory().VariableDeclarationList(
				[]tsgo.VariableDeclaration{
					context.Factory().VariableDeclaration(
						context.Factory().Identifier(name),
						nil,
						targetType,
						zero.Value(),
					),
				},
				tsgo.NodeFlagsLet,
			),
		),
	)
	return name,
		statements,
		api.CombineRequests(typeRequests, zero.Requests()),
		nil
}

func iteratorReturnPropagation(
	context api.Context,
	children api.ChildEmitter,
	source *ast.RangeStmt,
	stateName string,
	resultName string,
) (api.StatementEmission, error) {
	var result tsgo.Expression
	if resultName != "" {
		result = context.Factory().Identifier(resultName)
	}
	propagated, err := returnstatement.Propagate(
		context,
		children,
		source,
		result,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	return api.NewStatementEmission(
		[]tsgo.Statement{
			context.Factory().IfStatement(
				context.Factory().BinaryExpression(
					nil,
					context.Factory().Identifier(stateName),
					nil,
					context.Factory().BinaryOperatorToken(
						tsgo.BinaryOperatorEqualsEqualsEqualsToken,
					),
					stateLiteral(
						context,
						api.IteratorRangeStateReturned,
					),
				),
				context.Factory().Block(
					propagated.Statements(),
					true,
				),
				nil,
			),
		},
		propagated.Requests(),
	)
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
