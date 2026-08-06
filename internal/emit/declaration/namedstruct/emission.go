package namedstruct

import (
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func declarationEmission(
	declarations []tsgo.Statement,
	requests []api.RootRequest,
	className string,
	storageDemanded bool,
	moduleExport bool,
) (api.DeclarationEmission, error) {
	if !storageDemanded || !moduleExport {
		return api.NewDeclarationEmission(declarations, requests)
	}
	return api.NewDeclarationEmissionWithAdditionalPackageBindings(
		declarations,
		requests,
		[]string{className + api.StructStorageTypeSuffix},
	)
}
