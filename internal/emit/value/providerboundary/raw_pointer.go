package providerboundary

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	basictype "github.com/tsoniclang/gotots/internal/emit/type/basic"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
)

func fromProviderRawPointer(
	context api.Context,
	sourceType types.Type,
	value api.ExpressionEmission,
) (api.ExpressionEmission, bool, bool, error) {
	if !basictype.SupportsUnsafePointer(sourceType) {
		return api.ExpressionEmission{}, false, false, nil
	}
	if model, defined := definedtype.ResolveBasic(sourceType); defined {
		wrapped, err := model.Wrap(context, value)
		return wrapped, true, err == nil, err
	}
	return value, true, false, nil
}

func toProviderRawPointer(
	context api.Context,
	sourceType types.Type,
	value api.ExpressionEmission,
) (api.ExpressionEmission, bool, bool, error) {
	if !basictype.SupportsUnsafePointer(sourceType) {
		return api.ExpressionEmission{}, false, false, nil
	}
	if model, defined := definedtype.ResolveBasic(sourceType); defined {
		projected, err := model.Project(context, value)
		return projected, true, err == nil, err
	}
	return value, true, false, nil
}
