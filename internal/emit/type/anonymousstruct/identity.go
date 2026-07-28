package anonymousstruct

import (
	"github.com/tsoniclang/gotots/internal/emit/api"
)

const targetNameHexLength = 20

func TargetName(artifactKey string) (string, error) {
	if len(artifactKey) < targetNameHexLength {
		return "", &api.NameError{
			Reason: "anonymous-struct artifact key is invalid",
		}
	}
	return "$goStruct_" + artifactKey[:targetNameHexLength], nil
}
