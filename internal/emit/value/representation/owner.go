package representation

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	mapruntime "github.com/tsoniclang/gotots/internal/emit/runtime/map"
	basictype "github.com/tsoniclang/gotots/internal/emit/type/basic"
	pointertype "github.com/tsoniclang/gotots/internal/emit/type/pointer"
	arrayvalue "github.com/tsoniclang/gotots/internal/emit/value/array"
	"github.com/tsoniclang/gotots/internal/emit/value/maprepresentation"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type Owner struct{}

func (Owner) RequiresCustomEquality(
	context api.Context,
	sourceType types.Type,
) bool {
	if _, ok := arrayvalue.Resolve(context, sourceType); ok {
		return true
	}
	if scalarPointer(context, sourceType) {
		return true
	}
	_, _, ok := namedStruct(sourceType)
	return ok
}

func (Owner) RequiresExplicitType(
	context api.Context,
	sourceType types.Type,
) bool {
	return scalarPointer(context, sourceType)
}

func (Owner) Zero(
	context api.Context,
	source ast.Node,
	sourceType types.Type,
) (api.ExpressionEmission, error) {
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
	if alias, ok := primitive(context, sourceType); ok {
		var literal tsgo.Expression
		switch alias {
		case api.PrimitiveBool:
			literal = context.Factory().FalseLiteral()
		case api.PrimitiveString:
			literal = context.Factory().StringLiteral("", tsgo.TokenFlagsNone)
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
	if _, _, ok := pointertype.Scalar(context.TypesSizes(), sourceType); ok {
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
	if array, ok := arrayvalue.Resolve(context, sourceType); ok {
		return array.Copy(context, ownsFreshValue(context, source), value)
	}
	if _, ok := primitive(context, sourceType); ok ||
		callableValue(sourceType) ||
		scalarPointer(context, sourceType) ||
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

func (Owner) Assign(
	context api.Context,
	source ast.Node,
	sourceType types.Type,
	target tsgo.Expression,
	value api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	_, arrayOK := arrayvalue.Resolve(context, sourceType)
	_, primitiveOK := primitive(context, sourceType)
	_, _, structOK := namedStruct(sourceType)
	if !arrayOK &&
		!primitiveOK &&
		!callableValue(sourceType) &&
		!scalarPointer(context, sourceType) &&
		!isScalarSlice(context, sourceType) &&
		!mapValue(context, sourceType) &&
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
	if array, ok := arrayvalue.Resolve(context, sourceType); ok {
		return array.Equal(context, left, right), nil
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
	if scalarPointer(context, sourceType) {
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

func scalarPointer(context api.Context, sourceType types.Type) bool {
	_, _, ok := pointertype.Scalar(context.TypesSizes(), sourceType)
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
