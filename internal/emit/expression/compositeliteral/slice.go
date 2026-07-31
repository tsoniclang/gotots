package compositeliteral

import (
	"go/ast"
	"go/constant"
	"go/types"
	"strconv"

	"github.com/tsoniclang/gotots/internal/emit/api"
	runtimeslice "github.com/tsoniclang/gotots/internal/emit/runtime/slice"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	slicevalue "github.com/tsoniclang/gotots/internal/emit/value/slice"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type sliceElement struct {
	index    int64
	emission api.ExpressionEmission
}

func emitSlice(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CompositeLit,
	sourceType types.Type,
) (api.ExpressionEmission, bool, error) {
	_, elementType, represented := slicevalue.Resolve(sourceType)
	defined, definedOK := definedtype.ResolveSlice(sourceType)
	if definedOK {
		sliceType, _ := defined.Slice()
		elementType = sliceType.Elem()
		represented = true
	}
	if !represented {
		return api.ExpressionEmission{}, false, nil
	}
	if source.Incomplete ||
		context.ExpectedType() == nil ||
		!types.AssignableTo(sourceType, context.ExpectedType()) {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	elements, length, keyed, err := emitSliceElements(
		context,
		children,
		source,
		elementType,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	typeOwner := ast.Node(source)
	if source.Type != nil {
		typeOwner = source.Type
	}
	elementTarget, err := context.ContainerStorage().ContainerStorageType(
		context.WithRole(api.RoleSliceElementType),
		typeOwner,
		elementType,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	runtime, err := context.Names().Runtime(
		api.RuntimeSlice,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	if keyed {
		result, err := emitKeyedSlice(
			context,
			source,
			elementType,
			elementTarget,
			runtime,
			elements,
			length,
		)
		if err == nil && definedOK {
			result, err = defined.Wrap(context, result)
		}
		return result, true, err
	}
	emissions := make([]api.ExpressionEmission, 0, len(elements))
	capture := false
	for _, element := range elements {
		emissions = append(emissions, element.emission)
		capture = capture || len(element.emission.Before()) != 0
	}
	values, before, requests, err := unkeyedSliceElements(
		context,
		emissions,
		capture,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	target := runtimeSliceCall(
		context,
		runtime.Name(),
		runtimeslice.MemberName(runtimeslice.MemberLiteral),
		elementTarget.Value(),
		[]tsgo.Expression{
			context.Factory().ArrayLiteralExpression(values, false),
		},
	)
	result, err := api.NewExpressionEmission(
		before,
		target,
		api.CombineRequests(
			requests,
			elementTarget.Requests(),
			runtime.Requests(),
		),
	)
	if err == nil && definedOK {
		result, err = defined.Wrap(context, result)
	}
	return result, true, err
}

func emitSliceElements(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CompositeLit,
	elementType types.Type,
) ([]sliceElement, int64, bool, error) {
	result := make([]sliceElement, 0, len(source.Elts))
	seen := make(map[int64]struct{}, len(source.Elts))
	var nextIndex int64
	var length int64
	keyed := false
	for _, sourceElement := range source.Elts {
		index := nextIndex
		valueSource := sourceElement
		if keyedElement, ok := sourceElement.(*ast.KeyValueExpr); ok {
			keyed = true
			keyValue := context.TypesInfo().Types[keyedElement.Key].Value
			if keyValue == nil {
				return nil, 0, false,
					api.Unsupported(
						context.WithRole(api.RoleSliceElement),
						api.CategoryExpression,
						sourceElement,
					)
			}
			keyInteger, exact := constant.Int64Val(constant.ToInt(keyValue))
			if !exact || keyInteger < 0 {
				return nil, 0, false,
					api.Unsupported(
						context.WithRole(api.RoleSliceElement),
						api.CategoryExpression,
						sourceElement,
					)
			}
			index = keyInteger
			valueSource = keyedElement.Value
		}
		if _, duplicate := seen[index]; duplicate {
			return nil, 0, false,
				api.Unsupported(
					context.WithRole(api.RoleSliceElement),
					api.CategoryExpression,
					sourceElement,
				)
		}
		seen[index] = struct{}{}
		nextIndex = index + 1
		if nextIndex > length {
			length = nextIndex
		}
		actualType := context.TypesInfo().TypeOf(valueSource)
		if actualType == nil || !types.AssignableTo(actualType, elementType) {
			return nil, 0, false,
				api.Unsupported(
					context.WithRole(api.RoleSliceElement),
					api.CategoryExpression,
					valueSource,
				)
		}
		emission, err := children.Expression(
			context.
				WithRole(api.RoleSliceElement).
				WithExpectedType(elementType),
			valueSource,
		)
		if err != nil {
			return nil, 0, false, err
		}
		emission, err = context.Values().Transfer(
			context.WithRole(api.RoleSliceElement),
			valueSource,
			actualType,
			elementType,
			api.ValueTransferCopy,
			emission,
		)
		if err != nil {
			return nil, 0, false, err
		}
		emission, err = context.ContainerStorage().ToContainerStorage(
			context.WithRole(api.RoleSliceElement),
			valueSource,
			elementType,
			emission,
		)
		if err != nil {
			return nil, 0, false, err
		}
		result = append(result, sliceElement{
			index:    index,
			emission: emission,
		})
	}
	return result, length, keyed, nil
}

func emitKeyedSlice(
	context api.Context,
	source *ast.CompositeLit,
	elementType types.Type,
	elementTarget api.TypeEmission,
	runtime api.NameReference,
	elements []sliceElement,
	length int64,
) (api.ExpressionEmission, error) {
	name, err := context.Names().Temporary(api.TemporarySliceReceiver)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	lengthValue := context.Factory().NumericLiteral(
		strconv.FormatInt(length, 10),
		tsgo.TokenFlagsNone,
	)
	var result api.ExpressionEmission
	if context.Values().RequiresStructuralCopy(context, elementType) {
		result, err = slicevalue.MakeAggregate(
			context,
			elementTarget,
			source,
			elementType,
			lengthValue,
			context.Factory().NullLiteral(),
			nil,
			nil,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
	} else {
		zero, zeroErr := context.Values().Zero(
			context.WithRole(api.RoleSliceElement),
			source,
			elementType,
		)
		if zeroErr != nil {
			return api.ExpressionEmission{}, zeroErr
		}
		zero, zeroErr = context.ContainerStorage().ToContainerStorage(
			context.WithRole(api.RoleSliceElement),
			source,
			elementType,
			zero,
		)
		if zeroErr != nil {
			return api.ExpressionEmission{}, zeroErr
		}
		result, err = api.NewExpressionEmission(
			zero.Before(),
			runtimeSliceCall(
				context,
				runtime.Name(),
				runtimeslice.MemberName(runtimeslice.MemberMake),
				elementTarget.Value(),
				[]tsgo.Expression{
					lengthValue,
					context.Factory().NullLiteral(),
					zero.Value(),
				},
			),
			api.CombineRequests(
				elementTarget.Requests(),
				runtime.Requests(),
				zero.Requests(),
			),
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
	}
	before := append(
		result.Before(),
		context.Factory().VariableStatement(
			nil,
			context.Factory().VariableDeclarationList(
				[]tsgo.VariableDeclaration{
					context.Factory().VariableDeclaration(
						context.Factory().Identifier(name),
						nil,
						nil,
						result.Value(),
					),
				},
				tsgo.NodeFlagsConst,
			),
		),
	)
	requests := result.Requests()
	for _, element := range elements {
		before = append(before, element.emission.Before()...)
		before = append(before, context.Factory().ExpressionStatement(
			context.Factory().CallExpression(
				context.Factory().PropertyAccessExpression(
					context.Factory().Identifier(name),
					nil,
					context.Factory().Identifier(
						runtimeslice.MemberName(runtimeslice.MemberSet),
					),
					tsgo.NodeFlagsNone,
				),
				nil,
				nil,
				[]tsgo.Expression{
					context.Factory().NumericLiteral(
						strconv.FormatInt(element.index, 10),
						tsgo.TokenFlagsNone,
					),
					element.emission.Value(),
				},
				tsgo.NodeFlagsNone,
			),
		))
		requests = append(requests, element.emission.Requests()...)
	}
	return api.NewExpressionEmission(
		before,
		context.Factory().Identifier(name),
		requests,
	)
}

func runtimeSliceCall(
	context api.Context,
	runtimeName string,
	method string,
	elementType tsgo.TypeNode,
	arguments []tsgo.Expression,
) tsgo.CallExpression {
	return context.Factory().CallExpression(
		context.Factory().PropertyAccessExpression(
			context.Factory().Identifier(runtimeName),
			nil,
			context.Factory().Identifier(method),
			tsgo.NodeFlagsNone,
		),
		nil,
		[]tsgo.TypeNode{elementType},
		arguments,
		tsgo.NodeFlagsNone,
	)
}

func unkeyedSliceElements(
	context api.Context,
	emissions []api.ExpressionEmission,
	capture bool,
) (
	[]tsgo.Expression,
	[]tsgo.Statement,
	[]api.RootRequest,
	error,
) {
	values := make([]tsgo.Expression, 0, len(emissions))
	var before []tsgo.Statement
	var requests []api.RootRequest
	for _, emission := range emissions {
		requests = append(requests, emission.Requests()...)
		if !capture {
			values = append(values, emission.Value())
			continue
		}
		name, err := context.Names().Temporary(api.TemporarySliceElement)
		if err != nil {
			return nil, nil, nil, err
		}
		before = append(before, emission.Before()...)
		before = append(before, context.Factory().VariableStatement(
			nil,
			context.Factory().VariableDeclarationList(
				[]tsgo.VariableDeclaration{
					context.Factory().VariableDeclaration(
						context.Factory().Identifier(name),
						nil,
						nil,
						emission.Value(),
					),
				},
				tsgo.NodeFlagsConst,
			),
		))
		values = append(values, context.Factory().Identifier(name))
	}
	return values, before, requests, nil
}
