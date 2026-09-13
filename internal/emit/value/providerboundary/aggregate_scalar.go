package providerboundary

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	integervalue "github.com/tsoniclang/gotots/internal/emit/value/integer"
)

func validateAggregateScalarABI(context api.Context, sourceType types.Type) error {
	switch types.Unalias(sourceType).(type) {
	case *types.Array, *types.Struct:
	default:
		return nil
	}
	mismatch, err := aggregateScalarMismatch(context, sourceType)
	if err != nil || !mismatch {
		return err
	}
	return &api.UnsupportedError{
		Category: api.CategoryExpression,
		Role:     context.Role(),
		Construct: "provider aggregate " + types.TypeString(sourceType, nil) +
			" requires an alias-preserving scalar-ABI projection",
	}
}

func aggregateScalarMismatch(context api.Context, sourceType types.Type) (bool, error) {
	sourceType = types.Unalias(sourceType)
	if named, ok := sourceType.(*types.Named); ok {
		providerOwned, err := context.Names().ProviderOwnedDeclaration(named.Origin().Obj())
		if err != nil || providerOwned {
			return false, err
		}
		sourceType = named.Underlying()
	}
	if integer, ok := integervalue.DescribeUnderlying(context.TypesSizes(), sourceType); ok {
		providerABI, present := context.ProviderScalarABI()
		if !present {
			return false, boundaryInvariant(context, "provider scalar ABI is absent")
		}
		provider, err := providerABI.Carrier(integer.Alias())
		if err != nil {
			return false, err
		}
		product, err := context.ScalarABI().Carrier(integer.Alias())
		return provider != product, err
	}
	switch selected := sourceType.(type) {
	case *types.Array:
		return aggregateScalarMismatch(context, selected.Elem())
	case *types.Struct:
		for index := range selected.NumFields() {
			mismatch, err := aggregateScalarMismatch(context, selected.Field(index).Type())
			if err != nil || mismatch {
				return mismatch, err
			}
		}
	}
	return false, nil
}
