package defined

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

const ValueTypeParameter = "$Value"

func RequiresValueFacet(sourceType types.Type) bool {
	named, ok := types.Unalias(sourceType).(*types.Named)
	if !ok {
		return false
	}
	return namedRequiresValueFacet(
		named.Origin(),
		make(map[*types.Named]struct{}),
	)
}

func namedRequiresValueFacet(
	source *types.Named,
	seen map[*types.Named]struct{},
) bool {
	if source == nil {
		return false
	}
	source = source.Origin()
	if _, visited := seen[source]; visited {
		return false
	}
	seen[source] = struct{}{}
	return representedTypeVariesByProfile(source.Underlying(), seen)
}

func representedTypeVariesByProfile(
	sourceType types.Type,
	seen map[*types.Named]struct{},
) bool {
	if sourceType == nil {
		return false
	}
	sourceType = types.Unalias(sourceType)
	switch sourceType := sourceType.(type) {
	case *types.Signature:
		return true
	case *types.Array:
		return representedTypeVariesByProfile(sourceType.Elem(), seen)
	case *types.Slice:
		return representedTypeVariesByProfile(sourceType.Elem(), seen)
	case *types.Pointer:
		return representedTypeVariesByProfile(sourceType.Elem(), seen)
	case *types.Map:
		return representedTypeVariesByProfile(sourceType.Key(), seen) ||
			representedTypeVariesByProfile(sourceType.Elem(), seen)
	case *types.Chan:
		return representedTypeVariesByProfile(sourceType.Elem(), seen)
	case *types.Named:
		for index := range sourceType.TypeArgs().Len() {
			if representedTypeVariesByProfile(
				sourceType.TypeArgs().At(index),
				seen,
			) {
				return true
			}
		}
		if _, ok := Resolve(sourceType); !ok {
			return false
		}
		return namedRequiresValueFacet(sourceType, seen)
	case *types.Tuple:
		for index := range sourceType.Len() {
			if representedTypeVariesByProfile(
				sourceType.At(index).Type(),
				seen,
			) {
				return true
			}
		}
		return false
	default:
		return false
	}
}

func ValueTypeParameterDeclaration(
	factory tsgo.Factory,
	defaultType tsgo.TypeNode,
) tsgo.TypeParameterDeclaration {
	return factory.TypeParameterDeclaration(
		nil,
		factory.Identifier(ValueTypeParameter),
		nil,
		nil,
		defaultType,
	)
}

func ValueTypeParameterReference(
	factory tsgo.Factory,
) tsgo.TypeNode {
	return factory.TypeReferenceNode(
		factory.Identifier(ValueTypeParameter),
		nil,
	)
}
