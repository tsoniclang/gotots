package generic

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func Emit(
	context api.Context,
	declaration *ast.GenDecl,
	typeName *types.TypeName,
	requirements []api.DeclarationRequirement,
) (api.DeclarationEmission, bool, error) {
	spec, ok := sourceSpec(context, declaration, typeName)
	if !ok {
		return api.DeclarationEmission{}, false, nil
	}
	named, namedOK := types.Unalias(typeName.Type()).(*types.Named)
	if namedOK {
		if constraint, interfaceOK := named.Underlying().(*types.Interface); interfaceOK &&
			!constraint.Complete().IsMethodSet() {
			if len(requirements) != 0 ||
				spec.TypeParams != nil ||
				spec.Assign.IsValid() {
				return api.DeclarationEmission{}, true,
					api.Unsupported(
						context,
						api.CategoryDeclaration,
						declaration,
					)
			}
			return api.CoverageOnlyDeclarationEmission(), true, nil
		}
	}
	if spec.TypeParams == nil {
		return api.DeclarationEmission{}, false, nil
	}
	return api.DeclarationEmission{}, true,
		api.Unsupported(context, api.CategoryDeclaration, declaration)
}

func sourceSpec(
	context api.Context,
	declaration *ast.GenDecl,
	typeName *types.TypeName,
) (*ast.TypeSpec, bool) {
	if declaration == nil || typeName == nil {
		return nil, false
	}
	for _, candidate := range declaration.Specs {
		spec, ok := candidate.(*ast.TypeSpec)
		if ok && context.TypesInfo().Defs[spec.Name] == typeName {
			return spec, true
		}
	}
	return nil, false
}
