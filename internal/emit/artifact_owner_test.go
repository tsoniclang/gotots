package emit

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func sourceArtifactOwner(object types.Object) api.ArtifactOwner {
	return api.MustSourceArtifactOwner(object)
}
