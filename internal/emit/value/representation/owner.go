package representation

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	mapruntime "github.com/tsoniclang/gotots/internal/emit/runtime/map"
	pointerruntime "github.com/tsoniclang/gotots/internal/emit/runtime/pointer"
	basictype "github.com/tsoniclang/gotots/internal/emit/type/basic"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	pointertype "github.com/tsoniclang/gotots/internal/emit/type/pointer"
	arrayvalue "github.com/tsoniclang/gotots/internal/emit/value/array"
	complexvalue "github.com/tsoniclang/gotots/internal/emit/value/complex"
	"github.com/tsoniclang/gotots/internal/emit/value/maprepresentation"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type Owner struct{}

func (Owner) RequiresCustomEquality(
	context api.Context,
	sourceType types.Type,
) bool {
	if _, ok := definedtype.Resolve(sourceType); ok {
		return true
	}
	if _, ok := complexvalue.Describe(sourceType); ok {
		return true
	}
	if _, ok := arrayvalue.Resolve(context, sourceType); ok {
		return true
	}
	if _, ok := isAnonymousStruct(sourceType); ok {
		return true
	}
	if pointerValue(sourceType) {
		return true
	}
	if callableValue(sourceType) {
		return true
	}
	_, _, ok := namedStruct(sourceType)
	return ok
}

func (Owner) RequiresExplicitType(
	context api.Context,
	sourceType types.Type,
) bool {
	if defined, ok := definedtype.Resolve(sourceType); ok {
		return defined.NilCapable()
	}
	return pointerValue(sourceType) || callableValue(sourceType)
}

func (Owner) RequiresStructuralCopy(
	context api.Context,
	sourceType types.Type,
) bool {
	if defined, ok := definedtype.Resolve(sourceType); ok {
		return defined.Family() == definedtype.FamilyArray
	}
	if _, ok := arrayvalue.Resolve(context, sourceType); ok {
		return true
	}
	if _, ok := isAnonymousStruct(sourceType); ok {
		return true
	}
	_, _, ok := namedStruct(sourceType)
	return ok
}

