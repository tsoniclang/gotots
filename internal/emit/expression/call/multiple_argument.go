package call

import (
	"go/ast"
	"go/types"
	"strconv"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func emitMultipleArgument(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
	signature *types.Signature,
	results *types.Tuple,
) ([]tsgo.Expression, []tsgo.Statement, []api.PlacementRequest, error) {
	if results == nil || results.Len() != signature.Params().Len() {
		return nil, nil, nil,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	for index := range results.Len() {
		if !types.AssignableTo(results.At(index).Type(), signature.Params().At(index).Type()) {
			return nil, nil, nil,
				api.Unsupported(context, api.CategoryExpression, source)
		}
	}
	value, err := children.Expression(
		context.
			WithRole(api.RoleCallArgument).
			WithExpectedResults(results),
		source.Args[0],
	)
	if err != nil {
		return nil, nil, nil, err
	}
	temporaryName, err := context.Names().Temporary(api.TemporaryMultipleResults)
	if err != nil {
		return nil, nil, nil, err
	}
	declaration := context.Factory().VariableDeclaration(
		context.Factory().Identifier(temporaryName),
		nil,
		nil,
		value.Value(),
	)
	before := value.Before()
	before = append(
		before,
		context.Factory().VariableStatement(
			nil,
			context.Factory().VariableDeclarationList(
				[]tsgo.VariableDeclaration{declaration},
				tsgo.NodeFlagsConst,
			),
		),
	)

	arguments := make([]tsgo.Expression, 0, results.Len())
	requests := value.Requests()
	for index := range results.Len() {
		element := context.Factory().ElementAccessExpression(
			context.Factory().Identifier(temporaryName),
			nil,
			context.Factory().NumericLiteral(strconv.Itoa(index), tsgo.TokenFlagsNone),
			tsgo.NodeFlagsNone,
		)
		copied, err := context.Values().Copy(
			context.WithRole(api.RoleCallArgument),
			source.Args[0],
			signature.Params().At(index).Type(),
			api.DirectExpression(element),
		)
		if err != nil {
			return nil, nil, nil, err
		}
		if len(copied.Before()) != 0 {
			return nil, nil, nil,
				api.Unsupported(context, api.CategoryExpression, source)
		}
		arguments = append(arguments, copied.Value())
		requests = append(requests, copied.Requests()...)
	}
	return arguments, before, requests, nil
}
