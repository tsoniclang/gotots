package ordered

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	constantvalue "github.com/tsoniclang/gotots/internal/emit/constant"
	basictype "github.com/tsoniclang/gotots/internal/emit/type/basic"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	floatvalue "github.com/tsoniclang/gotots/internal/emit/value/float"
	integervalue "github.com/tsoniclang/gotots/internal/emit/value/integer"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type operation uint8

const (
	operationInvalid operation = iota
	operationMax
	operationMin
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
	builtin *types.Builtin,
) (api.ExpressionEmission, bool, error) {
	selected := operationInvalid
	switch types.Object(builtin) {
	case types.Universe.Lookup("max"):
		selected = operationMax
	case types.Universe.Lookup("min"):
		selected = operationMin
	default:
		return api.ExpressionEmission{}, false, nil
	}
	if source == nil ||
		source.Ellipsis != token.NoPos ||
		len(source.Args) == 0 {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	resultType := context.TypesInfo().TypeOf(source)
	resultFacts, factsOK := context.TypesInfo().TypeAndValue(source)
	if resultType == nil || !factsOK || resultFacts.Type == nil ||
		!types.Identical(resultFacts.Type, resultType) {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	targetType := resultType
	if expected := context.ExpectedType(); expected != nil {
		if !types.AssignableTo(resultType, expected) {
			return api.ExpressionEmission{}, true,
				api.Unsupported(context, api.CategoryExpression, source)
		}
		if resultFacts.Value != nil {
			targetType = expected
		}
	}
	if resultFacts.Value != nil {
		target, err := constantvalue.EmitValue(
			context,
			source,
			targetType,
			resultFacts.Value,
		)
		return target, true, err
	}
	classifiedType := resultType
	defined, definedResult := definedtype.ResolveBasic(resultType)
	if definedResult {
		classifiedType = defined.Underlying()
	}
	family, ok := classify(context, classifiedType)
	if !ok {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	emissions := make([]api.ExpressionEmission, 0, len(source.Args))
	for _, argument := range source.Args {
		argumentType := context.TypesInfo().TypeOf(argument)
		if argumentType == nil || !types.AssignableTo(argumentType, resultType) {
			return api.ExpressionEmission{}, true,
				api.Unsupported(context, api.CategoryExpression, source)
		}
		target, err := children.Expression(
			context.
				WithRole(api.RoleCallArgument).
				WithExpectedType(resultType),
			argument,
		)
		if err != nil {
			return api.ExpressionEmission{}, true, err
		}
		emissions = append(emissions, target)
	}
	if len(emissions) == 1 {
		return emissions[0], true, nil
	}
	if definedResult {
		for index, emission := range emissions {
			unwrapped, unwrapErr := defined.Project(context, emission)
			if unwrapErr != nil {
				return api.ExpressionEmission{}, true, unwrapErr
			}
			emissions[index] = unwrapped
		}
	}
	values, before, requests, err := arrange(context, emissions)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	var target tsgo.Expression
	switch family {
	case familyNumber:
		target = mathCall(context, selected, values)
	case familyBigInt, familyString:
		target, requests, err = runtimeFold(
			context,
			selected,
			family,
			values,
			requests,
		)
		if err != nil {
			return api.ExpressionEmission{}, true, err
		}
	default:
		return api.ExpressionEmission{}, true,
			&api.InvariantError{
				Role:   context.Role(),
				Reason: "ordered builtin family is invalid",
			}
	}
	result, err := api.NewExpressionEmission(before, target, requests)
	if err == nil && definedResult {
		result, err = defined.Wrap(context, result)
	}
	return result, true, err
}

type valueFamily uint8

const (
	familyInvalid valueFamily = iota
	familyNumber
	familyBigInt
	familyString
)

func classify(
	context api.Context,
	sourceType types.Type,
) (valueFamily, bool) {
	if _, ok := integervalue.Describe(
		context.TypesSizes(),
		sourceType,
	); ok {
		switch context.IntegerRepresentation() {
		case api.IntegerRepresentationNumber:
			return familyNumber, true
		case api.IntegerRepresentationBigInt:
			return familyBigInt, true
		default:
			return familyInvalid, false
		}
	}
	if _, ok := floatvalue.Describe(sourceType); ok {
		return familyNumber, true
	}
	if basictype.SupportsString(sourceType) {
		return familyString, true
	}
	return familyInvalid, false
}

func arrange(
	context api.Context,
	emissions []api.ExpressionEmission,
) (
	[]tsgo.Expression,
	[]tsgo.Statement,
	[]api.RootRequest,
	error,
) {
	capture := false
	for _, emission := range emissions {
		if len(emission.Before()) != 0 {
			capture = true
			break
		}
	}
	values := make([]tsgo.Expression, 0, len(emissions))
	var before []tsgo.Statement
	var requests []api.RootRequest
	for _, emission := range emissions {
		before = append(before, emission.Before()...)
		value := emission.Value()
		if capture {
			name, err := context.Names().Temporary(api.TemporaryCallArgument)
			if err != nil {
				return nil, nil, nil, err
			}
			before = append(
				before,
				context.Factory().VariableStatement(
					nil,
					context.Factory().VariableDeclarationList(
						[]tsgo.VariableDeclaration{
							context.Factory().VariableDeclaration(
								context.Factory().Identifier(name),
								nil,
								nil,
								value,
							),
						},
						tsgo.NodeFlagsConst,
					),
				),
			)
			value = context.Factory().Identifier(name)
		}
		values = append(values, value)
		requests = append(requests, emission.Requests()...)
	}
	return values, before, requests, nil
}

func mathCall(
	context api.Context,
	selected operation,
	values []tsgo.Expression,
) tsgo.Expression {
	member := "max"
	if selected == operationMin {
		member = "min"
	}
	factory := context.Factory()
	return factory.CallExpression(
		factory.PropertyAccessExpression(
			factory.Identifier("Math"),
			nil,
			factory.Identifier(member),
			tsgo.NodeFlagsNone,
		),
		nil,
		nil,
		values,
		tsgo.NodeFlagsNone,
	)
}

func runtimeFold(
	context api.Context,
	selected operation,
	family valueFamily,
	values []tsgo.Expression,
	requests []api.RootRequest,
) (tsgo.Expression, []api.RootRequest, error) {
	symbol := api.RuntimeInvalid
	switch {
	case family == familyBigInt && selected == operationMax:
		symbol = api.RuntimeIntegerMax
	case family == familyBigInt && selected == operationMin:
		symbol = api.RuntimeIntegerMin
	case family == familyString && selected == operationMax:
		symbol = api.RuntimeStringMax
	case family == familyString && selected == operationMin:
		symbol = api.RuntimeStringMin
	}
	reference, err := context.Names().Runtime(symbol, api.ImportPhaseValue)
	if err != nil {
		return nil, nil, err
	}
	result := values[0]
	for _, value := range values[1:] {
		result = context.Factory().CallExpression(
			reference.Expression(context.Factory()),
			nil,
			nil,
			[]tsgo.Expression{result, value},
			tsgo.NodeFlagsNone,
		)
	}
	return result,
		api.CombineRequests(requests, reference.Requests()),
		nil
}
