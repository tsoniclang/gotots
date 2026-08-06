package unsafecodec

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func (b *builder) operations(
	sourceType types.Type,
	storage tsgo.TypeNode,
) (tsgo.Expression, tsgo.Expression, error) {
	underlying := types.Unalias(sourceType).Underlying()
	switch selected := underlying.(type) {
	case *types.Basic:
		return b.basicOperations(sourceType, selected, storage)
	case *types.Array:
		return b.arrayOperations(selected, storage)
	case *types.Struct:
		return b.structOperations(sourceType, selected, storage)
	case *types.Pointer:
		return b.pointerOperations(selected, storage)
	case *types.Slice:
		return b.sliceOperations(selected, storage)
	default:
		return nil, nil, &api.GeneratedArtifactShapeError{
			Reason: "unsafe-codec source layout is unsupported",
		}
	}
}
