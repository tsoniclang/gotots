package providerboundary

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	rawpointermarker "github.com/tsoniclang/gotots/internal/emit/marker/rawpointer"
	basictype "github.com/tsoniclang/gotots/internal/emit/type/basic"
)

func fromProviderRawPointer(
	context api.Context,
	sourceType types.Type,
	value api.ExpressionEmission,
) (api.ExpressionEmission, bool, bool, error) {
	if !basictype.SupportsUnsafePointer(sourceType) {
		return api.ExpressionEmission{}, false, false, nil
	}
	target, err := rawpointermarker.BindNullable(context, value)
	return target, true, true, err
}

func toProviderRawPointer(
	context api.Context,
	sourceType types.Type,
	_ api.ExpressionEmission,
) (api.ExpressionEmission, bool, bool, error) {
	if !basictype.SupportsUnsafePointer(sourceType) {
		return api.ExpressionEmission{}, false, false, nil
	}
	return api.ExpressionEmission{}, true, false, boundaryInvariant(
		context,
		"provider raw-pointer input requires an exact raw-pointer identity extraction contract",
	)
}
