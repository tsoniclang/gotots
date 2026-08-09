package representation

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	basictype "github.com/tsoniclang/gotots/internal/emit/type/basic"
	channeltype "github.com/tsoniclang/gotots/internal/emit/type/channel"
	pointertype "github.com/tsoniclang/gotots/internal/emit/type/pointer"
	"github.com/tsoniclang/gotots/internal/emit/value/maprepresentation"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func representedAtDestination(
	actualType types.Type,
	destinationType types.Type,
) bool {
	basic, basicOK := types.Unalias(actualType).(*types.Basic)
	return basicOK && basic.Info()&types.IsUntyped != 0
}

func ownsFreshValue(context api.Context, source ast.Node) bool {
	switch source := source.(type) {
	case *ast.CallExpr, *ast.CompositeLit:
		return true
	case *ast.IndexExpr:
		sourceType := context.TypesInfo().TypeOf(source.X)
		if sourceType == nil {
			return false
		}
		_, ok := types.Unalias(sourceType).Underlying().(*types.Map)
		return ok
	case *ast.ParenExpr:
		return ownsFreshValue(context, source.X)
	case *ast.SelectorExpr:
		selection := context.TypesInfo().SelectionOf(source)
		return selection != nil &&
			selection.Kind() == types.FieldVal &&
			!selection.Indirect() &&
			ownsFreshValue(context, source.X)
	default:
		return false
	}
}

func primitive(
	context api.Context,
	sourceType types.Type,
) (api.PrimitiveAlias, bool) {
	return basictype.PrimitiveAlias(context.TypesSizes(), sourceType)
}

func callableValue(sourceType types.Type) bool {
	return callable.IsValue(sourceType)
}

func pointerValue(sourceType types.Type) bool {
	_, _, ok := pointertype.Resolve(sourceType)
	return ok
}

func rawPointerValue(sourceType types.Type) bool {
	return basictype.SupportsUnsafePointer(sourceType)
}

func channelValue(sourceType types.Type) bool {
	_, ok := channeltype.Resolve(sourceType)
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
	if !ok ||
		named.Obj() == nil ||
		(named.TypeParams().Len() != 0 &&
			named.TypeArgs().Len() != named.TypeParams().Len()) {
		return nil, nil, false
	}
	structType, ok := named.Underlying().(*types.Struct)
	if !ok {
		return nil, nil, false
	}
	return named.Origin().Obj(), structType, true
}

func staticStructOperationCall(
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
