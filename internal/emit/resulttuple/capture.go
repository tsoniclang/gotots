package resulttuple

import (
	"go/ast"
	"go/types"
	"slices"
	"strconv"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type Capture struct {
	name        string
	resultCount int
	statements  []tsgo.Statement
	requests    []api.RootRequest
}

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source ast.Expr,
	results *types.Tuple,
	role api.Role,
) (Capture, error) {
	if source == nil || results == nil || results.Len() < 2 {
		return Capture{}, api.Unsupported(
			context.WithRole(role),
			api.CategoryExpression,
			source,
		)
	}
	sourceResults, ok := context.TypesInfo().TypeOf(source).(*types.Tuple)
	if !ok || !types.Identical(sourceResults, results) {
		return Capture{}, api.Unsupported(
			context.WithRole(role),
			api.CategoryExpression,
			source,
		)
	}
	value, err := children.Expression(
		context.WithRole(role).WithExpectedResults(results),
		source,
	)
	if err != nil {
		return Capture{}, err
	}
	return capture(context, results.Len(), value, nil, nil)
}

func CaptureSynthesized(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	results *types.Tuple,
	value api.ExpressionEmission,
) (Capture, error) {
	if source == nil ||
		children == nil ||
		results == nil ||
		results.Len() < 2 {
		return Capture{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "synthesized multiple-result capture contract is invalid",
		}
	}
	targetType, err := children.RepresentedType(
		context.WithRole(api.RoleResultType),
		source,
		results,
	)
	if err != nil {
		return Capture{}, err
	}
	if _, ok := targetType.Value().(tsgo.TupleTypeNode); !ok {
		return Capture{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "multiple-result capture type is not a tuple",
		}
	}
	return capture(
		context,
		results.Len(),
		value,
		targetType.Value(),
		targetType.Requests(),
	)
}

func capture(
	context api.Context,
	resultCount int,
	value api.ExpressionEmission,
	targetType tsgo.TypeNode,
	typeRequests []api.RootRequest,
) (Capture, error) {
	if resultCount < 2 || value.Value() == nil {
		return Capture{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "multiple-result capture is invalid",
		}
	}
	name, err := context.Names().Temporary(api.TemporaryMultipleResults)
	if err != nil {
		return Capture{}, err
	}
	statements := value.Before()
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
						value.Value(),
					),
				},
				tsgo.NodeFlagsConst,
			),
		),
	)
	return Capture{
		name:        name,
		resultCount: resultCount,
		statements:  statements,
		requests: api.CombineRequests(
			value.Requests(),
			typeRequests,
		),
	}, nil
}

func (c Capture) Statements() []tsgo.Statement {
	return slices.Clone(c.statements)
}

func (c Capture) Requests() []api.RootRequest {
	return slices.Clone(c.requests)
}

func (c Capture) Element(
	context api.Context,
	index int,
) (tsgo.Expression, error) {
	if c.name == "" || index < 0 || index >= c.resultCount {
		return nil, &api.InvariantError{
			Role:   context.Role(),
			Reason: "multiple-result element index is invalid",
		}
	}
	return context.Factory().ElementAccessExpression(
		context.Factory().Identifier(c.name),
		nil,
		context.Factory().NumericLiteral(
			strconv.Itoa(index),
			tsgo.TokenFlagsNone,
		),
		tsgo.NodeFlagsNone,
	), nil
}
