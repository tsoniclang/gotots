package stagecheck

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
)

func independentSupportBindingRole(
	role catalog.Role,
) identity.SemanticBindingRole {
	switch role {
	case catalog.RoleTypeParameters:
		return identity.SemanticBindingTypeParameter
	case catalog.RoleParameters:
		return identity.SemanticBindingParameter
	case catalog.RoleResults:
		return identity.SemanticBindingResult
	case catalog.RoleReceiver:
		return identity.SemanticBindingReceiver
	case catalog.RoleImportAlias:
		return identity.SemanticBindingImport
	case catalog.RoleRangeKey, catalog.RoleRangeValue:
		return identity.SemanticBindingRange
	case catalog.RoleLabelDeclaration:
		return identity.SemanticBindingLabel
	default:
		return identity.SemanticBindingLocal
	}
}

func independentLexicalBinding(object types.Object) bool {
	if object == nil || object.Name() == "_" {
		return false
	}
	switch object := object.(type) {
	case *types.PkgName, *types.Label:
		return true
	case *types.TypeName:
		_, typeParameter := object.Type().(*types.TypeParam)
		return typeParameter
	case *types.Var:
		return !object.IsField() &&
			(object.Pkg() == nil ||
				object.Parent() != object.Pkg().Scope())
	default:
		return false
	}
}

func independentPackageName(object types.Object) bool {
	_, packageName := object.(*types.PkgName)
	return packageName
}

func independentTypelessBinding(
	role identity.SemanticBindingRole,
) bool {
	return role == identity.SemanticBindingImport ||
		role == identity.SemanticBindingLabel
}

func independentImplicitBindingRole(
	node ast.Node,
	object types.Object,
	role identity.SemanticBindingRole,
) identity.SemanticBindingRole {
	switch object.(type) {
	case *types.PkgName:
		return identity.SemanticBindingImport
	case *types.Var:
		if _, field := node.(*ast.Field); !field {
			return identity.SemanticBindingTypeSwitch
		}
	}
	return role
}
