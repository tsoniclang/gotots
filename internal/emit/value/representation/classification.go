package representation

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	basictype "github.com/tsoniclang/gotots/internal/emit/type/basic"
	pointertype "github.com/tsoniclang/gotots/internal/emit/type/pointer"
	"github.com/tsoniclang/gotots/internal/emit/value/maprepresentation"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

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
