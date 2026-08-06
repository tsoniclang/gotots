package representation

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	integervalue "github.com/tsoniclang/gotots/internal/emit/value/integer"
	integerconversion "github.com/tsoniclang/gotots/internal/emit/value/integer/conversion"
)

func transferIntegerRepresentation(
	context api.Context,
	sourceType types.Type,
	targetType types.Type,
	sourceABI api.ScalarABI,
	targetABI api.ScalarABI,
	value api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	if !types.Identical(sourceType, targetType) {
		return value, nil
	}
	carrier, integer := integervalue.Describe(
		context.TypesSizes(),
		sourceType,
	)
	if !integer {
		return value, nil
	}
	source, err := sourceABI.Carrier(carrier.Alias())
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	target, err := targetABI.Carrier(carrier.Alias())
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return integerconversion.ConvertRepresentation(
		context,
		carrier,
		source,
		target,
		value,
	)
}
