package structconstruction

import (
	"fmt"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func FieldName(
	names api.Names,
	field *types.Var,
	ordinal int,
) (string, error) {
	if names == nil || field == nil || ordinal < 0 {
		return "", &api.NameError{Reason: "struct construction field is invalid"}
	}
	if field.Name() == "_" {
		return fmt.Sprintf("$blank%d", ordinal), nil
	}
	return names.Member(field)
}
