package providerboundary

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	integervalue "github.com/tsoniclang/gotots/internal/emit/value/integer"
	integerconversion "github.com/tsoniclang/gotots/internal/emit/value/integer/conversion"
)

func fromProviderScalar(
	context api.Context,
	sourceType types.Type,
	value api.ExpressionEmission,
) (api.ExpressionEmission, bool, bool, error) {
	if _, scalar := integervalue.DescribeUnderlying(
		context.TypesSizes(),
		sourceType,
	); !scalar {
		return value, false, false, nil
	}
	model, representation, defined, err := definedScalarRepresentation(
		context,
		sourceType,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, false, err
	}
	if defined && representation != api.DefinedValueRepresentationGeneratedWrapper {
		return value, true, false, nil
	}
	provider, ok := context.ProviderScalarABI()
	if !ok {
		return api.ExpressionEmission{}, true, false, boundaryInvariant(
			context,
			"provider scalar ABI is absent",
		)
	}
	converted, changed, err := convertScalar(
		context,
		sourceType,
		provider,
		context.ScalarABI(),
		value,
	)
	if err != nil || !defined {
		return converted, true, changed, err
	}
	wrapped, err := model.Wrap(context, converted)
	return wrapped, true, err == nil, err
}

func toProviderScalar(
	context api.Context,
	sourceType types.Type,
	value api.ExpressionEmission,
) (api.ExpressionEmission, bool, bool, error) {
	if _, scalar := integervalue.DescribeUnderlying(
		context.TypesSizes(),
		sourceType,
	); !scalar {
		return value, false, false, nil
	}
	model, representation, defined, err := definedScalarRepresentation(
		context,
		sourceType,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, false, err
	}
	if defined && representation != api.DefinedValueRepresentationGeneratedWrapper {
		return value, true, false, nil
	}
	if defined {
		value, err = model.Project(context, value)
		if err != nil {
			return api.ExpressionEmission{}, true, false, err
		}
	}
	provider, ok := context.ProviderScalarABI()
	if !ok {
		return api.ExpressionEmission{}, true, false, boundaryInvariant(
			context,
			"provider scalar ABI is absent",
		)
	}
	converted, changed, err := convertScalar(
		context,
		sourceType,
		context.ScalarABI(),
		provider,
		value,
	)
	return converted, true, changed || defined, err
}

func definedScalarRepresentation(
	context api.Context,
	sourceType types.Type,
) (
	definedtype.Model,
	api.DefinedValueRepresentationKind,
	bool,
	error,
) {
	model, defined := definedtype.ResolveBasic(sourceType)
	if !defined {
		return definedtype.Model{}, api.DefinedValueRepresentationInvalid, false, nil
	}
	representation, err := context.Names().DefinedValueRepresentation(
		model.TypeName(),
	)
	if err != nil {
		return definedtype.Model{}, api.DefinedValueRepresentationInvalid, false, err
	}
	if !representation.Kind().Valid() {
		return definedtype.Model{}, api.DefinedValueRepresentationInvalid, false,
			boundaryInvariant(context, "defined scalar representation is invalid")
	}
	return model, representation.Kind(), true, nil
}

func convertScalar(
	context api.Context,
	sourceType types.Type,
	sourceABI api.ScalarABI,
	targetABI api.ScalarABI,
	value api.ExpressionEmission,
) (api.ExpressionEmission, bool, error) {
	carrier, integer := integervalue.DescribeUnderlying(
		context.TypesSizes(),
		sourceType,
	)
	if !integer {
		return value, false, nil
	}
	source, err := sourceABI.Carrier(carrier.Alias())
	if err != nil {
		return api.ExpressionEmission{}, false, err
	}
	target, err := targetABI.Carrier(carrier.Alias())
	if err != nil {
		return api.ExpressionEmission{}, false, err
	}
	if source == target {
		return value, false, nil
	}
	converted, err := integerconversion.ConvertRepresentation(
		context,
		carrier,
		source,
		target,
		value,
	)
	return converted, err == nil, err
}
