package maprepresentation

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func Convert(
	context api.Context,
	source ast.Node,
	sourceType types.Type,
	targetType types.Type,
	value api.ExpressionEmission,
) (api.ExpressionEmission, bool, error) {
	sourceMap, sourceOK := Source(context, sourceType)
	targetMap, targetOK := Source(context, targetType)
	if !sourceOK || !targetOK ||
		!types.Identical(sourceMap.Map(), targetMap.Map()) {
		return api.ExpressionEmission{}, false, nil
	}
	var err error
	if sourceMap.Nominal() {
		value, err = sourceMap.ReadReceiver(context, source, value)
		if err != nil {
			return api.ExpressionEmission{}, true, err
		}
	}
	value, err = targetMap.WrapConverted(context, value)
	return value, true, err
}
