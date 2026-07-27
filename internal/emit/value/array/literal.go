package array

import (
	"go/ast"
	"go/constant"
	"go/types"
	"strconv"

	"github.com/tsoniclang/gotots/internal/emit/api"
	arraymember "github.com/tsoniclang/gotots/internal/emit/runtime/array/member"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type literalElement struct {
	index int64
	value api.ExpressionEmission
}

func (a RuntimeArray) EmitLiteral(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CompositeLit,
) (api.ExpressionEmission, error) {
	if source.Type == nil ||
		!types.Identical(context.TypesInfo().TypeOf(source), a.source) ||
		context.ExpectedType() == nil ||
		!types.AssignableTo(a.source, context.ExpectedType()) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	elements, err := a.emitLiteralElements(context, children, source)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	elementZero, err := context.Values().Zero(
		context.WithRole(api.RoleCompositeElement),
		source,
		a.ElementType(),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if len(elementZero.Before()) != 0 {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	before, values, requests, err := arrangeLiteralElements(context, elements)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	indexes := make([]tsgo.Expression, 0, len(elements))
	for _, element := range elements {
		indexes = append(indexes, context.Factory().NumericLiteral(
			strconv.FormatInt(element.index, 10),
			tsgo.TokenFlagsNone,
		))
	}
	typeArguments, typeRequests, err := a.targetTypeArguments(context)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	target, runtimeRequests, err := a.callStatic(
		context,
		arraymember.Literal,
		typeArguments,
		a.lengthLiteral(context),
		elementZero.Value(),
		context.Factory().ArrayLiteralExpression(indexes, false),
		context.Factory().ArrayLiteralExpression(values, false),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.NewExpressionEmission(
		before,
		target,
		api.CombineRequests(
			elementZero.Requests(),
			typeRequests,
			requests,
			runtimeRequests,
		),
	)
}

func (a RuntimeArray) emitLiteralElements(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CompositeLit,
) ([]literalElement, error) {
	result := make([]literalElement, 0, len(source.Elts))
	next := int64(0)
	seen := make(map[int64]struct{}, len(source.Elts))
	for _, sourceElement := range source.Elts {
		index := next
		valueSource := sourceElement
		if keyed, ok := sourceElement.(*ast.KeyValueExpr); ok {
			keyValue := context.TypesInfo().Types[keyed.Key].Value
			var exact bool
			if keyValue != nil {
				index, exact = constant.Int64Val(keyValue)
			}
			if !exact {
				return nil, api.Unsupported(
					context.WithRole(api.RoleCompositeElement),
					api.CategoryExpression,
					keyed.Key,
				)
			}
			valueSource = keyed.Value
		}
		if index < 0 || index >= a.Length() {
			return nil, api.Unsupported(
				context.WithRole(api.RoleCompositeElement),
				api.CategoryExpression,
				sourceElement,
			)
		}
		if _, duplicate := seen[index]; duplicate {
			return nil, api.Unsupported(
				context.WithRole(api.RoleCompositeElement),
				api.CategoryExpression,
				sourceElement,
			)
		}
		seen[index] = struct{}{}
		value, err := children.Expression(
			context.
				WithRole(api.RoleCompositeElement).
				WithExpectedType(a.ElementType()),
			valueSource,
		)
		if err != nil {
			return nil, err
		}
		value, err = context.Values().Copy(
			context.WithRole(api.RoleCompositeElement),
			valueSource,
			a.ElementType(),
			value,
		)
		if err != nil {
			return nil, err
		}
		result = append(result, literalElement{index: index, value: value})
		next = index + 1
	}
	return result, nil
}

func arrangeLiteralElements(
	context api.Context,
	elements []literalElement,
) ([]tsgo.Statement, []tsgo.Expression, []api.RootRequest, error) {
	needsCapture := false
	for _, element := range elements {
		if len(element.value.Before()) != 0 {
			needsCapture = true
			break
		}
	}
	values := make([]tsgo.Expression, 0, len(elements))
	var before []tsgo.Statement
	var requests []api.RootRequest
	for _, element := range elements {
		requests = append(requests, element.value.Requests()...)
		if !needsCapture {
			values = append(values, element.value.Value())
			continue
		}
		name, err := context.Names().Temporary(api.TemporaryCompositeField)
		if err != nil {
			return nil, nil, nil, err
		}
		before = append(before, element.value.Before()...)
		before = append(before, context.Factory().VariableStatement(
			nil,
			context.Factory().VariableDeclarationList(
				[]tsgo.VariableDeclaration{
					context.Factory().VariableDeclaration(
						context.Factory().Identifier(name),
						nil,
						nil,
						element.value.Value(),
					),
				},
				tsgo.NodeFlagsConst,
			),
		))
		values = append(values, context.Factory().Identifier(name))
	}
	return before, values, requests, nil
}
