package naming

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func (r *Registry) semanticPackageToken(
	sourcePackage *types.Package,
) (string, error) {
	if r == nil || sourcePackage == nil {
		return "", &api.NameError{
			Reason: "generated-artifact package token owner is invalid",
		}
	}
	qualifier := r.ImportQualifier(sourcePackage)
	if qualifier == "" {
		return "", &api.NameError{
			Name:   sourcePackage.Path(),
			Reason: "generated-artifact package token is absent",
		}
	}
	return qualifier, nil
}
