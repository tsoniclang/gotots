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

type literalSourceElement struct {
	index      int64
	value      ast.Expr
	sourceType types.Type
}

func (a RuntimeArray) EmitLiteral(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CompositeLit,
	sourceType types.Type,
) (api.ExpressionEmission, error) {
	if !types.Identical(
		sourceType,
		a.sourceType,
	) ||
		context.ExpectedType() == nil ||
		!types.AssignableTo(a.sourceType, context.ExpectedType()) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	sourceElements, err := a.literalSourceElements(context, source)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	packed, selected, err := a.emitPackedLiteral(
		context,
		children,
		source,
		sourceElements,
	)
	if selected || err != nil {
		return packed, err
	}
	elements, err := a.emitLiteralElements(
		context,
		children,
		sourceElements,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	before, values, requests, err := arrangeLiteralElements(context, elements)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	var elementZero api.ExpressionEmission
	var target tsgo.Expression
	var typeRequests []api.RootRequest
	var runtimeRequests []api.RootRequest
	if a.aggregate {
		loopZero, zeroErr := context.Values().Zero(
			context.WithRole(api.RoleCompositeElement),
			source,
			a.ElementType(),
		)
		if zeroErr != nil {
			return api.ExpressionEmission{}, zeroErr
		}
		loopZero, zeroErr = context.ContainerStorage().ToContainerStorage(
			context.WithRole(api.RoleCompositeElement),
			source,
			a.ElementType(),
			loopZero,
		)
		if zeroErr != nil {
			return api.ExpressionEmission{}, zeroErr
		}
		resultName, nameErr := context.Names().Temporary(
			api.TemporaryArrayConstruction,
		)
		if nameErr != nil {
			return api.ExpressionEmission{}, nameErr
		}
		indexName, nameErr := context.Names().Temporary(
			api.TemporaryArrayConstruction,
		)
		if nameErr != nil {
			return api.ExpressionEmission{}, nameErr
		}
		target, runtimeRequests, err = a.runtimeOperation(
			context,
			children,
			api.RuntimeArrayAllocate,
			a.lengthLiteral(context),
		)
		if err == nil {
			result := context.Factory().Identifier(resultName)
			index := context.Factory().Identifier(indexName)
			before = append(
				before,
				arrayComparisonVariable(
					context,
					tsgo.NodeFlagsConst,
					resultName,
					target,
				),
				arrayConstructionLoop(
					context,
					index,
					a.lengthLiteral(context),
					"0",
					append(
						loopZero.Before(),
						context.Factory().ExpressionStatement(callMember(
							context,
							result,
							arraymember.Set,
							index,
							loopZero.Value(),
						)),
					),
				),
			)
			for index, element := range elements {
				before = append(
					before,
					context.Factory().ExpressionStatement(callMember(
						context,
						result,
						arraymember.Set,
						context.Factory().NumericLiteral(
							strconv.FormatInt(element.index, 10),
							tsgo.TokenFlagsNone,
						),
						values[index],
					)),
				)
			}
			target = result
			requests = api.CombineRequests(
				requests,
				loopZero.Requests(),
			)
		}
	} else {
		elementZero, err = context.Values().Zero(
			context.WithRole(api.RoleCompositeElement),
			source,
			a.ElementType(),
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		elementZero, err = context.ContainerStorage().ToContainerStorage(
			context.WithRole(api.RoleCompositeElement),
			source,
			a.ElementType(),
			elementZero,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		if len(elementZero.Before()) != 0 {
			return api.ExpressionEmission{},
				api.Unsupported(context, api.CategoryExpression, source)
		}
		indexes := make([]tsgo.Expression, 0, len(elements))
		for _, element := range elements {
			indexes = append(indexes, context.Factory().NumericLiteral(
				strconv.FormatInt(element.index, 10),
				tsgo.TokenFlagsNone,
			))
		}
		var typeArguments []tsgo.TypeNode
		typeArguments, typeRequests, err = a.targetTypeArguments(
			context,
			children,
		)
		if err == nil {
			target, runtimeRequests, err = a.callStatic(
				context,
				arraymember.Literal,
				typeArguments,
				a.lengthLiteral(context),
				elementZero.Value(),
				context.Factory().ArrayLiteralExpression(indexes, false),
				context.Factory().ArrayLiteralExpression(values, false),
			)
		}
	}
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	result, err := api.NewExpressionEmission(
		before,
		target,
		api.CombineRequests(
			elementZero.Requests(),
			typeRequests,
			requests,
			runtimeRequests,
		),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return a.wrap(context, result)
}

func (a RuntimeArray) emitLiteralElements(
	context api.Context,
	children api.ChildEmitter,
	sourceElements []literalSourceElement,
) ([]literalElement, error) {
	result := make([]literalElement, 0, len(sourceElements))
	for _, sourceElement := range sourceElements {
		value, err := children.Expression(
			context.
				WithRole(api.RoleCompositeElement).
				WithExpectedType(a.ElementType()),
			sourceElement.value,
		)
		if err != nil {
			return nil, err
		}
		value, err = context.Values().Transfer(
			context.WithRole(api.RoleCompositeElement),
			sourceElement.value,
			sourceElement.sourceType,
			a.ElementType(),
			api.ValueTransferCopy,
			value,
		)
		if err != nil {
			return nil, err
		}
		value, err = context.ContainerStorage().ToContainerStorage(
			context.WithRole(api.RoleCompositeElement),
			sourceElement.value,
			a.ElementType(),
			value,
		)
		if err != nil {
			return nil, err
		}
		result = append(result, literalElement{
			index: sourceElement.index,
			value: value,
		})
	}
	return result, nil
}

func (a RuntimeArray) literalSourceElements(
	context api.Context,
	source *ast.CompositeLit,
) ([]literalSourceElement, error) {
	result := make([]literalSourceElement, 0, len(source.Elts))
	next := int64(0)
	seen := make(map[int64]struct{}, len(source.Elts))
	for _, sourceElement := range source.Elts {
		index := next
		valueSource := sourceElement
		if keyed, ok := sourceElement.(*ast.KeyValueExpr); ok {
			keyFacts, _ := context.TypesInfo().TypeAndValue(keyed.Key)
			keyValue := keyFacts.Value
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
		valueType := context.TypesInfo().TypeOf(valueSource)
		if valueType == nil ||
			!types.AssignableTo(valueType, a.ElementType()) {
			return nil, api.Unsupported(
				context.WithRole(api.RoleCompositeElement),
				api.CategoryExpression,
				valueSource,
			)
		}
		result = append(result, literalSourceElement{
			index:      index,
			value:      valueSource,
			sourceType: valueType,
		})
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