func (Owner) Zero(
	context api.Context,
	source ast.Node,
	sourceType types.Type,
) (api.ExpressionEmission, error) {
	if defined, ok := definedtype.Resolve(sourceType); ok {
		zero, err := (Owner{}).Zero(
			context.WithRole(api.RoleDefinedValue),
			source,
			defined.Underlying(),
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		if defined.Family() == definedtype.FamilyCallable {
			return zero, nil
		}
		return defined.Wrap(context, zero)
	}
	if carrier, ok := complexvalue.Describe(sourceType); ok {
		return complexvalue.Construct(
			context,
			carrier,
			context.Factory().NumericLiteral("0", tsgo.TokenFlagsNone),
			context.Factory().NumericLiteral("0", tsgo.TokenFlagsNone),
		)
	}
	if mapType, ok := maprepresentation.Source(context, sourceType); ok {
		zero, err := (Owner{}).Zero(
			context.WithRole(api.RoleMapValue),
			source,
			mapType.Elem(),
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		if len(zero.Before()) != 0 {
			return api.ExpressionEmission{},
				api.Unsupported(context, api.CategoryExpression, source)
		}
		reference, typeArguments, err := maprepresentation.Reference(
			context,
			source,
			sourceType,
			api.ImportPhaseValue,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		nilName, err := mapruntime.Name(mapruntime.MemberNil)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		return api.DirectExpression(
			context.Factory().CallExpression(
				context.Factory().PropertyAccessExpression(
					context.Factory().Identifier(reference.Name()),
					nil,
					context.Factory().Identifier(nilName),
					tsgo.NodeFlagsNone,
				),
				nil,
				typeArguments,
				[]tsgo.Expression{zero.Value()},
				tsgo.NodeFlagsNone,
			),
			api.CombineRequests(
				reference.Requests(),
				zero.Requests(),
			)...,
		), nil
	}
	if array, ok := arrayvalue.Resolve(context, sourceType); ok {
		return array.Zero(context, source)
	}
	if structType, ok := isAnonymousStruct(sourceType); ok {
		return anonymousStructZero(context, source, structType)
	}
	if alias, ok := primitive(context, sourceType); ok {
		var literal tsgo.Expression
		switch alias {
		case api.PrimitiveBool:
			literal = context.Factory().FalseLiteral()
		case api.PrimitiveString:
			literal = context.Factory().StringLiteral("", tsgo.TokenFlagsNone)
		case api.PrimitiveFloat32, api.PrimitiveFloat64:
			literal = context.Factory().NumericLiteral("0", tsgo.TokenFlagsNone)
		default:
			var err error
			literal, err = api.IntegerLiteral(
				context.Factory(),
				context.IntegerRepresentation(),
				"0",
			)
			if err != nil {
				return api.ExpressionEmission{}, err
			}
		}
		return api.DirectExpression(literal), nil
	}
	if _, _, ok := pointertype.Resolve(sourceType); ok {
		return api.DirectExpression(
			context.Factory().VoidExpression(
				context.Factory().NumericLiteral("0", tsgo.TokenFlagsNone),
			),
		), nil
	}
	if callableValue(sourceType) {
		return api.DirectExpression(
			context.Factory().VoidExpression(
				context.Factory().NumericLiteral("0", tsgo.TokenFlagsNone),
			),
		), nil
	}
	if _, _, ok := scalarSlice(context, sourceType); ok {
		return sliceZero(context, source, sourceType)
	}
	typeName, _, ok := namedStruct(sourceType)
	if !ok {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	reference, err := context.Names().NamedStructOperation(
		typeName,
		api.NamedStructOperationZero,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	call, err := namedStructOperationCall(
		context,
		reference.Name(),
		api.NamedStructOperationZero,
		nil,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.DirectExpression(
		call,
		reference.Requests()...,
	), nil
}

func (Owner) Copy(
	context api.Context,
	source ast.Node,
	sourceType types.Type,
	value api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	if target, ok := definedtype.ResolveCallable(sourceType); ok {
		actual := expressionType(context, source)
		if actual == nil || types.Identical(actual, sourceType) {
			return api.NewExpressionEmission(
				value.Before(),
				value.Value(),
				value.Requests(),
			)
		}
		return target.Wrap(context, value)
	}
	if expression := expressionType(context, source); expression != nil &&
		!types.Identical(expression, sourceType) {
		if actual, ok := definedtype.ResolveCallable(expression); ok {
			return actual.Project(context, value)
		}
	}
	if defined, ok := definedtype.Resolve(sourceType); ok &&
		defined.Family() != definedtype.FamilyArray {
		return api.NewExpressionEmission(
			value.Before(),
			value.Value(),
			value.Requests(),
		)
	}
	if array, ok := arrayvalue.Resolve(context, sourceType); ok {
		return array.Copy(context, ownsFreshValue(context, source), value)
	}
	if structType, ok := isAnonymousStruct(sourceType); ok {
		if ownsFreshValue(context, source) {
			return value, nil
		}
		return anonymousStructCopy(context, source, structType, value)
	}
	_, complexOK := complexvalue.Describe(sourceType)
	_, primitiveOK := primitive(context, sourceType)
	if complexOK ||
		primitiveOK ||
		callableValue(sourceType) ||
		pointerValue(sourceType) ||
		isScalarSlice(context, sourceType) ||
		mapValue(context, sourceType) {
		return api.NewExpressionEmission(
			value.Before(),
			value.Value(),
			value.Requests(),
		)
	}
	typeName, _, ok := namedStruct(sourceType)
	if !ok {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if ownsFreshValue(context, source) {
		return api.NewExpressionEmission(
			value.Before(),
			value.Value(),
			value.Requests(),
		)
	}
	reference, err := context.Names().NamedStructOperation(
		typeName,
		api.NamedStructOperationCopy,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	call, err := namedStructOperationCall(
		context,
		reference.Name(),
		api.NamedStructOperationCopy,
		[]tsgo.Expression{value.Value()},
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.NewExpressionEmission(
		value.Before(),
		call,
		api.CombineRequests(value.Requests(), reference.Requests()),
	)
}

func ownsFreshValue(context api.Context, source ast.Node) bool {
	switch source := source.(type) {
	case *ast.CallExpr, *ast.CompositeLit:
		return true
	case *ast.ParenExpr:
		return ownsFreshValue(context, source.X)
	case *ast.SelectorExpr:
		selection := context.TypesInfo().Selections[source]
		return selection != nil &&
			selection.Kind() == types.FieldVal &&
			!selection.Indirect() &&
			len(selection.Index()) == 1 &&
			ownsFreshValue(context, source.X)
	default:
		return false
	}
}

func expressionType(context api.Context, source ast.Node) types.Type {
	expression, ok := source.(ast.Expr)
	if !ok {
		return nil
	}
	return context.TypesInfo().TypeOf(expression)
}

func (Owner) Assign(
	context api.Context,
	source ast.Node,
	sourceType types.Type,
	target tsgo.Expression,
	value api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	_, definedOK := definedtype.Resolve(sourceType)
	_, arrayOK := arrayvalue.Resolve(context, sourceType)
	_, complexOK := complexvalue.Describe(sourceType)
	_, primitiveOK := primitive(context, sourceType)
	_, _, structOK := namedStruct(sourceType)
	_, anonymousStructOK := isAnonymousStruct(sourceType)
	if !definedOK &&
		!arrayOK &&
		!complexOK &&
		!primitiveOK &&
		!callableValue(sourceType) &&
		!pointerValue(sourceType) &&
		!isScalarSlice(context, sourceType) &&
		!mapValue(context, sourceType) &&
		!anonymousStructOK &&
		!structOK {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	return api.NewExpressionEmission(
		value.Before(),
		context.Factory().BinaryExpression(
			nil,
			target,
			nil,
			context.Factory().BinaryOperatorToken(
				tsgo.BinaryOperatorEqualsToken,
			),
			value.Value(),
		),
		value.Requests(),
	)
}

func (Owner) Equal(
	context api.Context,
	source ast.Node,
	sourceType types.Type,
	left tsgo.Expression,
	right tsgo.Expression,
) (api.ExpressionEmission, error) {
	if defined, ok := definedtype.Resolve(sourceType); ok {
		if defined.Family() == definedtype.FamilyCallable {
			return api.DirectExpression(
				context.Factory().BinaryExpression(
					nil,
					left,
					nil,
					context.Factory().BinaryOperatorToken(
						tsgo.BinaryOperatorEqualsEqualsEqualsToken,
					),
					right,
				),
			), nil
		}
		leftValue := api.DirectExpression(left)
		rightValue := api.DirectExpression(right)
		var err error
		if defined.NilCapable() {
			leftValue, err = defined.Project(context, leftValue)
			if err == nil {
				rightValue, err = defined.Project(context, rightValue)
			}
		} else {
			leftValue = api.DirectExpression(
				defined.Unwrap(context.Factory(), left),
			)
			rightValue = api.DirectExpression(
				defined.Unwrap(context.Factory(), right),
			)
		}
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		result, err := (Owner{}).Equal(
			context.WithRole(api.RoleDefinedValue),
			source,
			defined.Underlying(),
			leftValue.Value(),
			rightValue.Value(),
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		return api.NewExpressionEmission(
			result.Before(),
			result.Value(),
			api.CombineRequests(
				leftValue.Requests(),
				rightValue.Requests(),
				result.Requests(),
			),
		)
	}
	if carrier, ok := complexvalue.Describe(sourceType); ok {
		symbol, valid := complexvalue.EqualSymbol(carrier)
		if !valid {
			return api.ExpressionEmission{}, &api.InvariantError{
				Role:   context.Role(),
				Reason: "complex equality has no runtime symbol",
			}
		}
		return complexvalue.Call(
			context,
			symbol,
			[]tsgo.Expression{left, right},
		)
	}
	if array, ok := arrayvalue.Resolve(context, sourceType); ok {
		return array.Equal(context, source, left, right)
	}
	if structType, ok := isAnonymousStruct(sourceType); ok {
		return anonymousStructEqual(context, source, structType, left, right)
	}
	if _, ok := primitive(context, sourceType); ok {
		return api.DirectExpression(context.Factory().BinaryExpression(
			nil,
			left,
			nil,
			context.Factory().BinaryOperatorToken(
				tsgo.BinaryOperatorEqualsEqualsEqualsToken,
			),
			right,
		)), nil
	}
	if pointerValue(sourceType) {
		reference, err := context.Names().Runtime(
			api.RuntimePointer,
			api.ImportPhaseValue,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		return api.DirectExpression(
			context.Factory().CallExpression(
				context.Factory().PropertyAccessExpression(
					context.Factory().Identifier(reference.Name()),
					nil,
					context.Factory().Identifier(pointerruntime.EqualName),
					tsgo.NodeFlagsNone,
				),
				nil,
				nil,
				[]tsgo.Expression{left, right},
				tsgo.NodeFlagsNone,
			),
			reference.Requests()...,
		), nil
	}
	if callableValue(sourceType) {
		return api.DirectExpression(
			context.Factory().BinaryExpression(
				nil,
				left,
				nil,
				context.Factory().BinaryOperatorToken(
					tsgo.BinaryOperatorEqualsEqualsEqualsToken,
				),
				right,
			),
		), nil
	}
	typeName, _, ok := namedStruct(sourceType)
	if !ok {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	reference, err := context.Names().NamedStructOperation(
		typeName,
		api.NamedStructOperationEqual,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	call, err := namedStructOperationCall(
		context,
		reference.Name(),
		api.NamedStructOperationEqual,
		[]tsgo.Expression{left, right},
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.DirectExpression(
		call,
		reference.Requests()...,
	), nil
}

func primitive(
	context api.Context,
	sourceType types.Type,
) (api.PrimitiveAlias, bool) {
	return basictype.PrimitiveAlias(context.TypesSizes(), sourceType)
}

func callableValue(sourceType types.Type) bool {
	_, ok := callable.Signature(sourceType)
	return ok
}

func pointerValue(sourceType types.Type) bool {
	_, _, ok := pointertype.Resolve(sourceType)
	return ok
}

func mapValue(context api.Context, sourceType types.Type) bool {
	_, ok := maprepresentation.Source(context, sourceType)
	return ok
}

func namedStruct(
	sourceType types.Type,
) (*types.TypeName, *types.Struct, bool) {
	if sourceType == nil {
		return nil, nil, false
	}
	named, ok := types.Unalias(sourceType).(*types.Named)
	if !ok || named.TypeParams().Len() != 0 {
		return nil, nil, false
	}
	structType, ok := named.Underlying().(*types.Struct)
	if !ok || named.Obj() == nil {
		return nil, nil, false
	}
	return named.Obj(), structType, true
}

func namedStructOperationCall(
	context api.Context,
	className string,
	operation api.NamedStructOperation,
	arguments []tsgo.Expression,
) (tsgo.CallExpression, error) {
	memberName, err := api.NamedStructOperationMemberName(operation)
	if err != nil {
		return nil, err
	}
	return context.Factory().CallExpression(
		context.Factory().PropertyAccessExpression(
			context.Factory().Identifier(className),
			nil,
			context.Factory().Identifier(memberName),
			tsgo.NodeFlagsNone,
		),
		nil,
		nil,
		arguments,
		tsgo.NodeFlagsNone,
	), nil
}
